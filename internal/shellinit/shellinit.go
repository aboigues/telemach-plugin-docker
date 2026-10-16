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
// du binaire docker-telemach à appeler ; vide, le binaire est cherché dans le PATH. version
// est celle du plugin, exportée avec le shell dans TELEMACH_SHELL (« zsh:0.3.0 ») ; pendingDir
// est le dossier des signaux affichés à l'invite (feature 004).
func Script(shell, binPath, version, pendingDir string) (string, error) {
	for _, s := range supported {
		if s == shell {
			data, err := scripts.ReadFile("scripts/" + shell + ".sh")
			if err != nil {
				return "", err
			}
			if binPath == "" {
				binPath = "docker-telemach"
			}
			s := strings.Replace(string(data), "__TELEMACH_BIN__", shellQuote(binPath), 1)
			s = strings.Replace(s, "__TELEMACH_SHELL__", shellQuote(ShellValue(shell, version)), 1)
			return strings.Replace(s, "__TELEMACH_PENDING__", shellQuote(pendingDir), 1), nil
		}
	}
	return "", fmt.Errorf("shell non pris en charge : %q (shells disponibles : bash, zsh)", shell)
}

// shellQuote protège une chaîne pour bash et zsh : entre apostrophes, chaque apostrophe
// étant remplacée par la séquence apostrophe, barre oblique inverse, apostrophe, apostrophe.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// EnvVar est la variable exportée par l'intégration shell.
const EnvVar = "TELEMACH_SHELL"

// ShellValue est la valeur de EnvVar pour un shell et une version (« zsh:0.3.0 »).
func ShellValue(shell, version string) string {
	return shell + ":" + version
}
