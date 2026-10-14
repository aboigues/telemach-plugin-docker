package diagnose

import (
	"testing"
	"time"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
	"github.com/aboigues/telemach-plugin-docker/internal/command"
)

// TestDiagnoseLatency vérifie SC-003 : chaque erreur du corpus est diagnostiquée bien en
// dessous de 2 s (objectif interne : 200 ms, chargement du catalogue compris).
func TestDiagnoseLatency(t *testing.T) {
	cases := loadCorpus(t)
	for _, c := range cases {
		start := time.Now()
		cat, err := catalog.Default()
		if err != nil {
			t.Fatal(err)
		}
		Diagnose(Input{Command: command.Parse(c.args), ExitCode: c.exitCode, Stderr: c.stderr}, cat)
		if d := time.Since(start); d > 2*time.Second {
			t.Errorf("%s : %s (> 2 s, SC-003)", c.name, d)
		} else if d > 200*time.Millisecond {
			t.Logf("%s : %s (> 200 ms, objectif interne)", c.name, d)
		}
	}
}

func BenchmarkDiagnose(b *testing.B) {
	cases := loadCorpus(&testing.T{})
	cat, err := catalog.Default()
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := cases[i%len(cases)]
		Diagnose(Input{Command: command.Parse(c.args), ExitCode: c.exitCode, Stderr: c.stderr}, cat)
	}
}

func BenchmarkCatalogLoad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := catalog.Default(); err != nil {
			b.Fatal(err)
		}
	}
}
