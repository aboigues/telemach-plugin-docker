//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// V12 — un secret passé en option n'apparaît ni à l'écran ni sur disque (SC-006).
func TestV12SecretNeverLeaks(t *testing.T) {
	h := newHarness(t)
	_, stderr := h.shell(t, `docker run --rm --user 1000:1000 -e DB_PASSWORD=SuperSecret123 alpine:3.24 sh -c 'exit 1'`)
	if strings.Contains(stderr, "SuperSecret123") {
		t.Errorf("le secret apparaît dans l'explication :\n%s", stderr)
	}
	err := filepath.Walk(h.state, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(p)
		if err == nil && strings.Contains(string(data), "SuperSecret123") {
			t.Errorf("le secret est écrit sur disque : %s", p)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// SC-007 / FR-015 — aucune sous-commande ne lance de commande docker. Un faux docker,
// placé en tête du PATH, journalise ses appels : seul « docker --version » est permis,
// une seule fois, par le script shell-init.
func TestSC007NoCommandExecuted(t *testing.T) {
	h := newHarness(t)
	fakeDir := t.TempDir()
	logFile := filepath.Join(fakeDir, "calls.log")
	fake := "#!/bin/sh\necho \"$*\" >> \"" + logFile + "\"\n" +
		"if [ \"$1\" = \"--version\" ]; then echo 'Docker version 29.8.0, build x'; exit 0; fi\n" +
		"echo 'Error response from daemon: No such container: web' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(fakeDir, "docker"), []byte(fake), 0o700); err != nil {
		t.Fatal(err)
	}
	composeDir, _ := filepath.Abs(filepath.Join("..", "..", "testdata", "compose", "tab-indent"))
	cmd := exec.Command("bash", "-c", `eval "$("$TELEMACH" shell-init bash)"
docker logs web
docker-telemach check "$COMPOSE_DIR"
docker-telemach deepen --exercise
true`)
	cmd.Env = append(os.Environ(),
		"PATH="+fakeDir+string(os.PathListSeparator)+h.bin+string(os.PathListSeparator)+"/usr/bin:/bin",
		"XDG_STATE_HOME="+h.state, "NO_COLOR=1", "COMPOSE_DIR="+composeDir,
		"TELEMACH="+filepath.Join(h.bin, "docker-telemach"))
	out, _ := cmd.CombinedOutput()
	data, _ := os.ReadFile(logFile)
	calls := strings.Split(strings.TrimSpace(string(data)), "\n")
	var unexpected []string
	versions := 0
	for _, c := range calls {
		switch c {
		case "--version":
			versions++
		case "logs web":
			// commande lancée par l'apprenant lui-même
		default:
			unexpected = append(unexpected, c)
		}
	}
	if versions != 1 || len(unexpected) > 0 {
		t.Errorf("appels docker inattendus (versions=%d) : %v\nsortie :\n%s", versions, unexpected, out)
	}
	if !strings.Contains(string(out), "Aucun conteneur ne s'appelle") {
		t.Errorf("l'erreur aurait dû être expliquée :\n%s", out)
	}
}
