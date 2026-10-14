package diagnose

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
	"github.com/aboigues/telemach-plugin-docker/internal/command"
	"github.com/aboigues/telemach-plugin-docker/internal/redact"
	"github.com/aboigues/telemach-plugin-docker/internal/sanitize"
)

type expectation struct {
	EntryID  string            `yaml:"entry_id"`
	Origin   Origin            `yaml:"origin"`
	Captures map[string]string `yaml:"captures"`
}

type corpusCase struct {
	name     string
	args     []string
	exitCode int
	stderr   string
	want     expectation
}

func loadCorpus(t *testing.T) []corpusCase {
	t.Helper()
	dir := filepath.Join("..", "..", "testdata", "errors")
	files, err := filepath.Glob(filepath.Join(dir, "*.expected.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []corpusCase
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".expected.yaml")
		read := func(ext string) string {
			data, err := os.ReadFile(filepath.Join(dir, name+ext))
			if err != nil {
				t.Fatalf("%s : %v", name, err)
			}
			return string(data)
		}
		var want expectation
		if err := yaml.Unmarshal([]byte(read(".expected.yaml")), &want); err != nil {
			t.Fatalf("%s : %v", name, err)
		}
		code, err := strconv.Atoi(strings.TrimSpace(read(".exit")))
		if err != nil {
			t.Fatalf("%s : code de sortie invalide : %v", name, err)
		}
		cases = append(cases, corpusCase{
			name:     name,
			args:     strings.Fields(read(".args")),
			exitCode: code,
			stderr:   redact.Redact(sanitize.Clean(read(".stderr.txt"))),
			want:     want,
		})
	}
	return cases
}

// TestCorpus vérifie le taux de reconnaissance du corpus de vraies erreurs (SC-001 ≥ 90 %)
// et qu'aucune erreur inconnue ne reçoit de cause (SC-005).
func TestCorpus(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatalf("catalogue invalide : %v", err)
	}
	cases := loadCorpus(t)
	if len(cases) < 30 {
		t.Fatalf("le corpus doit contenir au moins 30 erreurs, %d trouvées", len(cases))
	}
	ok := 0
	for _, c := range cases {
		d := Diagnose(Input{Command: command.Parse(c.args), ExitCode: c.exitCode, Stderr: c.stderr}, cat)
		got := ""
		if d.Entry != nil {
			got = d.Entry.ID
		}
		var problems []string
		if got != c.want.EntryID {
			problems = append(problems, "entrée "+strconv.Quote(got)+", attendu "+strconv.Quote(c.want.EntryID))
		}
		if d.Origin != c.want.Origin {
			problems = append(problems, "origine "+string(d.Origin)+", attendu "+string(c.want.Origin))
		}
		for k, v := range c.want.Captures {
			if d.Captures[k] != v {
				problems = append(problems, "capture "+k+"="+strconv.Quote(d.Captures[k])+", attendu "+strconv.Quote(v))
			}
		}
		if c.want.EntryID == "" && d.Entry != nil {
			problems = append(problems, "une erreur inconnue ne doit recevoir aucune cause")
		}
		if len(problems) == 0 {
			ok++
			continue
		}
		t.Logf("✖ %s : %s", c.name, strings.Join(problems, " ; "))
	}
	rate := float64(ok) / float64(len(cases)) * 100
	t.Logf("taux de reconnaissance exacte : %d/%d = %.1f %%", ok, len(cases), rate)
	if rate < 90 {
		t.Errorf("taux de reconnaissance %.1f %% inférieur à 90 %% (SC-001)", rate)
	}
}
