package diagnose

import (
	"testing"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
	"github.com/aboigues/telemach-plugin-docker/internal/command"
)

// FuzzDiagnose vérifie que le diagnostic ne panique jamais, quelle que soit la sortie
// d'erreur (texte non fiable, éventuellement hostile).
func FuzzDiagnose(f *testing.F) {
	cat, err := catalog.Default()
	if err != nil {
		f.Fatal(err)
	}
	for _, seed := range []string{"", "Bind for 0.0.0.0:80 failed: port is already allocated", "No such container: x", "\x00\xff{{.port}}"} {
		f.Add(seed, 125)
	}
	f.Fuzz(func(t *testing.T, stderr string, code int) {
		d := Diagnose(Input{Command: command.Parse([]string{"run", "x"}), ExitCode: code, Stderr: stderr}, cat)
		if d.Entry == nil && d.Confidence != Unknown {
			t.Fatalf("diagnostic incohérent : %+v", d)
		}
	})
}
