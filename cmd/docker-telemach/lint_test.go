package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
		"docker stop <conteneur> && docker rm <conteneur>",
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
	if strings.Contains(manual, "docker rm") || strings.Contains(manual, "…") {
		t.Errorf("à la main : pas de conseil de suppression ni de capture manquante :\n%s", manual)
	}
	if _, ok := a2.store.Load(); ok {
		t.Error("une vérification à la main ne doit pas remplacer le dernier diagnostic")
	}
}

// T025 : seules lectures locales, la présence d'une variable (jamais sa valeur) et
// l'existence d'un dossier de nom simple dans le dossier courant.
func TestLintLocalReads(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "data"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fichier"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv("TELEMACH_TEST_SET", "x")
	t.Setenv("TELEMACH_TEST_EMPTY", "")

	cases := []struct {
		args []string
		want string // identifiant attendu dans la sortie, ou "" pour aucune sortie
	}{
		{[]string{"run", "-v", "data:/d", "alpine:3.24"}, "volume nommé"},
		{[]string{"run", "-v", "autre:/d", "alpine:3.24"}, ""},
		{[]string{"run", "-v", "fichier:/d", "alpine:3.24"}, ""},
		{[]string{"run", "-e", "TELEMACH_TEST_SET", "alpine:3.24"}, ""},
		{[]string{"run", "-e", "TELEMACH_TEST_EMPTY", "alpine:3.24"}, ""},
		{[]string{"run", "-e", "TELEMACH_TEST_ABSENT", "alpine:3.24"}, "Variable TELEMACH_TEST_ABSENT absente"},
	}
	for _, c := range cases {
		a, _, errb := testApp(t, nil)
		a.lookupEnv = os.LookupEnv
		run(a, append([]string{"lint", "--color=never", "--width", "200", "--"}, c.args...))
		got := errb.String()
		if c.want == "" && got != "" || c.want != "" && !strings.Contains(got, c.want) {
			t.Errorf("docker %s : %q attendu, obtenu :\n%s", strings.Join(c.args, " "), c.want, got)
		}
		if strings.Contains(got, "TELEMACH_TEST_SET=x") {
			t.Errorf("la valeur d'une variable ne doit jamais être affichée :\n%s", got)
		}
	}

	a, _, _ := testApp(t, nil)
	env := a.lintEnv()
	for _, name := range []string{"..", ".", "data/", "../data", "/tmp", ""} {
		if env.DirExists(name) {
			t.Errorf("DirExists(%q) doit être faux : nom relatif simple uniquement", name)
		}
	}
	if !env.DirExists("data") {
		t.Error("DirExists(\"data\") attendu vrai")
	}
}

// T027 : répétition. Rappel d'une ligne dans les 10 minutes, complet ensuite ; un rappel ne
// remplace pas le dernier diagnostic.
func TestLintRepetition(t *testing.T) {
	a, _, errb := testApp(t, nil)
	start := a.now()
	at := func(d time.Duration) { a.now = func() time.Time { return start.Add(d) } }
	lintAfter := func(args ...string) string {
		errb.Reset()
		run(a, append([]string{"lint", "--color=never", "--width", "200", "--after", "--"}, args...))
		return errb.String()
	}
	issue := []string{"run", "-d", "nginx:1.30-alpine", "-p", "8090:80"}

	if full := lintAfter(issue...); !strings.Contains(full, "Cause probable") {
		t.Fatalf("premier affichage complet attendu :\n%s", full)
	}
	at(5 * time.Minute)
	reminder := lintAfter(issue...)
	if strings.Count(reminder, "\n") != 1 || !strings.Contains(reminder, "↺ Encore") ||
		!strings.Contains(reminder, "docker run -d -p 8090:80 nginx:1.30-alpine") || !strings.Contains(reminder, "docker telemach last") {
		t.Errorf("rappel d'une ligne avec la commande corrigée attendu :\n%s", reminder)
	}
	if last, _ := a.store.Load(); !last.RecordedAt.Equal(start) {
		t.Errorf("un rappel ne doit pas réécrire last.json (RecordedAt = %v)", last.RecordedAt)
	}
	at(16 * time.Minute) // plus de 10 minutes après le dernier affichage complet
	if again := lintAfter(issue...); !strings.Contains(again, "Cause probable") {
		t.Errorf("affichage complet attendu après 10 minutes :\n%s", again)
	}

	// Image sans tag : une fois par image et par 24 h.
	at(0)
	if first := lintAfter("run", "--rm", "nginx"); !strings.Contains(first, "Image sans tag explicite") {
		t.Fatalf("information attendue :\n%s", first)
	}
	at(time.Hour)
	if second := lintAfter("run", "--rm", "nginx:latest"); second != "" {
		t.Errorf("même image dans les 24 h : rien attendu :\n%s", second)
	}
	if other := lintAfter("run", "--rm", "redis"); !strings.Contains(other, "redis") {
		t.Errorf("autre image : information attendue :\n%s", other)
	}
	at(25 * time.Hour)
	if later := lintAfter("run", "--rm", "nginx"); !strings.Contains(later, "Image sans tag explicite") {
		t.Errorf("information attendue après 24 h :\n%s", later)
	}
}
