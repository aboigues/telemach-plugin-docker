package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
	"github.com/aboigues/telemach-plugin-docker/internal/render"
)

// deepenCommand approfondit le dernier diagnostic (US3, FR-009).
func (a *app) deepenCommand() *cobra.Command {
	var exercise bool
	cmd := &cobra.Command{
		Use:   "deepen [--exercise]",
		Short: "Approfondit le concept Docker de la dernière erreur expliquée",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			theme, w, width := a.output(a.stderr)
			last, ok := a.store.Load()
			cat, err := catalog.Default()
			var entry *catalog.Entry
			if ok && err == nil {
				entry, ok = cat.Get(last.EntryID)
			}
			if !ok || entry == nil {
				fmt.Fprintln(w, theme.Hint("Aucune erreur récente à approfondir."))
				fmt.Fprintln(w, render.Wrap("Active l'explication automatique avec « eval \"$(docker telemach shell-init zsh)\" » "+
					"(ou bash), puis lance une commande docker : dès qu'une erreur est expliquée, "+
					"« docker telemach deepen » permet d'aller plus loin.", width, "   "))
				return nil
			}
			return render.RenderDeepen(w, entry, last.Captures, exercise, render.Options{Theme: theme, Width: width})
		},
	}
	cmd.Flags().BoolVar(&exercise, "exercise", false, "affiche aussi un exercice court et sans risque")
	return cmd
}
