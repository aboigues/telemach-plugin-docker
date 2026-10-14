package shellinit

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed scripts/bash.sh scripts/zsh.sh
var scripts embed.FS

// Shells pris en charge.
var supported = []string{"bash", "zsh"}

// Script renvoie le code d'intégration à évaluer dans le shell donné. binPath est le chemin
// du binaire docker-telemach à appeler ; vide, le binaire est cherché dans le PATH.
func Script(shell, binPath string) (string, error) {
	for _, s := range supported {
		if s == shell {
			data, err := scripts.ReadFile("scripts/" + shell + ".sh")
			if err != nil {
				return "", err
			}
			if binPath == "" {
				binPath = "docker-telemach"
			}
			return strings.Replace(string(data), "__TELEMACH_BIN__", shellQuote(binPath), 1), nil
		}
	}
	return "", fmt.Errorf("shell non pris en charge : %q (shells disponibles : bash, zsh)", shell)
}

// shellQuote protège une chaîne pour bash et zsh : entre apostrophes, chaque apostrophe
// étant remplacée par la séquence apostrophe, barre oblique inverse, apostrophe, apostrophe.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
