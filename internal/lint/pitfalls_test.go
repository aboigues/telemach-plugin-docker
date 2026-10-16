package lint

import (
	"strings"
	"testing"
)

// pitfallEnv : seul le dossier « data » existe, aucune variable n'est définie.
var pitfallEnv = Env{HasVar: func(string) bool { return false }, DirExists: func(n string) bool { return n == "data" }}

func findingFor(t *testing.T, id string, args ...string) Finding {
	t.Helper()
	for _, f := range Analyze(args, pitfallEnv) {
		if f.EntryID == id {
			return f
		}
	}
	t.Fatalf("%s non trouvé pour docker %s", id, strings.Join(args, " "))
	return Finding{}
}

func TestPitfallFixedCommands(t *testing.T) {
	cases := []struct {
		id   string
		args []string
		want string
	}{
		{"cmdline.named-volume-not-dir", []string{"run", "-v", "data:/app/data", "alpine:3.24", "ls", "/app/data"}, "docker run -v ./data:/app/data alpine:3.24 ls /app/data"},
		{"cmdline.named-volume-not-dir", []string{"run", "--volume=data:/d:ro", "alpine:3.24"}, "docker run --volume=./data:/d:ro alpine:3.24"},
		{"cmdline.random-host-port", []string{"run", "-d", "-p", "80", "nginx:1.30-alpine"}, "docker run -d -p 8080:80 nginx:1.30-alpine"},
		{"cmdline.random-host-port", []string{"run", "--publish=53/udp", "alpine:3.24"}, "docker run --publish=8080:53/udp alpine:3.24"},
		{"cmdline.env-unset", []string{"run", "-e", "DB_HOST", "mon-app:1.0"}, "docker run -e 'DB_HOST=<valeur>' mon-app:1.0"},
		{"cmdline.ports-ignored-host-network", []string{"run", "-d", "--network", "host", "-p", "8080:80", "nginx:1.30-alpine"}, "docker run -d --network host nginx:1.30-alpine"},
		{"cmdline.ports-ignored-host-network", []string{"run", "-dP", "--net=host", "nginx:1.30-alpine"}, "docker run -d --net=host nginx:1.30-alpine"},
		{"cmdline.ports-ignored-host-network", []string{"run", "-dp", "8080:80", "--net=host", "nginx:1.30-alpine"}, "docker run -d --net=host nginx:1.30-alpine"},
		{"cmdline.ports-ignored-host-network", []string{"run", "-P", "--publish=9000:9000", "--network=host", "alpine:3.24"}, "docker run --network=host alpine:3.24"},
	}
	for _, c := range cases {
		f := findingFor(t, c.id, c.args...)
		if got := f.Captures["fixed_command"]; got != c.want {
			t.Errorf("docker %s\n    obtenu  %s\n    attendu %s", strings.Join(c.args, " "), got, c.want)
		}
	}
}

// Principe III : ni la valeur d'un secret ni rien de la ligne masquée ne sort de l'analyse.
func TestPitfallSecretNeverCaptured(t *testing.T) {
	const secret = "SuperSecret123"
	args := []string{"run", "-v", "data:/d", "-e", "DB_PASSWORD=" + secret, "-p", "80", "postgres:18-alpine"}
	findings := Analyze(args, pitfallEnv)
	if len(findings) != 3 {
		t.Fatalf("3 pièges attendus : %+v", findings)
	}
	for _, f := range findings {
		if strings.Contains(f.Fingerprint, secret) {
			t.Errorf("%s : secret dans l'empreinte %q", f.EntryID, f.Fingerprint)
		}
		for k, v := range f.Captures {
			if strings.Contains(v, secret) {
				t.Errorf("%s : secret dans la capture %s = %q", f.EntryID, k, v)
			}
		}
		if f.EntryID != "cmdline.secret-in-command" && f.Captures["masked"] != "1" {
			t.Errorf("%s : commande corrigée masquée sans le signaler", f.EntryID)
		}
	}
	if got := findingFor(t, "cmdline.secret-in-command", args...).Captures["name"]; got != "DB_PASSWORD" {
		t.Errorf("name = %q, attendu DB_PASSWORD", got)
	}
}

func TestPitfallPrivilegedOption(t *testing.T) {
	cases := map[string][]string{
		"--privileged": {"run", "--privileged", "alpine:3.24"},
		"-v /var/run/docker.sock:/var/run/docker.sock":     {"run", "-v", "/var/run/docker.sock:/var/run/docker.sock", "alpine:3.24"},
		"--mount type=bind,src=/run/docker.sock,target=/s": {"run", "--mount", "type=bind,src=/run/docker.sock,target=/s", "alpine:3.24"},
	}
	for want, args := range cases {
		if got := findingFor(t, "cmdline.privileged", args...).Captures["option"]; got != want {
			t.Errorf("option = %q, attendu %q", got, want)
		}
	}
}

// Sans lecture locale disponible, les règles qui en dépendent se taisent (doute = silence).
func TestPitfallNilEnvIsSilent(t *testing.T) {
	if f := Analyze([]string{"run", "-v", "data:/d", "-e", "DB_HOST", "alpine:3.24"}, Env{}); len(f) != 0 {
		t.Errorf("aucun piège attendu sans lecture locale : %+v", f)
	}
}

func TestSecretName(t *testing.T) {
	for name, want := range map[string]bool{
		"DB_PASSWORD": true, "apiKey": true, "AWS_SECRET_ACCESS_KEY": true, "GITHUB_TOKEN": true,
		"BASIC_AUTH": true, "db.password": true,
		"MONKEY": false, "AUTHOR": false, "KEYCLOAK_URL": false, "SECRETS_DIR": false,
		"SSH_KEY_PATH": false, "TOKEN_URL": false, "AWS_ACCESS_KEY_ID": false, "KEYBOARD_LAYOUT": false,
	} {
		if got := secretName(name); got != want {
			t.Errorf("secretName(%q) = %v, attendu %v", name, got, want)
		}
	}
}
