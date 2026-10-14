package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func corpusDir(name string) string { return filepath.Join("..", "..", "testdata", "compose", name) }

func TestCheckExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
		want     []string
	}{
		{"fichier valide", []string{"check", "--color=never", corpusDir("valid")}, 0, []string{"✔ Aucun problème détecté dans compose.yaml"}},
		{"tabulation", []string{"check", "--color=never", corpusDir("tab-indent")}, 1, []string{"compose.yaml:3:1", "→image", "Version corrigée", "YAML indentation"}},
		{"dossier sans fichier Compose", []string{"check", "--color=never", corpusDir("empty-dir")}, 2, []string{"compose.yaml, compose.yml, docker-compose.yaml, docker-compose.yml"}},
		{"dossier inexistant", []string{"check", "/telemach/inexistant"}, 2, []string{"n'existe pas"}},
		{"clé inconnue dans le fichier override", []string{"check", "--color=never", corpusDir("multi-file-override")}, 1, []string{"compose.override.yaml:3:5", "« ports »"}},
		{"secret en clair masqué", []string{"check", "--color=never", corpusDir("plaintext-password")}, 1, []string{"Remarque", "••••••", "${POSTGRES_PASSWORD}"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, _, errb := testApp(t, nil)
			code := run(a, tt.args)
			out := errb.String()
			if code != tt.wantCode {
				t.Errorf("code = %d, attendu %d\n%s", code, tt.wantCode, out)
			}
			for _, w := range tt.want {
				if !strings.Contains(out, w) {
					t.Errorf("sortie sans %q :\n%s", w, out)
				}
			}
			if strings.Contains(out, "example") && strings.Contains(tt.name, "secret") {
				t.Errorf("le secret ne doit pas apparaître :\n%s", out)
			}
		})
	}
}

// Plusieurs -f : chaque problème indique son fichier d'origine.
func TestCheckMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("base.yaml", "services:\n  web:\n    image: nginx:1.30-alpine\n")
	write("dev.yaml", "services:\n  web:\n    port:\n      - \"8080:80\"\n")
	a, _, errb := testApp(t, nil)
	code := run(a, []string{"check", "--color=never", "-f", "base.yaml", "-f", "dev.yaml", dir})
	if code != 1 || !strings.Contains(errb.String(), "dev.yaml:3:5") {
		t.Errorf("code %d, sortie :\n%s", code, errb.String())
	}
}

func TestExplainComposeShowsFileProblem(t *testing.T) {
	a, _, errb := testApp(t, nil)
	f := writeStderr(t, "yaml: while scanning for the next token at line 3: found character that cannot start any token\n")
	code := run(a, []string{
		"explain", "--color=never", "--exit-code", "1", "--stderr-file", f, "--",
		"compose", "--project-directory", corpusDir("tab-indent"), "up",
	})
	out := errb.String()
	if code != 0 || !strings.Contains(out, "compose.yaml:3:1") || !strings.Contains(out, "Version corrigée") {
		t.Errorf("code %d, sortie :\n%s", code, out)
	}
	last, ok := a.store.Load()
	if !ok || last.EntryID != "compose.tab-indent" {
		t.Errorf("le dernier diagnostic doit être compose.tab-indent : %+v", last)
	}
}
