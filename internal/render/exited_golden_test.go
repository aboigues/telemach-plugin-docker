package render

import (
	"strconv"
	"strings"
	"testing"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
)

// Entrée factice : le rendu est testé indépendamment du contenu du vrai catalogue.
var fakeExited = &catalog.Entry{
	ID: "exited.missing-required-env", Category: "exited", Fallback: "exited",
	Concept: catalog.ConceptRef{
		Name:       "variable d'environnement obligatoire",
		Definition: "Certaines images refusent de démarrer sans une variable de configuration, par exemple un mot de passe d'administration.",
	},
	Summary: "L'image {{.image}} exige la variable {{.variable}}.",
	Cause:   "Le conteneur a été lancé sans {{.variable}} : son programme s'est arrêté aussitôt.",
	Fixes: []catalog.Fix{
		{Description: "Mettre la variable dans un fichier lisible par toi seul, puis le passer avec --env-file.", Command: "docker run -d --env-file db.env {{.image}}", Safe: true},
		{Description: "Ou utiliser un secret monté et la variable {{.variable}}_FILE.", Safe: true},
	},
}

var pgView = ExitedView{
	Entry: fakeExited,
	Captures: map[string]string{
		"name": "essai-pg", "image": "postgres:18-alpine", "code": "1", "lifetime": "1 s", "age": "il y a 2 min",
		"variable": "POSTGRES_PASSWORD",
	},
	Log: []string{
		"       You must specify POSTGRES_PASSWORD to a non-empty value for the",
		`       superuser. For example, "-e POSTGRES_PASSWORD=password" on "docker run".`,
		"",
		`       You may also use "POSTGRES_HOST_AUTH_METHOD=trust" to allow all connections without a password. This is *not* recommended.`,
	},
}

func TestExitedGolden(t *testing.T) {
	others := pgView
	others.Others = []OtherLine{
		{Name: "api", Code: 127, Age: "il y a 1 h", Summary: "commande introuvable dans l'image"},
		{Name: "worker-très-long-nom", Code: 137, Age: "il y a 3 h", Summary: "arrêt forcé (docker kill, ou docker stop après délai)"},
	}
	noLog := pgView
	noLog.Log, noLog.LogUnavailable = nil, "configured logging driver does not support reading"
	created := pgView
	created.Captures = map[string]string{
		"name": "essai-127", "image": "alpine:3.24", "code": "127", "age": "il y a 5 s",
		"state": "created", "variable": "X", "restart_count": "", "compose_project": "app", "compose_service": "web",
	}
	restart := pgView
	restart.Captures = map[string]string{
		"name": "boucle", "image": "alpine:3.24", "code": "1", "lifetime": "0 s",
		"age": "il y a 1 s", "variable": "X", "restart_count": "3", "restart_policy": "on-failure",
	}

	cases := map[string]ExitedView{"analysis": pgView, "others": others, "nolog": noLog, "created": created, "restart": restart}
	for name, v := range cases {
		for _, width := range []int{60, 100} {
			for _, color := range []bool{false, true} {
				gname := name + "-" + strconv.Itoa(width)
				if color {
					gname += "-color"
				}
				var b strings.Builder
				if err := RenderExited(&b, v, Options{Theme: NewTheme(color, true), Width: width}); err != nil {
					t.Fatal(err)
				}
				golden(t, "exited", gname, b.String())
				if !color {
					for i, line := range strings.Split(b.String(), "\n") {
						if n := len([]rune(line)); n > width && !strings.Contains(line, "docker run") {
							t.Errorf("%s ligne %d : %d caractères > %d :\n%s", gname, i+1, n, width, line)
						}
					}
				}
			}
		}
	}
}

func TestNoExitedGolden(t *testing.T) {
	var b strings.Builder
	if err := RenderNoExited(&b, Options{Theme: NewTheme(false, true), Width: 80}); err != nil {
		t.Fatal(err)
	}
	golden(t, "exited", "none", b.String())
}

func TestSignalLine(t *testing.T) {
	th := NewTheme(false, true)
	got := SignalLine("essai-pg", 1, "3 s", false, th)
	want := "⚠ Le conteneur « essai-pg » s'est arrêté avec le code 1, 3 s après son lancement : docker telemach ps pour comprendre pourquoi."
	if got != want {
		t.Errorf("signal :\n%s\nattendu :\n%s", got, want)
	}
	rm := SignalLine("eager_x", 3, "2 s", true, th)
	if !strings.Contains(rm, "et a été supprimé (--rm) : relance-le sans --rm pour lire son journal.") || strings.Contains(rm, "telemach ps") {
		t.Errorf("signal --rm : %s", rm)
	}
}
