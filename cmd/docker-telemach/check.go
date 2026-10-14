package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
	"github.com/aboigues/telemach-plugin-docker/internal/compose"
)

// checkCommand vérifie un fichier Compose avant toute exécution (FR-014), en lecture seule.
// Codes de sortie : 0 aucun problème, 1 au moins un problème, 2 usage invalide ou fichier
// introuvable.
func (a *app) checkCommand() *cobra.Command {
	var files []string
	cmd := &cobra.Command{
		Use:   "check [-f <fichier>]... [<dossier>]",
		Short: "Vérifie un fichier Compose avant de le lancer",
		Long: `Vérifie un projet Docker Compose sans rien exécuter ni modifier : syntaxe YAML,
clés inconnues, types de valeurs, références non déclarées, dépendances en boucle,
variables non définies et secrets écrits en clair.

Sans -f, les fichiers sont cherchés comme le fait Docker Compose : compose.yaml,
compose.yml, docker-compose.yaml puis docker-compose.yml (et leur fichier override).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			if info, err := os.Stat(dir); err != nil || !info.IsDir() {
				fmt.Fprintf(a.stderr, "docker telemach : le dossier « %s » n'existe pas\n", dir)
				return exitError{code: 2}
			}
			theme, w, width := a.output(a.stderr)
			opts := a.renderOptions(theme, width)
			issues := compose.Analyze(dir, files)
			located, _ := compose.Locate(dir, files)
			cat, _ := catalog.Default()
			if err := compose.RenderIssues(w, dir, located, issues, cat, opts); err != nil {
				return err
			}
			switch {
			case len(issues) == 0:
				return nil
			case issues[0].Kind == compose.KindFileNotFound && len(files) == 0:
				return exitError{code: 2}
			default:
				return exitError{code: 1}
			}
		},
	}
	cmd.Flags().StringArrayVarP(&files, "file", "f", nil, "fichier Compose à vérifier (répétable)")
	return cmd
}
