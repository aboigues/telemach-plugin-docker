package compose

import (
	"strings"

	"github.com/aboigues/telemach-plugin-docker/internal/redact"
	"github.com/aboigues/telemach-plugin-docker/internal/render"
	"github.com/aboigues/telemach-plugin-docker/internal/sanitize"
)

// contextLines est le nombre de lignes affichées avant et après la ligne fautive.
const contextLines = 3

// attachExcerpt ajoute l'extrait du fichier autour du problème et, quand c'est possible,
// la version corrigée. Toutes les lignes sont nettoyées et masquées.
func attachExcerpt(is *Issue, pf *parsedFile) {
	if pf == nil || is.Line < 1 || is.Line > len(pf.lines) {
		return
	}
	from, to := max(1, is.Line-contextLines), min(len(pf.lines), is.Line+contextLines)
	for n := from; n <= to; n++ {
		text := pf.lines[n-1]
		if n == to && text == "" {
			break
		}
		is.Excerpt = append(is.Excerpt, render.Line{Number: n, Text: clean(text)})
	}
	if fixed, ok := fixLine(is, pf.lines[is.Line-1]); ok {
		is.Fixed = []render.Line{{Number: is.Line, Text: clean(fixed)}}
	}
}

func clean(s string) string { return redact.Redact(sanitize.Clean(s)) }

// fixLine propose la ligne corrigée pour les problèmes qui s'y prêtent.
func fixLine(is *Issue, line string) (string, bool) {
	switch is.Kind {
	case KindTabIndent:
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		return strings.ReplaceAll(indent, "\t", "    ") + strings.TrimLeft(line, " \t"), true
	case KindUnknownKey:
		key := is.KeyPath[strings.LastIndex(is.KeyPath, ".")+1:]
		if is.Suggestion != "" && strings.Contains(line, key+":") {
			return strings.Replace(line, key+":", is.Suggestion+":", 1), true
		}
	case KindYAMLSyntax:
		if is.Suggestion != "" {
			trimmed := strings.TrimSpace(line)
			return strings.Replace(line, trimmed, is.Suggestion, 1), true
		}
	case KindPlaintextSecret:
		name := is.KeyPath[strings.LastIndex(is.KeyPath, ".")+1:]
		indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
		if strings.HasPrefix(strings.TrimSpace(line), "-") {
			return indent + "- " + name + "=${" + name + "}", true
		}
		return indent + name + ": ${" + name + "}", true
	}
	return "", false
}
