//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Feature 004, US2 — quickstart V1 (signal), V2 et V5 : « watch » appelé comme par le script
// shell, après de vrais lancements ; un docker intercalé dans le PATH journalise les lectures.
func TestWatchRealDaemon(t *testing.T) {
	h := newHarness(t)
	real, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("docker non installé")
	}
	wrapDir := t.TempDir()
	log := filepath.Join(wrapDir, "calls.log")
	wrapper := "#!/bin/sh\necho \"$*\" >> \"" + log + "\"\nexec \"" + real + "\" \"$@\"\n"
	if err := os.WriteFile(filepath.Join(wrapDir, "docker"), []byte(wrapper), 0o700); err != nil { //nolint:gosec // script de test
		t.Fatal(err)
	}
	runtime := t.TempDir()
	shellPID := strconv.Itoa(os.Getpid()) // ce processus joue le rôle du shell

	watch := func(t *testing.T, since string, pid string, args ...string) string {
		t.Helper()
		pending := filepath.Join(runtime, "telemach")
		_ = os.RemoveAll(pending)
		cmd := exec.Command(filepath.Join(h.bin, "docker-telemach"), append([]string{"watch", "--since", since, "--shell-pid", pid, "--"}, args...)...) // #nosec G204 -- binaire compilé par le test
		cmd.Env = append(os.Environ(), "PATH="+wrapDir+":"+os.Getenv("PATH"), "XDG_RUNTIME_DIR="+runtime, "XDG_STATE_HOME="+h.state, "NO_COLOR=1")
		if out, err := cmd.CombinedOutput(); err != nil || len(out) != 0 {
			t.Fatalf("watch : %v, sortie %q (aucune attendue)", err, out)
		}
		files, _ := filepath.Glob(filepath.Join(pending, "pending-*"))
		var b strings.Builder
		for _, f := range files {
			data, _ := os.ReadFile(f) // #nosec G304 -- dossier de test
			b.Write(data)
		}
		return b.String()
	}
	t0 := func() string { return strconv.FormatFloat(float64(time.Now().UnixNano())/1e9, 'f', 6, 64) }
	u := []string{"--user", "1000:1000"}

	cases := []struct {
		name string
		args []string
		want string // vide : aucun signal
	}{
		{"V1 postgres sans mot de passe", []string{"run", "-d", "--name", "telemach-it-w-pg", "postgres:18-alpine"}, "« telemach-it-w-pg » s'est arrêté avec le code 1"},
		{"V2 nginx qui tourne", []string{"run", "-d", "--name", "telemach-it-w-web", "nginxinc/nginx-unprivileged:1.30-alpine"}, ""},
		{"V5 --rm sorti avec 3", append(append([]string{"run", "-d", "--rm"}, u...), "alpine:3.24", "sh", "-c", "exit 3"), "code 3"},
		{"--rm sorti avec 0", append(append([]string{"run", "-d", "--rm"}, u...), "alpine:3.24", "true"), ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, a := range c.args {
				if strings.HasPrefix(a, "telemach-it-w-") {
					name := a
					t.Cleanup(func() { _ = exec.Command("docker", "rm", "-f", name).Run() })
				}
			}
			since := t0()
			if out, err := exec.Command("docker", c.args...).CombinedOutput(); err != nil {
				t.Fatalf("docker %v : %v\n%s", c.args, err, out)
			}
			got := watch(t, since, shellPID, c.args...)
			if c.want == "" && got != "" || c.want != "" && !strings.Contains(got, c.want) {
				t.Errorf("signal %q attendu, obtenu %q", c.want, got)
			}
			if strings.Contains(c.name, "--rm sorti avec 3") && !strings.Contains(got, "supprimé (--rm)") {
				t.Errorf("mention de --rm attendue : %q", got)
			}
		})
	}

	// Shell disparu : aucun fichier écrit.
	dead := exec.Command("true")
	_ = dead.Run()
	since := t0()
	args := append(append([]string{"run", "-d", "--rm"}, u...), "alpine:3.24", "sh", "-c", "exit 4")
	_ = exec.Command("docker", args...).Run()
	if got := watch(t, since, strconv.Itoa(dead.Process.Pid), args...); got != "" {
		t.Errorf("aucun signal pour un shell disparu : %q", got)
	}

	// Seules lectures : « events » (pas d'inspect, pas de logs).
	data, _ := os.ReadFile(log) // #nosec G304 -- dossier de test
	for _, l := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if l != "" && !strings.HasPrefix(l, "events ") {
			t.Errorf("lecture non attendue : docker %s", l)
		}
	}
}
