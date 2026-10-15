package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	hintsName          = "hints.json"
	hintsSchemaVersion = 1
	// maxHints borne la taille de hints.json : au-delà, les entrées les plus anciennes sont
	// supprimées.
	maxHints = 200
	// hintTTL est la durée de conservation d'une empreinte, et la période pendant laquelle
	// l'information « image sans tag » n'est pas répétée pour la même image.
	hintTTL = 24 * time.Hour
)

// Display est la forme d'affichage d'un piège de ligne de commande.
type Display int

const (
	DisplayFull     Display = iota // explication complète
	DisplayReminder                // rappel d'une ligne (déjà affiché il y a moins de 10 minutes)
	DisplaySilent                  // rien (information déjà donnée pour cette image depuis moins de 24 h)
)

func (d Display) String() string {
	return [...]string{"complet", "rappel", "silence"}[d]
}

// HintStore mémorise les empreintes des pièges affichés en complet (feature 003). Les
// empreintes ne contiennent jamais de valeur saisie par l'apprenant.
type HintStore struct {
	Dir string
}

type hintsFile struct {
	SchemaVersion int
	Shown         map[string]time.Time
}

func (h HintStore) load() map[string]time.Time {
	data, err := os.ReadFile(filepath.Join(h.Dir, hintsName))
	if err != nil {
		return map[string]time.Time{}
	}
	var f hintsFile
	if err := json.Unmarshal(data, &f); err != nil || f.SchemaVersion != hintsSchemaVersion || f.Shown == nil {
		return map[string]time.Time{}
	}
	return f.Shown
}

// Decide choisit la forme d'affichage d'un piège selon son dernier affichage complet.
// imageRule vaut vrai pour l'information « image sans tag », jamais répétée avant 24 h.
func (h HintStore) Decide(fingerprint string, imageRule bool, now time.Time) Display {
	last, ok := h.load()[fingerprint]
	switch {
	case !ok:
		return DisplayFull
	case imageRule && now.Sub(last) < hintTTL:
		return DisplaySilent
	case !imageRule && now.Sub(last) < RepeatWindow:
		return DisplayReminder
	}
	return DisplayFull
}

// Record enregistre l'affichage complet des empreintes données, purge les entrées de plus
// de 24 h et ne garde que les maxHints plus récentes.
func (h HintStore) Record(fingerprints []string, now time.Time) error {
	shown := h.load()
	for _, fp := range fingerprints {
		shown[fp] = now
	}
	for fp, at := range shown {
		if now.Sub(at) > hintTTL {
			delete(shown, fp)
		}
	}
	if len(shown) > maxHints {
		keys := make([]string, 0, len(shown))
		for fp := range shown {
			keys = append(keys, fp)
		}
		sort.Slice(keys, func(i, j int) bool { return shown[keys[i]].After(shown[keys[j]]) })
		for _, fp := range keys[maxHints:] {
			delete(shown, fp)
		}
	}
	data, err := json.Marshal(hintsFile{SchemaVersion: hintsSchemaVersion, Shown: shown})
	if err != nil {
		return err
	}
	return writePrivate(h.Dir, hintsName, data)
}
