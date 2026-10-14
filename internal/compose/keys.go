package compose

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/compose-spec/compose-go/v2/schema"

	"github.com/aboigues/telemach-plugin-docker/internal/textdist"
)

// allowedKeys liste les clés autorisées à chaque niveau, extraites du schéma officiel
// embarqué par compose-go (racine, puis services, networks, volumes, secrets, configs).
var allowedKeys = sync.OnceValue(func() map[string][]string {
	var spec map[string]any
	if err := json.Unmarshal([]byte(schema.Schema), &spec); err != nil {
		return nil
	}
	defs, _ := spec["$defs"].(map[string]any)
	out := map[string][]string{"": keysOf(spec, defs, map[string]bool{})}
	for section, def := range map[string]string{
		"services": "service", "networks": "network", "volumes": "volume", "secrets": "secret", "configs": "config",
	} {
		if d, ok := defs[def].(map[string]any); ok {
			out[section] = keysOf(d, defs, map[string]bool{})
		}
	}
	return out
})

// keysOf réunit les propriétés d'une définition, en suivant $ref, allOf, oneOf et anyOf.
func keysOf(node map[string]any, defs map[string]any, visited map[string]bool) []string {
	set := map[string]bool{}
	var walk func(n map[string]any)
	walk = func(n map[string]any) {
		if props, ok := n["properties"].(map[string]any); ok {
			for k := range props {
				set[k] = true
			}
		}
		if ref, ok := n["$ref"].(string); ok {
			name := strings.TrimPrefix(ref, "#/$defs/")
			if d, ok := defs[name].(map[string]any); ok && !visited[name] {
				visited[name] = true
				walk(d)
			}
		}
		for _, comb := range []string{"allOf", "oneOf", "anyOf"} {
			if list, ok := n[comb].([]any); ok {
				for _, item := range list {
					if m, ok := item.(map[string]any); ok {
						walk(m)
					}
				}
			}
		}
	}
	walk(node)
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// checkKeys signale les clés inconnues, avec la clé autorisée la plus proche.
func checkKeys(pf *parsedFile) []Issue {
	if pf.root == nil {
		return nil
	}
	allowed := allowedKeys()
	var issues []Issue
	check := func(parent string, keys []string, m *mappingView) {
		for _, k := range m.keys {
			if strings.HasPrefix(k.name, "x-") || contains(keys, k.name) {
				continue
			}
			path := joinPath(parent, k.name)
			is := Issue{
				Kind: KindUnknownKey, File: pf.path, Line: k.line, Column: k.column,
				KeyPath: path, Service: serviceOf(path), Blocking: true,
				Detail: fmt.Sprintf("la clé « %s » n'existe pas à cet endroit", k.name),
			}
			if s, ok := textdist.Closest(k.name, keys, 3); ok {
				is.Suggestion = s
			}
			issues = append(issues, is)
		}
	}
	root := viewOf(pf.root)
	check("", allowed[""], root)
	for _, section := range []string{"services", "networks", "volumes", "secrets", "configs"} {
		sec := root.child(section)
		if sec == nil {
			continue
		}
		for _, item := range sec.keys {
			if m := viewOf(item.value); m != nil {
				check(section+"."+item.name, allowed[section], m)
			}
		}
	}
	return issues
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
