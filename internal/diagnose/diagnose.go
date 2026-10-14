package diagnose

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"slices"
	"strings"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
	"github.com/aboigues/telemach-plugin-docker/internal/command"
	"github.com/aboigues/telemach-plugin-docker/internal/redact"
)

// Confidence indique dans quelle mesure l'erreur est reconnue.
type Confidence string

const (
	Recognized Confidence = "recognized" // un motif du catalogue correspond au message
	Partial    Confidence = "partial"    // reconnue par le seul code de sortie
	Unknown    Confidence = "unknown"    // non reconnue : aucune cause n'est avancée
)

// Origin indique si l'erreur vient de Docker ou de l'application du conteneur.
type Origin string

const (
	OriginDocker      Origin = "docker"
	OriginApplication Origin = "application"
	OriginUnknown     Origin = "unknown"
)

// Diagnosis est le résultat de l'analyse d'une erreur capturée.
type Diagnosis struct {
	Confidence  Confidence
	Origin      Origin
	Entry       *catalog.Entry
	Captures    map[string]string // valeurs extraites, déjà masquées
	Fingerprint string
}

// Input regroupe ce qui est connu de la commande en échec.
type Input struct {
	Command     command.Command
	ExitCode    int
	Stderr      string // déjà nettoyé et masqué
	Interactive bool   // stderr non capturé
}

// Diagnose cherche dans le catalogue l'entrée correspondant à l'erreur. Les entrées à
// motifs sont essayées d'abord, dans l'ordre du catalogue, puis les entrées fondées sur le
// seul code de sortie. Sans correspondance, le diagnostic est « unknown » (FR-008).
func Diagnose(in Input, cat *catalog.Catalog) Diagnosis {
	d := Diagnosis{
		Confidence:  Unknown,
		Origin:      OriginUnknown,
		Captures:    baseCaptures(in.Command),
		Fingerprint: Fingerprint(in.Command, in.Stderr),
	}
	if cat != nil {
		if e, captures, ok := matchPatterns(in, cat); ok {
			d.Entry, d.Confidence, d.Origin = e, Recognized, OriginDocker
			for k, v := range captures {
				d.Captures[k] = v
			}
		} else if e, ok := matchExitCode(in, cat); ok {
			d.Entry, d.Confidence = e, Partial
		}
	}
	d.Origin = originFor(in, d)
	if d.Entry == nil && cat != nil && d.Origin == OriginApplication {
		if e, ok := matchFallback(d.Origin, in, cat); ok {
			d.Entry, d.Confidence = e, Partial
		}
	}
	enrich(in, &d)
	return d
}

func baseCaptures(cmd command.Command) map[string]string {
	c := map[string]string{"subcommand": cmd.Subcommand}
	if img := cmd.Image(); img != "" {
		c["image"] = img
	}
	return c
}

func matchPatterns(in Input, cat *catalog.Catalog) (*catalog.Entry, map[string]string, bool) {
	if in.Stderr == "" {
		return nil, nil, false
	}
	for _, e := range cat.Entries {
		if len(e.Compiled()) == 0 || !applies(e, in) {
			continue
		}
		for _, re := range e.Compiled() {
			m := re.FindStringSubmatch(in.Stderr)
			if m == nil {
				continue
			}
			captures := map[string]string{}
			for i, name := range re.SubexpNames() {
				if name != "" && i < len(m) && m[i] != "" {
					captures[name] = redact.Redact(m[i])
				}
			}
			return e, captures, true
		}
	}
	return nil, nil, false
}

// matchExitCode cherche une entrée qui déclare explicitement ce code de sortie.
func matchExitCode(in Input, cat *catalog.Catalog) (*catalog.Entry, bool) {
	for _, e := range cat.Entries {
		if len(e.ExitCodes) > 0 && applies(e, in) {
			return e, true
		}
	}
	return nil, false
}

// matchFallback renvoie l'entrée de repli associée à une origine (par exemple une erreur
// du programme du conteneur dont le code de sortie n'a pas d'entrée dédiée).
func matchFallback(origin Origin, in Input, cat *catalog.Catalog) (*catalog.Entry, bool) {
	for _, e := range cat.Entries {
		if e.Fallback == string(origin) && applies(e, in) {
			return e, true
		}
	}
	return nil, false
}

// applies vérifie la sous-commande et le code de sortie attendus par l'entrée.
func applies(e *catalog.Entry, in Input) bool {
	if len(e.ExitCodes) > 0 && !slices.Contains(e.ExitCodes, in.ExitCode) {
		return false
	}
	if len(e.Applies) == 0 {
		return true
	}
	sub := in.Command.Subcommand
	for _, a := range e.Applies {
		switch {
		case in.Command.Family == command.FamilyCompose && (a == "compose" || a == "compose "+sub):
			return true
		case in.Command.Family == command.FamilyDocker && a == sub:
			return true
		}
	}
	return false
}

var (
	hexID      = regexp.MustCompile(`\b[0-9a-f]{12,64}\b`)
	spaces     = regexp.MustCompile(`\s+`)
	timestamps = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z?`)
)

// Fingerprint calcule l'empreinte SHA-256 de la commande et de l'erreur normalisées
// (identifiants, horodatages et espaces neutralisés), à partir de données déjà masquées.
func Fingerprint(cmd command.Command, stderr string) string {
	norm := hexID.ReplaceAllString(stderr, "<id>")
	norm = timestamps.ReplaceAllString(norm, "<date>")
	norm = strings.TrimSpace(spaces.ReplaceAllString(norm, " "))
	h := sha256.New()
	h.Write([]byte(string(cmd.Family) + "\x00" + cmd.Subcommand + "\x00" + strings.Join(cmd.Positionals, " ") + "\x00" + norm))
	return hex.EncodeToString(h.Sum(nil))
}
