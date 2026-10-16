// Package shell vérifie les garanties de l'intégration shell (contracts/cli.md § shell-init)
// avec de faux binaires docker et docker-telemach.
package shell

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/aboigues/telemach-plugin-docker/internal/shellinit"
)

// Faux docker : écrit sur stdout et stderr, journalise ses appels et renvoie $FAKE_RC.
const fakeDocker = `#!/bin/sh
echo "docker $*" >> "$CALL_LOG"
if [ "$1" = "--version" ]; then
  echo "Docker version 29.8.0, build abc1234"
  exit 0
fi
if [ -n "$FAKE_SLEEP" ]; then
  echo "en cours" >&2
  sleep "$FAKE_SLEEP"
fi
printf 'STDOUT-LINE\n'
printf 'Error response from daemon: port is already allocated\n' >&2
exit "${FAKE_RC:-0}"
`

// Faux docker-telemach : journalise ses arguments, les droits et le contenu du fichier
// stderr, puis échoue (son code de sortie doit être ignoré).
const fakeTelemach = `#!/bin/sh
echo "telemach $*" >> "$CALL_LOG"
prev=""
for a in "$@"; do
  if [ "$prev" = "--stderr-file" ]; then
    echo "perm=$(stat -c %a "$a")" >> "$CALL_LOG"
    echo "content=$(cat "$a")" >> "$CALL_LOG"
  fi
  prev="$a"
done
exit 3
`

type env struct {
	bin, runtime, log string
}

func setup(t *testing.T, withTelemach bool) env {
	t.Helper()
	dir := t.TempDir()
	e := env{bin: filepath.Join(dir, "bin"), runtime: filepath.Join(dir, "run"), log: filepath.Join(dir, "calls.log")}
	for _, d := range []string{e.bin, e.runtime} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(e.bin, name), []byte(content), 0o700); err != nil { //nolint:gosec // script de test
			t.Fatal(err)
		}
	}
	write("docker", fakeDocker)
	if withTelemach {
		write("docker-telemach", fakeTelemach)
	}
	return e
}

// run évalue le script d'intégration puis exécute body dans le shell donné.
func run(t *testing.T, shell string, e env, extraEnv []string, body string) (stdout, stderr string, err error) {
	t.Helper()
	if _, lookErr := exec.LookPath(shell); lookErr != nil {
		t.Skipf("%s non installé", shell)
	}
	script, scriptErr := shellinit.Script(shell, os.Getenv("TELEMACH_TEST_BIN"), "9.9.9")
	if scriptErr != nil {
		t.Fatal(scriptErr)
	}
	scriptFile := filepath.Join(t.TempDir(), "init.sh")
	if err := os.WriteFile(scriptFile, []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(shell, "-c", `eval "$(cat "$INIT")"`+"\n"+body) //nolint:gosec // test
	cmd.Env = append([]string{
		"PATH=" + e.bin + ":/usr/bin:/bin",
		"CALL_LOG=" + e.log,
		"XDG_RUNTIME_DIR=" + e.runtime,
		"INIT=" + scriptFile,
		"HOME=" + t.TempDir(),
	}, extraEnv...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err = cmd.Run()
	return out.String(), errb.String(), err
}

func readLog(t *testing.T, e env) string {
	t.Helper()
	data, err := os.ReadFile(e.log)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(data)
}

func leftovers(t *testing.T, e env) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(e.runtime, "telemach.*"))
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func forEachShell(t *testing.T, f func(t *testing.T, shell string)) {
	for _, sh := range []string{"bash", "zsh"} {
		t.Run(sh, func(t *testing.T) { f(t, sh) })
	}
}

// Garanties 1, 2 et 3 : code de sortie, stdout intact, fichier temporaire 0600 supprimé.
func TestFailureIsExplained(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		e := setup(t, true)
		stdout, stderr, err := run(t, shell, e, []string{"FAKE_RC=125"}, `docker run -p 8080:80 nginx; echo "rc=$?"`)
		if err != nil {
			t.Fatalf("erreur du shell : %v\n%s", err, stderr)
		}
		if stdout != "STDOUT-LINE\nrc=125\n" {
			t.Errorf("stdout modifié : %q", stdout)
		}
		if !strings.Contains(stderr, "port is already allocated") {
			t.Errorf("le message Docker d'origine doit rester affiché : %q", stderr)
		}
		if strings.Contains(stderr, "STDOUT-LINE") {
			t.Errorf("stdout ne doit jamais être recopié sur stderr : %q", stderr)
		}
		log := readLog(t, e)
		for _, want := range []string{
			"telemach explain --exit-code 125 --stderr-file ",
			"--docker-version 29.8.0 -- run -p 8080:80 nginx",
			"perm=600",
			"content=Error response from daemon: port is already allocated\n",
		} {
			if !strings.Contains(log, want) {
				t.Errorf("journal sans %q :\n%s", want, log)
			}
		}
		if files := leftovers(t, e); len(files) != 0 {
			t.Errorf("fichiers temporaires non supprimés : %v", files)
		}
	})
}

