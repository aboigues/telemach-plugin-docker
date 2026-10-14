package compose

import (
	"crypto/sha256"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fileState struct {
	sum   [32]byte
	mtime time.Time
}

func snapshot(t *testing.T, dir string) map[string]fileState {
	t.Helper()
	out := map[string]fileState{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(p) //nolint:gosec // test
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		out[p] = fileState{sum: sha256.Sum256(data), mtime: info.ModTime()}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestReadOnly vérifie que l'analyse ne modifie ni ne crée aucun fichier (FR-018).
func TestReadOnly(t *testing.T) {
	dirs, err := filepath.Glob(filepath.Join("..", "..", "testdata", "compose", "*"))
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range dirs {
		before := snapshot(t, dir)
		Analyze(dir, nil)
		after := snapshot(t, dir)
		if len(after) != len(before) {
			t.Errorf("%s : nombre de fichiers modifié (%d → %d)", dir, len(before), len(after))
		}
		for p, st := range before {
			if after[p] != st {
				t.Errorf("%s : fichier modifié", p)
			}
		}
	}
}
