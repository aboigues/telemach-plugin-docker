package main

import (
	"strings"
	"testing"
)

func TestLintSilentCases(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		args []string
	}{
		{"TELEMACH_LINT=0", map[string]string{"TELEMACH_LINT": "0"}, []string{"lint", "--after", "--", "run", "-d", "nginx", "-p", "8090:80"}},
		{"sous-commande non analysée", nil, []string{"lint", "--after", "--", "ps", "-a"}},
		{"ligne correcte", nil, []string{"lint", "--", "run", "--rm", "-p", "8080:80", "nginx:1.30-alpine"}},
		{"sans arguments", nil, []string{"lint"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, out, errb := testApp(t, c.env)
			if code := run(a, c.args); code != 0 {
				t.Errorf("code = %d\n%s", code, errb.String())
			}
			if out.Len() != 0 || errb.Len() != 0 {
				t.Errorf("aucune sortie attendue :\nstdout=%q\nstderr=%q", out.String(), errb.String())
			}
		})
	}
}

func TestLintInHelp(t *testing.T) {
	a, out, _ := testApp(t, nil)
	run(a, []string{"--help"})
	if !strings.Contains(out.String(), "lint") {
		t.Errorf("lint absent de l'aide :\n%s", out.String())
	}
}
