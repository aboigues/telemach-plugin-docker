package exited

import (
	"errors"
	"regexp"
	"time"

	"github.com/aboigues/telemach-plugin-docker/internal/command"
	"github.com/aboigues/telemach-plugin-docker/internal/dockerread"
)

// Launch est un « docker run » lancé en arrière-plan, tel que tapé (research R5).
type Launch struct {
	Image      string
	Name       string // valeur de --name, si donnée
	AutoRemove bool   // --rm : le conteneur est supprimé à l'arrêt
}

// ParseLaunch reconnaît un « docker run » (ou « container run ») avec -d/--detach placé avant
// l'image ; toute autre ligne n'entraîne aucune lecture du démon.
func ParseLaunch(args []string) (Launch, bool) {
	c := command.Parse(args)
	if c.Subcommand != "run" || len(c.Positionals) == 0 {
		return Launch{}, false
	}
	_, short := c.Flags["d"]
	long, hasLong := c.Flags["detach"]
	if !short && (!hasLong || long == "false") {
		return Launch{}, false
	}
	rm, hasRm := c.Flags["rm"]
	return Launch{Image: c.Positionals[0], Name: c.Flags["name"], AutoRemove: hasRm && rm != "false"}, true
}

// WatchResult est l'issue d'un lancement en arrière-plan.
type WatchResult struct {
	Name    string
	Failed  bool          // arrêté avec un code ≠ 0
	Code    int           // code de sortie (si Failed)
	After   time.Duration // délai entre la création et l'arrêt
	Removed bool          // supprimé par --rm : son journal est perdu
}

// Match retrouve le conteneur lancé : premier « create » de cette image (et de ce nom, si
// --name a été donné), puis son « die ». Sans « die », il tourne toujours.
func Match(evs []dockerread.Event, l Launch) (WatchResult, bool) {
	var created *dockerread.Event
	for i, e := range evs {
		if e.Action == "create" && e.Image == l.Image && (l.Name == "" || e.Name == l.Name) {
			created = &evs[i]
			break
		}
	}
	if created == nil {
		return WatchResult{}, false
	}
	r := WatchResult{Name: created.Name}
	for _, e := range evs {
		if e.Action == "die" && e.ID == created.ID && e.ExitCode != 0 {
			r.Failed, r.Code, r.Removed = true, e.ExitCode, l.AutoRemove
			r.After = time.Duration(e.TimeNano - created.TimeNano)
			break
		}
	}
	return r, true
}

var sinceFormat = regexp.MustCompile(`^(\d{1,12})(?:[.,](\d{1,12}))?$`)

// ParseSince valide l'heure de début relevée par le shell ($EPOCHREALTIME), avec un point ou
// une virgule (bash en locale française), et la renvoie au format de « docker events » :
// décimales tronquées à la nanoseconde (zsh en donne 10).
func ParseSince(s string) (string, error) {
	m := sinceFormat.FindStringSubmatch(s)
	if m == nil {
		return "", errors.New("heure de début invalide")
	}
	if m[2] == "" {
		return m[1], nil
	}
	return m[1] + "." + m[2][:min(len(m[2]), 9)], nil
}
