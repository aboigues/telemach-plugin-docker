package shell

import (
	"strings"
	"testing"
)

// lintCalls renvoie les appels de « lint » journalisés par le faux plugin.
func lintCalls(log string) []string {
	var calls []string
	for _, l := range strings.Split(log, "\n") {
		if strings.HasPrefix(l, "telemach lint ") {
			calls = append(calls, l)
		}
	}
	return calls
}

// Garanties 8 et 9 (contracts/cli-lint.md) : lint est appelé une fois, après une commande
// run/create/exec réussie, et seulement pour elles ; sorties et code de Docker inchangés.
func TestLintCalledAfterLintableSuccess(t *testing.T) {
	lintable := []string{
		"run -d nginx -p 8090:80",
		"create --name web nginx",
		"exec web ls",
		"container run --rm alpine true",
		"container create alpine",
		"container exec web ls",
		"--context x run nginx",
		"-H tcp://h:2375 run nginx",
		"--log-level debug container run nginx",
		"--context=x run nginx",
		"run -it alpine sh",
	}
	for _, args := range lintable {
		forEachShell(t, func(t *testing.T, shell string) {
			e := setup(t, true)
			stdout, stderr, err := run(t, shell, e, nil, "docker "+args+`; echo "rc=$?"`)
			if err != nil {
				t.Fatal(err)
			}
			if stdout != "STDOUT-LINE\nrc=0\n" {
				t.Errorf("%s : stdout %q", args, stdout)
			}
			if strings.Count(stderr, "Error response from daemon") != 1 {
				t.Errorf("%s : stderr de Docker modifié : %q", args, stderr)
			}
			log := readLog(t, e)
			calls := lintCalls(log)
			if len(calls) != 1 || calls[0] != "telemach lint --after -- "+args {
				t.Errorf("%s : appels lint = %q", args, calls)
			}
			if strings.Contains(log, "telemach explain") {
				t.Errorf("%s : explain ne doit pas être appelé pour une commande réussie", args)
			}
			// lint est appelé après la commande, jamais avant.
			if di, li := strings.Index(log, "docker "+args), strings.Index(log, "telemach lint"); di < 0 || li < di {
				t.Errorf("%s : lint doit suivre la commande :\n%s", args, log)
			}
		})
	}
}

func TestLintNotCalled(t *testing.T) {
	cases := []struct {
		name, args string
		env        []string
	}{
		{"ps", "ps -a", nil},
		{"images", "images", nil},
		{"pull", "pull nginx", nil},
		{"build", "build -t app .", nil},
		{"compose up", "compose up -d", nil},
		{"image dont le nom est run", "pull run", nil},
		{"TELEMACH_LINT=0", "run -d nginx -p 8090:80", []string{"TELEMACH_LINT=0"}},
		{"TELEMACH_DISABLE=1", "run -d nginx -p 8090:80", []string{"TELEMACH_DISABLE=1"}},
	}
	for _, c := range cases {
		forEachShell(t, func(t *testing.T, shell string) {
			e := setup(t, true)
			stdout, _, err := run(t, shell, e, c.env, "docker "+c.args+`; echo "rc=$?"`)
			if err != nil {
				t.Fatal(err)
			}
			if stdout != "STDOUT-LINE\nrc=0\n" {
				t.Errorf("%s : stdout %q", c.name, stdout)
			}
			if calls := lintCalls(readLog(t, e)); len(calls) != 0 {
				t.Errorf("%s : lint ne doit pas être appelé : %q", c.name, calls)
			}
		})
	}
}

// En cas d'échec, seul explain est appelé (il intègre lui-même les pièges, FR-010).
func TestLintNotCalledOnFailure(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		e := setup(t, true)
		stdout, _, err := run(t, shell, e, []string{"FAKE_RC=125"}, `docker run -d nginx -p 8090:80; echo "rc=$?"`)
		if err != nil {
			t.Fatal(err)
		}
		if stdout != "STDOUT-LINE\nrc=125\n" {
			t.Errorf("stdout : %q", stdout)
		}
		log := readLog(t, e)
		if len(lintCalls(log)) != 0 || !strings.Contains(log, "telemach explain") {
			t.Errorf("explain seul attendu :\n%s", log)
		}
	})
}
