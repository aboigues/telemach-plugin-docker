package main

import (
	"strings"
	"testing"
)

// Issue #7 : last et deepen signalent un terminal sans intégration shell, ou chargé par une
// autre version du plugin ; rien n'est ajouté quand l'intégration est à jour.
func TestShellStatusWarning(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want []string // vide : aucun avertissement
	}{
		{"intégration à jour", map[string]string{"TELEMACH_SHELL": "zsh:1.2.3"}, nil},
		{
			"intégration absente (zsh)",
			map[string]string{"TELEMACH_SHELL": "", "SHELL": "/usr/bin/zsh"},
			[]string{"n'est pas active dans ce terminal", `eval "$(docker telemach shell-init zsh)"`, "~/.zshrc"},
		},
		{
			"intégration absente (bash)",
			map[string]string{"TELEMACH_SHELL": "", "SHELL": "/bin/bash"},
			[]string{`shell-init bash`, "~/.bashrc"},
		},
		{
			"ancienne version",
			map[string]string{"TELEMACH_SHELL": "bash:0.3.0"},
			[]string{"chargée par la version 0.3.0", "version installée : 1.2.3", "exec bash"},
		},
	}
	for _, c := range cases {
		for _, sub := range []string{"last", "deepen"} {
			t.Run(c.name+"/"+sub, func(t *testing.T) {
				a, out, errb := testApp(t, c.env)
				if code := run(a, []string{sub, "--color=never", "--width", "200"}); code != 0 {
					t.Fatalf("code = %d", code)
				}
				got := errb.String()
				if len(c.want) == 0 {
					if strings.Contains(got, "⚠") {
						t.Errorf("aucun avertissement attendu :\n%s", got)
					}
				} else if !strings.HasPrefix(got, "⚠ ") {
					t.Errorf("l'avertissement doit précéder le reste :\n%s", got)
				}
				for _, want := range c.want {
					if !strings.Contains(got, want) {
						t.Errorf("%q absent :\n%s", want, got)
					}
				}
				if out.Len() != 0 {
					t.Errorf("rien sur stdout : %q", out.String())
				}
			})
		}
	}
}
