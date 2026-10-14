package compose

import (
	"sort"

	"github.com/aboigues/telemach-plugin-docker/internal/render"
)

// Kind est la nature d'un problème détecté dans un fichier Compose.
type Kind string

const (
	KindYAMLSyntax         Kind = "yaml-syntax"
	KindTabIndent          Kind = "tab-indent"
	KindUnknownKey         Kind = "unknown-key"
	KindWrongType          Kind = "wrong-type"
	KindUndefinedReference Kind = "undefined-reference"
	KindDependencyCycle    Kind = "dependency-cycle"
	KindUnresolvedVariable Kind = "unresolved-variable"
	KindPlaintextSecret    Kind = "plaintext-secret"
	KindFileNotFound       Kind = "file-not-found"
	KindFileUnreadable     Kind = "file-unreadable"
)

// Issue est un problème localisé dans un fichier Compose (data-model.md, ComposeIssue).
type Issue struct {
	Kind       Kind
	File       string // chemin tel qu'indiqué ou trouvé
	Line       int    // à partir de 1 ; 0 si inconnue
	Column     int    // à partir de 1 ; 0 si inconnue
	Service    string
	KeyPath    string // ex. services.web.port
	Suggestion string // clé, valeur ou variable proposée
	Detail     string // précision en français (message traduit, référence manquante…)
	Excerpt    []render.Line
	Fixed      []render.Line
	Blocking   bool
}

// Sort trie les problèmes : bloquants d'abord, puis par fichier et par ligne.
func Sort(issues []Issue) {
	sort.SliceStable(issues, func(i, j int) bool {
		a, b := issues[i], issues[j]
		if a.Blocking != b.Blocking {
			return a.Blocking
		}
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Column < b.Column
	})
}
