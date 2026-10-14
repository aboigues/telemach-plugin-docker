package compose

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/sirupsen/logrus"

	"github.com/aboigues/telemach-plugin-docker/internal/redact"
	"github.com/aboigues/telemach-plugin-docker/internal/sanitize"
)

func init() {
	// compose-go écrit ses avertissements (variables non définies…) sur stderr : TELEMACH
	// les analyse lui-même et les présente en français.
	logrus.SetOutput(io.Discard)
}

// Traductions des types attendus par le schéma.
var typeNames = map[string]string{
	"array": "une liste (lignes commençant par « - »)", "mapping": "un dictionnaire (clé: valeur)",
	"object": "un dictionnaire (clé: valeur)", "string": "un texte", "number": "un nombre",
	"integer": "un nombre entier", "boolean": "un booléen (true ou false)",
}

var semanticRules = []struct {
	re    *regexp.Regexp
	build func(m []string) Issue
}{
	{regexp.MustCompile(`additional properties '([^']+)' not allowed`), func(m []string) Issue {
		return Issue{Kind: KindUnknownKey, KeyPath: m[1]}
	}},
	{regexp.MustCompile(`(\S+) false schema`), func(m []string) Issue {
		return Issue{Kind: KindUnknownKey, KeyPath: m[1]}
	}},
	{regexp.MustCompile(`(\S+) must be a (\w+)`), func(m []string) Issue {
		t := typeNames[m[2]]
		if t == "" {
			t = m[2]
		}
		return Issue{Kind: KindWrongType, KeyPath: m[1], Detail: "la valeur doit être " + t}
	}},
	{regexp.MustCompile(`service "([^"]+)" depends on undefined service "([^"]+)"`), func(m []string) Issue {
		return Issue{
			Kind: KindUndefinedReference, KeyPath: "services." + m[1] + ".depends_on",
			Detail: fmt.Sprintf("le service « %s » n'est pas déclaré", m[2]),
		}
	}},
	{regexp.MustCompile(`service "([^"]+)" refers to undefined network ([^\s:]+)`), func(m []string) Issue {
		return Issue{
			Kind: KindUndefinedReference, KeyPath: "services." + m[1] + ".networks",
			Detail: fmt.Sprintf("le réseau « %s » n'est pas déclaré dans la section networks", m[2]), Suggestion: m[2],
		}
	}},
	{regexp.MustCompile(`service "([^"]+)" refers to undefined volume ([^\s:]+)`), func(m []string) Issue {
		return Issue{
			Kind: KindUndefinedReference, KeyPath: "services." + m[1] + ".volumes",
			Detail: fmt.Sprintf("le volume « %s » n'est pas déclaré dans la section volumes", m[2]), Suggestion: m[2],
		}
	}},
	{regexp.MustCompile(`dependency cycle detected: (\S+) -> (.+)`), func(m []string) Issue {
		return Issue{
			Kind: KindDependencyCycle, KeyPath: "services." + m[1] + ".depends_on",
			Detail: "cycle : " + m[1] + " -> " + m[2],
		}
	}},
	{regexp.MustCompile(`service "([^"]+)" has neither an image nor a build context`), func(m []string) Issue {
		return Issue{
			Kind: KindUndefinedReference, KeyPath: "services." + m[1], Suggestion: "image",
			Detail: "le service n'indique ni « image » ni « build »",
		}
	}},
}

// checkSemantic charge le projet avec compose-go, le parseur de référence de Docker
// Compose, et traduit sa première erreur en problème localisé. Les valeurs des variables
// d'environnement ne sont utilisées qu'en mémoire, jamais affichées ni conservées.
func checkSemantic(dir string, files []*parsedFile) []Issue {
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.path
	}
	opts, err := cli.NewProjectOptions(paths,
		cli.WithWorkingDirectory(dir),
		cli.WithName("telemach-check"),
		cli.WithOsEnv,
		cli.WithDotEnv,
		cli.WithLoadOptions(func(o *loader.Options) {
			o.SkipResolveEnvironment = true
			o.ResourceLoaders = nil // aucun include distant (git, oci) : pas de réseau
		}),
	)
	if err != nil {
		return []Issue{genericIssue(err, files)}
	}
	if _, err := opts.LoadProject(context.Background()); err != nil {
		return []Issue{toIssue(err, files)}
	}
	return nil
}

func toIssue(err error, files []*parsedFile) Issue {
	msg := err.Error()
	for _, r := range semanticRules {
		if m := r.re.FindStringSubmatch(msg); m != nil {
			is := r.build(m)
			is.KeyPath = strings.TrimPrefix(is.KeyPath, ".")
			is.Blocking = true
			is.Service = serviceOf(is.KeyPath)
			locate(&is, msg, files)
			return is
		}
	}
	return genericIssue(err, files)
}

// genericIssue conserve une erreur non reconnue, nettoyée et masquée.
func genericIssue(err error, files []*parsedFile) Issue {
	is := Issue{Kind: KindWrongType, Blocking: true, Detail: redact.Redact(sanitize.Clean(err.Error()))}
	if len(files) > 0 {
		is.File = files[0].path
	}
	return is
}

// locate retrouve le fichier et la position d'un chemin de clé (ou de son parent).
func locate(is *Issue, msg string, files []*parsedFile) {
	ordered := files
	for _, f := range files {
		if strings.Contains(msg, f.path) {
			ordered = []*parsedFile{f}
			break
		}
	}
	for path := is.KeyPath; path != ""; path = parent(path) {
		for i := len(ordered) - 1; i >= 0; i-- {
			if p, ok := ordered[i].index[path]; ok {
				is.File, is.Line, is.Column = ordered[i].path, p.line, p.column
				return
			}
		}
	}
	if len(ordered) > 0 {
		is.File = ordered[0].path
	}
}

func parent(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		return path[:i]
	}
	return ""
}

// relative renvoie un chemin lisible, relatif au dossier analysé quand c'est possible.
func relative(dir, path string) string {
	if rel, err := filepath.Rel(dir, path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}
