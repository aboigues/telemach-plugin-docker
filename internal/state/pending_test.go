package state

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// deadPID renvoie le pid d'un processus terminé.
func deadPID(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	return cmd.Process.Pid
}

func TestPendingDir(t *testing.T) {
	env := map[string]string{"XDG_RUNTIME_DIR": "/run/user/1000"}
	if got := PendingDir(func(k string) string { return env[k] }, "/etat"); got != "/run/user/1000/telemach" {
		t.Errorf("PendingDir = %q", got)
	}
	if got := PendingDir(func(string) string { return "" }, "/etat"); got != "/etat/pending" {
		t.Errorf("PendingDir sans XDG_RUNTIME_DIR = %q", got)
	}
}

func TestWritePending(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "telemach")
	me := os.Getpid()
	now := time.Unix(1792156562, 0)
	if err := WritePending(dir, me, "⚠ signal 1", now); err != nil {
		t.Fatal(err)
	}
	if err := WritePending(dir, me, "⚠ signal 2\n", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := WritePending(dir, deadPID(t), "⚠ pour un shell disparu", now); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("dossier : %v %v", info.Mode(), err)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*"))
	if len(files) != 2 {
		t.Fatalf("2 signaux attendus (aucun pour un shell disparu, aucun temporaire) : %v", files)
	}
	for _, f := range files {
		fi, _ := os.Stat(f)
		data, _ := os.ReadFile(f) // #nosec G304 -- dossier de test
		if fi.Mode().Perm() != 0o600 || !strings.HasPrefix(filepath.Base(f), "pending-") || !strings.HasSuffix(string(data), "\n") || strings.Count(string(data), "\n") != 1 {
			t.Errorf("%s : droits %v, contenu %q", f, fi.Mode().Perm(), data)
		}
	}
}

func TestCleanStalePending(t *testing.T) {
	dir := t.TempDir()
	dead := deadPID(t)
	for _, name := range []string{
		"pending-" + itoa(os.Getpid()) + "-1",
		"pending-" + itoa(dead) + "-1",
		"pending-pasunpid-1",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	old := filepath.Join(dir, ".tmp-ancien")
	_ = os.WriteFile(old, nil, 0o600)
	_ = os.Chtimes(old, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour))
	CleanStalePending(dir)
	files, _ := filepath.Glob(filepath.Join(dir, "*"))
	if len(files) != 1 || !strings.Contains(files[0], itoa(os.Getpid())) {
		t.Errorf("seul le signal du shell vivant doit rester : %v", files)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
