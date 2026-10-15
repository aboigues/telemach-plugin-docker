package lint

import "strings"

// Pièges silencieux (FR-004). Ils ne portent que sur les options réellement lues par Docker,
// c'est-à-dire placées avant l'image : une option placée après relève de misplacedOptions.

func init() {
	rules = append(rules, imageWithoutTag)
}

// imageWithoutTag : image sans tag ni digest, ou avec le tag latest (information, 24 h par
// image).
func imageWithoutTag(l *line) (Finding, bool) {
	if l.layout.Subcommand == "exec" || l.layout.Target < 0 {
		return Finding{}, false
	}
	ref := l.args[l.layout.Target]
	if strings.Contains(ref, "@") {
		return Finding{}, false
	}
	name := ref
	last := ref[strings.LastIndex(ref, "/")+1:]
	if i := strings.LastIndex(last, ":"); i >= 0 {
		tag := last[i+1:]
		if tag != "latest" {
			return Finding{}, false
		}
		name = ref[:len(ref)-len(last)+i]
	}
	display := sanitizeDisplay(name)
	return Finding{
		EntryID: "cmdline.image-without-tag", Severity: SeverityInfo,
		Captures:    map[string]string{"image": display},
		Fingerprint: "cmdline.image-without-tag:" + display, pos: l.layout.Target,
	}, true
}
