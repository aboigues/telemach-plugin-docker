package lint

import (
	"regexp"
	"strings"

	"github.com/aboigues/telemach-plugin-docker/internal/redact"
	"github.com/aboigues/telemach-plugin-docker/internal/sanitize"
)

// FixedCommand reconstruit la commande corrigée à partir des arguments d'origine (sans le
// mot « docker ») : les arguments d'indice move sont déplacés, dans leur ordre d'origine,
// juste avant la cible (image ou conteneur, indice target) ; replace remplace un argument.
// Le résultat est nettoyé, masqué et cité pour le shell ; masked indique qu'un secret a été
// masqué (la commande n'est alors pas exécutable telle quelle).
func FixedCommand(args []string, target int, move []int, replace map[int]string) (cmd string, masked bool) {
	moved := make(map[int]bool, len(move))
	for _, i := range move {
		moved[i] = true
	}
	value := func(i int) string {
		if v, ok := replace[i]; ok {
			return v
		}
		return args[i]
	}
	out := make([]string, 0, len(args))
	for i := range args {
		if i == target {
			for _, j := range move {
				out = append(out, value(j))
			}
		}
		if !moved[i] {
			out = append(out, value(i))
		}
	}
	return Display(out)
}

// Display prépare des arguments pour l'affichage : nettoyage, masquage des secrets et
// citation shell ; masked indique qu'au moins une valeur a été masquée.
func Display(args []string) (cmd string, masked bool) {
	clean := make([]string, len(args))
	for i, a := range args {
		clean[i] = sanitize.Clean(a)
	}
	red := redact.RedactArgs(clean)
	parts := make([]string, 0, len(red)+1)
	parts = append(parts, "docker")
	for i, a := range red {
		if a != clean[i] {
			masked = true
		}
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " "), masked
}

// safeWord reconnaît un argument qui n'a pas besoin d'être cité (le marqueur de masquage
// « • » compris).
var safeWord = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./•-]+$`)

// shellQuote cite un argument pour bash et zsh : tel quel s'il est sûr, sinon entre
// guillemets simples (une apostrophe devient '\”).
func shellQuote(s string) string {
	if safeWord.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
