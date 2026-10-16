package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
	"github.com/aboigues/telemach-plugin-docker/internal/lint"
	"github.com/aboigues/telemach-plugin-docker/internal/render"
	"github.com/aboigues/telemach-plugin-docker/internal/state"
)

// imageRuleID est l'information « image sans tag », jamais répétée avant 24 h.
const imageRuleID = "cmdline.image-without-tag"

// lintCommand vérifie une ligne docker run/create/exec (feature 003) : à la main, sans rien
// exécuter, ou après coup (--after, appelé par l'intégration shell après une commande réussie).
// Codes de sortie : 0 aucun piège ou informations seulement, 1 au moins un avertissement,
// 2 usage invalide.
func (a *app) lintCommand() *cobra.Command {
	var after bool
	cmd := &cobra.Command{
		Use:   "lint [--after] -- <arguments docker…>",
		Short: "Vérifie une ligne docker run, create ou exec sans l'exécuter",
		Long: `Repère les écritures que Docker accepte mais qui ne font pas ce qui est voulu :
options placées après l'image ou le conteneur, volume nommé au lieu du dossier local,
port de la machine aléatoire, variable d'environnement non définie, image sans tag,
ports ignorés en réseau host, secret écrit en clair, privilèges excessifs.

Exemple : docker telemach lint -- run -d nginx -p 8090:80`,
		RunE: func(_ *cobra.Command, args []string) error {
			return a.lint(after, args)
		},
	}
	cmd.Flags().BoolVar(&after, "after", false, "la commande vient d'être exécutée (utilisé par l'intégration shell)")
	return cmd
}

// lintEnv fournit à l'analyse les deux seules lectures locales autorisées.
func (a *app) lintEnv() lint.Env {
	return lint.Env{
		HasVar: func(name string) bool {
			if a.lookupEnv != nil {
				_, ok := a.lookupEnv(name)
				return ok
			}
			return a.getenv(name) != ""
		},
		DirExists: func(name string) bool {
			if name == "" || strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
				return false
			}
			wd, err := a.getwd()
			if err != nil {
				return false
			}
			fi, err := os.Stat(filepath.Join(wd, name))
			return err == nil && fi.IsDir()
		},
	}
}

func (a *app) lint(after bool, args []string) error {
	if a.getenv("TELEMACH_LINT") == "0" {
		return nil
	}
	findings := lint.Analyze(args, a.lintEnv())
	if len(findings) == 0 {
		return nil // sortie rapide : ni catalogue ni thème chargés (R10)
	}
	cat, err := catalog.Default()
	if err != nil {
		return nil
	}
	now := a.now()
	hints := state.HintStore{Dir: a.store.Dir}
	var items []render.LintItem
	var shown []string
	var records []state.FindingRecord
	warning := false
	for _, f := range findings {
		e, ok := cat.Get(f.EntryID)
		if !ok {
			continue
		}
		if f.Severity == lint.SeverityWarning {
			warning = true
		}
		captures := lintCaptures(f.Captures, after)
		display := state.DisplayFull
		if after {
			display = hints.Decide(f.Fingerprint, f.EntryID == imageRuleID, now)
		}
		switch display {
		case state.DisplaySilent:
			continue
		case state.DisplayReminder:
			items = append(items, render.LintItem{Entry: e, Captures: captures, Reminder: true})
		default:
			items = append(items, render.LintItem{Entry: e, Captures: captures})
			shown = append(shown, f.Fingerprint)
			records = append(records, state.FindingRecord{EntryID: f.EntryID, Captures: captures})
		}
	}
	if len(items) > 0 {
		theme, w, width := a.output(a.stderr)
		if err := render.RenderFindings(w, items, render.Options{Theme: theme, Width: width}); err != nil {
			return err
		}
	}
	if after && len(shown) > 0 {
		_ = hints.Record(shown, now)
		_ = a.store.Save(state.LastDiagnosis{
			Kind: state.KindCmdline, Findings: records, Args: cleanArgs(args), RecordedAt: now,
			Fingerprint: "cmdline:" + strings.Join(shown, "|"),
		})
	}
	if warning {
		return exitError{code: 1}
	}
	return nil
}

// lintCaptures choisit la formulation : au passé après exécution (« le conteneur a
// démarré, mais… »), au conditionnel pour une vérification à la main.
func lintCaptures(c map[string]string, after bool) map[string]string {
	out := make(map[string]string, len(c))
	for k, v := range c {
		out[k] = v
	}
	out["transmis"] = "a été transmis"
	if after {
		out["cleanup"] = "Si le conteneur a été créé sans ces options, l'arrêter et le supprimer, puis le relancer avec la commande corrigée."
		// stop laisse au programme le temps de se terminer ; rm -f est un anti-pattern du catalogue.
		out["cleanup_command"] = "docker stop <conteneur> && docker rm <conteneur>"
	} else {
		out["effect"] = c["effect_cond"]
		out["transmis"] = "serait transmis"
		// Capture vide (et non absente, qui s'afficherait « … ») : correction sans objet.
		out["cleanup"], out["cleanup_command"] = "", ""
	}
	delete(out, "effect_cond")
	return out
}

// cmdlineWarnings renvoie les avertissements (jamais les informations) de la ligne de
// commande, pour les ajouter à l'explication d'une commande en échec.
func (a *app) cmdlineWarnings(args []string) ([]render.LintItem, []state.FindingRecord) {
	if a.getenv("TELEMACH_LINT") == "0" {
		return nil, nil
	}
	findings := lint.Analyze(args, a.lintEnv())
	if len(findings) == 0 {
		return nil, nil
	}
	cat, err := catalog.Default()
	if err != nil {
		return nil, nil
	}
	var items []render.LintItem
	var records []state.FindingRecord
	for _, f := range findings {
		e, ok := cat.Get(f.EntryID)
		if !ok || f.Severity != lint.SeverityWarning {
			continue
		}
		c := lintCaptures(f.Captures, true)
		// La commande a échoué : ni « le conteneur a démarré », ni conseil de suppression.
		c["effect"], c["cleanup"], c["cleanup_command"] = "", "", ""
		items = append(items, render.LintItem{Entry: e, Captures: c})
		records = append(records, state.FindingRecord{EntryID: f.EntryID, Captures: c})
	}
	return items, records
}
