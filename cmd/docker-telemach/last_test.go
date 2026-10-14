package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if got := errb.String(); got != full {
		t.Errorf("last doit reproduire l'explication complète.\n--- explain ---\n%s\n--- last ---\n%s", full, got)
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
