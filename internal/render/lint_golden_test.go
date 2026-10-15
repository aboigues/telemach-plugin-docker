package render

import (
	"strconv"
	"strings"
	"testing"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
)

// Entrées factices : le rendu est testé indépendamment du contenu du vrai catalogue.
var (
	fakeWarning = &catalog.Entry{
		ID: "cmdline.option-after-image", Fallback: "cmdline", Severity: "warning",
		Concept: catalog.ConceptRef{
			Name:       "ordre des arguments de docker run",
			Definition: "docker run [OPTIONS] IMAGE [COMMANDE] [ARG...] : les options de Docker se placent avant l'image ; tout ce qui suit l'image est donné au programme du conteneur.",
		},
		Summary: "Option Docker placée après l'image : {{.options}} a été transmis au conteneur, pas à Docker.",
		Cause:   "Docker arrête de lire ses options au nom de l'image.",
		Fixes:   []catalog.Fix{{Description: "Placer les options avant l'image.", Command: "{{.fixed_command}}", Safe: true}},
	}
	fakeInfo = &catalog.Entry{
		ID: "cmdline.image-without-tag", Fallback: "cmdline", Severity: "info",
		Concept: catalog.ConceptRef{Name: "image tag", Definition: "Le tag désigne une version précise."},
		Summary: "Image sans tag explicite : {{.image}} désigne {{.image}}:latest, qui change à chaque publication.",
		Cause:   "Sans tag, Docker utilise latest.",
		Fixes:   []catalog.Fix{{Description: "Préfère un tag précis, par exemple nginx:1.30-alpine.", Safe: true}},
	}
	issueCaptures = map[string]string{
		"options":       "-p 8090:80",
		"effect":        "Le conteneur a démarré, mais le port 8090 de ta machine n'est pas publié.",
		"fixed_command": "docker run -d -p 8090:80 nginx",
		"image":         "nginx",
	}
)

func TestLintGolden(t *testing.T) {
	warn := LintItem{Entry: fakeWarning, Captures: issueCaptures}
	info := LintItem{Entry: fakeInfo, Captures: issueCaptures}
	masked := LintItem{Entry: fakeWarning, Captures: map[string]string{
		"options": "-e API_TOKEN=••••••", "fixed_command": "docker run -e API_TOKEN=•••••• nginx", "masked": "1",
	}}
	cases := []struct {
		name  string
		items []LintItem
	}{
		{"warning", []LintItem{warn}},
		{"info", []LintItem{info}},
		{"reminder", []LintItem{{Entry: fakeWarning, Captures: issueCaptures, Reminder: true}}},
		{"several", []LintItem{warn, info}},
		{"masked", []LintItem{masked}},
	}
	for _, c := range cases {
		for _, width := range []int{60, 100} {
			for _, color := range []bool{false, true} {
				name := c.name + "-" + strconv.Itoa(width)
				if color {
					name += "-color"
				}
				var b strings.Builder
				o := Options{Theme: NewTheme(color, true), Width: width}
				if err := RenderFindings(&b, c.items, o); err != nil {
					t.Fatal(err)
				}
				golden(t, "lint", name, b.String())
			}
		}
	}
}

func TestLintInlineGolden(t *testing.T) {
	var b strings.Builder
	o := Options{Theme: NewTheme(false, true), Width: 80}
	if err := RenderInline(&b, []LintItem{{Entry: fakeWarning, Captures: issueCaptures}}, o); err != nil {
		t.Fatal(err)
	}
	golden(t, "lint", "inline", b.String())
	if !strings.Contains(b.String(), LabelInCommand) {
		t.Errorf("section « Dans ta commande » attendue :\n%s", b.String())
	}
}

// La couleur n'est jamais le seul porteur d'information : les symboles restent présents.
func TestLintSymbolsWithoutColor(t *testing.T) {
	var b strings.Builder
	o := Options{Theme: NewTheme(false, true), Width: 80}
	_ = RenderFindings(&b, []LintItem{{Entry: fakeWarning, Captures: issueCaptures}, {Entry: fakeInfo, Captures: issueCaptures}}, o)
	for _, sym := range []string{SymbolWarning, SymbolInfo, LabelConcept, LabelFix} {
		if !strings.Contains(b.String(), sym) {
			t.Errorf("symbole %q absent :\n%s", sym, b.String())
		}
	}
}
