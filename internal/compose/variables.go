package compose

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Références de variables : ${NOM}, ${NOM:-défaut}, ${NOM?erreur}, $NOM. « $$ » est un « $ »
// littéral et n'est pas une variable.
var varRef = regexp.MustCompile(`\$\$|\$\{([A-Za-z_][A-Za-z0-9_]*)((?::?[-?+])[^}]*)?\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

// checkVariables signale les variables utilisées sans valeur ni valeur par défaut. Seule
// leur présence est vérifiée : aucune valeur n'est lue dans le rapport.
func checkVariables(dir string, pf *parsedFile, getenv func(string) (string, bool)) []Issue {
	if pf.root == nil {
		return nil
	}
	dotenv := readDotEnvNames(filepath.Join(dir, ".env"))
	defined := func(name string) bool {
		if _, ok := getenv(name); ok {
			return true
		}
		return dotenv[name]
	}
	var issues []Issue
	var walk func(n *yaml.Node, path string)
	walk = func(n *yaml.Node, path string) {
		switch n.Kind {
		case yaml.MappingNode:
			for i := 0; i+1 < len(n.Content); i += 2 {
				walk(n.Content[i+1], joinPath(path, n.Content[i].Value))
			}
		case yaml.SequenceNode:
			for _, c := range n.Content {
				walk(c, path)
			}
		case yaml.ScalarNode:
			for _, m := range varRef.FindAllStringSubmatch(n.Value, -1) {
				if m[0] == "$$" {
					continue
				}
				name, modifier := m[1], m[2]
				if name == "" {
					name = m[3]
				}
				if strings.HasPrefix(modifier, "-") || strings.HasPrefix(modifier, ":-") ||
					strings.HasPrefix(modifier, "+") || strings.HasPrefix(modifier, ":+") || defined(name) {
					continue
				}
				required := strings.Contains(modifier, "?")
				issues = append(issues, Issue{
					Kind: KindUnresolvedVariable, File: pf.path, Line: n.Line, Column: n.Column,
					KeyPath: path, Service: serviceOf(path), Suggestion: name, Blocking: required,
					Detail: fmt.Sprintf("la variable « %s » n'est définie ni dans l'environnement ni dans le fichier .env", name),
				})
			}
		}
	}
	walk(pf.root, "")
	return issues
}

// readDotEnvNames lit seulement les noms des variables d'un fichier .env.
func readDotEnvNames(path string) map[string]bool {
	names := map[string]bool{}
	f, err := os.Open(path) //nolint:gosec // fichier .env du projet analysé, lu en lecture seule
	if err != nil {
		return names
	}
	defer f.Close() //nolint:errcheck
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		if name, _, ok := strings.Cut(line, "="); ok {
			names[strings.TrimSpace(name)] = true
		}
	}
	return names
}
