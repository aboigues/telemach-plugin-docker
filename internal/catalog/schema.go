package catalog

import "regexp"

// Category regroupe les erreurs par domaine Docker.
type Category string

// Catégories autorisées.
var Categories = map[Category]bool{
	"network": true, "image": true, "container": true, "volume": true, "daemon": true,
	"permission": true, "build": true, "compose": true, "runtime": true,
}

// Entry est une entrée du catalogue d'erreurs. Les textes sont en français et peuvent
// utiliser les valeurs capturées par les motifs, par exemple {{.port}}.
type Entry struct {
	ID          string           `yaml:"id"`
	Category    Category         `yaml:"category"`
	Applies     []string         `yaml:"applies"`
	ExitCodes   []int            `yaml:"exit_codes"`
	Fallback    string           `yaml:"fallback"` // « application » : entrée de repli pour cette origine
	Patterns    []string         `yaml:"patterns"`
	Concept     ConceptRef       `yaml:"concept"`
	Summary     string           `yaml:"summary"`
	Cause       string           `yaml:"cause"`
	Fixes       []Fix            `yaml:"fixes"`
	AntiPattern *AntiPatternNote `yaml:"anti_pattern"`
	Deepen      Deepening        `yaml:"deepen"`
	DocURL      string           `yaml:"doc_url"`
	Tests       []string         `yaml:"tests"`

	compiled []*regexp.Regexp
}

// Fix est une action corrective. Command est un gabarit facultatif ; Safe vaut false pour
// une solution rapide mais dangereuse, alors présentée comme anti-pattern.
type Fix struct {
	Description string `yaml:"description"`
	Command     string `yaml:"command"`
	Safe        bool   `yaml:"safe"`
}

// ConceptRef est le concept Docker enseigné : terme d'origine et définition en français.
type ConceptRef struct {
	Name       string `yaml:"name"`
	Definition string `yaml:"definition"`
}

// AntiPatternNote met en garde contre une solution rapide mais dangereuse.
type AntiPatternNote struct {
	Description string `yaml:"description"`
}

// Deepening est le contenu d'approfondissement (docker telemach deepen).
type Deepening struct {
	Explanation string `yaml:"explanation"`
	Example     string `yaml:"example"`
	Exercise    string `yaml:"exercise"`
}

// Compiled renvoie les motifs compilés de l'entrée.
func (e *Entry) Compiled() []*regexp.Regexp { return e.compiled }
