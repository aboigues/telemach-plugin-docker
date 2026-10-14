package compose

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/aboigues/telemach-plugin-docker/internal/redact"
)

// checkSecrets signale les secrets écrits en clair dans « environment » (FR-016). Une
// valeur qui fait référence à une variable (${…}) n'est pas un secret en clair.
func checkSecrets(pf *parsedFile) []Issue {
	services := viewOf(pf.root).child("services")
	if services == nil {
		return nil
	}
	var issues []Issue
	add := func(svc, name string, line, col int) {
		issues = append(issues, Issue{
			Kind: KindPlaintextSecret, File: pf.path, Line: line, Column: col, Service: svc,
			KeyPath: "services." + svc + ".environment." + name, Suggestion: ".env",
			Detail: fmt.Sprintf("la variable sensible « %s » contient une valeur écrite en clair", name),
		})
	}
	for _, svc := range services.keys {
		env := viewOf(svc.value).child("environment")
		if env != nil {
			for _, k := range env.keys {
				if redact.IsSensitiveName(k.name) && isLiteral(k.value) {
					add(svc.name, k.name, k.line, k.column)
				}
			}
			continue
		}
		// Forme liste : « - NOM=valeur ».
		for _, k := range viewOf(svc.value).keys {
			if k.name != "environment" || k.value.Kind != yaml.SequenceNode {
				continue
			}
			for _, item := range k.value.Content {
				name, value, ok := strings.Cut(item.Value, "=")
				if ok && value != "" && !strings.Contains(value, "${") && redact.IsSensitiveName(name) {
					add(svc.name, name, item.Line, item.Column)
				}
			}
		}
	}
	return issues
}

func isLiteral(n *yaml.Node) bool {
	return n != nil && n.Kind == yaml.ScalarNode && n.Value != "" && !strings.Contains(n.Value, "${")
}
