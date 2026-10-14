// Commande docker-telemach : plugin Docker CLI pédagogique qui explique en français les
// erreurs de Docker et de Docker Compose.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/aboigues/telemach-plugin-docker/internal/plugin"
	"github.com/aboigues/telemach-plugin-docker/internal/render"
	"github.com/aboigues/telemach-plugin-docker/internal/state"
)

// version est renseignée à la compilation (-ldflags "-X main.version=…").
var version = "dev"

// exitError porte un code de sortie précis jusqu'à main.
type exitError struct{ code int }

func (e exitError) Error() string { return fmt.Sprintf("code de sortie %d", e.code) }

// app regroupe les dépendances injectables (pour les tests).
type app struct {
	stdout, stderr io.Writer
	getenv         func(string) string
	now            func() time.Time
	store          state.Store

	colorFlag string
	width     int
}

func newApp() *app {
	return &app{
		stdout: os.Stdout,
		stderr: os.Stderr,
		getenv: os.Getenv,
		now:    time.Now,
		store:  state.Store{Dir: state.DefaultDir(os.Getenv)},
	}
}

func main() {
	args := os.Args[1:]
	// Appelé par la CLI Docker (« docker telemach … »), le plugin reçoit son nom en premier.
	if len(args) > 0 && args[0] == "telemach" {
		args = args[1:]
	}
	os.Exit(run(newApp(), args))
}

// run exécute la commande et renvoie le code de sortie.
func run(a *app, args []string) int {
	root := a.rootCommand()
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		var ee exitError
		if errors.As(err, &ee) {
			return ee.code
		}
		fmt.Fprintf(a.stderr, "docker telemach : %v\n", err)
		return 2
	}
	return 0
}

func (a *app) rootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "docker-telemach",
		Short:         "Explique en français les erreurs Docker et Docker Compose",
		Long:          "TELEMACH aide à apprendre Docker : il explique en français les erreurs des commandes docker et docker compose, et vérifie les fichiers Compose.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(*cobra.Command, []string) error {
			if _, err := render.ParseColorMode(a.colorFlag); err != nil {
				return err
			}
			if a.width != 0 && a.width < 40 {
				return fmt.Errorf("--width doit valoir au moins 40 (reçu %d)", a.width)
			}
			return nil
		},
	}
	root.SetOut(a.stdout)
	root.SetErr(a.stderr)
	root.PersistentFlags().StringVar(&a.colorFlag, "color", "auto", "couleurs : auto, always ou never")
	root.PersistentFlags().IntVar(&a.width, "width", 0, "largeur de mise en page (40 minimum, largeur du terminal par défaut)")
	root.SetHelpCommand(&cobra.Command{Hidden: true})
	root.CompletionOptions.DisableDefaultCmd = true
	frenchHelp(root)

	root.AddCommand(&cobra.Command{
		Use:    plugin.MetadataSubcommand,
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			return plugin.WriteMetadata(a.stdout, version)
		},
	})
	root.AddCommand(a.shellInitCommand(), a.explainCommand(), a.checkCommand(), a.deepenCommand())
	return root
}

// frenchHelp traduit les gabarits d'aide de cobra.
func frenchHelp(root *cobra.Command) {
	root.SetUsageTemplate(`Utilisation :{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [commande]{{end}}{{if .HasAvailableSubCommands}}

Commandes :{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Options :
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Options globales :
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
`)
	root.Flags().BoolP("help", "h", false, "affiche l'aide")
}

// output prépare le thème et l'écrivain coloré pour w.
func (a *app) output(w io.Writer) (render.Theme, io.Writer, int) {
	colorOn := render.ResolveColor(a.colorFlag, a.getenv, render.IsTerminal(w))
	dark := true
	if colorOn {
		dark = render.HasDarkBackground(w, a.getenv)
	}
	width := a.width
	if width == 0 {
		width = render.TerminalWidth(w)
	}
	return render.NewTheme(colorOn, dark), render.NewWriter(w, colorOn), width
}
