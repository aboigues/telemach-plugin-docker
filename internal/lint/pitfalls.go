package lint

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/aboigues/telemach-plugin-docker/internal/redact"
)

// Pièges silencieux (FR-004). Ils ne portent que sur les options réellement lues par Docker,
// c'est-à-dire placées avant l'image : une option placée après relève de misplacedOptions.

var (
	// namedSource reconnaît la source d'un volume nommé (ni chemin absolu ni relatif).
	namedSource = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	// containerPortOnly reconnaît une publication sans port de la machine (« 80 », « 53/udp »).
	containerPortOnly = regexp.MustCompile(`^\d+(/(tcp|udp|sctp))?$`)
	// varName reconnaît un nom de variable d'environnement seul (sans « = »).
	varName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// dockerSockets sont les chemins du socket du démon Docker sur la machine.
var dockerSockets = map[string]bool{"/var/run/docker.sock": true, "/run/docker.sock": true}

func init() {
	rules = append(rules, namedVolumeNotDir, randomHostPort, envUnset, imageWithoutTag,
		portsIgnoredHostNetwork, secretInCommand, privileged)
}

// pitfallLine indique si la ligne relève des pièges de FR-004 (run et create, image trouvée).
func pitfallLine(l *line) bool {
	return l.layout.Subcommand != "exec" && l.layout.Target >= 0
}

// is indique si l'option porte l'un des noms donnés.
func (o option) is(names ...string) bool {
	for _, n := range names {
		if o.name == n {
			return true
		}
	}
	return false
}

// replaceValue renvoie la commande corrigée où la valeur de o devient value (option écrite
// en un ou deux arguments, courte ou longue).
func (l *line) replaceValue(o option, value string) (string, bool) {
	i := o.end - 1
	raw := l.args[i]
	return FixedCommand(l.args, l.layout.Target, nil, map[int]string{i: raw[:len(raw)-len(o.value)] + value})
}

// optionText affiche l'option telle qu'écrite (nettoyée, masquée).
func (l *line) optionText(o option) string {
	s, _ := Display(l.args[o.start:o.end])
	return strings.TrimPrefix(s, "docker ")
}

// namedVolumeNotDir : « -v data:/d » alors qu'un dossier « data » existe dans le dossier
// courant ; Docker crée un volume nommé et le dossier n'est pas monté.
func namedVolumeNotDir(l *line) (Finding, bool) {
	if !pitfallLine(l) || l.env.DirExists == nil {
		return Finding{}, false
	}
	for _, o := range l.opts {
		if !o.is("v", "volume") || !o.hasValue {
			continue
		}
		source, target, ok := strings.Cut(o.value, ":")
		if !ok || !namedSource.MatchString(source) || !l.env.DirExists(source) {
			continue
		}
		target, _, _ = strings.Cut(target, ":")
		fixed, masked := l.replaceValue(o, "./"+o.value)
		return Finding{
			EntryID: "cmdline.named-volume-not-dir", Severity: SeverityWarning,
			Captures: withMasked(map[string]string{
				"name": sanitizeDisplay(source), "target": sanitizeDisplay(target), "fixed_command": fixed,
			}, masked),
			Fingerprint: "cmdline.named-volume-not-dir:" + o.name, pos: o.start,
		}, true
	}
	return Finding{}, false
}

// randomHostPort : « -p 80 » publie le port du conteneur sur un port de la machine choisi
// au hasard (information). Sans objet en réseau host (voir portsIgnoredHostNetwork).
func randomHostPort(l *line) (Finding, bool) {
	if !pitfallLine(l) || hostNetwork(l) {
		return Finding{}, false
	}
	for _, o := range l.opts {
		if !o.is("p", "publish") || !o.hasValue || !containerPortOnly.MatchString(o.value) {
			continue
		}
		fixed, masked := l.replaceValue(o, "8080:"+o.value)
		return Finding{
			EntryID: "cmdline.random-host-port", Severity: SeverityInfo,
			Captures:    withMasked(map[string]string{"cport": o.value, "fixed_command": fixed}, masked),
			Fingerprint: "cmdline.random-host-port:" + o.name, pos: o.start,
		}, true
	}
	return Finding{}, false
}

// envUnset : « -e NOM » sans valeur alors que NOM n'est pas définie dans le shell ; la
// variable sera absente du conteneur.
func envUnset(l *line) (Finding, bool) {
	if !pitfallLine(l) || l.env.HasVar == nil {
		return Finding{}, false
	}
	for _, o := range l.opts {
		if !o.is("e", "env") || !o.hasValue || !varName.MatchString(o.value) || l.env.HasVar(o.value) {
			continue
		}
		fixed, masked := l.replaceValue(o, o.value+"=<valeur>")
		return Finding{
			EntryID: "cmdline.env-unset", Severity: SeverityWarning,
			Captures:    withMasked(map[string]string{"name": o.value, "fixed_command": fixed}, masked),
			Fingerprint: "cmdline.env-unset:" + o.value, pos: o.start,
		}, true
	}
	return Finding{}, false
}

// imageWithoutTag : image sans tag ni digest, ou avec le tag latest (information, 24 h par
// image).
func imageWithoutTag(l *line) (Finding, bool) {
	if !pitfallLine(l) {
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

// hostNetwork indique un conteneur en réseau host (--network host, --net=host…).
func hostNetwork(l *line) bool {
	for _, o := range l.opts {
		if o.is("network", "net") && o.hasValue && o.value == "host" {
			return true
		}
	}
	return false
}

// portsIgnoredHostNetwork : publication de ports avec --network host ; Docker l'ignore, le
// conteneur utilise directement les ports de la machine.
func portsIgnoredHostNetwork(l *line) (Finding, bool) {
	if !pitfallLine(l) || !hostNetwork(l) {
		return Finding{}, false
	}
	var published []option
	var names []string
	for _, o := range l.opts {
		if o.is("p", "publish", "P", "publish-all") {
			published = append(published, o)
			names = append(names, o.name)
		}
	}
	if len(published) == 0 {
		return Finding{}, false
	}
	texts := make([]string, len(published))
	for i, o := range published {
		texts[i] = l.optionText(o)
	}
	fixed, masked := l.withoutOptions(published)
	return Finding{
		EntryID: "cmdline.ports-ignored-host-network", Severity: SeverityWarning,
		Captures:    withMasked(map[string]string{"options": strings.Join(texts, " "), "fixed_command": fixed}, masked),
		Fingerprint: "cmdline.ports-ignored-host-network:" + strings.Join(names, ","), pos: published[0].start,
	}, true
}

// withoutOptions renvoie la commande sans les options données. Une option courte groupée
// (« -dP », « -dp 80:80 ») est retirée du groupe, qui garde ses autres lettres.
func (l *line) withoutOptions(opts []option) (string, bool) {
	drop := map[int]bool{}
	replace := map[int]string{}
	for _, o := range opts {
		raw := l.args[o.start]
		if strings.HasPrefix(raw, "--") || len(raw) == 2 {
			for i := o.start; i < o.end; i++ {
				drop[i] = true
			}
			continue
		}
		group := raw
		if r, ok := replace[o.start]; ok {
			group = r
		}
		k := strings.Index(group[1:], o.name) + 1
		if o.hasValue {
			group = group[:k] // la lettre à valeur termine le groupe
			if o.end == o.start+2 {
				drop[o.start+1] = true
			}
		} else {
			group = group[:k] + group[k+1:]
		}
		if group == "-" {
			drop[o.start] = true
		} else {
			replace[o.start] = group
		}
	}
	args := make([]string, 0, len(l.args))
	for i, a := range l.args {
		if drop[i] {
			continue
		}
		if r, ok := replace[i]; ok {
			a = r
		}
		args = append(args, a)
	}
	return Display(args)
}

// secretInCommand : valeur de -e/--env reconnue comme secret par les règles de masquage ;
// elle reste visible dans l'historique du shell et dans docker inspect. La valeur n'est
// jamais capturée.
func secretInCommand(l *line) (Finding, bool) {
	if !pitfallLine(l) {
		return Finding{}, false
	}
	for _, o := range l.opts {
		if !o.is("e", "env") || !o.hasValue {
			continue
		}
		name, value, ok := strings.Cut(o.value, "=")
		if !ok || redact.Redact(o.value) == o.value {
			continue
		}
		// Le masquage reste large (mieux vaut masquer à tort qu'afficher un secret), mais
		// l'avertissement exige un nom sans ambiguïté ou une valeur reconnaissable par
		// elle-même (jeton connu, identifiants dans une URL) : MONKEY, AUTHOR ou
		// KEYCLOAK_URL ne sont pas des secrets.
		if !secretName(name) && redact.Redact(value) == value {
			continue
		}
		return Finding{
			EntryID: "cmdline.secret-in-command", Severity: SeverityWarning,
			Captures:    map[string]string{"name": sanitizeDisplay(name)},
			Fingerprint: "cmdline.secret-in-command:" + o.name, pos: o.start,
		}, true
	}
	return Finding{}, false
}

// Mots qui désignent un secret quand ils apparaissent entiers dans un nom de variable
// (DB_PASSWORD, apiKey, AWS_SECRET_ACCESS_KEY), et derniers mots qui en font une simple
// référence (SSH_KEY_PATH, TOKEN_URL, AWS_ACCESS_KEY_ID).
var (
	secretWords = map[string]bool{
		"PASSWORD": true, "PASSWD": true, "PASSPHRASE": true, "SECRET": true, "TOKEN": true,
		"KEY": true, "APIKEY": true, "CREDENTIAL": true, "CREDENTIALS": true, "AUTH": true,
	}
	referenceWords = map[string]bool{
		"FILE": true, "PATH": true, "DIR": true, "URL": true, "URI": true, "HOST": true,
		"PORT": true, "NAME": true, "ID": true, "TYPE": true, "SIZE": true, "LENGTH": true,
	}
	camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)
	nameSeparator = regexp.MustCompile(`[^A-Za-z0-9]+`)
)

// secretName indique si un nom de variable désigne sans ambiguïté un secret.
func secretName(name string) bool {
	words := nameSeparator.Split(strings.ToUpper(camelBoundary.ReplaceAllString(name, "${1}_${2}")), -1)
	if len(words) == 0 || referenceWords[words[len(words)-1]] {
		return false
	}
	for _, w := range words {
		if secretWords[w] {
			return true
		}
	}
	return false
}

// privileged : --privileged, ou montage du socket Docker ; le conteneur obtient un contrôle
// quasi total de la machine (anti-pattern pour un usage courant).
func privileged(l *line) (Finding, bool) {
	if !pitfallLine(l) {
		return Finding{}, false
	}
	for _, o := range l.opts {
		hit := false
		switch {
		case o.is("privileged"):
			on, err := strconv.ParseBool(o.value)
			hit = !o.hasValue || (err == nil && on)
		case o.is("v", "volume") && o.hasValue:
			source, _, _ := strings.Cut(o.value, ":")
			hit = dockerSockets[source]
		case o.is("mount") && o.hasValue:
			for _, field := range strings.Split(o.value, ",") {
				k, v, _ := strings.Cut(field, "=")
				if (k == "source" || k == "src") && dockerSockets[v] {
					hit = true
				}
			}
		}
		if hit {
			return Finding{
				EntryID: "cmdline.privileged", Severity: SeverityWarning,
				Captures:    map[string]string{"option": l.optionText(o)},
				Fingerprint: "cmdline.privileged:" + o.name, pos: o.start,
			}, true
		}
	}
	return Finding{}, false
}

// withMasked signale dans les captures qu'un secret a été masqué dans la commande corrigée.
func withMasked(c map[string]string, masked bool) map[string]string {
	if masked {
		c["masked"] = "1"
	}
	return c
}
