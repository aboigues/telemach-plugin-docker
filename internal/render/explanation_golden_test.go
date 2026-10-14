package render

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
	"github.com/aboigues/telemach-plugin-docker/internal/command"
	"github.com/aboigues/telemach-plugin-docker/internal/diagnose"
	"github.com/aboigues/telemach-plugin-docker/internal/redact"
	"github.com/aboigues/telemach-plugin-docker/internal/sanitize"
)

var update = flag.Bool("update", false, "régénère les fichiers golden")

// golden compare got au fichier testdata/golden/<dir>/<name>.golden (ou le réécrit).
func golden(t *testing.T, dir, name, got string) {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "golden", dir, name+".golden")
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("fichier golden manquant (relancer avec -update) : %v", err)
	}
	if diff := cmp.Diff(string(want), got); diff != "" {
		t.Errorf("%s (-attendu +obtenu) :\n%s", path, diff)
	}
}

// corpusDiagnosis rejoue un cas du corpus testdata/errors.
func corpusDiagnosis(t *testing.T, name string) (diagnose.Diagnosis, string, int) {
	t.Helper()
	dir := filepath.Join("..", "..", "testdata", "errors")
	read := func(ext string) string {
		data, err := os.ReadFile(filepath.Join(dir, name+ext))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	code, _ := strconv.Atoi(strings.TrimSpace(read(".exit")))
	stderr := redact.Redact(sanitize.Clean(read(".stderr.txt")))
	in := diagnose.Input{Command: command.Parse(strings.Fields(read(".args"))), ExitCode: code, Stderr: stderr}
	return diagnose.Diagnose(in, cat), stderr, code
}

func TestExplanationGolden(t *testing.T) {
	cases := []struct {
		name     string
		corpus   string
		repeated bool
	}{
		{"port-alloue", "run-port-allocated", false},
		{"image-introuvable", "image-not-found-typo", false},
		{"demon-arrete", "daemon-not-running", false},
		{"erreur-applicative", "runtime-application-error", false},
		{"erreur-inconnue", "unknown-invented", false},
		{"repetition", "run-port-allocated", true},
	}
	for _, c := range cases {
		for _, mode := range []struct {
			suffix string
			color  bool
		}{{"sans-couleur", false}, {"couleur", true}} {
			t.Run(c.name+"/"+mode.suffix, func(t *testing.T) {
				d, stderr, code := corpusDiagnosis(t, c.corpus)
				var b bytes.Buffer
				opts := Options{Theme: NewTheme(mode.color, true), Width: 80, DockerVersion: "29.8.0", Repeated: c.repeated, ExitCode: code}
				if err := RenderExplanation(&b, d, stderr, opts); err != nil {
					t.Fatal(err)
				}
				out := b.String()
				if !mode.color && strings.Contains(out, "\x1b") {
					t.Error("aucune séquence ESC attendue sans couleur")
				}
				golden(t, "us1", c.name+"."+mode.suffix, out)
			})
		}
	}
}
