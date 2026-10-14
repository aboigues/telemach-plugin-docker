package compose

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
	"github.com/aboigues/telemach-plugin-docker/internal/render"
)

// RenderIssues affiche les problèmes Compose : emplacement, extrait coloré avec la ligne
// fautive mise en évidence, version corrigée, puis la pédagogie du type de problème
// (une seule fois par type). Sans problème, un message de succès explicite est affiché.
func RenderIssues(w io.Writer, dir string, files []string, issues []Issue, cat *catalog.Catalog, o render.Options) error {
	t := o.Theme
	dir = absDir(dir)
	var b strings.Builder
	if len(issues) == 0 {
		names := make([]string, len(files))
		for i, f := range files {
			names[i] = relative(dir, absPath(f))
		}
		b.WriteString(t.Success(render.SymbolSuccess+" Aucun problème détecté dans "+strings.Join(names, ", ")) + "\n")
		_, err := io.WriteString(w, b.String())
		return err
	}

	explained := map[Kind]bool{}
	for i, is := range issues {
		if i > 0 {
			b.WriteString("\n")
		}
		writeIssueHeader(&b, i+1, len(issues), dir, is, t)
		if len(is.Excerpt) > 0 {
			b.WriteString("\n" + indentBlock(t.HighlightLines(is.Excerpt, "yaml", is.Line, is.Column), "   ") + "\n")
		}
		if len(is.Fixed) > 0 {
			b.WriteString("\n   " + t.Fix(render.LabelFixedPart) + "\n")
			b.WriteString(indentBlock(t.HighlightLines(is.Fixed, "yaml", 0, 0), "   ") + "\n")
		}
		if explained[is.Kind] || cat == nil {
			continue
		}
		explained[is.Kind] = true
		if e, ok := cat.Get("compose." + string(is.Kind)); ok {
			b.WriteString("\n")
			writePedagogy(&b, e, is, t, o.Width)
		}
	}
	b.WriteString("\n" + t.Hint(render.LabelDeepen+" : docker telemach deepen") + "\n")
	_, err := io.WriteString(w, b.String())
	return err
}

func writeIssueHeader(b *strings.Builder, n, total int, dir string, is Issue, t render.Theme) {
	loc := relative(dir, absPath(is.File))
	if is.Line > 0 {
		loc += fmt.Sprintf(":%d", is.Line)
		if is.Column > 0 {
			loc += fmt.Sprintf(":%d", is.Column)
		}
	}
	header := fmt.Sprintf("%s %d/%d — %s", render.LabelProblem, n, total, loc)
	if !is.Blocking {
		header = fmt.Sprintf("⚠ Remarque %d/%d — %s", n, total, loc)
	}
	b.WriteString(t.Title(header) + "\n")
	var ctx []string
	if is.Service != "" {
		ctx = append(ctx, "service « "+is.Service+" »")
	}
	if is.KeyPath != "" {
		ctx = append(ctx, "clé « "+is.KeyPath+" »")
	}
	if len(ctx) > 0 {
		b.WriteString("   " + t.Dim(strings.Join(ctx, ", ")) + "\n")
	}
	detail := is.Detail
	switch {
	case is.Kind == KindFileNotFound:
		detail = "aucun des fichiers attendus n'existe ici : " + strings.Join(DefaultFileNames(), ", ")
	case is.Kind == KindUnknownKey && is.Suggestion != "":
		detail += fmt.Sprintf(" — tu voulais sans doute écrire « %s »", is.Suggestion)
	case is.Kind == KindTabIndent:
		detail = "tabulation (→) dans l'indentation"
	}
	if detail != "" {
		b.WriteString("   " + detail + "\n")
	}
}

func writePedagogy(b *strings.Builder, e *catalog.Entry, is Issue, t render.Theme, width int) {
	captures := map[string]string{"service": is.Service, "key": is.KeyPath, "suggestion": is.Suggestion}
	b.WriteString(t.Concept(render.LabelConcept+" — "+e.Concept.Name) + "\n")
	b.WriteString(render.Wrap(e.Concept.Definition, width, "   ") + "\n\n")
	b.WriteString(t.Cause(render.LabelCause) + "\n")
	b.WriteString(render.Wrap(catalog.Render(e.Cause, captures), width, "   ") + "\n\n")
	b.WriteString(t.Fix(render.LabelFixes) + "\n")
	for i, f := range e.Fixes {
		desc := catalog.Render(f.Description, captures)
		if !f.Safe {
			desc = t.Anti("(déconseillé)") + " " + desc
		}
		fmt.Fprintf(b, "   %d. %s\n", i+1, desc)
		if f.Command != "" {
			b.WriteString("      " + t.Highlight(catalog.Render(f.Command, captures), "bash") + "\n")
		}
	}
	if e.AntiPattern != nil {
		b.WriteString("\n" + t.Anti(render.LabelAnti) + "\n")
		b.WriteString(render.Wrap(e.AntiPattern.Description, width, "   ") + "\n")
	}
}

func indentBlock(s, prefix string) string {
	return prefix + strings.ReplaceAll(s, "\n", "\n"+prefix)
}

func absPath(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}
