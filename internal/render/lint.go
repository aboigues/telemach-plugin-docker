package render

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
)

// LintItem est un piège de la ligne de commande à afficher (feature 003).
type LintItem struct {
	Entry    *catalog.Entry
	Captures map[string]string
	Reminder bool // rappel d'une ligne (même piège affiché en complet il y a moins de 10 minutes)
}

func (it LintItem) info() bool { return it.Entry.Severity == "info" }

// RenderFindings écrit les pièges d'une ligne de commande en un seul affichage :
// avertissements puis informations (l'ordre reçu est conservé), rappels en une ligne.
func RenderFindings(w io.Writer, items []LintItem, o Options) error {
	t := o.Theme
	width := max(o.Width, 40)
	var b strings.Builder
	full := false
	for i, it := range items {
		switch {
		case it.Reminder:
			writeReminder(&b, it, t)
		case it.info():
			if i > 0 {
				b.WriteString("\n")
			}
			writeInfo(&b, it, t, width)
		default:
			if i > 0 {
				b.WriteString("\n")
			}
			writeWarning(&b, it, t, width)
			full = true
		}
	}
	if full {
		b.WriteString("\n" + t.Hint(LabelDeepen+" : docker telemach deepen") + "\n")
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// RenderInline ajoute à une explication d'erreur les pièges de la ligne de commande,
// présentés comme cause probable (FR-010).
func RenderInline(w io.Writer, items []LintItem, o Options) error {
	t := o.Theme
	width := max(o.Width, 40)
	var b strings.Builder
	b.WriteString("\n" + t.Title(LabelInCommand) + "\n")
	for _, it := range items {
		c := it.Captures
		b.WriteString(wrap(catalog.Render(it.Entry.Summary, c), width, indent) + "\n")
		b.WriteString(indent + t.Concept(LabelConcept+" — "+it.Entry.Concept.Name) + "\n")
		b.WriteString(wrap(it.Entry.Concept.Definition, width, indent+indent) + "\n")
		writeFixes(&b, it, t, width, indent)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func writeWarning(b *strings.Builder, it LintItem, t Theme, width int) {
	e, c := it.Entry, it.Captures
	b.WriteString(t.Title(wrapRest(SymbolWarning+" "+catalog.Render(e.Summary, c), width, indent)) + "\n")
	if eff := c["effect"]; eff != "" {
		b.WriteString(wrap(eff, width, indent) + "\n")
	}
	b.WriteString("\n" + t.Concept(LabelConcept+" — "+e.Concept.Name) + "\n")
	b.WriteString(wrap(e.Concept.Definition, width, indent) + "\n\n")
	b.WriteString(t.Cause(LabelCause) + "\n")
	b.WriteString(wrap(catalog.Render(e.Cause, c), width, indent) + "\n\n")
	b.WriteString(t.Fix(LabelFix) + "\n")
	writeFixes(b, it, t, width, "")
	if e.AntiPattern != nil {
		b.WriteString("\n" + t.Anti(LabelAnti) + "\n")
		b.WriteString(wrap(catalog.Render(e.AntiPattern.Description, c), width, indent) + "\n")
	}
}

// writeFixes écrit les corrections : description, commande à copier, et la note de
// masquage quand un secret a été remplacé par ••••••.
func writeFixes(b *strings.Builder, it LintItem, t Theme, width int, base string) {
	c := it.Captures
	for _, f := range it.Entry.Fixes {
		prefix := base + indent
		b.WriteString(prefix + wrapRest(catalog.Render(f.Description, c), width, strings.Repeat(" ", utf8.RuneCountInString(prefix))) + "\n")
		if f.Command != "" {
			b.WriteString(prefix + t.Highlight(catalog.Render(f.Command, c), "bash") + "\n")
		}
	}
	if c["masked"] == "1" {
		b.WriteString(base + indent + t.Dim("Remplace •••••• par ta valeur, ou mieux : passe-la avec --env-file.") + "\n")
	}
}

func writeInfo(b *strings.Builder, it LintItem, t Theme, width int) {
	e, c := it.Entry, it.Captures
	b.WriteString(t.Hint(wrapRest(SymbolInfo+" "+catalog.Render(e.Summary, c), width, indent)) + "\n")
	if len(e.Fixes) > 0 {
		f := e.Fixes[0]
		text := "💡 " + catalog.Render(f.Description, c)
		if f.Command != "" {
			text += " " + catalog.Render(f.Command, c)
		}
		b.WriteString(wrap(text, width, indent) + "\n")
	}
}

func writeReminder(b *strings.Builder, it LintItem, t Theme) {
	c := it.Captures
	text := fmt.Sprintf("%s Encore : %s", SymbolReminder, catalog.Render(it.Entry.Summary, c))
	if cmd := c["fixed_command"]; cmd != "" {
		text += " → " + cmd
	}
	b.WriteString(t.Dim(text+" — détails : docker telemach last") + "\n")
}
