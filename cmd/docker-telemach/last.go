package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
	"github.com/aboigues/telemach-plugin-docker/internal/command"
	"github.com/aboigues/telemach-plugin-docker/internal/diagnose"
	"github.com/aboigues/telemach-plugin-docker/internal/render"
	"github.com/aboigues/telemach-plugin-docker/internal/state"
)

// lastCommand réaffiche l'explication complète de la dernière erreur (FR-009a), par exemple
// après une version courte pour une erreur répétée, ou après avoir effacé le terminal.
func (a *app) lastCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "last",
		Short: "Réaffiche l'explication complète de la dernière erreur ou du dernier avertissement",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			theme, w, width := a.output(a.stderr)
			a.shellStatusWarning(w, theme, width)
			last, ok := a.store.Load()
			if !ok {
				fmt.Fprintln(w, theme.Hint("Aucune erreur récente à réafficher."))
				fmt.Fprintln(w, render.Wrap("Dès qu'une commande docker échoue avec l'intégration shell activée, "+
					"« docker telemach last » réaffiche son explication complète.", width, "   "))
				return nil
			}
			if last.IsExited() {
				// Analyse d'un conteneur arrêté (feature 004) : réaffichée sans interroger le démon.
				fmt.Fprintln(w, theme.Dim(lastHeader(last, a.now())))
				view := render.ExitedView{Captures: last.Captures, Log: last.LogExcerpt}
				if cat, err := catalog.Default(); err == nil {
					view.Entry, _ = cat.Get(last.EntryID)
				}
				return render.RenderExited(w, view, render.Options{Theme: theme, Width: width})
			}
			if last.IsCmdline() {
				// Avertissement d'une commande réussie (feature 003) : même rendu que lint.
				fmt.Fprintln(w, theme.Dim(lastHeader(last, a.now())))
				return render.RenderFindings(w, findingItems(last.Findings), render.Options{Theme: theme, Width: width})
			}
			cmd := command.Command{
				Family:       command.Family(last.Family),
				Subcommand:   last.Subcommand,
				Flags:        map[string]string{},
				ComposeFiles: last.ComposeFiles,
			}
			if last.ComposeDir != "" {
				cmd.Flags["project-directory"] = last.ComposeDir
			}
			d := diagnose.Diagnosis{
				Confidence:  diagnose.Confidence(last.Confidence),
				Origin:      diagnose.Origin(last.Origin),
				Captures:    last.Captures,
				Fingerprint: last.Fingerprint,
			}
			if d.Captures == nil {
				d.Captures = map[string]string{}
			}
			if cat, err := catalog.Default(); err == nil && last.MatchedEntryID != "" {
				if e, ok := cat.Get(last.MatchedEntryID); ok {
					d.Entry = e
				}
			}
			// Quand et pour quelle commande : une erreur ancienne ne doit pas passer pour la
			// dernière commande tapée (issue #7).
			fmt.Fprintln(w, theme.Dim(lastHeader(last, a.now())))
			in := diagnose.Input{Command: cmd, ExitCode: last.ExitCode, Stderr: last.Stderr, Interactive: last.Interactive}
			opts := render.Options{
				Theme: theme, Width: width, DockerVersion: last.DockerVersion,
				Interactive: last.Interactive, ExitCode: last.ExitCode,
			}
			// Même affichage que explain : le piège de la ligne de commande, cause probable,
			// avant l'explication de l'erreur, ou à sa place si l'erreur n'est pas reconnue.
			items := findingItems(last.Findings)
			if len(items) > 0 && d.Entry == nil && cmd.Family != command.FamilyCompose {
				fmt.Fprintln(w)
				if err := render.RenderFindings(w, items, opts); err != nil {
					return err
				}
				return render.RenderOriginal(w, last.Stderr, opts)
			}
			if len(items) > 0 {
				if err := render.RenderInline(w, items, opts); err != nil {
					return err
				}
			}
			_, err := a.renderDiagnosis(w, in, d, last.Stderr, opts)
			return err
		},
	}
}

// findingItems reconstruit les pièges enregistrés (captures déjà nettoyées et masquées).
func findingItems(records []state.FindingRecord) []render.LintItem {
	if len(records) == 0 {
		return nil
	}
	cat, err := catalog.Default()
	if err != nil {
		return nil
	}
	var items []render.LintItem
	for _, r := range records {
		if e, ok := cat.Get(r.EntryID); ok {
			items = append(items, render.LintItem{Entry: e, Captures: r.Captures})
		}
	}
	return items
}

// maxHeaderCommand limite la longueur de la commande affichée dans l'en-tête.
const maxHeaderCommand = 120

// lastHeader décrit l'erreur réaffichée : date, ancienneté et commande (arguments masqués).
func lastHeader(last state.LastDiagnosis, now time.Time) string {
	command := "docker"
	switch {
	case last.IsExited():
		command = "conteneur " + last.Container.Name
	case len(last.Args) > 0:
		command += " " + strings.Join(last.Args, " ")
	case last.Family == "compose":
		command += " compose " + last.Subcommand
	case last.Subcommand != "":
		command += " " + last.Subcommand
	}
	if r := []rune(command); len(r) > maxHeaderCommand {
		command = string(r[:maxHeaderCommand-1]) + "…"
	}
	if last.RecordedAt.IsZero() {
		return render.LabelLast + " de la dernière erreur enregistrée : " + command
	}
	what := "erreur enregistrée"
	switch {
	case last.IsCmdline():
		what = "avertissement enregistré"
	case last.IsExited():
		what = "analyse enregistrée"
	}
	at := last.RecordedAt.In(now.Location())
	return fmt.Sprintf("%s — %s le %s à %s (%s) : %s", render.LabelLast, what,
		at.Format("02/01/2006"), at.Format("15:04"), age(now.Sub(last.RecordedAt)), command)
}

// age exprime une durée écoulée en français, de façon approximative.
func age(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "à l'instant"
	case d < time.Hour:
		return fmt.Sprintf("il y a %d min", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("il y a %d h", int(d.Hours()))
	default:
		return fmt.Sprintf("il y a %d j", int(d.Hours()/24))
	}
}
