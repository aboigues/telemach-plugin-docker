package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aboigues/telemach-plugin-docker/internal/state"
)

func testApp(t *testing.T, env map[string]string) (*app, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var out, errb bytes.Buffer
	return &app{
		stdout:     &out,
		stderr:     &errb,
		getenv:     func(k string) string { return env[k] },
		now:        func() time.Time { return time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC) },
		executable: func() string { return "/opt/telemach/docker-telemach" },
		store:      state.Store{Dir: filepath.Join(t.TempDir(), "telemach")},
		version:    "1.2.3",
	}, &out, &errb
}

func writeStderr(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "stderr")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// Scénario V5 : erreur inventée → bloc « non reconnue », sans cause avancée.
func TestExplainUnknown(t *testing.T) {
	a, _, errb := testApp(t, nil)
	f := writeStderr(t, "erreur totalement inventée\n")
	code := run(a, []string{"explain", "--color=never", "--exit-code", "1", "--stderr-file", f, "--docker-version", "29.8.0", "--", "pull", "x"})
	if code != 0 {
		t.Fatalf("code = %d, attendu 0 ; sortie :\n%s", code, errb)
	}
	out := errb.String()
	for _, want := range []string{"✖ Erreur non reconnue", "préfère ne pas deviner", "Docker 29.8.0", "erreur totalement inventée", "https://docs.docker.com/"} {
		if !strings.Contains(out, want) {
			t.Errorf("sortie sans %q :\n%s", want, out)
		}
	}
	if strings.Contains(out, "Cause probable") {
		t.Errorf("aucune cause ne doit être avancée pour une erreur inconnue :\n%s", out)
	}
	if f2, _ := os.ReadFile(f); string(f2) != "erreur totalement inventée\n" {
		t.Error("le fichier stderr ne doit pas être modifié")
	}
}

func TestExplainColorAlways(t *testing.T) {
	a, _, errb := testApp(t, nil)
	f := writeStderr(t, "Unable to find image 'nginx-inventee:latest' locally\n")
	if code := run(a, []string{"explain", "--color=always", "--exit-code", "125", "--stderr-file", f, "--", "run", "nginx-inventee"}); code != 0 {
		t.Fatalf("code = %d", code)
	}
	out := errb.String()
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("séquences de couleur attendues :\n%q", out)
	}
	if !strings.Contains(out, "✖") || !strings.Contains(out, "Message Docker d'origine") {
		t.Errorf("symboles et libellés attendus même en couleur :\n%s", out)
	}
	// Le fragment reconnu (l'image) est surligné : il est entouré d'une séquence de style
	// distincte du reste de la ligne, qui est atténuée (faint = code 2).
	if !strings.Contains(out, "\x1b[2") {
		t.Errorf("le message d'origine doit être atténué :\n%q", out)
	}
	if !strings.Contains(out, "nginx-inventee\x1b[") {
		t.Errorf("le fragment reconnu doit être surligné :\n%q", out)
	}
}

func TestExplainNoColor(t *testing.T) {
	for name, tc := range map[string]struct {
		flag string
		env  map[string]string
	}{
		"--color=never": {"--color=never", nil},
		"NO_COLOR=1":    {"--color=auto", map[string]string{"NO_COLOR": "1"}},
	} {
		t.Run(name, func(t *testing.T) {
			a, _, errb := testApp(t, tc.env)
			f := writeStderr(t, "quelque chose a échoué\n")
			run(a, []string{"explain", tc.flag, "--exit-code", "1", "--stderr-file", f, "--", "ps"})
			out := errb.String()
			if strings.Contains(out, "\x1b") {
				t.Errorf("aucune séquence ESC attendue :\n%q", out)
			}
			if !strings.Contains(out, "✖ Erreur non reconnue") || !strings.Contains(out, "Message Docker d'origine") {
				t.Errorf("symboles et libellés attendus sans couleur :\n%s", out)
			}
		})
	}
}

func TestExplainInvalidUsage(t *testing.T) {
	a, _, _ := testApp(t, nil)
	if code := run(a, []string{"explain", "--exit-code", "0", "--", "ps"}); code != 2 {
		t.Errorf("--exit-code 0 : code = %d, attendu 2", code)
	}
	if code := run(a, []string{"explain", "--", "ps"}); code != 2 {
		t.Errorf("sans --exit-code : code = %d, attendu 2", code)
	}
}

