package dockerread

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// fakeDocker journalise ses arguments, puis renvoie le contenu de <sous-commande>.out (stdout),
// <sous-commande>.err (stderr) et le code de <sous-commande>.rc.
const fakeDocker = `#!/bin/sh
echo "$*" >> "$FAKE_DIR/calls.log"
sub=""
for a in "$@"; do
  case "$a" in ps|inspect|logs|events) sub="$a"; break ;; esac
done
[ -n "$FAKE_SLEEP" ] && sleep "$FAKE_SLEEP"
[ -f "$FAKE_DIR/$sub.out" ] && cat "$FAKE_DIR/$sub.out"
[ -f "$FAKE_DIR/$sub.err" ] && cat "$FAKE_DIR/$sub.err" >&2
exit "$(cat "$FAKE_DIR/$sub.rc" 2>/dev/null || echo 0)"
`

type fake struct {
	dir string
	r   Runner
}

func newFake(t *testing.T) fake {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "docker")
	if err := os.WriteFile(bin, []byte(fakeDocker), 0o700); err != nil { //nolint:gosec // script de test
		t.Fatal(err)
	}
	t.Setenv("FAKE_DIR", dir)
	return fake{dir: dir, r: Runner{Bin: bin, Global: []string{"--context", "essai"}, Timeout: 5 * time.Second}}
}

