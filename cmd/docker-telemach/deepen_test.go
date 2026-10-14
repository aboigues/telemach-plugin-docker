package main

import (
	"strings"
	"testing"

	"github.com/aboigues/telemach-plugin-docker/internal/state"
)

func TestDeepen(t *testing.T) {
	t.Run("sans diagnostic récent", func(t *testing.T) {
		a, _, errb := testApp(t, nil)
		if code := run(a, []string{"deepen", "--color=never"}); code != 0 {
			t.Errorf("code = %d", code)
		}
		if !strings.Contains(errb.String(), "Aucune erreur récente") || !strings.Contains(errb.String(), "shell-init") {
			t.Errorf("message d'aide attendu :\n%s", errb.String())
		}
	})

	save := func(a *app, id string) {
		if err := a.store.Save(state.LastDiagnosis{Fingerprint: "x", EntryID: id, Captures: map[string]string{"port": "8080", "alt_port": "8081"}, RecordedAt: a.now()}); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("approfondissement du port déjà utilisé", func(t *testing.T) {
		a, _, errb := testApp(t, nil)
		save(a, "run.port-already-allocated")
		run(a, []string{"deepen", "--color=never"})
		out := errb.String()
		for _, want := range []string{"📘 Concept — port publishing", "▶ Exemple", "nginx-unprivileged", "docker telemach deepen --exercise", "📖 Documentation : https://docs.docker.com/"} {
			if !strings.Contains(out, want) {
				t.Errorf("sortie sans %q :\n%s", want, out)
			}
		}
		if strings.Contains(out, `{{"`) {
			t.Errorf("gabarit non rendu :\n%s", out)
		}
	})

	t.Run("avec exercice", func(t *testing.T) {
		a, _, errb := testApp(t, nil)
		save(a, "run.port-already-allocated")
		run(a, []string{"deepen", "--exercise", "--color=never"})
		if !strings.Contains(errb.String(), "🧪 Exercice") || !strings.Contains(errb.String(), "8082") {
			t.Errorf("exercice attendu :\n%s", errb.String())
		}
	})

	t.Run("entrée inconnue du catalogue", func(t *testing.T) {
		a, _, errb := testApp(t, nil)
		save(a, "entree.inexistante")
		run(a, []string{"deepen", "--color=never"})
		if !strings.Contains(errb.String(), "Aucune erreur récente") {
			t.Errorf("doit être traité comme « pas de diagnostic » :\n%s", errb.String())
		}
	})
}
