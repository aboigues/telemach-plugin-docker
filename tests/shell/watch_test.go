package shell

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// watchCalls renvoie les appels de « watch » journalisés par le faux plugin.
func watchCalls(log string) []string {
	var calls []string
	for _, l := range strings.Split(log, "\n") {
		if strings.HasPrefix(l, "telemach watch ") {
			calls = append(calls, l)
		}
	}
	return calls
}

// waitLog attend qu'une ligne apparaisse dans le journal des appels (tâche de fond).
func waitLog(t *testing.T, e env, want string) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if log := readLog(t, e); strings.Contains(log, want) {
			return log
		}
		time.Sleep(50 * time.Millisecond)
	}
	return readLog(t, e)
}

// Garantie 11 : après un « run » réussi, en session interactive, « watch » est lancé une fois,
// en arrière-plan (jamais attendu), entrées et sorties sur /dev/null ; sorties et code de
// docker inchangés.
func TestWatchLaunchedAfterRun(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		e := setup(t, true)
		start := time.Now()
		stdout, _, err := runInteractive(t, shell, e, []string{"FAKE_WATCH_SLEEP=3"}, `docker run -d --name essai alpine:3.24; echo "rc=$?"; x=$(docker run -d alpine:3.24); echo "capturé=$x"`)
		if err != nil {
			t.Fatal(err)
		}
		if d := time.Since(start); d > 2*time.Second {
			t.Errorf("la commande a attendu la vérification (%v)", d)
		}
		if stdout != "STDOUT-LINE\nrc=0\ncapturé=STDOUT-LINE\n" {
			t.Errorf("stdout modifié : %q", stdout)
		}
		log := waitLog(t, e, "watch-fd2=")
		calls := watchCalls(log)
		if len(calls) != 2 {
			t.Fatalf("2 appels de watch attendus :\n%s", log)
		}
		re := regexp.MustCompile(`^telemach watch --since \d+[.,]\d+ --shell-pid \d+ -- run -d --name essai alpine:3\.24$`)
		if !re.MatchString(calls[0]) {
			t.Errorf("appel : %q", calls[0])
		}
		for _, fd := range []string{"watch-fd0=/dev/null", "watch-fd1=/dev/null", "watch-fd2=/dev/null"} {
			if !strings.Contains(log, fd) {
				t.Errorf("%s attendu :\n%s", fd, log)
			}
		}
	})
}

// Garantie 14 : pas de vérification hors session interactive, après un échec, pour une autre
// commande, ni quand elle est désactivée ; TELEMACH_LINT=0 ne la désactive pas.
func TestWatchNotLaunched(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		cases := []struct {
			name        string
			interactive bool
			env         []string
			body        string
		}{
			{"session non interactive", false, nil, "docker run -d alpine:3.24"},
			{"commande en échec", true, []string{"FAKE_RC=125"}, "docker run -d alpine:3.24"},
			{"autre commande", true, nil, "docker ps -a"},
			{"TELEMACH_WATCH=0", true, []string{"TELEMACH_WATCH=0"}, "docker run -d alpine:3.24"},
			{"TELEMACH_DISABLE=1", true, []string{"TELEMACH_DISABLE=1"}, "docker run -d alpine:3.24"},
		}
		for _, c := range cases {
			e := setup(t, true)
			runner := run
			if c.interactive {
				runner = runInteractive
			}
			if _, _, err := runner(t, shell, e, c.env, c.body+"; true"); err != nil {
				t.Fatal(err)
			}
			time.Sleep(200 * time.Millisecond)
			if calls := watchCalls(readLog(t, e)); len(calls) != 0 {
				t.Errorf("%s : aucun appel attendu : %q", c.name, calls)
			}
		}
		e := setup(t, true)
		if _, _, err := runInteractive(t, shell, e, []string{"TELEMACH_LINT=0"}, "docker run -d alpine:3.24"); err != nil {
			t.Fatal(err)
		}
		log := waitLog(t, e, "telemach watch")
		if len(watchCalls(log)) != 1 || len(lintCalls(log)) != 0 {
			t.Errorf("TELEMACH_LINT=0 : watch sans lint attendu :\n%s", log)
		}
	})
}

