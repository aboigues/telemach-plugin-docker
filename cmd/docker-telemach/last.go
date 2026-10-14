package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
	"github.com/aboigues/telemach-plugin-docker/internal/command"
	"github.com/aboigues/telemach-plugin-docker/internal/diagnose"
	"github.com/aboigues/telemach-plugin-docker/internal/render"
)

// lastCommand réaffiche l'explication complète de la dernière erreur (FR-009a), par exemple
// après une version courte pour une erreur répétée, ou après avoir effacé le terminal.
func (a *app) lastCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "last",
		Short: "Réaffiche l'explication complète de la dernière erreur",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			theme, w, width := a.output(a.stderr)
			last, ok := a.store.Load()
			if !ok {
				fmt.Fprintln(w, theme.Hint("Aucune erreur récente à réafficher."))
				fmt.Fprintln(w, render.Wrap("Dès qu'une commande docker échoue avec l'intégration shell activée, "+
					"« docker telemach last » réaffiche son explication complète.", width, "   "))
				return nil
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
			in := diagnose.Input{Command: cmd, ExitCode: last.ExitCode, Stderr: last.Stderr, Interactive: last.Interactive}
			opts := render.Options{
				Theme: theme, Width: width, DockerVersion: last.DockerVersion,
				Interactive: last.Interactive, ExitCode: last.ExitCode,
			}
			_, err := a.renderDiagnosis(w, in, d, last.Stderr, opts)
			return err
		},
	}
}
