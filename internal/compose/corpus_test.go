package compose

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

type expected struct {
	Kind       Kind   `yaml:"kind"`
	File       string `yaml:"file"`
	Line       int    `yaml:"line"`
	Column     int    `yaml:"column"`
	Service    string `yaml:"service"`
	KeyPath    string `yaml:"key_path"`
	Suggestion string `yaml:"suggestion"`
}

// TestCorpus vérifie la localisation exacte des problèmes (SC-002 ≥ 90 %) et l'absence
// de secret en clair dans les extraits affichés (SC-006).
func TestCorpus(t *testing.T) {
	dirs, err := filepath.Glob(filepath.Join("..", "..", "testdata", "compose", "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) < 20 {
		t.Fatalf("le corpus doit contenir au moins 20 cas, %d trouvés", len(dirs))
	}
	if _, set := os.LookupEnv("TELEMACH_TEST_TAG"); set {
		t.Skip("TELEMACH_TEST_TAG ne doit pas être définie pour le cas unresolved-variable")
	}

	ok, total := 0, 0
	for _, dir := range dirs {
		name := filepath.Base(dir)
		data, err := os.ReadFile(filepath.Join(dir, "expected.yaml"))
		if err != nil {
			t.Fatalf("%s : %v", name, err)
		}
		var want expected
		if err := yaml.Unmarshal(data, &want); err != nil {
			t.Fatalf("%s : %v", name, err)
		}
		total++
		issues := Analyze(dir, nil)
		if msg := compare(want, issues); msg != "" {
			t.Logf("✖ %s : %s", name, msg)
		} else {
			ok++
		}
		for _, is := range issues {
			for _, l := range append(is.Excerpt, is.Fixed...) {
				if strings.Contains(l.Text, "example") && name == "plaintext-password" {
					t.Errorf("%s : secret affiché en clair : %q", name, l.Text)
				}
			}
		}
	}
	rate := float64(ok) / float64(total) * 100
	t.Logf("localisation exacte : %d/%d = %.1f %%", ok, total, rate)
	if rate < 90 {
		t.Errorf("taux de localisation %.1f %% inférieur à 90 %% (SC-002)", rate)
	}
}

func compare(want expected, issues []Issue) string {
	if want.Kind == "" {
		if len(issues) > 0 {
			return "aucun problème attendu, obtenu " + string(issues[0].Kind) + " : " + issues[0].Detail
		}
		return ""
	}
	if len(issues) == 0 {
		return "aucun problème détecté, attendu " + string(want.Kind)
	}
	got := issues[0]
	var diffs []string
	check := func(field string, g, w any) {
		if g != w {
			diffs = append(diffs, field+"="+toString(g)+" (attendu "+toString(w)+")")
		}
	}
	check("kind", got.Kind, want.Kind)
	if want.File != "" {
		check("file", filepath.Base(got.File), want.File)
	}
	check("line", got.Line, want.Line)
	check("column", got.Column, want.Column)
	check("service", got.Service, want.Service)
	check("key_path", got.KeyPath, want.KeyPath)
	check("suggestion", got.Suggestion, want.Suggestion)
	if len(diffs) > 0 {
		return strings.Join(diffs, ", ") + " — " + got.Detail
	}
	return ""
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return "«" + x + "»"
	case Kind:
		return string(x)
	case int:
		return strconv.Itoa(x)
	}
	return ""
}

// TestUnreadablePermissions vérifie un fichier sans droit de lecture (créé ici, car git
// ne conserve pas ces droits).
func TestUnreadablePermissions(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root peut lire un fichier en 0000")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")
	if err := os.WriteFile(path, []byte("services: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
	issues := Analyze(dir, nil)
	if len(issues) == 0 || issues[0].Kind != KindFileUnreadable || !strings.Contains(issues[0].Detail, "droits") {
		t.Errorf("file-unreadable (droits) attendu : %+v", issues)
	}
}