func TestSuccessIsNotExplained(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		e := setup(t, true)
		stdout, _, err := run(t, shell, e, nil, `docker ps; echo "rc=$?"`)
		if err != nil {
			t.Fatal(err)
		}
		if stdout != "STDOUT-LINE\nrc=0\n" {
			t.Errorf("stdout : %q", stdout)
		}
		if strings.Contains(readLog(t, e), "telemach") {
			t.Error("explain ne doit pas être appelé pour une commande réussie")
		}
	})
}

// Garantie 3 (interruption) : le fichier temporaire est supprimé après SIGINT.
func TestInterruptCleansUp(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		e := setup(t, true)
		if _, err := exec.LookPath(shell); err != nil {
			t.Skipf("%s non installé", shell)
		}
		script, _ := shellinit.Script(shell, "", "9.9.9")
		initFile := filepath.Join(t.TempDir(), "init.sh")
		if err := os.WriteFile(initFile, []byte(script), 0o600); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(shell, "-c", `eval "$(cat "$INIT")"`+"\ndocker run nginx") //nolint:gosec // test
		cmd.Env = []string{"PATH=" + e.bin + ":/usr/bin:/bin", "CALL_LOG=" + e.log, "XDG_RUNTIME_DIR=" + e.runtime, "INIT=" + initFile, "FAKE_SLEEP=5"}
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		// Attendre que docker tourne réellement : son premier message est déjà dans le
		// fichier temporaire.
		deadline := time.Now().Add(3 * time.Second)
		running := func() bool {
			for _, f := range leftovers(t, e) {
				if data, _ := os.ReadFile(f); len(data) > 0 {
					return true
				}
			}
			return false
		}
		for !running() && time.Now().Before(deadline) {
			time.Sleep(20 * time.Millisecond)
		}
		if !running() {
			t.Fatal("la commande docker aurait dû être en cours")
		}
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
		_ = cmd.Wait()
		time.Sleep(100 * time.Millisecond)
		if files := leftovers(t, e); len(files) != 0 {
			t.Errorf("fichiers temporaires non supprimés après SIGINT : %v", files)
		}
	})
}

// Garantie 4 : TELEMACH_DISABLE=1 désactive la capture.
func TestDisabled(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		e := setup(t, true)
		stdout, _, err := run(t, shell, e, []string{"FAKE_RC=1", "TELEMACH_DISABLE=1"}, `docker run nginx; echo "rc=$?"`)
		if err != nil {
			t.Fatal(err)
		}
		if stdout != "STDOUT-LINE\nrc=1\n" {
			t.Errorf("stdout : %q", stdout)
		}
		if strings.Contains(readLog(t, e), "telemach") {
			t.Error("explain ne doit pas être appelé quand TELEMACH_DISABLE=1")
		}
	})
}

// Garantie 5 : sans docker-telemach, la fonction se comporte comme docker.
func TestMissingBinary(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		e := setup(t, false)
		stdout, _, err := run(t, shell, e, []string{"FAKE_RC=2"}, `docker run nginx; echo "rc=$?"`)
		if err != nil {
			t.Fatal(err)
		}
		if stdout != "STDOUT-LINE\nrc=2\n" {
			t.Errorf("stdout : %q", stdout)
		}
		if files := leftovers(t, e); len(files) != 0 {
			t.Errorf("aucun fichier temporaire attendu : %v", files)
		}
	})
}

