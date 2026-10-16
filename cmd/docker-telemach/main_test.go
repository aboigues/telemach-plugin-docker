package main

import (
	"reflect"
	"strings"
	"testing"
)

// La CLI Docker transmet au plugin les options globales placées avant son nom
// (« docker --context x telemach ps » → « --context x telemach ps »).
func TestSplitPluginArgs(t *testing.T) {
	cases := []struct {
		args           []string
		global, plugin []string
	}{
		{[]string{"telemach", "ps"}, nil, []string{"ps"}},
		{[]string{"--context", "x", "telemach", "ps"}, []string{"--context", "x"}, []string{"ps"}},
		{[]string{"-H", "tcp://h:2375", "--log-level=debug", "telemach", "last"}, []string{"-H", "tcp://h:2375", "--log-level=debug"}, []string{"last"}},
		{[]string{"--tlsverify", "telemach", "--version"}, []string{"--tlsverify"}, []string{"--version"}},
		{[]string{"--inconnue", "telemach", "ps"}, []string{"--inconnue"}, []string{"ps"}},
		// Complétion (docker completion bash|zsh) : la CLI appelle « __complete <plugin> … »
		// après avoir retiré les options globales (formes observées sur Docker 29.8.0).
		{[]string{"__complete", "telemach", "p"}, nil, []string{"__complete", "p"}},
		{[]string{"__complete", "telemach", "ps", "--c"}, nil, []string{"__complete", "ps", "--c"}},
		{[]string{"__complete", "telemach", ""}, nil, []string{"__complete", ""}},
		// Appel direct du binaire (intégration shell) : tout appartient au plugin.
		{[]string{"explain", "--exit-code", "1"}, nil, []string{"explain", "--exit-code", "1"}},
		{[]string{"--version"}, nil, []string{"--version"}},
		{nil, nil, nil},
	}
	for _, c := range cases {
		global, plugin := splitPluginArgs(c.args)
		if !reflect.DeepEqual(global, c.global) || !reflect.DeepEqual(plugin, c.plugin) {
			t.Errorf("splitPluginArgs(%q) = %q, %q ; attendu %q, %q", c.args, global, plugin, c.global, c.plugin)
		}
	}
}

// La complétion des sous-commandes passe par le plugin : sous-commandes visibles et options
// proposées, sous-commandes cachées (explain, watch, métadonnées) jamais.
func TestPluginCompletion(t *testing.T) {
	_, args := splitPluginArgs([]string{"__complete", "telemach", ""})
	a, out, _ := testApp(t, nil)
	if code := run(a, args); code != 0 {
		t.Fatalf("code = %d", code)
	}
	got := out.String()
	for _, want := range []string{"ps\t", "last\t", "lint\t", "check\t", "deepen\t", "shell-init\t"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q absent :\n%s", want, got)
		}
	}
	for _, hidden := range []string{"watch", "docker-cli-plugin-metadata", "completion"} {
		if strings.Contains(got, hidden+"\t") || strings.Contains(got, "\n"+hidden+"\n") {
			t.Errorf("%s ne doit pas être proposé :\n%s", hidden, got)
		}
	}
	_, args = splitPluginArgs([]string{"__complete", "telemach", "ps", "--c"})
	a, out, _ = testApp(t, nil)
	run(a, args)
	if !strings.Contains(out.String(), "--color") {
		t.Errorf("--color attendu :\n%s", out.String())
	}
	_, args = splitPluginArgs([]string{"__complete", "telemach", "ps", "--h"})
	a, out, _ = testApp(t, nil)
	run(a, args)
	if !strings.Contains(out.String(), "--help\taffiche l'aide") || strings.Contains(out.String(), "help for") {
		t.Errorf("aide en français attendue :\n%s", out.String())
	}
}
