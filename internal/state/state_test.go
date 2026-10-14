package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newStore(t *testing.T) (Store, string) {
	t.Helper()
	dir := t.TempDir()
	return Store{Dir: filepath.Join(dir, "telemach")}, dir
}

func TestDirFromEnv(t *testing.T) {
	env := map[string]string{"XDG_STATE_HOME": "/xdg/state", "HOME": "/home/x"}
	if got := DefaultDir(func(k string) string { return env[k] }); got != "/xdg/state/telemach" {
		t.Errorf("DefaultDir avec XDG_STATE_HOME = %q", got)
	}
	delete(env, "XDG_STATE_HOME")
	if got := DefaultDir(func(k string) string { return env[k] }); got != "/home/x/.local/state/telemach" {
		t.Errorf("DefaultDir sans XDG_STATE_HOME = %q", got)
	}
}

func TestSaveLoadPermissions(t *testing.T) {
	s, _ := newStore(t)
	now := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	in := LastDiagnosis{Fingerprint: "abc", EntryID: "run.port-already-allocated", Captures: map[string]string{"port": "8080"}, RecordedAt: now}
	if err := s.Save(in); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(s.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("droits du répertoire = %o, attendu 700", perm)
	}
	finfo, err := os.Stat(filepath.Join(s.Dir, fileName))
	if err != nil {
		t.Fatal(err)
	}
	if perm := finfo.Mode().Perm(); perm != 0o600 {
		t.Errorf("droits du fichier = %o, attendu 600", perm)
	}
	out, ok := s.Load()
	if !ok {
		t.Fatal("Load : diagnostic attendu")
	}
	if out.SchemaVersion != SchemaVersion || out.EntryID != in.EntryID || out.Captures["port"] != "8080" || !out.RecordedAt.Equal(now) {
		t.Errorf("Load = %+v", out)
	}
}

func TestLoadCorruptedOrUnknownSchema(t *testing.T) {
	s, _ := newStore(t)
	if _, ok := s.Load(); ok {
		t.Error("aucun diagnostic attendu sans fichier")
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(s.Dir, fileName)
	for _, content := range []string{"{pas du json", `{"SchemaVersion": 99, "Fingerprint": "x"}`} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, ok := s.Load(); ok {
			t.Errorf("contenu %q : doit être ignoré", content)
		}
		if err := s.Save(LastDiagnosis{Fingerprint: "y", RecordedAt: time.Now()}); err != nil {
			t.Errorf("le fichier doit pouvoir être réécrit : %v", err)
		}
	}
}

func TestIsRepeat(t *testing.T) {
	s, _ := newStore(t)
	now := time.Now()
	if s.IsRepeat("fp", now) {
		t.Error("pas de répétition sans diagnostic précédent")
	}
	if err := s.Save(LastDiagnosis{Fingerprint: "fp", RecordedAt: now}); err != nil {
		t.Fatal(err)
	}
	if !s.IsRepeat("fp", now.Add(9*time.Minute)) {
		t.Error("répétition attendue à 9 minutes")
	}
	if s.IsRepeat("fp", now.Add(11*time.Minute)) {
		t.Error("pas de répétition au-delà de 10 minutes")
	}
	if s.IsRepeat("autre", now.Add(time.Minute)) {
		t.Error("pas de répétition pour une autre empreinte")
	}
}
