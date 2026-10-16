package main

import (
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/aboigues/telemach-plugin-docker/internal/dockerread"
	"github.com/aboigues/telemach-plugin-docker/internal/exited"
	"github.com/aboigues/telemach-plugin-docker/internal/render"
	"github.com/aboigues/telemach-plugin-docker/internal/state"
)

// watchDelay est l'attente avant de vérifier l'état du conteneur lancé (FR-014).
const watchDelay = 3 * time.Second

// watchCommand vérifie, une seule fois, qu'un conteneur lancé avec « docker run -d » ne s'est
// pas arrêté en erreur (feature 004, US2). Lancée en arrière-plan par l'intégration shell, sans
// entrée ni sortie ; seule lecture du démon : « docker events --until » (FR-010). Code de
// sortie toujours 0.
func (a *app) watchCommand() *cobra.Command {
	var since string
	var shellPID int
	cmd := &cobra.Command{
		Use:    "watch --since T0 --shell-pid PID -- <arguments de docker run…>",
		Hidden: true,
		RunE: func(_ *cobra.Command, args []string) error {
			a.watch(since, shellPID, args)
			return nil
		},
	}
	cmd.Flags().StringVar(&since, "since", "", "début de la commande (secondes depuis l'époque)")
	cmd.Flags().IntVar(&shellPID, "shell-pid", 0, "shell où afficher le signal")
	return cmd
}

func (a *app) watch(since string, shellPID int, args []string) {
	if a.getenv("TELEMACH_WATCH") == "0" || a.getenv("TELEMACH_DISABLE") == "1" {
		return
	}
	launch, ok := exited.ParseLaunch(args)
	if !ok {
		return // pas un lancement en arrière-plan : aucune lecture du démon
	}
	t0, err := exited.ParseSince(since)
	if err != nil || !state.Alive(shellPID) {
		return
	}
	a.sleep(watchDelay)
	r := dockerread.Runner{Bin: a.dockerBin, Global: a.dockerGlobal}
	evs, err := r.Events(t0, strconv.FormatInt(a.now().Unix()+1, 10))
	if err != nil {
		return // démon injoignable : la vérification se tait (FR-015)
	}
	res, ok := exited.Match(evs, launch)
	if !ok || !res.Failed {
		return
	}
	theme := render.NewTheme(render.ResolveColor("auto", a.getenv, true), true)
	line := render.SignalLine(res.Name, res.Code, exited.Duration(res.After), res.Removed, theme)
	_ = state.WritePending(state.PendingDir(a.getenv, a.store.Dir), shellPID, line, a.now())
}
