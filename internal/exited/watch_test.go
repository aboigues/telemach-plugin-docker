package exited

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aboigues/telemach-plugin-docker/internal/dockerread"
)

// loadEvents lit les événements réels capturés par testdata/exited/capture.sh (« events ») :
// postgres sans mot de passe (code 1), alpine --rm « exit 3 », alpine --rm « true » (code 0),
// nginx qui continue de tourner.
func loadEvents(t *testing.T) ([]dockerread.Event, string) {
	t.Helper()
	dir := filepath.Join("..", "..", "testdata", "exited", "events")
	f, err := os.Open(filepath.Join(dir, "run-d.jsonl")) // #nosec G304 -- corpus de test
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close() //nolint:errcheck
	var evs []dockerread.Event
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var raw json.RawMessage = sc.Bytes()
		ev, err := dockerread.DecodeEvent(raw)
		if err != nil {
			t.Fatal(err)
		}
		evs = append(evs, ev)
	}
	t0, err := os.ReadFile(filepath.Join(dir, "run-d.t0")) // #nosec G304 -- corpus de test
	if err != nil {
		t.Fatal(err)
	}
	return evs, strings.TrimPrefix(strings.TrimSpace(string(t0)), "t0=")
}

func TestParseLaunch(t *testing.T) {
	cases := []struct {
		args        []string
		ok, rm      bool
		image, name string
	}{
		{[]string{"run", "-d", "--name", "essai-pg", "postgres:18-alpine"}, true, false, "postgres:18-alpine", "essai-pg"},
		{[]string{"run", "-dit", "--rm", "alpine:3.24", "sh"}, true, true, "alpine:3.24", ""},
		{[]string{"--context", "x", "container", "run", "--detach=true", "nginx:1.30"}, true, false, "nginx:1.30", ""},
		{[]string{"run", "--detach=false", "alpine:3.24"}, false, false, "", ""},
		{[]string{"run", "--rm", "alpine:3.24", "-d"}, false, false, "", ""}, // -d après l'image : donné au conteneur
		{[]string{"run", "alpine:3.24"}, false, false, "", ""},
		{[]string{"create", "-d", "alpine:3.24"}, false, false, "", ""},
		{[]string{"ps", "-a"}, false, false, "", ""},
	}
	for _, c := range cases {
		l, ok := ParseLaunch(c.args)
		if ok != c.ok || (ok && (l.Image != c.image || l.Name != c.name || l.AutoRemove != c.rm)) {
			t.Errorf("ParseLaunch(%q) = %+v, %v", c.args, l, ok)
		}
	}
}

func TestParseSince(t *testing.T) {
	for in, want := range map[string]string{
		"1792151265.045881":     "1792151265.045881",
		"1792151265,045881":     "1792151265.045881", // bash en locale française
		"1792151265":            "1792151265",
		"1792151265.04588163":   "1792151265.04588163",
		"1792151258.4049170017": "1792151258.404917001", // zsh : 10 décimales
	} {
		got, err := ParseSince(in)
		if err != nil || got != want {
			t.Errorf("ParseSince(%q) = %q, %v ; attendu %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "abc", "12;rm -rf /", "-5", "1.2.3"} {
		if _, err := ParseSince(bad); err == nil {
			t.Errorf("ParseSince(%q) : erreur attendue", bad)
		}
	}
}

func TestMatchRealEvents(t *testing.T) {
	evs, _ := loadEvents(t)
	cases := []struct {
		name    string
		launch  Launch
		failed  bool
		code    int
		removed bool
	}{
		{"postgres sans mot de passe", Launch{Image: "postgres:18-alpine", Name: "telemach-exited-ev-pg"}, true, 1, false},
		{"--rm sorti avec 3", Launch{Image: "alpine:3.24", AutoRemove: true}, true, 3, true},
		{"nginx toujours démarré", Launch{Image: "nginxinc/nginx-unprivileged:1.30-alpine", Name: "telemach-exited-ev-web"}, false, 0, false},
	}
	for _, c := range cases {
		r, ok := Match(evs, c.launch)
		if !ok {
			t.Errorf("%s : conteneur lancé non retrouvé", c.name)
			continue
		}
		if r.Failed != c.failed || (c.failed && (r.Code != c.code || r.Removed != c.removed)) {
			t.Errorf("%s : %+v", c.name, r)
		}
		if c.failed && (r.After <= 0 || r.After > 10*time.Second) {
			t.Errorf("%s : délai depuis le lancement = %v", c.name, r.After)
		}
	}
	// Deux lancements de alpine:3.24 : le premier « create » est celui de « exit 3 ».
	// Le second (« true », code 0), désigné seul, ne doit pas être signalé.
	var later []dockerread.Event
	seen := 0
	for _, e := range evs {
		if e.Image == "alpine:3.24" && e.Action == "create" {
			seen++
		}
		if seen >= 2 && e.Image == "alpine:3.24" {
			later = append(later, e)
		}
	}
	if r, ok := Match(later, Launch{Image: "alpine:3.24", AutoRemove: true}); !ok || r.Failed {
		t.Errorf("--rm sorti avec 0 : aucun signal attendu, obtenu %+v (%v)", r, ok)
	}
}
