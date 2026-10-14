package render

import (
	"bytes"
	"testing"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
)

func TestDeepenGolden(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"run.port-already-allocated", "compose.plaintext-secret", "build.copy-source-missing"} {
		e, ok := cat.Get(id)
		if !ok {
			t.Fatalf("entrée %s absente", id)
		}
		for _, mode := range []struct {
			suffix string
			color  bool
		}{{"sans-couleur", false}, {"couleur", true}} {
			t.Run(id+"/"+mode.suffix, func(t *testing.T) {
				var b bytes.Buffer
				opts := Options{Theme: NewTheme(mode.color, true), Width: 80}
				if err := RenderDeepen(&b, e, map[string]string{"port": "8080"}, true, opts); err != nil {
					t.Fatal(err)
				}
				golden(t, "us3", id+"."+mode.suffix, b.String())
			})
		}
	}
}
