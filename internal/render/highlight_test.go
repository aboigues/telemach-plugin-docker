package render

import (
	"strings"
	"testing"
)

func TestHighlightWithoutColor(t *testing.T) {
	th := NewTheme(false, true)
	code := "docker run -p 8081:80 nginx"
	if got := th.Highlight(code, "bash"); got != code {
		t.Errorf("sans couleur, le code doit être inchangé : %q", got)
	}
}

func TestHighlightWithColor(t *testing.T) {
	th := NewTheme(true, true)
	got := th.Highlight("services:\n  web:\n    image: nginx", "yaml")
	if !strings.Contains(got, "\x1b[") {
		t.Errorf("une séquence de couleur était attendue : %q", got)
	}
	if strings.Count(got, "\n") != 2 {
		t.Errorf("le nombre de lignes doit être conservé : %q", got)
	}
}

func TestHighlightLinesPlain(t *testing.T) {
	th := NewTheme(false, true)
	lines := []Line{{8, "services:"}, {9, "  web:"}, {10, "\tport: 80"}}
	got := th.HighlightLines(lines, "yaml", 10, 1)
	want := "   8 │ services:\n   9 │   web:\n➤ 10 │ →port: 80\n       ^"
	if got != want {
		t.Errorf("rendu inattendu :\n%s\n--- attendu ---\n%s", got, want)
	}
	if strings.Contains(got, "\x1b") {
		t.Error("aucune séquence ESC attendue sans couleur")
	}
}

func TestThemeWithoutColorKeepsText(t *testing.T) {
	th := NewTheme(false, false)
	for _, s := range []string{th.Title("a"), th.Concept("a"), th.Dim("a"), th.Highlight("a", "bash")} {
		if s != "a" {
			t.Errorf("sans couleur, le texte doit être inchangé : %q", s)
		}
	}
}
