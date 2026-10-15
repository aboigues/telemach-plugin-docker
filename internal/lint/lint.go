package lint

import (
	"sort"
	"strings"

	"github.com/aboigues/telemach-plugin-docker/internal/command"
	"github.com/aboigues/telemach-plugin-docker/internal/sanitize"
)

// Niveaux d'un piège (repris de l'entrée de catalogue correspondante).
const (
	SeverityWarning = "warning"
	SeverityInfo    = "info"
)

// Finding est un piège reconnu dans une ligne de commande. Les captures sont nettoyées et
// masquées ; l'empreinte ne contient jamais de valeur saisie par l'apprenant.
type Finding struct {
	EntryID     string
	Severity    string
	Captures    map[string]string
	Fingerprint string

	pos int // position dans la ligne, pour l'ordre d'affichage
}

// Env fournit les deux seules lectures locales autorisées : présence d'une variable
// d'environnement (jamais sa valeur) et existence d'un dossier dans le dossier courant.
type Env struct {
	HasVar    func(name string) bool
	DirExists func(name string) bool
}

// line est la ligne analysée, découpée une fois pour toutes les règles.
type line struct {
	args   []string // arguments nettoyés (non masqués : les règles en ont besoin)
	layout command.Layout
	opts   []option // options entre la sous-commande et la cible
	after  []int    // indices des arguments qui suivent la cible (avant un éventuel --)
	env    Env
}

// option est une option de la ligne, avec sa valeur éventuelle.
type option struct {
	name     string // sans tirets
	value    string
	hasValue bool
	start    int // indice du premier argument de l'option
	end      int // indice suivant le dernier argument (valeur séparée comprise)
}

// rule est une règle de détection : elle renvoie au plus un Finding par ligne.
type rule func(l *line) (Finding, bool)

// rules est le registre des règles, dans l'ordre de leur identifiant.
var rules []rule

// lintable sont les sous-commandes analysées (FR-001).
var lintable = map[string]bool{"run": true, "create": true, "exec": true}

// Analyze analyse les arguments d'une commande docker (sans le mot « docker ») et renvoie
// les pièges trouvés : avertissements d'abord, puis informations, chacun dans l'ordre
// d'apparition dans la ligne.
func Analyze(args []string, env Env) []Finding {
	clean := make([]string, len(args))
	for i, a := range args {
		clean[i] = sanitize.Clean(a)
	}
	l := &line{args: clean, layout: command.Locate(clean), env: env}
	if !lintable[l.layout.Subcommand] {
		return nil
	}
	end := l.layout.Target
	if end < 0 {
		end = len(clean)
	}
	l.opts = parseOptions(clean, l.layout.OptsStart, end, l.layout.Subcommand)
	if l.layout.Target >= 0 {
		for i := l.layout.Target + 1; i < len(clean) && clean[i] != "--"; i++ {
			l.after = append(l.after, i)
		}
	}
	var out []Finding
	for _, r := range rules {
		if f, ok := r(l); ok {
			out = append(out, f)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return out[i].Severity == SeverityWarning
		}
		return out[i].pos < out[j].pos
	})
	return out
}

// parseOptions découpe les options de args[from:to] (options courtes groupées comprises).
func parseOptions(args []string, from, to int, sub string) []option {
	var opts []option
	for i := from; i < to; i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") || a == "-" || a == "--" {
			continue
		}
		if strings.HasPrefix(a, "--") {
			name, value, has := strings.Cut(a[2:], "=")
			o := option{name: name, value: value, hasValue: has, start: i, end: i + 1}
			if !has && command.TakesValue(sub, name) && i+1 < to {
				o.value, o.hasValue, o.end = args[i+1], true, i+2
				i++
			}
			opts = append(opts, o)
			continue
		}
		short := a[1:]
		for j, r := range short {
			name := string(r)
			o := option{name: name, start: i, end: i + 1}
			if command.TakesValue(sub, name) {
				if rest := short[j+1:]; rest != "" {
					o.value, o.hasValue = strings.TrimPrefix(rest, "="), true
					opts = append(opts, o)
					break
				}
				if i+1 < to {
					o.value, o.hasValue, o.end = args[i+1], true, i+2
					opts = append(opts, o)
					i++
					break
				}
			}
			opts = append(opts, o)
		}
	}
	return opts
}
