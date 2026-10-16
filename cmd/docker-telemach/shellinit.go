package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aboigues/telemach-plugin-docker/internal/shellinit"
)

func (a *app) shellInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "shell-init bash|zsh",
		Short: "Affiche le code à évaluer pour activer l'explication automatique des erreurs",
		Long: `Affiche le code d'intégration à ajouter à ton fichier de configuration du shell :

  eval "$(docker telemach shell-init zsh)"    # dans ~/.zshrc
  eval "$(docker telemach shell-init bash)"   # dans ~/.bashrc

La fonction « docker » ainsi définie explique chaque commande docker en échec. Désactivation
ponctuelle : TELEMACH_DISABLE=1.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			script, err := shellinit.Script(args[0], a.executable(), a.version)
			if err != nil {
				fmt.Fprintf(a.stderr, "docker telemach : %v\n", err)
				return exitError{code: 2}
			}
			_, err = fmt.Fprint(a.stdout, script)
			return err
		},
	}
}
