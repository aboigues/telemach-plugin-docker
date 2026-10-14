package compose

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
	"github.com/aboigues/telemach-plugin-docker/internal/render"
)

var update = flag.Bool("update", false, "régénère les fichiers golden")

func TestRenderGolden(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"tab-indent", "unknown-key-port", "missing-colon", "undefined-depends-on", "plaintext-password", "valid"} {
		for _, mode := range []struct {
			suffix string
			color  bool
		}{{"sans-couleur", false}, {"couleur", true}} {
			t.Run(name+"/"+mode.suffix, func(t *testing.T) {
				dir := filepath.Join("..", "..", "testdata", "compose", name)
				issues := Analyze(dir, nil)
				files, _ := Locate(dir, nil)
				var b bytes.Buffer
				opts := render.Options{Theme: render.NewTheme(mode.color, true), Width: 80}
				if err := RenderIssues(&b, dir, files, issues, cat, opts); err != nil {
					t.Fatal(err)
				}
				got := b.String()
				if !mode.color && strings.Contains(got, "\x1b") {
					t.Error("aucune séquence ESC attendue sans couleur")
				}
				path := filepath.Join("..", "..", "testdata", "golden", "us2", name+"."+mode.suffix+".golden")
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
			})
		}
	}
}