// Garantie 12 : le crochet d'invite affiche puis supprime les signaux de ce shell seulement,
// et préserve le code de retour de la dernière commande.
func TestPromptHook(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		e := setup(t, true)
		dir := filepath.Join(e.runtime, "telemach")
		body := `mkdir -p "` + dir + `"
printf '⚠ signal A\n' > "` + dir + `/pending-$$-1"
printf '⚠ signal B\n' > "` + dir + `/pending-$$-2"
printf '⚠ autre shell\n' > "` + dir + `/pending-1-1"
false; __telemach_prompt; echo "rc=$?"
__telemach_prompt; echo "second passage"
ls "` + dir + `"`
		stdout, stderr, err := runInteractive(t, shell, e, nil, body)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(stderr, "⚠ signal A\n⚠ signal B\n") || strings.Contains(stderr, "autre shell") {
			t.Errorf("signaux de ce shell attendus sur stderr :\n%s", stderr)
		}
		if strings.Count(stderr, "signal A") != 1 {
			t.Errorf("un signal ne s'affiche qu'une fois :\n%s", stderr)
		}
		if !strings.Contains(stdout, "rc=1\n") {
			t.Errorf("code de retour non préservé : %q", stdout)
		}
		if !strings.HasSuffix(stdout, "second passage\npending-1-1\n") {
			t.Errorf("seul le signal d'un autre shell doit rester : %q", stdout)
		}
	})
}

// Chargement répété de shell-init : un seul crochet ; PROMPT_COMMAND existant conservé, sous
// ses formes chaîne et tableau.
func TestPromptHookIdempotent(t *testing.T) {
	e := setup(t, true)
	stdout, _, err := runInteractive(t, "zsh", e, nil, `eval "$(cat "$INIT")"; print -l $precmd_functions | grep -c __telemach_prompt`)
	if err == nil && strings.TrimSpace(stdout) != "1" {
		t.Errorf("zsh : %s crochets", strings.TrimSpace(stdout))
	}
	e = setup(t, true)
	stdout, _, err = runInteractive(t, "bash", e, []string{"PROMPT_COMMAND=mon_invite"}, `eval "$(cat "$INIT")"; echo "$PROMPT_COMMAND"`)
	if err != nil || strings.TrimSpace(stdout) != "__telemach_prompt;mon_invite" {
		t.Errorf("bash (chaîne) : %q %v", stdout, err)
	}
	e = setup(t, true)
	body := `PROMPT_COMMAND=(mon_invite); eval "$(cat "$INIT")"; eval "$(cat "$INIT")"; printf '%s\n' "${PROMPT_COMMAND[@]}"`
	stdout, _, err = runShell(t, "bash", interactiveFlags("bash"), e, nil, body)
	if err != nil || stdout != "mon_invite\n__telemach_prompt\n" {
		t.Errorf("bash (tableau) : %q %v", stdout, err)
	}
}

// SC-005 : le lancement de la vérification ajoute moins de 50 ms au p95 (LL-012).
func TestWatchLaunchLatency(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		e := setup(t, true)
		body := `zmodload zsh/datetime 2>/dev/null
for i in $(seq 40); do
  a=$EPOCHREALTIME; docker run -d alpine:3.24 >/dev/null; b=$EPOCHREALTIME
  c=$EPOCHREALTIME; command docker run -d alpine:3.24 >/dev/null; d=$EPOCHREALTIME
  echo "$a $b $c $d"
done`
		stdout, _, err := runInteractive(t, shell, e, nil, body)
		if err != nil {
			t.Fatal(err)
		}
		var with, without []float64
		for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
			f := strings.Fields(strings.ReplaceAll(line, ",", "."))
			if len(f) != 4 {
				continue
			}
			v := make([]float64, 4)
			for i := range f {
				v[i], _ = strconv.ParseFloat(f[i], 64)
			}
			with, without = append(with, v[1]-v[0]), append(without, v[3]-v[2])
		}
		if len(with) < 30 {
			t.Fatalf("mesures insuffisantes :\n%s", stdout)
		}
		sort.Float64s(with)
		sort.Float64s(without)
		added := (with[len(with)*95/100-1] - without[len(without)/2]) * 1000
		t.Logf("%s : p95 avec intégration %.1f ms, médiane sans %.1f ms, ajout %.1f ms", shell,
			with[len(with)*95/100-1]*1000, without[len(without)/2]*1000, added)
		limit := 50.0
		if os.Getenv("CI") != "" {
			limit = 100
		}
		if added > limit {
			t.Errorf("%s : %.1f ms ajoutés au p95 (limite %.0f ms)", shell, added, limit)
		}
	})
}