func (f fake) set(t *testing.T, sub, out, errOut string, rc int) {
	t.Helper()
	for name, content := range map[string]string{sub + ".out": out, sub + ".err": errOut, sub + ".rc": fmt.Sprint(rc)} {
		if err := os.WriteFile(filepath.Join(f.dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func (f fake) calls(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(f.dir, "calls.log"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

// Lignes réelles relevées le 2026-10-16 (Docker 29.8.0).
const (
	psExited           = `{"Command":"\"docker-entrypoint.sh postgres\"","CreatedAt":"2026-10-16 15:01:03 +0200 CEST","ID":"ccb3e54ad7a21811ad29e6b7e677805b206638992a267ec90043163cc8a3ef7d","Image":"postgres:18-alpine","Names":"it8-pg","State":"exited","Status":"Exited (1) 3 seconds ago"}`
	psCreatedSample    = `{"CreatedAt":"2026-10-16 15:02:00 +0200 CEST","ID":"aaa","Image":"alpine:3.24","Names":"essai-127","State":"created","Status":"Created"}`
	psRestartingSample = `{"CreatedAt":"2026-10-16 15:03:00 +0200 CEST","ID":"bbb","Image":"alpine:3.24","Names":"boucle","State":"restarting","Status":"Restarting (2) 1 second ago"}`
	inspectSample      = `[{"Id":"ccb3e54ad7a21811ad29e6b7e677805b206638992a267ec90043163cc8a3ef7d","Name":"/it8-pg","Path":"docker-entrypoint.sh","Args":["postgres","--password=Secret123"],"Created":"2026-10-16T13:01:03.787156843Z","RestartCount":0,
"State":{"Status":"exited","Running":false,"Restarting":false,"OOMKilled":false,"ExitCode":1,"Error":"","StartedAt":"2026-10-16T13:01:04.096717096Z","FinishedAt":"2026-10-16T13:01:04.467718611Z"},
"Config":{"Image":"postgres:18-alpine","Env":["DB_PASSWORD=Secret123","PATH=/usr/bin","EMPTY="],"Labels":{"com.docker.compose.project":"app","com.docker.compose.service":"db"}},
"HostConfig":{"RestartPolicy":{"Name":"no","MaximumRetryCount":0},"Memory":8388608,"LogConfig":{"Type":"json-file","Config":{}}}}]`
	eventsSample = `{"Type":"container","Action":"create","Actor":{"ID":"ccb3","Attributes":{"image":"postgres:18-alpine","name":"it8-pg"}},"scope":"local","time":1792155663,"timeNano":1792155663874368400}
{"Type":"container","Action":"die","Actor":{"ID":"ccb3","Attributes":{"execDuration":"0","exitCode":"1","image":"postgres:18-alpine","name":"it8-pg"}},"scope":"local","time":1792155664,"timeNano":1792155664686222117}
`
)

func TestReadsUseOnlyAllowedArguments(t *testing.T) {
	f := newFake(t)
	f.set(t, "ps", psExited+"\n"+psCreatedSample+"\n"+psRestartingSample+"\n", "", 0)
	f.set(t, "inspect", inspectSample, "", 0)
	f.set(t, "logs", "ligne 1\nmot de passe : Secret123\nAPI_TOKEN=Secret123\n", "", 0)
	f.set(t, "events", eventsSample, "", 0)

	cands, err := f.r.ListCandidates()
	if err != nil || len(cands) != 3 {
		t.Fatalf("ListCandidates = %+v, %v", cands, err)
	}
	if c := cands[0]; c.Name != "it8-pg" || !c.HasCode || c.Code != 1 || c.Image != "postgres:18-alpine" {
		t.Errorf("candidat exited : %+v", c)
	}
	if c := cands[1]; c.HasCode || c.State != "created" {
		t.Errorf("candidat created (code inconnu sans inspection) : %+v", c)
	}
	if c := cands[2]; !c.HasCode || c.Code != 2 || c.State != "restarting" {
		t.Errorf("candidat restarting : %+v", c)
	}
	if _, err := f.r.Inspect("ccb3"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.r.Logs("ccb3", 200); err != nil {
		t.Fatal(err)
	}
	if _, err := f.r.Events("1792155660.5", "1792155670"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"--context essai inspect ccb3",
		"--context essai logs --tail 200 ccb3",
		"--context essai events --since 1792155660.5 --until 1792155670 --filter type=container --filter event=create --filter event=die --format {{json .}}",
	}
	// Les deux lectures de ps sont parallèles : leur ordre dans le journal varie.
	ps := map[string]bool{
		"--context essai ps -a --no-trunc --filter status=exited --filter status=restarting --last " + fmt.Sprint(MaxStopped) + " --format {{json .}}": true,
		"--context essai ps -a --no-trunc --filter status=created --last " + fmt.Sprint(MaxCreated) + " --format {{json .}}":                           true,
	}
	got := f.calls(t)
	if len(got) < 2 || !ps[got[0]] || !ps[got[1]] || got[0] == got[1] {
		t.Errorf("lectures de ps : %q", got)
	} else {
		got = got[2:]
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("appels :\n%s\nattendu :\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestInspectDecodesWithoutEnvValues(t *testing.T) {
	f := newFake(t)
	f.set(t, "inspect", inspectSample, "", 0)
	cs, err := f.r.Inspect("ccb3")
	if err != nil || len(cs) != 1 {
		t.Fatalf("Inspect = %+v, %v", cs, err)
	}
	c := cs[0]
	if !reflect.DeepEqual([]string(c.EnvNames), []string{"DB_PASSWORD", "PATH", "EMPTY"}) {
		t.Errorf("EnvNames = %q", c.EnvNames)
	}
	if dump := fmt.Sprintf("%#v", c); strings.Contains(dump, "Secret123") {
		t.Errorf("une valeur secrète a survécu au décodage :\n%s", dump)
	}
	if c.Name != "it8-pg" || c.Image != "postgres:18-alpine" || c.ExitCode != 1 || c.RestartPolicy != "no" ||
		c.MemoryLimit != 8388608 || c.LogDriver != "json-file" || c.ComposeProject != "app" || c.ComposeService != "db" {
		t.Errorf("champs décodés : %+v", c)
	}
	if !strings.Contains(c.Command, "docker-entrypoint.sh postgres") || strings.Contains(c.Command, "Secret123") {
		t.Errorf("commande non masquée : %q", c.Command)
	}
	if c.FinishedAt.Sub(c.StartedAt) <= 0 {
		t.Errorf("dates : %v → %v", c.StartedAt, c.FinishedAt)
	}
}

func TestLogsAreCleanedAndMasked(t *testing.T) {
	f := newFake(t)
	f.set(t, "logs", "\x1b[31mrouge\x1b[0m\nAPI_TOKEN=Secret123\n", "erreur sur stderr\n", 0)
	ex, err := f.r.Logs("ccb3", 200)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(ex.Lines, "\n")
	if strings.Contains(joined, "Secret123") || strings.Contains(joined, "\x1b") {
		t.Errorf("journal non nettoyé ou non masqué : %q", ex.Lines)
	}
	if !strings.Contains(joined, "erreur sur stderr") {
		t.Errorf("stderr du conteneur absent : %q", ex.Lines)
	}
}

func TestLogsUnavailable(t *testing.T) {
	f := newFake(t)
	f.set(t, "logs", "", `Error response from daemon: configured logging driver does not support reading`+"\n", 1)
	ex, err := f.r.Logs("ccb3", 200)
	if err != nil || !ex.Unavailable || !strings.Contains(ex.Reason, "does not support reading") {
		t.Errorf("journal illisible attendu : %+v, %v", ex, err)
	}
}

func TestEventsDecoded(t *testing.T) {
	f := newFake(t)
	f.set(t, "events", eventsSample, "", 0)
	evs, err := f.r.Events("1", "2")
	if err != nil || len(evs) != 2 {
		t.Fatalf("Events = %+v, %v", evs, err)
	}
	if e := evs[1]; e.Action != "die" || e.ExitCode != 1 || e.Name != "it8-pg" || e.Image != "postgres:18-alpine" || e.TimeNano != 1792155664686222117 {
		t.Errorf("événement die : %+v", e)
	}
	if evs[0].ExitCode != -1 {
		t.Errorf("create sans code : ExitCode = %d, attendu -1", evs[0].ExitCode)
	}
}

func TestOnlyAllowedSubcommands(t *testing.T) {
	f := newFake(t)
	for _, sub := range []string{"rm", "stop", "start", "kill", "exec", "run", "pull", "restart", "update", "cp"} {
		if _, err := f.r.output(sub, "x"); !errors.Is(err, ErrNotAllowed) {
			t.Errorf("%s : ErrNotAllowed attendu, obtenu %v", sub, err)
		}
	}
	if got := f.calls(t); len(got) != 1 || got[0] != "" {
		t.Errorf("aucun appel attendu : %q", got)
	}
}

func TestTimeout(t *testing.T) {
	f := newFake(t)
	f.r.Timeout = 200 * time.Millisecond
	t.Setenv("FAKE_SLEEP", "5")
	start := time.Now()
	_, err := f.r.ListCandidates()
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("ErrTimeout attendu, obtenu %v", err)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("le délai n'a pas interrompu la lecture (%v)", d)
	}
}

func TestCLIErrorIsMasked(t *testing.T) {
	f := newFake(t)
	f.set(t, "ps", "", "Cannot connect to the Docker daemon at unix:///var/run/docker.sock. token=Secret123\n", 1)
	_, err := f.r.ListCandidates()
	var ce *CLIError
	if !errors.As(err, &ce) || ce.Code != 1 || !strings.Contains(ce.Stderr, "Cannot connect") || strings.Contains(ce.Stderr, "Secret123") {
		t.Errorf("CLIError masquée attendue : %#v", err)
	}
}
