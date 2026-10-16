package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeDockerPS simule le démon à partir du corpus testdata/exited/ : journalise ses appels,
// répond à ps, inspect et logs. Il n'est pas dans le PATH : le test passe son chemin (LL-006).
const fakeDockerPS = `#!/bin/sh
echo "$*" >> "$FAKE_DIR/calls.log"
[ -f "$FAKE_DIR/fail" ] && { cat "$FAKE_DIR/fail" >&2; exit 1; }
for a in "$@"; do
  case "$a" in
  ps) cat "$FAKE_DIR/ps.out" 2>/dev/null; exit 0 ;;
  inspect) shift_done=1 ;;
  logs) cat "$FAKE_DIR/logs.out" 2>/dev/null; exit 0 ;;
  esac
done
# inspect <nom> : renvoie <nom>.json, ou « No such object ».
last=""; for a in "$@"; do last="$a"; done
if [ -f "$FAKE_DIR/$last.json" ]; then cat "$FAKE_DIR/$last.json"; exit 0; fi
echo "Error: No such object: $last" >&2; exit 1
`

type psFake struct{ dir string }

func newPSFake(t *testing.T, a *app) psFake {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "docker")
	if err := os.WriteFile(bin, []byte(fakeDockerPS), 0o700); err != nil { //nolint:gosec // script de test
		t.Fatal(err)
	}
	t.Setenv("FAKE_DIR", dir)
	a.dockerBin = bin
	return psFake{dir: dir}
}

func (f psFake) write(t *testing.T, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// useCase installe un cas du corpus : ligne de ps, inspect (par identifiant et par nom), journal.
func (f psFake) useCase(t *testing.T, name string) {
	t.Helper()
	base := filepath.Join("..", "..", "testdata", "exited", name)
	inspect, err := os.ReadFile(filepath.Join(base, "inspect.json")) // #nosec G304 -- corpus de test
	if err != nil {
		t.Fatal(err)
	}
	logs, _ := os.ReadFile(filepath.Join(base, "logs.txt")) // #nosec G304 -- corpus de test
	id, ctrName := "ccb3e54ad7a2", "telemach-exited-"+name
	f.write(t, "ps.out", fmt.Sprintf(`{"ID":%q,"Names":%q,"Image":"x","State":"exited","Status":"Exited (1) 2 minutes ago"}`+"\n", id, ctrName))
	f.write(t, id+".json", string(inspect))
	f.write(t, ctrName+".json", string(inspect))
	f.write(t, "logs.out", string(logs))
}

func (f psFake) calls(t *testing.T) []string {
	t.Helper()
	data, _ := os.ReadFile(filepath.Join(f.dir, "calls.log"))
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

func TestPSExplainsLatest(t *testing.T) {
	a, out, errb := testApp(t, nil)
	f := newPSFake(t, a)
	f.useCase(t, "missing-env-postgres")
	if code := run(a, []string{"ps", "--color=never", "--width", "200"}); code != 0 {
		t.Fatalf("code = %d\n%s", code, errb.String())
	}
	got := errb.String()
	for _, want := range []string{
		// Les guillemets sont suivis et précédés d'espaces insécables (typographie française).
		"Le conteneur «\u00a0telemach-exited-missing-env-postgres\u00a0» (postgres:18-alpine) s'est arrêté avec le code 1",
		"Fin du journal", "You must specify POSTGRES_PASSWORD",
		"exige la variable POSTGRES_PASSWORD", "--env-file", "Concept", "Pour aller plus loin",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("%q absent :\n%s", want, got)
		}
	}
	if out.Len() != 0 {
		t.Errorf("rien sur stdout : %q", out.String())
	}
	// Lectures autorisées seulement (FR-010, SC-002).
	for _, c := range f.calls(t) {
		sub := strings.Fields(c)[0]
		if sub != "ps" && sub != "inspect" && sub != "logs" {
			t.Errorf("appel non autorisé : docker %s", c)
		}
	}
	last, ok := a.store.Load()
	if !ok || !last.IsExited() || last.EntryID != "exited.missing-required-env" || last.Container.Name != "telemach-exited-missing-env-postgres" || len(last.LogExcerpt) == 0 {
		t.Errorf("analyse non enregistrée : %+v", last)
	}
}

func TestPSNothingToReport(t *testing.T) {
	a, _, errb := testApp(t, nil)
	f := newPSFake(t, a)
	f.write(t, "ps.out", `{"ID":"z","Names":"ok","State":"exited","Status":"Exited (0) 1 minute ago"}`+"\n")
	if code := run(a, []string{"ps", "--color=never"}); code != 0 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(errb.String(), "Aucun conteneur arrêté en erreur") {
		t.Errorf("message attendu :\n%s", errb.String())
	}
	for _, c := range f.calls(t) {
		if strings.HasPrefix(c, "inspect") || strings.HasPrefix(c, "logs") {
			t.Errorf("aucune inspection attendue pour un code 0 : %s", c)
		}
	}
}

func TestPSDaemonUnreachable(t *testing.T) {
	a, _, errb := testApp(t, nil)
	f := newPSFake(t, a)
	f.write(t, "fail", "Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?\n")
	if code := run(a, []string{"ps", "--color=never"}); code != 1 {
		t.Errorf("code = %d, attendu 1", code)
	}
	if !strings.Contains(errb.String(), "démon") {
		t.Errorf("explication du démon arrêté attendue :\n%s", errb.String())
	}
}

func TestPSNamedContainer(t *testing.T) {
	a, _, errb := testApp(t, nil)
	f := newPSFake(t, a)
	f.useCase(t, "oom")
	if code := run(a, []string{"ps", "--color=never", "--width", "200", "telemach-exited-oom"}); code != 0 {
		t.Fatalf("code = %d\n%s", code, errb.String())
	}
	if !strings.Contains(errb.String(), "faute de mémoire (limite : 8 Mo)") {
		t.Errorf("analyse du conteneur désigné attendue :\n%s", errb.String())
	}
	errb.Reset()
	if code := run(a, []string{"ps", "--color=never", "--width", "200", "telemach-exited-omm"}); code != 1 {
		t.Errorf("introuvable : code = %d, attendu 1", code)
	}
	if !strings.Contains(errb.String(), "telemach-exited-oom") {
		t.Errorf("nom proche attendu :\n%s", errb.String())
	}
}

func TestPSGlobalOptionsPassedToReads(t *testing.T) {
	a, _, _ := testApp(t, nil)
	f := newPSFake(t, a)
	f.useCase(t, "generic-1")
	a.dockerGlobal = []string{"--context", "essai"}
	run(a, []string{"ps", "--color=never"})
	for _, c := range f.calls(t) {
		if !strings.HasPrefix(c, "--context essai ") {
			t.Errorf("options globales absentes : docker %s", c)
		}
	}
}
