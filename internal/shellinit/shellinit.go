package shellinit

import (
	"embed"
	"fmt"
)

//go:embed scripts/bash.sh scripts/zsh.sh
var scripts embed.FS

// Shells pris en charge.
var supported = []string{"bash", "zsh"}

// Script renvoie le code d'intégration à évaluer dans le shell donné.
func Script(shell string) (string, error) {
	for _, s := range supported {
		if s == shell {
			data, err := scripts.ReadFile("scripts/" + shell + ".sh")
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
	}
	return "", fmt.Errorf("shell non pris en charge : %q (shells disponibles : bash, zsh)", shell)
}