// Garantie 6 : en mode interactif, stderr n'est pas capturé et --interactive est transmis.
func TestInteractive(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		for _, body := range []string{`docker run -it alpine sh`, `docker exec -it web sh`, `docker attach web`, `docker login registry.example`} {
			e := setup(t, true)
			stdout, _, err := run(t, shell, e, []string{"FAKE_RC=127"}, body+`; echo "rc=$?"`)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasSuffix(stdout, "rc=127\n") {
				t.Errorf("%s : code de sortie non préservé : %q", body, stdout)
			}
			log := readLog(t, e)
			if !strings.Contains(log, "--interactive") || strings.Contains(log, "--stderr-file") {
				t.Errorf("%s : explain doit recevoir --interactive sans --stderr-file :\n%s", body, log)
			}
		}
	})
}

// Garantie 7 : docker --version n'est lu qu'une fois, à l'évaluation du script.
func TestVersionReadOnce(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		e := setup(t, true)
		_, _, err := run(t, shell, e, []string{"FAKE_RC=1"}, `docker run a; docker run b; docker ps; true`)
		if err != nil {
			t.Fatal(err)
		}
		log := readLog(t, e)
		if n := strings.Count(log, "docker --version"); n != 1 {
			t.Errorf("docker --version appelé %d fois, attendu 1 :\n%s", n, log)
		}
		var others []string
		for _, line := range strings.Split(strings.TrimSpace(log), "\n") {
			if strings.HasPrefix(line, "docker ") && line != "docker --version" &&
				line != "docker run a" && line != "docker run b" && line != "docker ps" {
				others = append(others, line)
			}
		}
		if len(others) != 0 {
			t.Errorf("appels docker inattendus : %v", others)
		}
	})
}

func TestUnsupportedShell(t *testing.T) {
	if _, err := shellinit.Script("fish", "", "9.9.9"); err == nil || !strings.Contains(err.Error(), "non pris en charge") {
		t.Errorf("erreur en français attendue pour fish, obtenu %v", err)
	}
}

// Les commandes « docker telemach … » ne sont jamais expliquées par TELEMACH.
func TestTelemachSubcommandNotExplained(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		e := setup(t, true)
		stdout, _, err := run(t, shell, e, []string{"FAKE_RC=1"}, `docker telemach check; echo "rc=$?"`)
		if err != nil {
			t.Fatal(err)
		}
		if stdout != "STDOUT-LINE\nrc=1\n" {
			t.Errorf("stdout : %q", stdout)
		}
		if strings.Contains(readLog(t, e), "telemach explain") {
			t.Error("explain ne doit pas être appelé pour docker telemach")
		}
	})
}

// Le binaire installé dans ~/.docker/cli-plugins n'est pas dans le PATH : le script généré
// doit l'appeler par son chemin absolu, même s'il contient espace ou apostrophe (LL-006).
func TestBinaryOutsidePath(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		e := setup(t, false) // aucun docker-telemach dans le PATH
		plugins := filepath.Join(t.TempDir(), "cli-plugins avec espace et l'apostrophe")
		if err := os.MkdirAll(plugins, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(plugins, "docker-telemach"), []byte(fakeTelemach), 0o700); err != nil { //nolint:gosec // script de test
			t.Fatal(err)
		}
		t.Setenv("TELEMACH_TEST_BIN", filepath.Join(plugins, "docker-telemach"))
		stdout, _, err := run(t, shell, e, []string{"FAKE_RC=125"}, `docker run nginx; echo "rc=$?"`)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(stdout, "rc=125\n") || !strings.Contains(readLog(t, e), "telemach explain --exit-code 125") {
			t.Errorf("explain doit être appelé par son chemin absolu :\nstdout=%q\n%s", stdout, readLog(t, e))
		}
	})
}

// Issue #7 : l'intégration exporte TELEMACH_SHELL (shell et version du plugin), visible des
// programmes lancés ensuite, dont « docker telemach last ».
func TestShellVariableExported(t *testing.T) {
	forEachShell(t, func(t *testing.T, shell string) {
		e := setup(t, true)
		stdout, stderr, err := run(t, shell, e, nil, `sh -c 'echo "vu=$TELEMACH_SHELL"'`)
		if err != nil {
			t.Fatal(err, stderr)
		}
		if want := "vu=" + shell + ":9.9.9\n"; stdout != want {
			t.Errorf("stdout = %q, attendu %q", stdout, want)
		}
	})
}
