//go:build integration

package integration

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// ps lance « docker-telemach ps » comme le ferait la CLI Docker (sans l'intégration shell).
func (h harness) ps(t *testing.T, args ...string) (stderr string, code int) {
	t.Helper()
	cmd := exec.Command(filepath.Join(h.bin, "docker-telemach"), append([]string{"ps", "--width", "200"}, args...)...) // #nosec G204 -- binaire compilé par le test
	cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+h.state, "NO_COLOR=1")
	var outb, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &outb, &errb
	err := cmd.Run()
	if outb.Len() > 0 {
		errb.WriteString("\n[stdout] " + outb.String())
	}
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return errb.String(), code
}

// snapshot relève l'état observable de tous les conteneurs : statut, code, redémarrages,
// empreinte du journal (SC-002).
func snapshot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("docker", "ps", "-a", "--no-trunc", "--format", "{{.ID}}").Output()
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, id := range strings.Fields(string(out)) {
		st, _ := exec.Command("docker", "inspect", "--format", "{{.State.Status}} {{.State.ExitCode}} {{.RestartCount}} {{.State.FinishedAt}}", id).Output()
		logs, _ := exec.Command("docker", "logs", id).CombinedOutput()
		fmt.Fprintf(&b, "%s %s %x\n", id, strings.TrimSpace(string(st)), sha256.Sum256(logs))
	}
	return b.String()
}

// launch lance un conteneur en arrière-plan et attend son arrêt (ou 3 s s'il reste créé).
func launch(t *testing.T, name string, args ...string) {
	t.Helper()
	t.Cleanup(func() { _ = exec.Command("docker", "rm", "-f", name).Run() })
	_ = exec.Command("docker", append([]string{"run", "-d", "--name", name}, args...)...).Run()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		st, _ := exec.Command("docker", "inspect", "--format", "{{.State.Status}}", name).Output()
		if s := strings.TrimSpace(string(st)); s == "exited" || s == "created" {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("%s ne s'est pas arrêté", name)
}

// Feature 004 — quickstart V1 (partie ps), V3 et V6, sur un vrai démon.
func TestExitedScenarios(t *testing.T) {
	h := newHarness(t)
	u := []string{"--user", "1000:1000"}
	launch(t, "telemach-it-pg", "postgres:18-alpine")
	launch(t, "telemach-it-127", append(u, "alpine:3.24", "commande-inexistante")...)
	launch(t, "telemach-it-oom", append(u, "--memory", "8m", "alpine:3.24", "sh", "-c", "x=a; while :; do x=$x$x; done")...)
	launch(t, "telemach-it-1", append(u, "alpine:3.24", "sh", "-c", "echo démarrage; exit 1")...)
	launch(t, "telemach-it-secret", append(u, "-e", "API_TOKEN=Secret123", "alpine:3.24", "sh", "-c", "echo token=Secret123; exit 1")...)

	before := snapshot(t)
	cases := []struct {
		name string
		want []string
	}{
		{"telemach-it-pg", []string{"exige la variable POSTGRES_PASSWORD", "--env-file", "Fin du journal"}},
		{"telemach-it-127", []string{"n'a pas pu démarrer", "commande-inexistante est introuvable"}},
		{"telemach-it-oom", []string{"faute de mémoire (limite : 8 Mo)"}},
		{"telemach-it-1", []string{"code 1", "démarrage", "docker logs telemach-it-1"}},
		{"telemach-it-secret", []string{"code 1"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, code := h.ps(t, c.name)
			if code != 0 {
				t.Errorf("code = %d\n%s", code, got)
			}
			for _, w := range c.want {
				if !strings.Contains(got, w) {
					t.Errorf("%q absent :\n%s", w, got)
				}
			}
			if strings.Contains(got, "Secret123") {
				t.Errorf("secret affiché (SC-003) :\n%s", got)
			}
		})
	}
	// Sans argument : le dernier arrêté (le secret, lancé en dernier), et la liste des autres.
	got, code := h.ps(t)
	if code != 0 || !strings.Contains(got, "telemach-it-secret") || !strings.Contains(got, "Autres conteneurs arrêtés en erreur") {
		t.Errorf("ps sans argument (code %d) :\n%s", code, got)
	}
	if got, code := h.ps(t, "telemach-it-inconnu"); code != 1 || !strings.Contains(got, "Aucun conteneur") {
		t.Errorf("conteneur introuvable : code %d\n%s", code, got)
	}

	if after := snapshot(t); after != before {
		t.Errorf("l'état des conteneurs a changé (SC-002) :\n--- avant ---\n%s--- après ---\n%s", before, after)
	}
	_ = filepath.Walk(h.state, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			if data, rerr := os.ReadFile(p); rerr == nil && bytes.Contains(data, []byte("Secret123")) { // #nosec G304 -- dossier de test
				t.Errorf("secret écrit dans %s (SC-003)", p)
			}
		}
		return nil
	})
}

