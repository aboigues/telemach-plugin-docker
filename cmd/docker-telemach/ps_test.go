package main

import (
	"encoding/json"
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
while [ $# -gt 0 ]; do
  case "$1" in
  ps) cat "$FAKE_DIR/ps.out" 2>/dev/null; exit 0 ;;
  logs) cat "$FAKE_DIR/logs.out" 2>/dev/null; exit 0 ;;
  inspect) shift; break ;;
  esac
  shift
done
# inspect <id>… : un tableau JSON qui regroupe les <id>.json, ou « No such object ».
for id in "$@"; do
  [ -f "$FAKE_DIR/$id.json" ] || { echo "Error: No such object: $id" >&2; exit 1; }
done
printf '['; sep=""
for id in "$@"; do printf '%s' "$sep"; sed '1s/^\[//; $s/\]$//' "$FAKE_DIR/$id.json"; sep=","; done
printf ']'
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

// US3 : plusieurs conteneurs en erreur — le plus récent est expliqué, les autres listés ;
// redémarrages et projet Compose mentionnés ; désignation par préfixe d'identifiant.
func TestPSListAndContext(t *testing.T) {
	a, _, errb := testApp(t, nil)
	f := newPSFake(t, a)
	var ps strings.Builder
	load := func(name string) map[string]any {
		raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "exited", name, "inspect.json"))
		if err != nil {
			t.Fatal(err)
		}
		var v []map[string]any
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatal(err)
		}
		return v[0]
	}
	put := func(id string, c map[string]any, status string) {
		c["Id"] = id
		data, _ := json.Marshal([]any{c})
		f.write(t, id+".json", string(data))
		f.write(t, strings.TrimPrefix(c["Name"].(string), "/")+".json", string(data))
		fmt.Fprintf(&ps, `{"ID":%q,"Names":%q,"State":"exited","Status":%q}`+"\n", id, strings.TrimPrefix(c["Name"].(string), "/"), status)
	}
	loop := load("restart-loop")
	loop["State"].(map[string]any)["FinishedAt"] = "2026-10-16T13:59:00Z"
	compose := load("generic-1")
	compose["Name"] = "/boutique-api-1"
	compose["Config"].(map[string]any)["Labels"] = map[string]any{"com.docker.compose.project": "boutique", "com.docker.compose.service": "api"}
	compose["State"].(map[string]any)["FinishedAt"] = "2026-10-16T13:00:00Z"
	put("aaaa1111", loop, "Exited (1) 1 minute ago")
	put("bbbb2222", compose, "Exited (1) 1 hour ago")
	f.write(t, "ps.out", ps.String())

	if code := run(a, []string{"ps", "--color=never", "--width", "200"}); code != 0 {
		t.Fatalf("code = %d\n%s", code, errb.String())
	}
	got := errb.String()
	for _, want := range []string{
		"telemach-exited-restart-loop", "Redémarré 3 fois (politique de redémarrage : on-failure)",
		"Autres conteneurs arrêtés en erreur", "boutique-api-1", "docker telemach ps <nom>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("%q absent :\n%s", want, got)
		}
	}
	errb.Reset()
	if code := run(a, []string{"ps", "--color=never", "--width", "200", "bbbb2222"}); code != 0 {
		t.Fatalf("préfixe : code = %d\n%s", code, errb.String())
	}
	if !strings.Contains(errb.String(), "Projet Compose « boutique », service « api »") {
		t.Errorf("projet Compose attendu :\n%s", errb.String())
	}
}

func TestPSNotAnalyzable(t *testing.T) {
	a, _, errb := testApp(t, nil)
	f := newPSFake(t, a)
	for name, state := range map[string]string{"web": "running", "fini": "exited"} {
		f.write(t, name+".json", fmt.Sprintf(`[{"Id":"%s","Name":"/%s","State":{"Status":%q,"ExitCode":0}}]`, name, name, state))
	}
	for name, want := range map[string]string{"web": "en cours d'exécution", "fini": "sans erreur (code 0)"} {
		errb.Reset()
		if code := run(a, []string{"ps", "--color=never", "--width", "200", name}); code != 1 {
			t.Errorf("%s : code = %d, attendu 1", name, code)
		}
		if !strings.Contains(errb.String(), want) {
			t.Errorf("%s : %q attendu :\n%s", name, want, errb.String())
		}
	}
}

// Complétion de « docker telemach ps <Tab> » : seulement les conteneurs que ps sait analyser
// (arrêtés avec un code ≠ 0, en boucle de redémarrage), par une seule lecture « docker ps ».
func TestPSCompletion(t *testing.T) {
	complete := func(t *testing.T, fail bool, args ...string) (string, psFake) {
		t.Helper()
		a, out, _ := testApp(t, nil)
		f := newPSFake(t, a)
		f.write(t, "ps.out", `{"ID":"a","Names":"essai-pg","State":"exited","Status":"Exited (1) 2 minutes ago"}
{"ID":"b","Names":"fini","State":"exited","Status":"Exited (0) 1 hour ago"}
{"ID":"c","Names":"boucle","State":"restarting","Status":"Restarting (2) 3 seconds ago"}
{"ID":"d","Names":"cree","State":"created","Status":"Created"}
{"ID":"e","Names":"essai-oom","State":"exited","Status":"Exited (137) 5 minutes ago"}
`)
		if fail {
			f.write(t, "fail", "Cannot connect to the Docker daemon at unix:///var/run/docker.sock.\n")
		}
		_, plugin := splitPluginArgs(append([]string{"__complete", "telemach"}, args...))
		if code := run(a, plugin); code != 0 {
			t.Fatalf("code = %d", code)
		}
		return out.String(), f
	}

	got, f := complete(t, false, "ps", "")
	for _, want := range []string{"essai-pg\tcode 1", "essai-oom\tcode 137", "boucle\tcode 2, redémarre en boucle"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q absent :\n%s", want, got)
		}
	}
	for _, absent := range []string{"fini", "cree"} {
		if strings.Contains(got, absent+"\t") {
			t.Errorf("%s ne doit pas être proposé :\n%s", absent, got)
		}
	}
	if calls := f.calls(t); len(calls) != 1 || !strings.HasPrefix(calls[0], "ps -a") || !strings.Contains(calls[0], "--last") {
		t.Errorf("une seule lecture « docker ps » bornée attendue : %q", calls)
	}
	if !strings.Contains(got, ":4") {
		t.Errorf("aucune complétion de fichiers attendue (directive 4) :\n%s", got)
	}

	if got, _ := complete(t, false, "ps", "essai-o"); !strings.Contains(got, "essai-oom") || strings.Contains(got, "essai-pg") {
		t.Errorf("filtrage sur le début du nom attendu :\n%s", got)
	}
	if got, f := complete(t, false, "ps", "essai-pg", ""); strings.Contains(got, "essai") || len(f.calls(t)) != 1 || f.calls(t)[0] != "" {
		t.Errorf("un seul conteneur par commande, sans lecture du démon :\n%s %q", got, f.calls(t))
	}
	if got, _ := complete(t, true, "ps", ""); strings.Contains(got, "essai") || strings.Contains(got, "Cannot connect") {
		t.Errorf("démon injoignable : rien proposé, sans message :\n%s", got)
	}
}
