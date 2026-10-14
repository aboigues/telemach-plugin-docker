//go:build integration

// Package integration rejoue les scénarios du quickstart contre un vrai démon Docker,
// au travers de l'intégration shell. Lancement : go test -tags integration ./tests/integration/...
package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
	cmd := exec.Command("bash", "-c", `eval "$(docker-telemach shell-init bash)"`+"\n"+body)
	cmd.Env = append(os.Environ(),
		"PATH="+h.bin+string(os.PathListSeparator)+os.Getenv("PATH"),
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
	docker(t, "run", "-d", "--name", "telemach-it-web", "-p", "18090:8080", "nginxinc/nginx-unprivileged:1.29-alpine")
	t.Cleanup(func() { _ = exec.Command("docker", "rm", "-f", "telemach-it-web").Run() })

	stdout, stderr := h.shell(t, `docker run --rm -p 18090:8080 nginxinc/nginx-unprivileged:1.29-alpine; echo "rc=$?"`)
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
	stdout, stderr := h.shell(t, `docker run --rm --user 1000:1000 alpine:3.22 sh -c 'exit 3'; echo "rc=$?"`)
	if !strings.Contains(stdout, "rc=3") {
		t.Errorf("code 3 attendu : %q", stdout)
	}
	if !strings.Contains(stderr, "pas de Docker") || !strings.Contains(stderr, "docker logs") {
		t.Errorf("l'erreur doit être attribuée au programme du conteneur :\n%s", stderr)
	}
}
