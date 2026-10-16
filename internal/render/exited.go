package render

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
)

// Libellés de l'analyse d'un conteneur arrêté (feature 004).
const (
	LabelLogTail = "Fin du journal"
	LabelOthers  = "Autres conteneurs arrêtés en erreur :"
)

// ExitedView est l'analyse d'un conteneur arrêté, prête à afficher : captures nettoyées et
// masquées (name, image, code, lifetime, age, state, variable, restart_count, restart_policy,
// compose_project, compose_service…), fin du journal et autres conteneurs en erreur.
type ExitedView struct {
	Entry          *catalog.Entry
	Captures       map[string]string
	Log            []string // au plus 20 lignes, déjà nettoyées et masquées
	LogUnavailable string   // raison, si le journal ne peut pas être lu
	Others         []OtherLine
}

// OtherLine est un autre conteneur en erreur, résumé en une ligne.
type OtherLine struct {
	Name    string
	Code    int
	Age     string
	Summary string
}

// RenderExited affiche l'analyse : conteneur, fin du journal, concept, cause, corrections.
func RenderExited(w io.Writer, v ExitedView, o Options) error {
	t := o.Theme
	width := max(o.Width, 40)
	c := v.Captures
	var b strings.Builder

	b.WriteString(t.Title(wrapRest(SymbolError+" "+exitedHeader(c), width, indent)) + "\n")
	if v.Entry != nil {
		b.WriteString(wrap(catalog.Render(v.Entry.Summary, c), width, indent) + "\n")
	}
	if p, s := c["compose_project"], c["compose_service"]; p != "" {
		b.WriteString(t.Dim(wrap(fmt.Sprintf("Projet Compose « %s », service « %s ».", p, s), width, indent)) + "\n")
	}
	if n := c["restart_count"]; n != "" && n != "0" {
		b.WriteString(t.Dim(wrap(fmt.Sprintf("Redémarré %s fois (politique de redémarrage : %s).", n, c["restart_policy"]), width, indent)) + "\n")
	}

	writeLogTail(&b, v, t, width)

	if e := v.Entry; e != nil {
		b.WriteString("\n" + t.Concept(LabelConcept+" — "+e.Concept.Name) + "\n")
		b.WriteString(wrap(e.Concept.Definition, width, indent) + "\n\n")
		b.WriteString(t.Cause(LabelCause) + "\n")
		b.WriteString(wrap(catalog.Render(e.Cause, c), width, indent) + "\n\n")
		b.WriteString(t.Fix(LabelFixes) + "\n")
		for i, f := range e.Fixes {
			prefix := fmt.Sprintf("%s%d. ", indent, i+1)
			pad := strings.Repeat(" ", utf8.RuneCountInString(prefix))
			desc := catalog.Render(f.Description, c)
			if !f.Safe {
				desc = t.Anti("(déconseillé)") + " " + desc
			}
			b.WriteString(prefix + wrapRest(desc, width, pad) + "\n")
			if f.Command != "" {
				b.WriteString(pad + t.Highlight(catalog.Render(f.Command, c), "bash") + "\n")
			}
		}
		if e.AntiPattern != nil {
			b.WriteString("\n" + t.Anti(LabelAnti) + "\n")
			b.WriteString(wrap(catalog.Render(e.AntiPattern.Description, c), width, indent) + "\n")
		}
		b.WriteString("\n" + t.Hint(LabelDeepen+" : docker telemach deepen") + "\n")
	}

	writeOthers(&b, v.Others, t, width)
	_, err := io.WriteString(w, b.String())
	return err
}

// nbsp lie les mots d'une durée (« 2 min », « il y a 3 h ») pour que le repli ne les sépare pas.
func nbsp(s string) string { return strings.ReplaceAll(s, " ", "\u00a0") }

// exitedHeader décrit le conteneur : nom, image, code, durée de vie et ancienneté.
func exitedHeader(c map[string]string) string {
	who := fmt.Sprintf("Le conteneur « %s » (%s)", c["name"], c["image"])
	if c["state"] == "created" {
		return fmt.Sprintf("%s n'a pas pu démarrer (code %s, %s).", who, c["code"], nbsp(c["age"]))
	}
	s := fmt.Sprintf("%s s'est arrêté avec le code %s", who, c["code"])
	if l := c["lifetime"]; l != "" {
		s += ", " + nbsp(l) + " après son démarrage"
	}
	if a := c["age"]; a != "" {
		s += " (" + nbsp(a) + ")"
	}
	return s + "."
}

// writeLogTail affiche la fin du journal, une ligne par ligne d'origine : jamais repliée
// (l'alignement du journal compte), tronquée à la largeur si besoin.
func writeLogTail(b *strings.Builder, v ExitedView, t Theme, width int) {
	title := "── " + LabelLogTail
	switch {
	case v.LogUnavailable != "":
		b.WriteString("\n" + t.Dim(title+" : illisible") + "\n")
		b.WriteString(t.Dim(wrap(v.LogUnavailable, width, indent)) + "\n")
		return
	case len(v.Log) == 0:
		b.WriteString("\n" + t.Dim(title+" : vide") + "\n")
		return
	}
	title += fmt.Sprintf(" (%d dernières lignes) ", len(v.Log))
	b.WriteString("\n" + t.Dim(title+strings.Repeat("─", max(3, width-utf8.RuneCountInString(title)))) + "\n")
	for _, line := range v.Log {
		b.WriteString(truncate(indent+line, width) + "\n")
	}
}

func truncate(s string, width int) string {
	r := []rune(strings.TrimRight(s, " "))
	if len(r) <= width {
		return string(r)
	}
	return string(r[:width-1]) + "…"
}

func writeOthers(b *strings.Builder, others []OtherLine, t Theme, width int) {
	if len(others) == 0 {
		return
	}
	nameW := 0
	for _, o := range others {
		nameW = max(nameW, utf8.RuneCountInString(o.Name))
	}
	b.WriteString("\n" + t.Title(LabelOthers) + "\n")
	for _, o := range others {
		pad := strings.Repeat(" ", nameW-utf8.RuneCountInString(o.Name))
		line := fmt.Sprintf("%s%s%s   code %-3d   %-12s %s", indent, o.Name, pad, o.Code, o.Age, o.Summary)
		b.WriteString(truncate(line, width) + "\n")
	}
	b.WriteString(t.Hint("→ Pour en expliquer un : docker telemach ps <nom>") + "\n")
}

// RenderNoExited indique qu'aucun conteneur n'est arrêté en erreur.
func RenderNoExited(w io.Writer, o Options) error {
	t := o.Theme
	width := max(o.Width, 40)
	text := t.Success(SymbolSuccess+" Aucun conteneur arrêté en erreur.") + "\n" +
		wrap("Un conteneur lancé avec --rm est supprimé à l'arrêt : il ne peut plus être analysé.", width, indent) + "\n"
	_, err := io.WriteString(w, text)
	return err
}

// SignalLine est la ligne affichée à l'invite quand un conteneur lancé en arrière-plan s'est
// arrêté en erreur (FR-013) ; after est le délai depuis son lancement (« 3 s »).
func SignalLine(name string, code int, after string, removed bool, t Theme) string {
	s := fmt.Sprintf("%s Le conteneur « %s » s'est arrêté avec le code %d, %s après son lancement", SymbolWarning, name, code, after)
	if removed {
		s += ", et a été supprimé (--rm) : relance-le sans --rm pour lire son journal."
	} else {
		s += " : docker telemach ps pour comprendre pourquoi."
	}
	return t.Title(s)
}
