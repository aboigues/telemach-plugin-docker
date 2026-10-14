//go:build integration

package integration

import (
	"path/filepath"
	"strings"
	"testing"
)

// V6 — docker compose up sur un fichier erroné : ligne exacte et correction affichées.
func TestV6ComposeErrors(t *testing.T) {
	h := newHarness(t)
	for name, want := range map[string][]string{
		"tab-indent":       {"compose.yaml:3:1", "Version corrigée", "YAML indentation"},
		"unknown-key-port": {"compose.yaml:4:5", "« ports »", "Version corrigée"},
	} {
		t.Run(name, func(t *testing.T) {
			dir, err := filepath.Abs(filepath.Join("..", "..", "testdata", "compose", name))
			if err != nil {
				t.Fatal(err)
			}
			stdout, stderr := h.shell(t, `cd "$DIR" && docker compose up --no-start; echo "rc=$?"`, "DIR="+dir)
			if strings.Contains(stdout, "rc=0") {
				t.Fatalf("docker compose aurait dû échouer :\n%s", stdout)
			}
			for _, w := range want {
				if !strings.Contains(stderr, w) {
					t.Errorf("explication sans %q :\n%s", w, stderr)
				}
			}
		})
	}
}
