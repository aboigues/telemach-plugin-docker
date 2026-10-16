package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aboigues/telemach-plugin-docker/internal/state"
)

func TestLastWithoutHistory(t *testing.T) {
	a, _, errb := testApp(t, nil)
	if code := run(a, []string{"last", "--color=never"}); code != 0 {
		t.Errorf("code = %d", code)
	}
	if !strings.Contains(errb.String(), "Aucune erreur récente") {
		t.Errorf("message d'aide attendu :\n%s", errb.String())
	}
}

// Après une version courte (erreur répétée), « last » réaffiche exactement l'explication
// complète d'origine.
func TestLastReprintsFullExplanation(t *testing.T) {
	a, _, errb := testApp(t, nil)
	f := writeStderr(t, "docker: Error response from daemon: pull access denied for ngnix, repository does not exist or may require 'docker login'\n")
	args := []string{"explain", "--color=never", "--exit-code", "125", "--stderr-file", f, "--docker-version", "29.8.0", "--", "run", "--rm", "-e", "DB_PASSWORD=SuperSecret123", "ngnix"}

	run(a, args)
	full := errb.String()
	if !strings.Contains(full, "Cause probable") {
		t.Fatalf("explication complète attendue :\n%s", full)
	}

	errb.Reset()
	run(a, args)
	short := errb.String()
	if strings.Contains(short, "Cause probable") || !strings.Contains(short, "docker telemach last") {
		t.Errorf("version courte avec renvoi à « last » attendue :\n%s", short)
	}

	errb.Reset()
	if code := run(a, []string{"last", "--color=never"}); code != 0 {
		t.Fatalf("code = %d", code)
	}
	got := errb.String()
	// En-tête : quand et pour quelle commande l'erreur a été enregistrée (issue #7), secret masqué.
	header, body, _ := strings.Cut(got, "\n")
	for _, want := range []string{"14/10/2026 à 10:00", "à l'instant", "docker run --rm -e DB_PASSWORD=", "ngnix"} {
		if !strings.Contains(header, want) {
			t.Errorf("en-tête sans %q : %q", want, header)
		}
	}
	if strings.Contains(header, "SuperSecret123") {
		t.Errorf("le secret ne doit pas apparaître dans l'en-tête : %q", header)
	}
	if body != full {
		t.Errorf("last doit reproduire l'explication complète.\n--- explain ---\n%s\n--- last ---\n%s", full, body)
	}

	data, err := os.ReadFile(filepath.Join(a.store.Dir, "last.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "SuperSecret123") {
		t.Error("le secret ne doit pas être conservé sur disque")
	}
}

func TestLastCompose(t *testing.T) {
	a, _, errb := testApp(t, nil)
	f := writeStderr(t, "yaml: line 3: found character that cannot start any token\n")
	run(a, []string{"explain", "--color=never", "--exit-code", "1", "--stderr-file", f, "--", "compose", "--project-directory", corpusDir("tab-indent"), "up"})
	errb.Reset()
	run(a, []string{"last", "--color=never"})
	if out := errb.String(); !strings.Contains(out, "compose.yaml:3:1") || !strings.Contains(out, "Version corrigée") {
		t.Errorf("le problème Compose doit être réaffiché :\n%s", out)
	}
}

// L'ancienneté de l'erreur réaffichée est visible : une erreur d'hier ne passe pas pour la
// dernière commande tapée (issue #7).
func TestLastShowsAge(t *testing.T) {
	a, _, errb := testApp(t, nil)
	f := writeStderr(t, "docker: Error response from daemon: pull access denied for ngnix, repository does not exist or may require 'docker login'\n")
	run(a, []string{"explain", "--color=never", "--exit-code", "125", "--stderr-file", f, "--", "run", "ngnix"})

	cases := []struct {
		after time.Duration
		want  string
	}{
		{30 * time.Second, "à l'instant"},
		{5 * time.Minute, "il y a 5 min"},
		{21 * time.Hour, "il y a 21 h"},
		{3 * 24 * time.Hour, "il y a 3 j"},
	}
	for _, c := range cases {
		a.now = func() time.Time { return time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC).Add(c.after) }
		errb.Reset()
		run(a, []string{"last", "--color=never"})
		header, _, _ := strings.Cut(errb.String(), "\n")
		if !strings.Contains(header, c.want) || !strings.Contains(header, "docker run ngnix") {
			t.Errorf("après %v : en-tête %q, attendu %q et la commande", c.after, header, c.want)
		}
	}
}

// Un fichier d'état antérieur (sans arguments enregistrés) affiche la sous-commande seule.
func TestLastOldStateWithoutArgs(t *testing.T) {
	a, _, errb := testApp(t, nil)
	if err := a.store.Save(state.LastDiagnosis{
		RecordedAt: time.Date(2026, 10, 13, 9, 30, 0, 0, time.UTC), Family: "docker", Subcommand: "run",
		Stderr: "erreur inventée", Confidence: "unknown", Origin: "docker",
	}); err != nil {
		t.Fatal(err)
	}
	run(a, []string{"last", "--color=never"})
	header, _, _ := strings.Cut(errb.String(), "\n")
	if !strings.Contains(header, "13/10/2026 à 09:30") || !strings.Contains(header, "docker run") {
		t.Errorf("en-tête inattendu : %q", header)
	}
}

// Erreur non reconnue dont la cause probable est un piège de la ligne de commande :
// « last » réaffiche le même rendu qu'explain.
func TestLastReprintsCmdlineCause(t *testing.T) {
	a, _, errb := testApp(t, nil)
	f := writeStderr(t, "quelque chose d'inattendu\n")
	run(a, []string{"explain", "--color=never", "--exit-code", "1", "--stderr-file", f, "--", "run", "nginx:1.30-alpine", "--rm", "-p", "8090:80"})
	full := errb.String()
	if !strings.Contains(full, "Option Docker placée après l'image") {
		t.Fatalf("piège attendu comme explication principale :\n%s", full)
	}
	errb.Reset()
	run(a, []string{"last", "--color=never"})
	if _, body, _ := strings.Cut(errb.String(), "\n"); body != full {
		t.Errorf("last doit reproduire l'explication.\n--- explain ---\n%s\n--- last ---\n%s", full, body)
	}
}
