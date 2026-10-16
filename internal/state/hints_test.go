package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)

func TestLastDiagnosisKindAndFindings(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	if err := s.Save(LastDiagnosis{Fingerprint: "x", RecordedAt: t0}); err != nil {
		t.Fatal(err)
	}
	d, _ := s.Load()
	if d.Kind != KindError || d.IsCmdline() {
		t.Errorf("Kind par défaut = %q, attendu %q", d.Kind, KindError)
	}
	in := LastDiagnosis{Kind: KindCmdline, RecordedAt: t0, Findings: []FindingRecord{
		{EntryID: "cmdline.option-after-image", Captures: map[string]string{"options": "-p 8090:80"}},
	}}
	if err := s.Save(in); err != nil {
		t.Fatal(err)
	}
	d, _ = s.Load()
	if !d.IsCmdline() || len(d.Findings) != 1 || d.Findings[0].Captures["options"] != "-p 8090:80" {
		t.Errorf("Findings non conservés : %+v", d)
	}
}

func TestHintDecide(t *testing.T) {
	h := HintStore{Dir: t.TempDir()}
	if got := h.Decide("a", false, t0); got != DisplayFull {
		t.Errorf("empreinte inconnue : %v", got)
	}
	if err := h.Record([]string{"a", "img"}, t0); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		fp    string
		image bool
		after time.Duration
		want  Display
	}{
		{"a", false, 5 * time.Minute, DisplayReminder},
		{"a", false, 11 * time.Minute, DisplayFull},
		{"img", true, 5 * time.Minute, DisplaySilent},
		{"img", true, 23 * time.Hour, DisplaySilent},
		{"img", true, 25 * time.Hour, DisplayFull},
		{"b", false, time.Minute, DisplayFull},
	}
	for _, c := range cases {
		if got := h.Decide(c.fp, c.image, t0.Add(c.after)); got != c.want {
			t.Errorf("%s après %v : %v, attendu %v", c.fp, c.after, got, c.want)
		}
	}
}

func TestHintStoreFile(t *testing.T) {
	dir := t.TempDir()
	h := HintStore{Dir: dir}
	if err := h.Record([]string{"vieux"}, t0); err != nil {
		t.Fatal(err)
	}
	// Une écriture plus de 24 h après purge l'ancienne entrée.
	if err := h.Record([]string{"recent"}, t0.Add(25*time.Hour)); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "hints.json")
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("droits = %v, attendu 0600", fi.Mode().Perm())
	}
	var f hintsFile
	data, _ := os.ReadFile(path)
	if err := json.Unmarshal(data, &f); err != nil || f.SchemaVersion != 1 {
		t.Fatalf("fichier inattendu : %s (%v)", data, err)
	}
	if _, ok := f.Shown["vieux"]; ok {
		t.Error("l'entrée de plus de 24 h doit être purgée")
	}

	// Plafond de 200 entrées : les plus anciennes sont supprimées.
	for i := range 210 {
		if err := h.Record([]string{fmt.Sprintf("fp%03d", i)}, t0.Add(26*time.Hour+time.Duration(i)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	data, _ = os.ReadFile(path)
	f = hintsFile{}
	_ = json.Unmarshal(data, &f)
	if len(f.Shown) != maxHints {
		t.Errorf("%d entrées, attendu %d", len(f.Shown), maxHints)
	}
	if _, ok := f.Shown["fp209"]; !ok {
		t.Error("l'entrée la plus récente doit être conservée")
	}
	if _, ok := f.Shown["fp000"]; ok {
		t.Error("l'entrée la plus ancienne doit être supprimée")
	}

	// Fichier illisible : ignoré, sans erreur.
	if err := os.WriteFile(path, []byte("{pas du json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := h.Decide("x", false, t0); got != DisplayFull {
		t.Errorf("fichier illisible : %v", got)
	}
	if err := h.Record([]string{"x"}, t0); err != nil {
		t.Errorf("écriture après fichier illisible : %v", err)
	}
}
