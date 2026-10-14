package diagnose

import (
	_ "embed"
	"strconv"
	"strings"

	"github.com/aboigues/telemach-plugin-docker/internal/textdist"
)

//go:embed data/official-images.txt
var officialImagesFile string

// officialImages est la liste embarquée des images officielles courantes. Aucune
// commande docker n'est exécutée pour obtenir des suggestions (FR-015).
var officialImages = func() []string {
	var out []string
	for _, line := range strings.Split(officialImagesFile, "\n") {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
			out = append(out, line)
		}
	}
	return out
}()

func init() {
	enrichers = append(enrichers, suggestImage, alternativePort)
}

// suggestImage propose l'image officielle la plus proche d'un nom mal orthographié.
func suggestImage(_ Input, d *Diagnosis) {
	repo := d.Captures["repo"]
	if repo == "" || strings.Contains(repo, "/") {
		return
	}
	name, _, _ := strings.Cut(repo, ":")
	if s, ok := textdist.Closest(name, officialImages, 2); ok {
		d.Captures["suggestion"] = s
	}
}

// alternativePort propose le port suivant quand un port est déjà utilisé, et reprend le
// port du conteneur indiqué par l'apprenant dans « -p MACHINE:CONTENEUR ».
func alternativePort(in Input, d *Diagnosis) {
	if p, err := strconv.Atoi(d.Captures["port"]); err == nil && p > 0 && p < 65535 {
		d.Captures["alt_port"] = strconv.Itoa(p + 1)
	}
	for _, name := range []string{"p", "publish"} {
		spec := in.Command.Flags[name]
		if i := strings.LastIndex(spec, ":"); i >= 0 && i+1 < len(spec) {
			d.Captures["cport"] = spec[i+1:]
			return
		}
	}
}
