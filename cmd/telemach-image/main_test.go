package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// execCall mémorise l'appel au binaire cible.
type execCall struct {
	path string
	argv []string
}

// testApp construit une application isolée : sorties en mémoire, exécution simulée.
func testApp(t *testing.T) (*app, *bytes.Buffer, *bytes.Buffer, *execCall) {
	t.Helper()
	var out, errb bytes.Buffer
	call := &execCall{}
	return &app{
		stdout:  &out,
		stderr:  &errb,
		getenv:  func(string) string { return "" },
		environ: func() []string { return []string{"TERM=xterm"} },
		exec: func(path string, argv, _ []string) error {
			call.path, call.argv = path, argv
			return nil
		},
		version: "1.2.3",
		root:    t.TempDir(),
		dest:    t.TempDir(),
	}, &out, &errb, call
}

func TestHelp(t *testing.T) {
	for _, args := range [][]string{nil, {"--help"}, {"-h"}, {"help"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			a, out, _, call := testApp(t)
			if code := run(a, args); code != 0 {
				t.Fatalf("code = %d", code)
			}
			if call.path != "" {
				t.Errorf("le plugin ne doit pas être exécuté pour l'aide")
			}
			for _, want := range []string{
				`docker run --rm -t -v "$PWD":/projet:ro telemachlearning/telemach:1.2.3 check /projet`,
				"mkdir -p ~/.docker/cli-plugins",
				`docker run --rm --user "$(id -u):$(id -g)" -v ~/.docker/cli-plugins:/sortie telemachlearning/telemach:1.2.3 install`,
				`docker run --rm --user "$(id -u):$(id -g)" -e HOST_OS=darwin -v ~/.docker/cli-plugins:/sortie telemachlearning/telemach:1.2.3 install`,
			} {
				if !strings.Contains(out.String(), want) {
					t.Errorf("commande absente de l'aide : %s\n%s", want, out.String())
				}
			}
		})
	}
}

// Une version de développement ne doit pas produire de commande avec un tag invalide.
func TestHelpDevVersion(t *testing.T) {
	a, out, _, _ := testApp(t)
	a.version = "dev"
	run(a, nil)
	if !strings.Contains(out.String(), "telemachlearning/telemach:X.Y.Z") {
		t.Errorf("tag générique attendu pour une version de développement :\n%s", out.String())
	}
}

func TestPassThrough(t *testing.T) {
	for _, args := range [][]string{{"--version"}, {"explain", "--exit-code", "1", "--", "ps"}} {
		t.Run(args[0], func(t *testing.T) {
			a, _, _, call := testApp(t)
			if code := run(a, args); code != 0 {
				t.Fatalf("code = %d", code)
			}
			if call.path != linuxPlugin(a.root) {
				t.Errorf("chemin = %q, attendu %q", call.path, linuxPlugin(a.root))
			}
			want := append([]string{"docker-telemach"}, args...)
			if strings.Join(call.argv, "\x00") != strings.Join(want, "\x00") {
				t.Errorf("argv = %q, attendu %q", call.argv, want)
			}
		})
	}
}

func TestPassThroughFailure(t *testing.T) {
	a, _, errb, _ := testApp(t)
	a.exec = func(string, []string, []string) error { return errors.New("introuvable") }
	if code := run(a, []string{"--version"}); code != 2 {
		t.Errorf("code = %d, attendu 2", code)
	}
	if !strings.Contains(errb.String(), "introuvable") {
		t.Errorf("cause attendue dans le message :\n%s", errb.String())
	}
}

// « check » sur un dossier sans fichier Compose : message propre à l'image, sans lancer le
// plugin (contracts/image-cli.md).
func TestCheckPreflight(t *testing.T) {
	withCompose := t.TempDir()
	if err := os.WriteFile(filepath.Join(withCompose, "compose.yaml"), []byte("services: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	empty := t.TempDir()

	cases := []struct {
		name     string
		args     []string
		wantExec bool
		wantCode int
	}{
		{"dossier inexistant", []string{"check", filepath.Join(empty, "absent")}, false, 2},
		{"dossier sans fichier Compose", []string{"check", empty}, false, 2},
		{"options avant le dossier", []string{"check", "--color", "never", empty}, false, 2},
		{"dossier avec compose.yaml", []string{"check", withCompose}, true, 0},
		{"fichier explicite -f", []string{"check", "-f", "autre.yaml", empty}, true, 0},
		{"fichier explicite --file=", []string{"check", "--file=autre.yaml", empty}, true, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, _, errb, call := testApp(t)
			if code := run(a, c.args); code != c.wantCode {
				t.Errorf("code = %d, attendu %d\n%s", code, c.wantCode, errb.String())
			}
			if (call.path != "") != c.wantExec {
				t.Errorf("plugin exécuté = %v, attendu %v", call.path != "", c.wantExec)
			}
			if !c.wantExec && !strings.Contains(errb.String(), "Aucun fichier Compose") {
				t.Errorf("message de l'image attendu :\n%s", errb.String())
			}
		})
	}
}
