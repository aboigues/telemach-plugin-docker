package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	// SchemaVersion est la version du format du fichier d'état.
	SchemaVersion = 1
	// RepeatWindow est la durée pendant laquelle une même erreur est considérée comme répétée.
	RepeatWindow = 10 * time.Minute
	fileName     = "last.json"
)

// LastDiagnosis est le dernier diagnostic, dont les captures sont déjà masquées.
type LastDiagnosis struct {
	SchemaVersion int
	Fingerprint   string
	EntryID       string
	Captures      map[string]string
	RecordedAt    time.Time
}

// Store lit et écrit l'état dans un répertoire privé.
type Store struct {
	Dir string
}

// DefaultDir renvoie $XDG_STATE_HOME/telemach, ou ~/.local/state/telemach par défaut.
func DefaultDir(env func(string) string) string {
	if d := env("XDG_STATE_HOME"); d != "" {
		return filepath.Join(d, "telemach")
	}
	return filepath.Join(env("HOME"), ".local", "state", "telemach")
}

// Load lit le dernier diagnostic. Un fichier absent, illisible ou d'un schéma inconnu
// est ignoré sans erreur.
func (s Store) Load() (LastDiagnosis, bool) {
	data, err := os.ReadFile(filepath.Join(s.Dir, fileName))
	if err != nil {
		return LastDiagnosis{}, false
	}
	var d LastDiagnosis
	if err := json.Unmarshal(data, &d); err != nil || d.SchemaVersion != SchemaVersion {
		return LastDiagnosis{}, false
	}
	return d, true
}

// Save écrit le diagnostic de façon atomique (fichier temporaire puis renommage), avec
// les droits 0700 pour le répertoire et 0600 pour le fichier.
func (s Store) Save(d LastDiagnosis) error {
	d.SchemaVersion = SchemaVersion
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(s.Dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.Dir, ".last-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) //nolint:errcheck // sans effet après le renommage
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close() //nolint:errcheck,gosec
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close() //nolint:errcheck,gosec
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(s.Dir, fileName))
}

// IsRepeat indique si l'empreinte est identique à celle du dernier diagnostic, enregistré
// il y a moins de RepeatWindow.
func (s Store) IsRepeat(fingerprint string, now time.Time) bool {
	d, ok := s.Load()
	if !ok || d.Fingerprint != fingerprint {
		return false
	}
	return now.Sub(d.RecordedAt) < RepeatWindow
}
