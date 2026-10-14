package render

import (
	"io"
	"strings"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
)

// RenderDeepen affiche l'approfondissement d'une entrée : concept détaillé, exemple
// exécutable coloré, exercice (sur demande) et lien vers la documentation officielle.
func RenderDeepen(w io.Writer, e *catalog.Entry, captures map[string]string, exercise bool, o Options) error {
	t := o.Theme
	width := max(o.Width, 40)
	var b strings.Builder
	b.WriteString(t.Concept(LabelConcept+" — "+e.Concept.Name) + "\n")
	b.WriteString(wrap(e.Concept.Definition, width, indent) + "\n\n")
	b.WriteString(wrap(catalog.Render(e.Deepen.Explanation, captures), width, indent) + "\n\n")

	example := catalog.Render(e.Deepen.Example, captures)
	b.WriteString(t.Fix(LabelExample) + "\n")
	b.WriteString(indentLines(t.Highlight(example, lexerFor(example)), indent+"  ") + "\n")

	if exercise {
		b.WriteString("\n" + t.Cause(LabelExercise) + "\n")
		b.WriteString(wrap(catalog.Render(e.Deepen.Exercise, captures), width, indent) + "\n")
	} else {
		b.WriteString("\n" + t.Hint(LabelExercise+" : docker telemach deepen --exercise") + "\n")
	}
	b.WriteString("\n" + t.Dim(LabelDocLink+" : "+e.DocURL) + "\n")
	_, err := io.WriteString(w, b.String())
	return err
}

// lexerFor choisit la coloration syntaxique d'un exemple.
func lexerFor(code string) string {
	first := strings.TrimSpace(strings.SplitN(code, "\n", 2)[0])
	switch {
	case strings.HasPrefix(first, "FROM "):
		return "docker"
	case strings.HasSuffix(first, ":") || strings.HasPrefix(first, "services:") || strings.HasPrefix(first, "depends_on:"):
		return "yaml"
	}
	return "bash"
}

func indentLines(s, prefix string) string {
	return prefix + strings.ReplaceAll(s, "\n", "\n"+prefix)
}
