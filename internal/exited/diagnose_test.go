package exited

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
	"github.com/aboigues/telemach-plugin-docker/internal/dockerread"
)

type expected struct {
	EntryID  string            `yaml:"entry_id"`
	Source   string            `yaml:"source"`
	Captures map[string]string `yaml:"captures"`
}

// loadCase lit un cas du corpus testdata/exited/<nom>/ comme le ferait dockerread.
func loadCase(t *testing.T, dir string) (Input, expected) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "inspect.json")) // #nosec G304 -- corpus de test
	if err != nil {
		t.Fatal(err)
	}
	cs, err := dockerread.DecodeInspect(raw)
	if err != nil || len(cs) != 1 {
		t.Fatalf("%s : inspect.json illisible : %v", dir, err)
	}
	logs, err := os.ReadFile(filepath.Join(dir, "logs.txt")) // #nosec G304 -- corpus de test
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, l := range strings.Split(strings.TrimRight(string(logs), "\n"), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	var want expected
	data, err := os.ReadFile(filepath.Join(dir, "expected.yaml")) // #nosec G304 -- corpus de test
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	return Input{Container: cs[0], Log: dockerread.LogExcerpt{Lines: lines}}, want
}

// TestCorpus vérifie SC-001 : au moins 90 % de causes correctes, et jamais de cause précise
// pour un cas attendu « unknown ».
func TestCorpus(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	dirs, _ := filepath.Glob(filepath.Join("..", "..", "testdata", "exited", "*", "expected.yaml"))
	if len(dirs) < 12 {
		t.Fatalf("corpus trop petit : %d cas (12 au moins, SC-001)", len(dirs))
	}
	sort.Strings(dirs)
	now := time.Date(2026, 10, 16, 14, 0, 0, 0, time.UTC)
	correct := 0
	for _, f := range dirs {
		dir := filepath.Dir(f)
		name := filepath.Base(dir)
		in, want := loadCase(t, dir)
		d := Diagnose(in, cat, now)
		ok := d.Entry != nil && d.Entry.ID == want.EntryID && d.Source == want.Source
		for k, v := range want.Captures {
			if d.Captures[k] != v {
				ok = false
				t.Errorf("%s : capture %s = %q, attendu %q", name, k, d.Captures[k], v)
			}
		}
		if ok {
			correct++
		} else {
			got := "aucune"
			if d.Entry != nil {
				got = d.Entry.ID
			}
			t.Errorf("%s : %s (%s), attendu %s (%s)", name, got, d.Source, want.EntryID, want.Source)
		}
		if want.Source == "unknown" && d.Source != "unknown" {
			t.Errorf("%s : cause précise donnée à un cas inconnu (FR-005)", name)
		}
	}
	t.Logf("corpus : %d/%d causes correctes", correct, len(dirs))
	if correct*10 < len(dirs)*9 {
		t.Errorf("SC-001 : %d/%d < 90 %%", correct, len(dirs))
	}
}

// Ordre de décision : OOMKilled (fait enregistré par le noyau) l'emporte sur un motif du
// journal ; un motif ancien (au-delà des 10 dernières lignes) ne compte pas.
func TestDecisionOrder(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	oom := Input{
		Container: dockerread.Container{Name: "x", Image: "alpine:3.24", ExitCode: 137, OOMKilled: true, MemoryLimit: 64 << 20},
		Log:       dockerread.LogExcerpt{Lines: []string{"sh: can't create /etc/x: Permission denied"}},
	}
	if d := Diagnose(oom, cat, now); d.Entry == nil || d.Entry.ID != "exited.oom" || d.Captures["memory_limit"] != "64 Mo" {
		t.Errorf("OOM attendu en premier : %+v", d)
	}
	lines := []string{"Error: listen tcp :8080: bind: address already in use"}
	for i := 0; i < 10; i++ {
		lines = append(lines, "ligne normale")
	}
	old := Input{Container: dockerread.Container{Name: "x", ExitCode: 1}, Log: dockerread.LogExcerpt{Lines: lines}}
	if d := Diagnose(old, cat, now); d.Entry == nil || d.Entry.ID != "exited.code-1" {
		t.Errorf("motif ancien ignoré attendu : %+v", d.Entry)
	}
}

// Captures de contexte : durée de vie, ancienneté, redémarrages, Compose ; aucune valeur
// de variable d'environnement.
func TestCaptures(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	finished := time.Date(2026, 10, 16, 13, 58, 0, 0, time.UTC)
	in := Input{Container: dockerread.Container{
		Name: "api", Image: "app:1.0", ExitCode: 1, StartedAt: finished.Add(-3 * time.Second), FinishedAt: finished,
		RestartCount: 2, RestartPolicy: "always", ComposeProject: "boutique", ComposeService: "api",
		EnvNames: dockerread.EnvNames{"DB_PASSWORD"},
	}}
	d := Diagnose(in, cat, finished.Add(2*time.Minute))
	for k, v := range map[string]string{
		"name": "api", "image": "app:1.0", "code": "1", "lifetime": "3 s", "age": "il y a 2 min",
		"restart_count": "2", "restart_policy": "always", "compose_project": "boutique", "compose_service": "api",
	} {
		if d.Captures[k] != v {
			t.Errorf("capture %s = %q, attendu %q", k, d.Captures[k], v)
		}
	}
	b, _ := json.Marshal(d.Captures)
	if strings.Contains(string(b), "DB_PASSWORD=") {
		t.Errorf("valeur de variable dans les captures : %s", b)
	}
}
