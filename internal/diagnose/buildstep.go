package diagnose

import (
	"regexp"
	"strings"
)

// failedStep repère, dans la sortie de BuildKit, l'en-tête de l'étape en échec :
//
//	> [3/3] RUN commande-inexistante --version:
var failedStep = regexp.MustCompile(`(?m)^ *> (\[[^\]]+\] [^\n]*?):?\s*$`)

func init() {
	enrichers = append(enrichers, isolateBuildStep)
}

// isolateBuildStep isole l'étape en échec d'une sortie « docker build » sur plusieurs
// étapes et les lignes d'erreur qui la suivent (captures « step » et « step_error »).
func isolateBuildStep(in Input, d *Diagnosis) {
	if in.Command.Subcommand != "build" || in.Stderr == "" {
		return
	}
	m := failedStep.FindStringSubmatchIndex(in.Stderr)
	if m == nil {
		return
	}
	d.Captures["step"] = strings.TrimSpace(in.Stderr[m[2]:m[3]])
	var errLines []string
	for _, line := range strings.Split(in.Stderr[m[1]:], "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "---") {
			if len(errLines) > 0 {
				break
			}
			continue
		}
		errLines = append(errLines, line)
		if len(errLines) == 3 {
			break
		}
	}
	d.Captures["step_error"] = strings.Join(errLines, "\n")
}