// SC-004 — « ps » en moins de 2 s au p95 (4 s en CI, runners plus lents) avec 200 conteneurs
// arrêtés, dont 20 en erreur, et un journal de 120 000 lignes (LL-012).
func TestSC004PSLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("mesure longue")
	}
	h := newHarness(t)
	const prefix = "telemach-sc004-"
	// Nettoyage installé avant la création (LL-003).
	cleanup := func() {
		out, _ := exec.Command("docker", "ps", "-aq", "--filter", "name=^"+prefix).Output()
		if ids := strings.Fields(string(out)); len(ids) > 0 {
			_ = exec.Command("docker", append([]string{"rm", "-f"}, ids...)...).Run()
		}
	}
	cleanup()
	t.Cleanup(cleanup)

	jobs := make(chan []string)
	done := make(chan struct{})
	for w := 0; w < 8; w++ {
		go func() {
			for args := range jobs {
				_ = exec.Command("docker", args...).Run()
			}
			done <- struct{}{}
		}()
	}
	u := []string{"--user", "1000:1000"}
	for i := 0; i < 160; i++ { // simples « docker create » (state created, sans erreur)
		jobs <- append([]string{"create", "--name", fmt.Sprintf("%screated-%03d", prefix, i)}, append(u, "alpine:3.24", "true")...)
	}
	for i := 0; i < 20; i++ { // arrêtés sans erreur
		jobs <- append([]string{"run", "--name", fmt.Sprintf("%sok-%02d", prefix, i)}, append(u, "alpine:3.24", "true")...)
	}
	for i := 0; i < 19; i++ { // arrêtés en erreur
		jobs <- append([]string{"run", "--name", fmt.Sprintf("%sko-%02d", prefix, i)}, append(u, "alpine:3.24", "sh", "-c", "exit 1")...)
	}
	close(jobs)
	for w := 0; w < 8; w++ {
		<-done
	}
	// Le dernier arrêté : un journal de 120 000 lignes.
	_ = exec.Command("docker", append([]string{"run", "--name", prefix + "gros-journal"}, append(u, "alpine:3.24", "sh", "-c",
		`i=0; while [ $i -lt 120000 ]; do echo "ligne $i"; i=$((i+1)); done; exit 1`)...)...).Run()

	// Mise en route : laisser le démon absorber la création en rafale (un apprenant ne lance
	// pas ps à la seconde où 200 conteneurs viennent d'être créés), deux exécutions à vide.
	time.Sleep(20 * time.Second)
	_, _ = h.ps(t)
	_, _ = h.ps(t)
	durations := make([]time.Duration, 20)
	for i := range durations {
		start := time.Now()
		out, code := h.ps(t)
		durations[i] = time.Since(start)
		if code != 0 || !strings.Contains(out, prefix+"gros-journal") || !strings.Contains(out, "Autres conteneurs arrêtés en erreur") {
			t.Fatalf("analyse inattendue (code %d, %v, %d octets) :\n%q", code, durations[i], len(out), out)
		}
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p95 := durations[len(durations)*95/100-1]
	limit := 2 * time.Second
	if os.Getenv("CI") != "" {
		limit = 4 * time.Second
	}
	t.Logf("ps : médiane %v, p95 %v (limite %v)", durations[len(durations)/2], p95, limit)
	if p95 > limit {
		t.Errorf("SC-004 : p95 %v au-delà de %v", p95, limit)
	}
}
