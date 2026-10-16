package command

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/aboigues/telemach-plugin-docker/internal/redact"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want Command
	}{
		{
			name: "run avec publication de port",
			args: []string{"run", "-p", "8080:80", "nginx"},
			want: Command{Family: FamilyDocker, Subcommand: "run", Flags: map[string]string{"p": "8080:80"}, Positionals: []string{"nginx"}},
		},
		{
			name: "compose avec plusieurs fichiers",
			args: []string{"compose", "-f", "a.yaml", "-f", "b.yaml", "up"},
			want: Command{Family: FamilyCompose, Subcommand: "up", Flags: map[string]string{"f": "b.yaml"}, ComposeFiles: []string{"a.yaml", "b.yaml"}},
		},
		{
			name: "option globale --context",
			args: []string{"--context", "x", "run", "alpine"},
			want: Command{Family: FamilyDocker, Subcommand: "run", Flags: map[string]string{}, Positionals: []string{"alpine"}},
		},
		{
			name: "option globale -H",
			args: []string{"-H", "tcp://127.0.0.1:2375", "ps"},
			want: Command{Family: FamilyDocker, Subcommand: "ps", Flags: map[string]string{}},
		},
		{
			name: "exec interactif",
			args: []string{"exec", "-it", "web", "sh"},
			want: Command{Family: FamilyDocker, Subcommand: "exec", Flags: map[string]string{"i": "", "t": ""}, Positionals: []string{"web", "sh"}, Interactive: true},
		},
		{
			name: "attach est interactif",
			args: []string{"attach", "web"},
			want: Command{Family: FamilyDocker, Subcommand: "attach", Flags: map[string]string{}, Positionals: []string{"web"}, Interactive: true},
		},
		{
			name: "login sans --password-stdin est interactif",
			args: []string{"login", "registry.example"},
			want: Command{Family: FamilyDocker, Subcommand: "login", Flags: map[string]string{}, Positionals: []string{"registry.example"}, Interactive: true},
		},
		{
			name: "login avec --password-stdin n'est pas interactif",
			args: []string{"login", "--password-stdin", "-u", "bob"},
			want: Command{Family: FamilyDocker, Subcommand: "login", Flags: map[string]string{"password-stdin": "", "u": "bob"}},
		},
		{
			name: "build avec -f",
			args: []string{"build", "-f", "Dockerfile", "."},
			want: Command{Family: FamilyDocker, Subcommand: "build", Flags: map[string]string{"f": "Dockerfile"}, Positionals: []string{"."}},
		},
		{
			name: "valeur -e masquée",
			args: []string{"run", "-e", "DB_PASSWORD=secret", "alpine"},
			want: Command{Family: FamilyDocker, Subcommand: "run", Flags: map[string]string{"e": "DB_PASSWORD=" + redact.Mask}, Positionals: []string{"alpine"}},
		},
		{
			name: "sous-commande de gestion container rm",
			args: []string{"container", "rm", "web"},
			want: Command{Family: FamilyDocker, Subcommand: "rm", Flags: map[string]string{}, Positionals: []string{"web"}},
		},
		{
			name: "les arguments après l'image ne sont pas des options",
			args: []string{"run", "--rm", "alpine", "sh", "-c", "exit 3"},
			want: Command{Family: FamilyDocker, Subcommand: "run", Flags: map[string]string{"rm": ""}, Positionals: []string{"alpine", "sh", "-c", "exit 3"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, Parse(tt.args)); diff != "" {
				t.Errorf("Parse(%q) (-attendu +obtenu) :\n%s", tt.args, diff)
			}
		})
	}
}

func TestImage(t *testing.T) {
	if got := Parse([]string{"run", "-d", "--name", "web", "-p", "8080:80", "nginx:1.27", "sh"}).Image(); got != "nginx:1.27" {
		t.Errorf("Image() = %q, attendu nginx:1.27", got)
	}
	if got := Parse([]string{"ps"}).Image(); got != "" {
		t.Errorf("Image() = %q, attendu vide", got)
	}
}

// create accepte les options à valeur de run ; une option à valeur peu courante ne doit
// jamais faire prendre sa valeur pour l'image (feature 003, U1).
func TestCreateAndRareValueFlags(t *testing.T) {
	for _, args := range [][]string{
		{"create", "-p", "8080:80", "--name", "web", "nginx"},
		{"container", "create", "-p", "8080:80", "--name", "web", "nginx"},
		{"run", "--cpuset-cpus", "0", "--memory-swap", "1g", "--pids-limit", "100", "nginx"},
		{"run", "--gpus", "all", "--health-interval", "5s", "--group-add", "audio", "nginx"},
	} {
		if got := Parse(args).Image(); got != "nginx" {
			t.Errorf("%v : image = %q, attendu nginx", args, got)
		}
	}
}

func TestLocate(t *testing.T) {
	cases := []struct {
		args   []string
		sub    string
		target int
	}{
		{[]string{"run", "-d", "nginx", "-p", "8090:80"}, "run", 2},
		{[]string{"--context", "x", "run", "-p", "1:2", "nginx"}, "run", 5},
		{[]string{"container", "run", "--rm", "alpine", "ls"}, "run", 3},
		{[]string{"exec", "-it", "web", "sh"}, "exec", 2},
		{[]string{"create", "--cpuset-cpus", "0", "nginx"}, "create", 3},
		{[]string{"run", "-d"}, "run", -1},
		{[]string{"ps", "-a"}, "ps", -1},
	}
	for _, c := range cases {
		l := Locate(c.args)
		if l.Subcommand != c.sub || l.Target != c.target {
			t.Errorf("%v : %+v, attendu sous-commande %s et cible %d", c.args, l, c.sub, c.target)
		}
	}
}
