//go:build integration

// Package integration rejoue les scénarios du quickstart contre un vrai démon Docker,
// au travers de l'intégration shell. Lancement : go test -tags integration ./tests/integration/...
package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// harness compile docker-telemach et prépare un environnement isolé.
type harness struct {
	bin, state string
}

func newHarness(t *testing.T) harness {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker non installé")
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("démon Docker indisponible")
	}
	dir := t.TempDir()
	h := harness{bin: filepath.Join(dir, "bin"), state: filepath.Join(dir, "state")}
	build := exec.Command("go", "build", "-o", filepath.Join(h.bin, "docker-telemach"), "../../cmd/docker-telemach")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("compilation : %v\n%s", err, out)
	}
	return h
}

// shell exécute body dans bash après activation de l'intégration TELEMACH.
func (h harness) shell(t *testing.T, body string, extraEnv ...string) (stdout, stderr string) {
	t.Helper()
	// Comme une vraie installation : le binaire n'est PAS dans le PATH (LL-006).
	cmd := exec.Command("bash", "-c", `eval "$("$TELEMACH" shell-init bash)"`+"\n"+body)
	cmd.Env = append(os.Environ(),
		"TELEMACH="+filepath.Join(h.bin, "docker-telemach"),
		"XDG_STATE_HOME="+h.state,
		"NO_COLOR=1",
	)
	cmd.Env = append(cmd.Env, extraEnv...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	_ = cmd.Run()
	return out.String(), errb.String()
}

func docker(t *testing.T, args ...string) {
	t.Helper()
	if out, err := exec.Command("docker", args...).CombinedOutput(); err != nil {
		t.Fatalf("docker %v : %v\n%s", args, err, out)
	}
}

// V1 — port déjà utilisé : explication en français et code 125 préservé.
func TestV1PortAllocated(t *testing.T) {
	h := newHarness(t)
	docker(t, "run", "-d", "--name", "telemach-it-web", "-p", "18090:8080", "nginxinc/nginx-unprivileged:1.30-alpine")
	t.Cleanup(func() { _ = exec.Command("docker", "rm", "-f", "telemach-it-web").Run() })

	stdout, stderr := h.shell(t, `docker run --rm -p 18090:8080 nginxinc/nginx-unprivileged:1.30-alpine; echo "rc=$?"`)
	if !strings.Contains(stdout, "rc=125") {
		t.Errorf("code 125 attendu : %q", stdout)
	}
	for _, want := range []string{"Le port 18090 de ta machine est déjà utilisé", "port publishing", "docker run -p 18091:8080"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("explication sans %q :\n%s", want, stderr)
		}
	}
}

// V2 — image mal orthographiée : suggestion de « nginx ».
func TestV2ImageTypo(t *testing.T) {
	h := newHarness(t)
	_, stderr := h.shell(t, `docker run --rm ngnix`)
	for _, want := range []string{"L'image « ngnix » est introuvable", "nginx"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("explication sans %q :\n%s", want, stderr)
		}
	}
}

// V4 — erreur applicative : l'origine est le programme du conteneur, pas Docker.
func TestV4ApplicationError(t *testing.T) {
	h := newHarness(t)
	stdout, stderr := h.shell(t, `docker run --rm --user 1000:1000 alpine:3.24 sh -c 'exit 3'; echo "rc=$?"`)
	if !strings.Contains(stdout, "rc=3") {
		t.Errorf("code 3 attendu : %q", stdout)
	}
	if !strings.Contains(stderr, "pas de Docker") || !strings.Contains(stderr, "docker logs") {
		t.Errorf("l'erreur doit être attribuée au programme du conteneur :\n%s", stderr)
	}
}

// Issue #9 — option placée après l'image : la commande réussit, l'avertissement s'affiche,
// et la commande corrigée publie réellement le port.
func TestIssue9OptionAfterImage(t *testing.T) {
	h := newHarness(t)
	const name = "telemach-it-issue9"
	t.Cleanup(func() { _ = exec.Command("docker", "rm", "-f", name).Run() })

	stdout, stderr := h.shell(t, `docker run -d --name `+name+` nginxinc/nginx-unprivileged:1.30-alpine -p 18092:8080; echo "rc=$?"`)
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 || len(lines[0]) != 64 || lines[1] != "rc=0" {
		t.Errorf("stdout = identifiant du conteneur puis rc=0 attendu : %q", stdout)
	}
	fixed := "docker run -d --name " + name + " -p 18092:8080 nginxinc/nginx-unprivileged:1.30-alpine"
	for _, want := range []string{"Option Docker placée après l'image", fixed} {
		if !strings.Contains(stderr, want) {
			t.Errorf("avertissement sans %q :\n%s", want, stderr)
		}
	}

	docker(t, "rm", "-f", name)
	docker(t, strings.Fields(fixed)[1:]...)
	out, err := exec.Command("docker", "port", name, "8080").CombinedOutput()
	if err != nil || !strings.Contains(string(out), "18092") {
		t.Errorf("port 18092 non publié par la commande corrigée : %v\n%s", err, out)
	}
}

