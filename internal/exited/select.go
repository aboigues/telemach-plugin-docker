package exited

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/aboigues/telemach-plugin-docker/internal/dockerread"
	"github.com/aboigues/telemach-plugin-docker/internal/textdist"
)

// MaxInspect est le nombre de candidats inspectés en un appel : de 12 à 25 ms par conteneur
// selon la charge (research R4, T031) ; 12 couvrent l'analyse et les 10 autres listés.
const MaxInspect = 12

// Lister est la partie de dockerread.Runner utilisée pour choisir les conteneurs.
type Lister interface {
	ListCandidates() ([]dockerread.Candidate, error)
	Inspect(ids ...string) ([]dockerread.Container, error)
}

var (
	// ErrRunning : le conteneur désigné est en cours d'exécution.
	ErrRunning = errors.New("conteneur en cours d'exécution")
	// ErrNotFailed : le conteneur désigné s'est arrêté avec le code 0.
	ErrNotFailed = errors.New("conteneur arrêté sans erreur")
)

// NotFoundError : aucun conteneur ne porte ce nom ; Suggestions propose des noms proches.
type NotFoundError struct {
	Name        string
	Suggestions []string
}

func (e *NotFoundError) Error() string { return "conteneur introuvable : " + e.Name }

// Select renvoie les conteneurs en erreur, du plus récemment arrêté au plus ancien : arrêtés
// ou en boucle de redémarrage avec un code ≠ 0, et créés dont le démarrage a échoué.
func Select(r Lister) ([]dockerread.Container, error) {
	cands, err := r.ListCandidates()
	if err != nil {
		return nil, err
	}
	// Le code de « Exited (n) » suffit à écarter les arrêts sans erreur. Un « Created » doit
	// être inspecté pour savoir si son démarrage a échoué : il ne prend que les places
	// restantes, pour que de simples « docker create » ne masquent pas de vrais arrêts.
	var ids, created []string
	for _, c := range cands {
		switch {
		case c.HasCode && c.Code != 0 && len(ids) < MaxInspect:
			ids = append(ids, c.ID)
		case c.State == "created":
			created = append(created, c.ID)
		}
	}
	ids = append(ids, created[:min(len(created), MaxInspect-len(ids))]...)
	if len(ids) == 0 {
		return nil, nil
	}
	all, err := r.Inspect(ids...)
	if err != nil {
		return nil, err
	}
	var out []dockerread.Container
	for _, c := range all {
		if failed(c) {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return stoppedAt(out[i]).After(stoppedAt(out[j])) })
	return out, nil
}

// Find renvoie le conteneur désigné par son nom ou un préfixe de son identifiant.
func Find(r Lister, name string) (dockerread.Container, error) {
	cs, err := r.Inspect(name)
	var ce *dockerread.CLIError
	if errors.As(err, &ce) && strings.Contains(strings.ToLower(ce.Stderr), "no such") {
		nf := &NotFoundError{Name: name}
		if cands, lerr := r.ListCandidates(); lerr == nil {
			names := make([]string, 0, len(cands))
			for _, c := range cands {
				names = append(names, c.Name)
			}
			if s, ok := textdist.Closest(name, names, 3); ok {
				nf.Suggestions = []string{s}
			}
		}
		return dockerread.Container{}, nf
	}
	if err != nil {
		return dockerread.Container{}, err
	}
	if len(cs) == 0 {
		return dockerread.Container{}, &NotFoundError{Name: name}
	}
	c := cs[0]
	switch {
	case failed(c):
		return c, nil
	case c.Status == "running" || c.Status == "paused":
		return c, ErrRunning
	default:
		return c, ErrNotFailed
	}
}

// failed indique un conteneur en erreur (code ≠ 0, démarrage échoué compris).
func failed(c dockerread.Container) bool {
	if c.ExitCode == 0 {
		return false
	}
	if c.Status == "created" {
		return c.Error != ""
	}
	return c.Status == "exited" || c.Status == "restarting" || c.Status == "dead"
}

// stoppedAt est le moment du dernier arrêt (la création pour un démarrage échoué).
func stoppedAt(c dockerread.Container) time.Time {
	if !c.FinishedAt.IsZero() {
		return c.FinishedAt
	}
	return c.Created
}
