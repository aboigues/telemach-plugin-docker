package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Faux docker : journalise ses appels et renvoie les événements réels capturés.
const fakeDockerEvents = `#!/bin/sh
echo "$*" >> "$FAKE_DIR/calls.log"
cat "$FAKE_DIR/events.out"
`

func watchApp(t *testing.T, env map[string]string) (*app, string, string) {
	t.Helper()
	runtime := t.TempDir()
	e := map[string]string{"XDG_RUNTIME_DIR": runtime, "NO_COLOR": "1"}
	for k, v := range env {
		e[k] = v
	}
	a, _, _ := testApp(t, e)
	dir := t.TempDir()
	bin := filepath.Join(dir, "docker")
	if err := os.WriteFile(bin, []byte(fakeDockerEvents), 0o700); err != nil { //nolint:gosec // script de test
		t.Fatal(err)
	}
	events, err := os.ReadFile(filepath.Join("..", "..", "testdata", "exited", "events", "run-d.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "events.out"), events, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_DIR", dir)
	a.dockerBin = bin
	a.sleep = func(time.Duration) {}
	return a, dir, filepath.Join(runtime, "telemach")
}

func pending(t *testing.T, dir string) string {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join(dir, "pending-*"))
	var b strings.Builder
	for _, f := range files {
		data, _ := os.ReadFile(f) // #nosec G304 -- dossier de test
		b.Write(data)
	}
	return b.String()
}

func TestWatchSignalsFailedLaunch(t *testing.T) {
	a, fake, dir := watchApp(t, nil)
	pid := strconvItoa(os.Getpid())
	if code := run(a, []string{"watch", "--since", "1792156562,228663", "--shell-pid", pid, "--", "run", "-d", "--name", "telemach-exited-ev-pg", "postgres:18-alpine"}); code != 0 {
		t.Fatalf("code = %d", code)
	}
	got := pending(t, dir)
	if !strings.Contains(got, "⚠ Le conteneur « telemach-exited-ev-pg » s'est arrêté avec le code 1") || !strings.Contains(got, "docker telemach ps") {
		t.Errorf("signal attendu :\n%q", got)
	}
	calls, _ := os.ReadFile(filepath.Join(fake, "calls.log")) // #nosec G304 -- dossier de test
	if !strings.HasPrefix(string(calls), "events --since 1792156562.228663 --until ") || strings.Count(string(calls), "\n") != 1 {
		t.Errorf("une seule lecture, « events », attendue : %q", calls)
	}
}

func TestWatchRemovedLaunch(t *testing.T) {
	a, _, dir := watchApp(t, nil)
	run(a, []string{"watch", "--since", "1792156562", "--shell-pid", strconvItoa(os.Getpid()), "--", "run", "-d", "--rm", "--user", "1000:1000", "alpine:3.24", "sh", "-c", "exit 3"})
	if got := pending(t, dir); !strings.Contains(got, "code 3") || !strings.Contains(got, "supprimé (--rm)") {
		t.Errorf("signal --rm attendu :\n%q", got)
	}
}

func TestWatchSilentCases(t *testing.T) {
	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	dead := strconvItoa(cmd.Process.Pid)
	me := strconvItoa(os.Getpid())
	cases := []struct {
		name string
		env  map[string]string
		args []string
	}{
		{"conteneur toujours démarré", nil, []string{"--shell-pid", me, "--", "run", "-d", "--name", "telemach-exited-ev-web", "nginxinc/nginx-unprivileged:1.30-alpine"}},
		{"sans -d", nil, []string{"--shell-pid", me, "--", "run", "--name", "telemach-exited-ev-pg", "postgres:18-alpine"}},
		{"TELEMACH_WATCH=0", map[string]string{"TELEMACH_WATCH": "0"}, []string{"--shell-pid", me, "--", "run", "-d", "--name", "telemach-exited-ev-pg", "postgres:18-alpine"}},
		{"TELEMACH_DISABLE=1", map[string]string{"TELEMACH_DISABLE": "1"}, []string{"--shell-pid", me, "--", "run", "-d", "--name", "telemach-exited-ev-pg", "postgres:18-alpine"}},
		{"shell disparu", nil, []string{"--shell-pid", dead, "--", "run", "-d", "--name", "telemach-exited-ev-pg", "postgres:18-alpine"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, fake, dir := watchApp(t, c.env)
			if code := run(a, append([]string{"watch", "--since", "1792156562"}, c.args...)); code != 0 {
				t.Errorf("code = %d", code)
			}
			if got := pending(t, dir); got != "" {
				t.Errorf("aucun signal attendu : %q", got)
			}
			if c.name != "conteneur toujours démarré" {
				if _, err := os.Stat(filepath.Join(fake, "calls.log")); err == nil {
					t.Error("aucune lecture du démon attendue")
				}
			}
		})
	}
}

func strconvItoa(n int) string { return strconv.Itoa(n) }
