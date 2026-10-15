//go:build integration

package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

// imageVersion est la version injectée dans l'image de test.
const imageVersion = "0.0.0-test"

// testImage construit, une fois par test, l'image du Dockerfile à partir d'un contexte au
// format GoReleaser dockers_v2 (binaires linux sous linux/<arch>/, dossiers de compilation
// darwin sous dist/), avec les options de compilation de .goreleaser.yaml.
func testImage(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker non installé")
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("démon Docker indisponible")
	}
	ctx := t.TempDir()
	arch := runtime.GOARCH
	builds := []struct{ goos, pkg, out string }{
		{"linux", "docker-telemach", filepath.Join(ctx, "linux", arch, "docker-telemach")},
		{"linux", "telemach-image", filepath.Join(ctx, "linux", arch, "telemach-image")},
		{"darwin", "docker-telemach", filepath.Join(ctx, "dist", "docker-telemach_darwin_"+arch+"_test", "docker-telemach")},
	}
	for _, b := range builds {
		cmd := exec.Command("go", "build", "-trimpath",
			"-ldflags", "-s -w -buildid= -X main.version="+imageVersion,
			"-o", b.out, "../../cmd/"+b.pkg)
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+b.goos, "GOARCH="+arch)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("compilation de %s (%s) : %v\n%s", b.pkg, b.goos, err, out)
		}
	}
	dockerfile, err := filepath.Abs("../../Dockerfile")
	if err != nil {
		t.Fatal(err)
	}
	tag := "telemach-test:" + imageVersion
	build := exec.Command("docker", "buildx", "build", "--load", "--platform", "linux/"+arch,
		"-t", tag, "-f", dockerfile, ctx)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("construction de l'image : %v\n%s", err, out)
	}
	t.Cleanup(func() { _ = exec.Command("docker", "rmi", "-f", tag).Run() })
	return tag
}

// bash exécute script dans bash avec HOME isolé, comme un apprenant dans son terminal.
func bash(t *testing.T, home, script string, env ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command("bash", "-c", script)
	cmd.Env = append(os.Environ(), append([]string{"HOME=" + home, "NO_COLOR=1"}, env...)...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	code = 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("bash : %v", err)
	}
	return out.String(), errb.String(), code
}

// installCmd reproduit la commande d'installation du README (Linux et WSL2).
func installCmd(image string, extra string) string {
	return `docker run --rm --user "$(id -u):$(id -g)" ` + extra + `-v ~/.docker/cli-plugins:/sortie ` + image + ` install`
}

func TestImageInstall(t *testing.T) {
	image := testImage(t)

	t.Run("procédure du README", func(t *testing.T) {
		home := t.TempDir()
		out, errb, code := bash(t, home, "mkdir -p ~/.docker/cli-plugins\n"+installCmd(image, ""))
		if code != 0 {
			t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out, errb)
		}
		if !strings.Contains(out, "TELEMACH "+imageVersion+" installé") {
			t.Errorf("message de succès attendu :\n%s", out)
		}
		plugin := filepath.Join(home, ".docker", "cli-plugins", "docker-telemach")
		fi, err := os.Stat(plugin)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != 0o755 {
			t.Errorf("droits = %v, attendu 0755", fi.Mode().Perm())
		}
		if st, ok := fi.Sys().(*syscall.Stat_t); ok && int(st.Uid) != os.Getuid() {
			t.Errorf("propriétaire = %d, attendu l'utilisateur %d", st.Uid, os.Getuid())
		}
		version, err := exec.Command(plugin, "--version").Output()
		if err != nil {
			t.Fatal(err)
		}
		if first, _, _ := strings.Cut(string(version), "\n"); first != "TELEMACH PLUGIN DOCKER "+imageVersion {
			t.Errorf("--version = %q", first)
		}
		entries, _ := os.ReadDir(filepath.Dir(plugin))
		if len(entries) != 1 {
			t.Errorf("seul docker-telemach doit être présent : %v", entries)
		}

		// Réinstallation : déjà à jour.
		out, _, code = bash(t, home, installCmd(image, ""))
		if code != 0 || !strings.Contains(out, "déjà à jour") {
			t.Errorf("réinstallation : code %d\n%s", code, out)
		}
	})

	t.Run("macOS : binaire darwin", func(t *testing.T) {
		home := t.TempDir()
		_, errb, code := bash(t, home, "mkdir -p ~/.docker/cli-plugins\n"+installCmd(image, "-e HOST_OS=darwin "))
		if code != 0 {
			t.Fatalf("code = %d\n%s", code, errb)
		}
		data, err := os.ReadFile(filepath.Join(home, ".docker", "cli-plugins", "docker-telemach"))
		if err != nil {
			t.Fatal(err)
		}
		// En-tête Mach-O 64 bits (0xfeedfacf, petit-boutiste).
		if !bytes.HasPrefix(data, []byte{0xcf, 0xfa, 0xed, 0xfe}) {
			t.Errorf("binaire macOS (Mach-O) attendu")
		}
	})

	refusals := []struct {
		name, script, want string
		code               int
	}{
		{"sans montage", `docker run --rm --user "$(id -u):$(id -g)" ` + image + ` install`, "n'est pas monté", 1},
		{"sans --user", "mkdir -p ~/.docker/cli-plugins\ndocker run --rm -v ~/.docker/cli-plugins:/sortie " + image + " install", "--user", 1},
		// ~/.docker existe dès qu'on a utilisé Docker ; seul cli-plugins manque.
		{"dossier créé par Docker", "mkdir -p ~/.docker\n" + installCmd(image, ""), "appartient à root", 1},
		{"Windows", "mkdir -p ~/.docker/cli-plugins\n" + installCmd(image, "-e HOST_OS=windows "), "WSL2", 2},
		{"fichier étranger", "mkdir -p ~/.docker/cli-plugins\nprintf '#!/bin/sh\\n' > ~/.docker/cli-plugins/docker-telemach\n" + installCmd(image, ""), "n'est pas un binaire TELEMACH", 1},
	}
	for _, r := range refusals {
		t.Run(r.name, func(t *testing.T) {
			home := t.TempDir()
			_, errb, code := bash(t, home, r.script)
			if code != r.code {
				t.Errorf("code = %d, attendu %d\n%s", code, r.code, errb)
			}
			if !strings.Contains(errb, r.want) {
				t.Errorf("message attendu (%q) :\n%s", r.want, errb)
			}
			if r.name == "fichier étranger" {
				data, _ := os.ReadFile(filepath.Join(home, ".docker", "cli-plugins", "docker-telemach"))
				if string(data) != "#!/bin/sh\n" {
					t.Errorf("le fichier étranger a été modifié")
				}
			}
			// Un dossier créé par Docker appartient à root : vide, il reste supprimable par
			// l'utilisateur (droits du dossier parent), ce que fait le nettoyage de t.TempDir.
			if r.name == "dossier créé par Docker" {
				entries, _ := os.ReadDir(filepath.Join(home, ".docker", "cli-plugins"))
				if len(entries) != 0 {
					t.Errorf("rien ne doit être écrit : %v", entries)
				}
			}
		})
	}
}
