package exited

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
	"github.com/aboigues/telemach-plugin-docker/internal/dockerread"
	"github.com/aboigues/telemach-plugin-docker/internal/redact"
)

// Sources d'une cause, de la plus sûre à la plus générale.
const (
	SourceState   = "state"   // fait enregistré par Docker ou le noyau (OOM, erreur de démarrage)
	SourceLog     = "log"     // motif reconnu dans la fin du journal
	SourceCode    = "code"    // signification du code de sortie
	SourceUnknown = "unknown" // aucune cause précise : TELEMACH ne devine pas (FR-005)
)

// LogWindow est le nombre de dernières lignes du journal où un motif est cherché : une ligne
// plus ancienne a peu de chances d'expliquer l'arrêt (LL-010).
const LogWindow = 10

// Input est ce qui est connu du conteneur : son état et la fin de son journal.
type Input struct {
	Container dockerread.Container
	Log       dockerread.LogExcerpt
}

// Diagnosis est la cause retenue et les valeurs pour les gabarits du catalogue.
type Diagnosis struct {
	Entry    *catalog.Entry
	Source   string
	Captures map[string]string
}

// Diagnose cherche la cause de l'arrêt (research R7), dans cet ordre : manque de mémoire
// enregistré par le noyau ; motif sur l'erreur de démarrage enregistrée par Docker ; motif sur
// les dernières lignes du journal, de la plus récente à la plus ancienne ; erreur de démarrage
// sans motif ; code de sortie ; sinon cause inconnue.
func Diagnose(in Input, cat *catalog.Catalog, now time.Time) Diagnosis {
	c := in.Container
	d := Diagnosis{Captures: baseCaptures(c, now)}
	set := func(id, source string) Diagnosis {
		d.Entry, _ = cat.Get(id)
		d.Source = source
		return d
	}
	if c.OOMKilled {
		return set("exited.oom", SourceState)
	}
	if c.Error != "" {
		if e, caps, ok := matchLine(c.Error, cat); ok {
			return d.with(e, SourceState, caps, c.Error)
		}
	}
	lines := in.Log.Lines
	for i := len(lines) - 1; i >= 0 && i >= len(lines)-LogWindow; i-- {
		if e, caps, ok := matchLine(lines[i], cat); ok {
			return d.with(e, SourceLog, caps, lines[i])
		}
	}
	if c.Error != "" {
		return set("exited.start-error", SourceState)
	}
	for _, e := range cat.Entries {
		if e.Fallback == "exited" && containsCode(e.ExitCodes, c.ExitCode) {
			d.Entry, d.Source = e, SourceCode
			return d
		}
	}
	return set("exited.unknown", SourceUnknown)
}

func (d Diagnosis) with(e *catalog.Entry, source string, caps map[string]string, line string) Diagnosis {
	d.Entry, d.Source = e, source
	for k, v := range caps {
		d.Captures[k] = v
	}
	d.Captures["line"] = strings.TrimSpace(line)
	return d
}

// matchLine essaie les motifs des entrées « exited », dans l'ordre du catalogue.
func matchLine(line string, cat *catalog.Catalog) (*catalog.Entry, map[string]string, bool) {
	for _, e := range cat.Entries {
		if e.Fallback != "exited" {
			continue
		}
		for _, re := range e.Compiled() {
			m := re.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			caps := map[string]string{}
			for i, name := range re.SubexpNames() {
				if name != "" && m[i] != "" {
					caps[name] = redact.Redact(m[i])
				}
			}
			return e, caps, true
		}
	}
	return nil, nil, false
}

func containsCode(codes []int, code int) bool {
	for _, c := range codes {
		if c == code {
			return true
		}
	}
	return false
}

// baseCaptures prépare les valeurs communes à toutes les causes (déjà nettoyées et masquées
// par dockerread) ; jamais de valeur de variable d'environnement.
func baseCaptures(c dockerread.Container, now time.Time) map[string]string {
	caps := map[string]string{
		"name": c.Name, "image": c.Image, "code": strconv.Itoa(c.ExitCode), "command": c.Command,
		"error": c.Error, "restart_policy": c.RestartPolicy,
		"compose_project": c.ComposeProject, "compose_service": c.ComposeService,
	}
	if c.Status == "created" {
		caps["state"] = "created"
	}
	if !c.StartedAt.IsZero() && !c.FinishedAt.IsZero() && !c.FinishedAt.Before(c.StartedAt) {
		caps["lifetime"] = Duration(c.FinishedAt.Sub(c.StartedAt))
	}
	switch {
	case !c.FinishedAt.IsZero():
		caps["age"] = Age(now.Sub(c.FinishedAt))
	case !c.Created.IsZero():
		caps["age"] = Age(now.Sub(c.Created))
	}
	if c.RestartCount > 0 {
		caps["restart_count"] = strconv.Itoa(c.RestartCount)
	}
	if c.MemoryLimit > 0 {
		caps["memory_limit"] = Bytes(c.MemoryLimit)
	}
	return caps
}

// Duration exprime une durée courte en français : « 3 s », « 2 min », « 4 h », « 2 j ».
func Duration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%d s", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%d min", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d h", int(d.Hours()))
	default:
		return fmt.Sprintf("%d j", int(d.Hours()/24))
	}
}

// Age exprime une ancienneté : « à l'instant », « il y a 2 min »…
func Age(d time.Duration) string {
	if d < 5*time.Second {
		return "à l'instant"
	}
	return "il y a " + Duration(d)
}

// Bytes exprime une taille mémoire : « 8 Mo », « 1,5 Go ».
func Bytes(n int64) string {
	const mo, go_ = 1 << 20, 1 << 30
	if n >= go_ {
		return strings.Replace(strconv.FormatFloat(float64(n)/go_, 'f', -1, 64), ".", ",", 1) + " Go"
	}
	return strconv.FormatInt(n/mo, 10) + " Mo"
}
