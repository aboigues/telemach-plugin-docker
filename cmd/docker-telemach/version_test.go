package main

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	for _, flag := range []string{"--version", "-v"} {
		t.Run(flag, func(t *testing.T) {
			a, out, _ := testApp(t, nil)
			if code := run(a, []string{flag}); code != 0 {
				t.Fatalf("code = %d", code)
			}
			want := "TELEMACH PLUGIN DOCKER 1.2.3\nLicence Apache-2.0 — https://www.telemach-learning.fr\n"
			if out.String() != want {
				t.Errorf("sortie :\n%q\nattendu :\n%q", out.String(), want)
			}
		})
	}
}

func TestVersionInHelp(t *testing.T) {
	a, out, _ := testApp(t, nil)
	run(a, []string{"--help"})
	if !strings.Contains(out.String(), "affiche la version") {
		t.Errorf("option --version absente de l'aide :\n%s", out.String())
	}
}

func TestResolveVersion(t *testing.T) {
	buildInfo := func(v string) func() (*debug.BuildInfo, bool) {
		return func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Version: v}}, true
		}
	}
	noBuildInfo := func() (*debug.BuildInfo, bool) { return nil, false }

	tests := []struct {
		name     string
		injected string
		read     func() (*debug.BuildInfo, bool)
		want     string
	}{
		{"injectée à la compilation", "1.2.3", buildInfo("v9.9.9"), "1.2.3"},
		{"tag git", "dev", buildInfo("v0.2.0"), "0.2.0"},
		{"pseudo-version", "dev", buildInfo("v0.0.0-20261015090000-babdd81a1b2c+dirty"), "0.0.0-20261015090000-babdd81a1b2c+dirty"},
		{"sans information VCS", "dev", buildInfo("(devel)"), "dev"},
		{"sans build info", "dev", noBuildInfo, "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveVersion(tt.injected, tt.read); got != tt.want {
				t.Errorf("resolveVersion = %q, attendu %q", got, tt.want)
			}
		})
	}
}
