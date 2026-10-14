package compose

import (
	"errors"
	"os"
	"path/filepath"
)

// Noms recherchés par défaut, dans l'ordre de Docker Compose, avec leur fichier
// de surcharge (override) chargé automatiquement s'il existe.
var defaultFiles = []struct{ base, override string }{
	{"compose.yaml", "compose.override.yaml"},
	{"compose.yml", "compose.override.yml"},
	{"docker-compose.yaml", "docker-compose.override.yaml"},
	{"docker-compose.yml", "docker-compose.override.yml"},
}

// DefaultFileNames renvoie les noms de fichiers recherchés par défaut.
func DefaultFileNames() []string {
	out := make([]string, len(defaultFiles))
	for i, f := range defaultFiles {
		out[i] = f.base
	}
	return out
}

// ErrNotFound signale qu'aucun fichier Compose n'a été trouvé.
var ErrNotFound = errors.New("aucun fichier Compose trouvé")

// Locate renvoie les fichiers Compose à analyser : ceux passés par -f (relatifs à dir),
// sinon le premier nom par défaut trouvé dans dir, suivi de son fichier de surcharge.
// Aucun fichier n'est ouvert en écriture.
func Locate(dir string, files []string) ([]string, error) {
	if len(files) > 0 {
		out := make([]string, len(files))
		for i, f := range files {
			if !filepath.IsAbs(f) {
				f = filepath.Join(dir, f)
			}
			out[i] = f
		}
		return out, nil
	}
	for _, f := range defaultFiles {
		base := filepath.Join(dir, f.base)
		if _, err := os.Stat(base); err != nil {
			continue
		}
		out := []string{base}
		if _, err := os.Stat(filepath.Join(dir, f.override)); err == nil {
			out = append(out, filepath.Join(dir, f.override))
		}
		return out, nil
	}
	return nil, ErrNotFound
}