func TestExplainSanitizesAndRedacts(t *testing.T) {
	a, _, errb := testApp(t, nil)
	f := writeStderr(t, "\x1b]0;titre piégé\x07échec avec DB_PASSWORD=SuperSecret123\n")
	run(a, []string{"explain", "--color=never", "--exit-code", "1", "--stderr-file", f, "--", "run", "-e", "DB_PASSWORD=SuperSecret123", "alpine"})
	out := errb.String()
	if strings.Contains(out, "SuperSecret123") || strings.Contains(out, "titre piégé") || strings.Contains(out, "\x1b") {
		t.Errorf("sortie non nettoyée ou non masquée :\n%q", out)
	}
	data, _ := os.ReadFile(filepath.Join(a.store.Dir, "last.json"))
	if strings.Contains(string(data), "SuperSecret123") {
		t.Error("le secret ne doit pas être écrit sur disque")
	}
}

func TestReadTailKeepsEnd(t *testing.T) {
	p := writeStderr(t, strings.Repeat("a", 100)+"FIN")
	got, err := readTail(p, 10)
	if err != nil || got != "aaaaaaaFIN" {
		t.Errorf("readTail = %q, %v", got, err)
	}
}

func TestShellInitAndMetadata(t *testing.T) {
	a, out, _ := testApp(t, nil)
	if code := run(a, []string{"shell-init", "zsh"}); code != 0 || !strings.Contains(out.String(), "docker() (") ||
		!strings.Contains(out.String(), "__telemach_bin='/opt/telemach/docker-telemach'") {
		t.Errorf("shell-init zsh : code %d, sortie %q", code, out.String())
	}
	a, _, errb := testApp(t, nil)
	if code := run(a, []string{"shell-init", "fish"}); code != 2 || !strings.Contains(errb.String(), "non pris en charge") {
		t.Errorf("shell-init fish : code %d, erreur %q", code, errb.String())
	}
	a, out, _ = testApp(t, nil)
	if code := run(a, []string{"docker-cli-plugin-metadata"}); code != 0 || !strings.Contains(out.String(), `"SchemaVersion":"0.1.0"`) {
		t.Errorf("métadonnées : code %d, sortie %q", code, out.String())
	}
}

// Une commande en échec dont la ligne contient un piège : un seul affichage, le piège
// présenté comme cause probable (feature 003, FR-010).
func TestExplainIncludesCmdlinePitfall(t *testing.T) {
	msg := "OCI runtime exec failed: exec failed: unable to start container process: exec: \"-it\": executable file not found in $PATH: unknown\n"
	args := []string{"explain", "--color=never", "--exit-code", "126", "--stderr-file", "", "--", "exec", "web", "-it", "sh"}

	a, _, errb := testApp(t, nil)
	args[5] = writeStderr(t, msg)
	if code := run(a, args); code != 0 {
		t.Fatalf("code = %d", code)
	}
	out := errb.String()
	if n := strings.Count(out, "Option Docker placée après le conteneur"); n != 1 {
		t.Errorf("piège attendu une fois, trouvé %d fois :\n%s", n, out)
	}
	if !strings.Contains(out, "docker exec -it web sh") {
		t.Errorf("commande corrigée attendue :\n%s", out)
	}
	last, ok := a.store.Load()
	if !ok || len(last.Findings) != 1 || last.Findings[0].EntryID != "cmdline.option-after-container" {
		t.Errorf("le piège doit être conservé pour last : %+v", last.Findings)
	}

	// TELEMACH_LINT=0 : explain se comporte exactement comme avant.
	a2, _, errb2 := testApp(t, map[string]string{"TELEMACH_LINT": "0"})
	args[5] = writeStderr(t, msg)
	run(a2, args)
	if strings.Contains(errb2.String(), "Dans ta commande") || strings.Contains(errb2.String(), "Option Docker placée") {
		t.Errorf("aucun piège attendu avec TELEMACH_LINT=0 :\n%s", errb2.String())
	}

	// Erreur non reconnue : le piège devient l'explication principale.
	a3, _, errb3 := testApp(t, nil)
	args[5] = writeStderr(t, "erreur totalement inventée\n")
	run(a3, args)
	out3 := errb3.String()
	if strings.Contains(out3, "ne reconnaît pas encore") || !strings.Contains(out3, "Option Docker placée après le conteneur") || !strings.Contains(out3, "erreur totalement inventée") {
		t.Errorf("piège principal et message d'origine attendus :\n%s", out3)
	}
}
