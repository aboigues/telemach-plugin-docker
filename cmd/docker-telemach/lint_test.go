package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "réécrit les fichiers golden")

// golden compare got au fichier testdata/golden/<name>.golden (réécrit avec -update).
func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", "golden", name+".golden")
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden %s : %v (relancer avec -update)", path, err)
	}
	if got != string(want) {
		t.Errorf("sortie différente de %s :\n--- obtenu ---\n%s\n--- attendu ---\n%s", path, got, want)
	}
}

func TestLintSilentCases(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		args []string
	}{
		{"TELEMACH_LINT=0", map[string]string{"TELEMACH_LINT": "0"}, []string{"lint", "--after", "--", "run", "-d", "nginx", "-p", "8090:80"}},
		{"sous-commande non analysée", nil, []string{"lint", "--after", "--", "ps", "-a"}},
		{"ligne correcte", nil, []string{"lint", "--", "run", "--rm", "-p", "8080:80", "nginx:1.30-alpine"}},
		{"sans arguments", nil, []string{"lint"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, out, errb := testApp(t, c.env)
			if code := run(a, c.args); code != 0 {
				t.Errorf("code = %d\n%s", code, errb.String())
			}
			if out.Len() != 0 || errb.Len() != 0 {
				t.Errorf("aucune sortie attendue :\nstdout=%q\nstderr=%q", out.String(), errb.String())
			}
		})
	}
}

func TestLintInHelp(t *testing.T) {
	a, out, _ := testApp(t, nil)
	run(a, []string{"--help"})
	if !strings.Contains(out.String(), "lint") {
		t.Errorf("lint absent de l'aide :\n%s", out.String())
	}
}

// Exemple de l'issue #9 : formulation au passé après exécution, au conditionnel à la main.
func TestLintIssue9Phrasing(t *testing.T) {
	args := []string{"run", "-d", "nginx", "-p", "8090:80"}

	a, out, errb := testApp(t, nil)
	if code := run(a, append([]string{"lint", "--color=never", "--width", "200", "--after", "--"}, args...)); code != 1 {
		t.Errorf("code = %d, attendu 1 (avertissement)", code)
	}
	after := errb.String()
	golden(t, "issue9-after", after)
	for _, want := range []string{
		"Option Docker placée après l'image : -p 8090:80 a été transmis au conteneur",
		"le port 8090 de ta machine n'est pas publié",
		"docker run -d -p 8090:80 nginx",
		"docker rm -f <conteneur>",
		"Image sans tag explicite",
	} {
		if !strings.Contains(after, want) {
			t.Errorf("--after : %q absent :\n%s", want, after)
		}
	}
	if out.Len() != 0 {
		t.Errorf("rien ne doit être écrit sur stdout : %q", out.String())
	}

	a2, _, errb2 := testApp(t, nil)
	run(a2, append([]string{"lint", "--color=never", "--width", "200", "--"}, args...))
	manual := errb2.String()
	golden(t, "issue9-manual", manual)
	for _, want := range []string{"serait transmis", "ne serait pas publié", "docker run -d -p 8090:80 nginx"} {
		if !strings.Contains(manual, want) {
			t.Errorf("à la main : %q absent :\n%s", want, manual)
		}
	}
	if strings.Contains(manual, "docker rm -f") || strings.Contains(manual, "…") {
		t.Errorf("à la main : pas de conseil de suppression ni de capture manquante :\n%s", manual)
	}
	if _, ok := a2.store.Load(); ok {
		t.Error("une vérification à la main ne doit pas remplacer le dernier diagnostic")
	}
}
