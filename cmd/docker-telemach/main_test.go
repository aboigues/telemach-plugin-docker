package main

import (
	"reflect"
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
