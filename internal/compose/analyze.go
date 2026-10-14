package compose

import (
	"errors"
	"os"
	"path/filepath"
)

// Analyze vérifie, en lecture seule, les fichiers Compose d'un dossier : ceux passés par
// -f, sinon les fichiers par défaut. Les passes s'enchaînent ; une erreur bloquante
// (fichier illisible, syntaxe YAML) arrête l'analyse, car la suite n'aurait pas de sens.
func Analyze(dir string, files []string) []Issue {
	dir = absDir(dir)
	paths, err := Locate(dir, files)
	if errors.Is(err, ErrNotFound) {
		return []Issue{{Kind: KindFileNotFound, File: dir, Blocking: true}}
	}

	var parsed []*parsedFile
	var issues []Issue
	byPath := map[string]*parsedFile{}
	for _, p := range paths {
		pf, found := parseFile(p)
		issues = append(issues, found...)
		if pf != nil {
			parsed = append(parsed, pf)
			byPath[p] = pf
		}
	}
	if hasBlocking(issues) {
		return finish(issues, byPath)
	}

	for _, pf := range parsed {
		issues = append(issues, checkKeys(pf)...)
	}
	keyErrors := hasBlocking(issues)
	for _, pf := range parsed {
		issues = append(issues, checkVariables(dir, pf, os.LookupEnv)...)
		issues = append(issues, checkSecrets(pf)...)
	}
	if !keyErrors {
		issues = append(issues, checkSemantic(dir, parsed)...)
	}
	return finish(issues, byPath)
}

func finish(issues []Issue, byPath map[string]*parsedFile) []Issue {
	for i := range issues {
		attachExcerpt(&issues[i], byPath[issues[i].File])
	}
	Sort(issues)
	return issues
}

func hasBlocking(issues []Issue) bool {
	for _, is := range issues {
		if is.Blocking {
			return true
		}
	}
	return false
}

func absDir(dir string) string {
	if abs, err := filepath.Abs(dir); err == nil {
		return abs
	}
	return dir
}