// V4 — pièges silencieux (US2) : la commande réussit, le piège est signalé après coup.
func TestV4SilentPitfalls(t *testing.T) {
	h := newHarness(t)
	work := t.TempDir()
	t.Cleanup(func() { _ = exec.Command("docker", "volume", "rm", "-f", "data").Run() })

	cases := []struct {
		name, body string
		want       []string
	}{
		{
			"volume nommé", `cd "$WORK" && mkdir -p data && docker run --rm --user 1000:1000 -v data:/d alpine:3.24 ls /d`,
			[]string{"désigne un volume nommé", "docker run --rm --user 1000:1000 -v ./data:/d alpine:3.24 ls /d"},
		},
		{
			"port aléatoire", `docker run --rm --user 1000:1000 -p 80 alpine:3.24 true`,
			[]string{"Port de la machine choisi au hasard", "docker port <conteneur>"},
		},
		{
			"variable non définie", `unset DB_HOST; docker run --rm --user 1000:1000 -e DB_HOST alpine:3.24 true`,
			[]string{"Variable DB_HOST absente"},
		},
		{
			"réseau host", `docker run --rm --user 1000:1000 --network host -p 8080:80 alpine:3.24 true`,
			[]string{"Ports ignorés en réseau host"},
		},
		{
			"secret en clair", `docker run --rm --user 1000:1000 -e DB_PASSWORD=Secret123 alpine:3.24 true`,
			[]string{"Secret écrit en clair", "DB_PASSWORD"},
		},
		{
			"privilèges", `docker run --rm --user 1000:1000 --privileged alpine:3.24 true`,
			[]string{"Privilèges excessifs", "Anti-pattern"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stdout, stderr := h.shell(t, c.body+`; echo "rc=$?"`, "WORK="+work)
			if !strings.Contains(stdout, "rc=0") {
				t.Errorf("la commande doit réussir : %q\n%s", stdout, stderr)
			}
			for _, want := range c.want {
				if !strings.Contains(stderr, want) {
					t.Errorf("sortie sans %q :\n%s", want, stderr)
				}
			}
			if strings.Contains(stderr, "Secret123") {
				t.Errorf("le secret ne doit jamais être affiché :\n%s", stderr)
			}
		})
	}

	// SC-001 : les commandes corrigées produisent l'effet voulu.
	if err := os.WriteFile(filepath.Join(work, "data", "message.txt"), []byte("bonjour\n"), 0o644); err != nil { // #nosec G306 -- lu par l'utilisateur 1000 du conteneur
		t.Fatal(err)
	}
	out, _ := h.shell(t, `cd "$WORK" && docker run --rm --user 1000:1000 -v ./data:/d alpine:3.24 ls /d`, "WORK="+work)
	if !strings.Contains(out, "message.txt") {
		t.Errorf("le dossier local doit être monté par la commande corrigée : %q", out)
	}
	out, _ = h.shell(t, `docker run --rm --user 1000:1000 -e DB_HOST=db alpine:3.24 printenv DB_HOST`)
	if strings.TrimSpace(out) != "db" {
		t.Errorf("la variable doit être présente avec la commande corrigée : %q", out)
	}

	// Le secret n'est écrit nulle part dans l'état de TELEMACH.
	_ = filepath.WalkDir(h.state, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if data, rerr := os.ReadFile(path); rerr == nil && bytes.Contains(data, []byte("Secret123")) { // #nosec G304 -- dossier de test
				t.Errorf("secret écrit dans %s", path)
			}
		}
		return nil
	})

	// Image sans tag : information la première fois seulement (24 h par image).
	_, first := h.shell(t, `docker run --rm --user 1000:1000 alpine true`)
	_, second := h.shell(t, `docker run --rm --user 1000:1000 alpine true`)
	if !strings.Contains(first, "Image sans tag explicite") || strings.Contains(second, "Image sans tag explicite") {
		t.Errorf("information attendue une seule fois :\n--- 1 ---\n%s\n--- 2 ---\n%s", first, second)
	}
}

// SC-003 — délai ajouté par lint après une commande réussie : p95 < 50 ms sur un poste de
// développement, < 100 ms en CI (runners plus lents et variables).
func TestSC003LintLatency(t *testing.T) {
	h := newHarness(t)
	limit := 50 * time.Millisecond
	if os.Getenv("CI") != "" {
		limit = 100 * time.Millisecond
	}
	bin := filepath.Join(h.bin, "docker-telemach")
	for _, c := range []struct {
		name string
		args []string
	}{
		{"avec piège", []string{"lint", "--after", "--", "run", "-d", "nginx", "-p", "8090:80"}},
		{"sans piège", []string{"lint", "--after", "--", "run", "-d", "-p", "8090:80", "nginx:1.30-alpine"}},
	} {
		durations := make([]time.Duration, 40)
		for i := range durations {
			cmd := exec.Command(bin, c.args...) // #nosec G204 -- binaire compilé par le test
			cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+t.TempDir(), "NO_COLOR=1")
			start := time.Now()
			_ = cmd.Run() // code 1 attendu avec piège
			durations[i] = time.Since(start)
		}
		sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
		p95 := durations[len(durations)*95/100-1]
		t.Logf("%s : médiane %v, p95 %v (limite %v)", c.name, durations[len(durations)/2], p95, limit)
		if p95 > limit {
			t.Errorf("%s : p95 %v au-delà de %v", c.name, p95, limit)
		}
	}
}
