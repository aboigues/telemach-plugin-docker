package lint

import (
	"regexp"
	"sort"
	"strings"
)

// Formats de valeur propres à Docker : leur présence lève l'ambiguïté avec les options du
// programme du conteneur (research R4).
var (
	portMapping = regexp.MustCompile(`^((\d{1,3}\.){3}\d{1,3}:|\[[0-9a-fA-F:]+\]:)?(\d+)(-\d+)?:\d+(-\d+)?(/(tcp|udp|sctp))?$`)
	volumeSpec  = regexp.MustCompile(`^[^:\s]+:/([^/:][^:]*)?(:[a-zA-Z,]+)?$`)
	urlScheme   = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*://`)
	envAssign   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	restartMode = regexp.MustCompile(`^(no|always|unless-stopped|on-failure(:\d+)?)$`)
)

// strength classe une option trouvée après l'image ou le conteneur.
type strength int

const (
	notDocker strength = iota // option du programme du conteneur, ou doute : on s'abstient
	weak                      // signalée seulement si un signal fort est présent dans la ligne
	strong                    // forme propre à Docker : suffit seule
)

func init() {
	rules = append(rules, misplacedOptions)
}

// classify évalue une option placée après la cible ; raw est l'argument tel qu'écrit.
func classify(sub string, o option, raw string) strength {
	if raw == "-it" || raw == "-ti" {
		return strong
	}
	if sub == "exec" {
		switch o.name {
		case "e", "env":
			if o.hasValue && envAssign.MatchString(o.value) {
				return strong
			}
		case "env-file", "detach", "interactive", "tty", "privileged":
			return strong
		case "d", "i", "t", "u", "user", "w", "workdir":
			return weak
		}
		return notDocker
	}
	switch o.name {
	case "p", "publish":
		if o.hasValue && portMapping.MatchString(o.value) {
			return strong
		}
	case "v", "volume":
		if o.hasValue && volumeSpec.MatchString(o.value) && !urlScheme.MatchString(o.value) {
			return strong
		}
	case "e", "env":
		if o.hasValue && envAssign.MatchString(o.value) {
			return strong
		}
	case "restart":
		if o.hasValue && restartMode.MatchString(o.value) {
			return strong
		}
	case "mount":
		if o.hasValue && strings.HasPrefix(o.value, "type=") {
			return strong
		}
	case "env-file", "network", "net":
		if o.hasValue {
			return strong
		}
	case "rm", "detach", "interactive", "tty", "privileged":
		return strong
	case "d", "i", "t", "name", "u", "user", "w", "workdir", "m", "memory":
		return weak
	}
	return notDocker
}

// misplacedOptions repère les options Docker placées après l'image (run, create) ou le
// conteneur (exec) : Docker les a transmises au programme du conteneur (issue #9).
func misplacedOptions(l *line) (Finding, bool) {
	target := l.layout.Target
	if target < 0 || len(l.after) == 0 {
		return Finding{}, false
	}
	sub := l.layout.Subcommand
	from, to := l.after[0], l.after[len(l.after)-1]+1
	hasStrong := false
	picked := map[int]bool{}
	var names []string
	ports := []string{}
	for _, o := range parseOptions(l.args, from, to, sub) {
		s := classify(sub, o, l.args[o.start])
		if s == notDocker {
			continue
		}
		if s == strong {
			hasStrong = true
		}
		for i := o.start; i < o.end; i++ {
			picked[i] = true
		}
		names = append(names, o.name)
		if (o.name == "p" || o.name == "publish") && o.hasValue {
			if m := portMapping.FindStringSubmatch(o.value); m != nil {
				ports = append(ports, m[3])
			}
		}
	}
	if !hasStrong {
		return Finding{}, false
	}
	move := make([]int, 0, len(picked))
	for i := range picked {
		move = append(move, i)
	}
	sort.Ints(move)
	movedArgs := make([]string, len(move))
	for k, i := range move {
		movedArgs[k] = l.args[i]
	}
	options, _ := Display(movedArgs)
	options = strings.TrimPrefix(options, "docker ")
	fixed, masked := FixedCommand(l.args, target, move, nil)

	id, targetKey := "cmdline.option-after-image", "image"
	effect := "Docker ne les a pas appliquées : elles ont été données au programme du conteneur."
	effectCond := "Docker ne les appliquerait pas : elles seraient données au programme du conteneur."
	switch {
	case sub == "exec":
		id, targetKey = "cmdline.option-after-container", "container"
		effect = "Docker ne les a pas appliquées : elles ont été données à la commande lancée dans le conteneur."
		effectCond = "Docker ne les appliquerait pas : elles seraient données à la commande lancée dans le conteneur."
	case sub == "create":
		effect = "Le conteneur a été créé, mais Docker n'a pas appliqué ces options : elles seront données au programme du conteneur."
	case len(ports) > 0:
		effect = "Le conteneur a pu démarrer, mais le port " + strings.Join(ports, ", ") + " de ta machine n'est pas publié."
		effectCond = "Le conteneur démarrerait, mais le port " + strings.Join(ports, ", ") + " de ta machine ne serait pas publié."
	}
	sort.Strings(names)
	c := map[string]string{
		"options":       options,
		targetKey:       sanitizeDisplay(l.args[target]),
		"fixed_command": fixed,
		"effect":        effect,
		"effect_cond":   effectCond,
	}
	if masked {
		c["masked"] = "1"
	}
	return Finding{
		EntryID: id, Severity: SeverityWarning, Captures: c,
		Fingerprint: id + ":" + strings.Join(names, ","), pos: move[0],
	}, true
}

// sanitizeDisplay prépare un seul argument pour l'affichage (nettoyé, masqué).
func sanitizeDisplay(arg string) string {
	s, _ := Display([]string{arg})
	return strings.TrimPrefix(s, "docker ")
}
