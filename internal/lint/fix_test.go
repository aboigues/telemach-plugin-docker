package lint

import (
	"strings"
	"testing"
)

func TestFixedCommand(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		move    []int
		replace map[int]string
		want    string
	}{
		{
			"issue #9",
			[]string{"run", "-d", "nginx", "-p", "8090:80"},
			[]int{3, 4},
			nil,
			"docker run -d -p 8090:80 nginx",
		},
		{
			"ordre d'origine conservé",
			[]string{"run", "nginx:1.30-alpine", "--name", "web", "-p", "8080:80", "nginx", "-g", "daemon off;"},
			[]int{2, 3, 4, 5},
			nil,
			"docker run --name web -p 8080:80 nginx:1.30-alpine nginx -g 'daemon off;'",
		},
		{
			"options globales et exec",
			[]string{"--context", "x", "exec", "web", "-it", "sh"},
			[]int{4},
			nil,
			"docker --context x exec -it web sh",
		},
		{
			"remplacement",
			[]string{"run", "-v", "data:/d", "alpine:3.24", "ls"},
			nil,
			map[int]string{2: "./data:/d"},
			"docker run -v ./data:/d alpine:3.24 ls",
		},
		{
			"double tiret conservé",
			[]string{"run", "alpine:3.24", "--rm", "--", "-p"},
			[]int{2},
			nil,
			"docker run --rm alpine:3.24 -- -p",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			target := -1
			for i, a := range c.args {
				if a == "nginx" || a == "web" || strings.HasPrefix(a, "nginx:") || strings.HasPrefix(a, "alpine:") {
					target = i
					break
				}
			}
			got, masked := FixedCommand(c.args, target, c.move, c.replace)
			if got != c.want || masked {
				t.Errorf("FixedCommand = %q (masqué=%v), attendu %q", got, masked, c.want)
			}
		})
	}
}

func TestFixedCommandMasksSecrets(t *testing.T) {
	got, masked := FixedCommand([]string{"run", "nginx", "-e", "API_TOKEN=abc123secret"}, 1, []int{2, 3}, nil)
	if strings.Contains(got, "abc123secret") || !strings.Contains(got, "API_TOKEN=••••••") || !masked {
		t.Errorf("secret non masqué : %q (masqué=%v)", got, masked)
	}
}

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"nginx":            "nginx",
		"8090:80":          "8090:80",
		"./data:/d":        "./data:/d",
		"API_TOKEN=••••••": "API_TOKEN=••••••",
		"daemon off;":      "'daemon off;'",
		"it's":             `'it'\''s'`,
		"$HOME":            "'$HOME'",
		"a|b":              "'a|b'",
		"":                 "''",
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %q, attendu %q", in, got, want)
		}
	}
}
