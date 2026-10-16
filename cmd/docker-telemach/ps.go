package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
	"github.com/aboigues/telemach-plugin-docker/internal/command"
	"github.com/aboigues/telemach-plugin-docker/internal/diagnose"
	"github.com/aboigues/telemach-plugin-docker/internal/dockerread"
	"github.com/aboigues/telemach-plugin-docker/internal/exited"
	"github.com/aboigues/telemach-plugin-docker/internal/render"
	"github.com/aboigues/telemach-plugin-docker/internal/state"
)

// maxOthers est le nombre d'autres conteneurs en erreur listés sous l'analyse (FR-002).
const maxOthers = 10

// psCommand explique le dernier conteneur arrêté en erreur (feature 004, issue #8). Seules
// lectures du démon : ps, inspect et logs (FR-010). Codes de sortie : 0 analyse affichée ou
// rien à signaler, 1 démon injoignable ou conteneur désigné introuvable, 2 usage invalide.
func (a *app) psCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ps [CONTENEUR]",
		Short: "Explique pourquoi le dernier conteneur s'est arrêté en erreur",
		Long: `Retrouve le dernier conteneur arrêté avec un code de sortie différent de 0 (ou celui que
tu désignes par son nom), montre la fin de son journal et explique la cause probable.

TELEMACH ne fait que lire l'état des conteneurs et la fin de leur journal : il ne démarre,
n'arrête, ne supprime ni ne modifie aucun conteneur.

Exemple : docker telemach ps`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error { return a.ps(args) },
	}
}

func (a *app) ps(args []string) error {
	theme, w, width := a.output(a.stderr)
	opts := render.Options{Theme: theme, Width: width}
	r := dockerread.Runner{Bin: a.dockerBin, Global: a.dockerGlobal}

	var target dockerread.Container
	var others []dockerread.Container
	if len(args) == 1 {
		c, err := exited.Find(r, args[0])
		if err != nil {
			return a.psFindError(w, theme, width, args[0], c, err)
		}
		target = c
	} else {
		list, err := exited.Select(r)
		if err != nil {
			return a.daemonError(w, opts, err)
		}
		if len(list) == 0 {
			return render.RenderNoExited(w, opts)
		}
		target, others = list[0], list[1:min(len(list), 1+maxOthers)]
	}

	logs, err := r.Logs(target.ID, 200)
	if err != nil {
		return a.daemonError(w, opts, err)
	}
	cat, err := catalog.Default()
	if err != nil {
		return err
	}
	now := a.now()
	d := exited.Diagnose(exited.Input{Container: target, Log: logs}, cat, now)
	view := render.ExitedView{Entry: d.Entry, Captures: d.Captures, Log: tail(logs.Lines, state.MaxLogExcerpt)}
	if logs.Unavailable {
		view.LogUnavailable = logs.Reason
	}
	for _, o := range others {
		od := exited.Diagnose(exited.Input{Container: o}, cat, now)
		line := render.OtherLine{Name: o.Name, Code: o.ExitCode, Age: od.Captures["age"]}
		if od.Entry != nil {
			line.Summary = catalog.Render(od.Entry.Summary, od.Captures)
		}
		view.Others = append(view.Others, line)
	}
	if err := render.RenderExited(w, view, opts); err != nil {
		return err
	}
	entryID := ""
	if d.Entry != nil {
		entryID = d.Entry.ID
	}
	// L'ancienneté (« il y a 2 min ») vieillirait : « last » la recalcule dans son en-tête.
	saved := make(map[string]string, len(d.Captures))
	for k, v := range d.Captures {
		if k != "age" {
			saved[k] = v
		}
	}
	_ = a.store.Save(state.LastDiagnosis{
		Kind: state.KindExited, EntryID: entryID, Captures: saved, RecordedAt: now,
		Container:  state.ContainerRef{Name: target.Name, ID: shortID(target.ID)},
		LogExcerpt: view.Log, Fingerprint: "exited:" + target.ID,
	})
	return nil
}

// psFindError explique pourquoi le conteneur désigné ne peut pas être analysé (code 1).
func (a *app) psFindError(w io.Writer, theme render.Theme, width int, name string, c dockerread.Container, err error) error {
	var nf *exited.NotFoundError
	var msg string
	switch {
	case errors.As(err, &nf):
		msg = fmt.Sprintf("Aucun conteneur ne s'appelle « %s ».", name)
		if len(nf.Suggestions) > 0 {
			msg += fmt.Sprintf(" Tu pensais peut-être à « %s » ?", nf.Suggestions[0])
		}
		msg += " Un conteneur lancé avec --rm est supprimé à l'arrêt."
	case errors.Is(err, exited.ErrRunning):
		msg = fmt.Sprintf("Le conteneur « %s » est en cours d'exécution : rien à analyser. Son journal : docker logs %s", c.Name, c.Name)
	case errors.Is(err, exited.ErrNotFailed):
		msg = fmt.Sprintf("Le conteneur « %s » s'est arrêté sans erreur (code 0) : rien à analyser.", c.Name)
	default:
		return a.daemonError(w, render.Options{Theme: theme, Width: width}, err)
	}
	fmt.Fprintln(w, theme.Title(render.Wrap(render.SymbolError+" "+msg, width, "   ")[3:]))
	return exitError{code: 1}
}

// daemonError explique un échec de lecture : erreur de la CLI (démon arrêté, permission
// refusée…) par le diagnostic habituel des erreurs, délai dépassé par un message dédié.
func (a *app) daemonError(w io.Writer, opts render.Options, err error) error {
	var ce *dockerread.CLIError
	if errors.As(err, &ce) {
		cat, _ := catalog.Default()
		in := diagnose.Input{Command: command.Parse([]string{"ps", "-a"}), ExitCode: ce.Code, Stderr: ce.Stderr}
		opts.ExitCode = ce.Code
		if _, rerr := a.renderDiagnosis(w, in, diagnose.Diagnose(in, cat), ce.Stderr, opts); rerr != nil {
			return rerr
		}
		return exitError{code: 1}
	}
	if errors.Is(err, dockerread.ErrTimeout) {
		fmt.Fprintln(w, opts.Theme.Title(render.SymbolError+" Le démon Docker ne répond pas (délai de 10 s dépassé)."))
		fmt.Fprintln(w, render.Wrap("Vérifie qu'il est démarré avec « docker info », puis relance la commande.", opts.Width, "   "))
		return exitError{code: 1}
	}
	return err
}

func tail(lines []string, n int) []string {
	if len(lines) > n {
		return lines[len(lines)-n:]
	}
	return lines
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
