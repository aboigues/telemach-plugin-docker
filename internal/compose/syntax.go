package compose

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

// position est l'emplacement d'une clé ou d'une valeur dans un fichier.
type position struct {
	line, column int
	value        *yaml.Node // nœud de la valeur associée à la clé
}

// parsedFile est un fichier Compose lu et indexé.
type parsedFile struct {
	path  string
	lines []string
	root  *yaml.Node          // nœud racine (mapping), nil si vide
	index map[string]position // chemin de clé (services.web.ports) → position de la clé
}

var yamlErrLine = regexp.MustCompile(`line (\d+)(?:, column (\d+))?`)

// parseFile lit un fichier en lecture seule et effectue la passe syntaxique : lisibilité,
// encodage, tabulations, YAML valide et clés en double.
func parseFile(path string) (*parsedFile, []Issue) {
	data, err := os.ReadFile(path) //nolint:gosec // fichier Compose désigné par l'apprenant
	if err != nil {
		if os.IsNotExist(err) {
			return nil, []Issue{{Kind: KindFileNotFound, File: path, Blocking: true, Detail: "le fichier n'existe pas"}}
		}
		detail := "le fichier ne peut pas être lu"
		if os.IsPermission(err) {
			detail = "les droits du fichier ne permettent pas de le lire"
		}
		return nil, []Issue{{Kind: KindFileUnreadable, File: path, Blocking: true, Detail: detail}}
	}
	if !utf8.Valid(data) {
		return nil, []Issue{{
			Kind: KindFileUnreadable, File: path, Blocking: true,
			Detail: "le fichier n'est pas encodé en UTF-8 (accents enregistrés dans un autre encodage)",
		}}
	}
	pf := &parsedFile{path: path, lines: strings.Split(string(data), "\n"), index: map[string]position{}}

	var issues []Issue
	for i, l := range pf.lines {
		indent := l[:len(l)-len(strings.TrimLeft(l, " \t"))]
		if col := strings.IndexByte(indent, '\t'); col >= 0 {
			issues = append(issues, Issue{Kind: KindTabIndent, File: path, Line: i + 1, Column: col + 1, Blocking: true})
		}
	}
	if len(issues) > 0 {
		return pf, issues
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return pf, []Issue{pf.syntaxIssue(err)}
	}
	if len(doc.Content) > 0 && doc.Content[0].Kind == yaml.MappingNode {
		pf.root = doc.Content[0]
		issues = append(issues, pf.indexNode(pf.root, "")...)
	}
	return pf, issues
}

// syntaxIssue traduit une erreur de l'analyseur YAML en problème localisé.
func (pf *parsedFile) syntaxIssue(err error) Issue {
	msg := err.Error()
	is := Issue{Kind: KindYAMLSyntax, File: pf.path, Blocking: true, Detail: msg}
	if m := yamlErrLine.FindStringSubmatch(msg); m != nil {
		is.Line, _ = strconv.Atoi(m[1])
	}
	// « mapping values are not allowed » à la ligne N vient souvent d'un « : » oublié à la
	// ligne précédente (« web » au lieu de « web: »).
	if strings.Contains(msg, "mapping values are not allowed") && is.Line > 1 {
		for prev := is.Line - 1; prev >= 1; prev-- {
			t := strings.TrimSpace(pf.lines[prev-1])
			if t == "" || strings.HasPrefix(t, "#") {
				continue
			}
			if !strings.Contains(t, ":") && !strings.HasPrefix(t, "-") {
				is.Line = prev
				is.Suggestion = t + ":"
				is.Detail = fmt.Sprintf("il manque « : » après « %s »", t)
			}
			break
		}
	}
	return is
}

// indexNode enregistre la position de chaque clé et détecte les clés en double.
func (pf *parsedFile) indexNode(n *yaml.Node, prefix string) []Issue {
	if n.Kind != yaml.MappingNode {
		return nil
	}
	var issues []Issue
	seen := map[string]bool{}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		path := joinPath(prefix, k.Value)
		if seen[k.Value] {
			issues = append(issues, Issue{
				Kind: KindYAMLSyntax, File: pf.path, Line: k.Line, Column: k.Column,
				KeyPath: path, Service: serviceOf(path), Blocking: true,
				Detail: fmt.Sprintf("la clé « %s » est définie deux fois", k.Value),
			})
			continue
		}
		seen[k.Value] = true
		pf.index[path] = position{line: k.Line, column: k.Column, value: v}
		issues = append(issues, pf.indexNode(v, path)...)
	}
	return issues
}

func joinPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

// serviceOf extrait le nom du service d'un chemin services.<nom>…
func serviceOf(path string) string {
	parts := strings.SplitN(path, ".", 3)
	if len(parts) >= 2 && parts[0] == "services" {
		return parts[1]
	}
	return ""
}
