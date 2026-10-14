package command

import (
	"strings"

	"github.com/aboigues/telemach-plugin-docker/internal/redact"
)

// Family distingue les commandes docker des commandes docker compose.
type Family string

const (
	FamilyDocker  Family = "docker"
	FamilyCompose Family = "compose"
)

// Command est la commande analysée, dérivée des arguments capturés (sans « docker »).
type Command struct {
	Family       Family
	Subcommand   string            // run, build, pull, up, « network rm »…
	Flags        map[string]string // nom sans tirets → valeur (vide pour un booléen), masquée si sensible
	Positionals  []string          // arguments hors options (image, conteneur, commande…)
	ComposeFiles []string          // fichiers passés par -f/--file à docker compose
	Interactive  bool              // la commande utilise le terminal (stderr non capturé)
}

// Options globales de la CLI Docker qui attendent une valeur.
var globalValueFlags = set("c", "context", "H", "host", "config", "l", "log-level", "tlscacert", "tlscert", "tlskey")

// Options globales de docker compose qui attendent une valeur.
var composeGlobalValueFlags = set("f", "file", "p", "project-name", "project-directory", "env-file", "profile", "parallel", "progress", "ansi")

// Options qui attendent une valeur, par sous-commande. Les options absentes de ces listes
// sont considérées comme booléennes, sauf si elles sont écrites sous la forme --nom=valeur.
var valueFlags = map[string]map[string]bool{
	"run": set("e", "env", "env-file", "p", "publish", "v", "volume", "w", "workdir", "u", "user",
		"l", "label", "label-file", "m", "memory", "h", "hostname", "name", "network", "net", "entrypoint",
		"mount", "platform", "restart", "cpus", "c", "cpu-shares", "add-host", "dns", "expose", "health-cmd",
		"log-driver", "log-opt", "ulimit", "cap-add", "cap-drop", "device", "tmpfs", "pull", "gpus",
		"cidfile", "stop-signal", "stop-timeout", "shm-size", "pid", "ipc", "runtime", "security-opt",
		"sysctl", "volumes-from", "link", "network-alias", "ip", "mac-address", "a", "attach"),
	"exec":  set("e", "env", "env-file", "u", "user", "w", "workdir"),
	"build": set("t", "tag", "f", "file", "build-arg", "target", "platform", "label", "network", "secret", "ssh", "cache-from", "cache-to", "o", "output", "progress", "iidfile", "builder"),
	"login": set("u", "username", "p", "password"),
	"logs":  set("n", "tail", "since", "until"),
	"push":  set(),
	"pull":  set("platform"),
	"rm":    set(),
	"rmi":   set(),
	"stop":  set("t", "time", "s", "signal"),
	"start": set("detach-keys"),
	"ps":    set("f", "filter", "format", "n", "last"),
}

// Commandes dont les arguments après le premier argument positionnel appartiennent au
// conteneur (ils ne sont pas analysés comme des options).
var stopAtFirstPositional = set("run", "create", "exec")

// Commandes de gestion dont le second mot est l'action.
var managementCommands = set("container", "image", "network", "volume", "system", "builder", "buildx", "context", "plugin")

// Équivalents historiques des commandes de gestion container/image.
var legacyNames = map[string]string{
	"image rm": "rmi", "image remove": "rmi", "image ls": "images", "image list": "images",
	"image pull": "pull", "image push": "push", "image build": "build", "buildx build": "build",
	"container ls": "ps", "container list": "ps", "container remove": "rm",
}

// Parse analyse les arguments d'une commande docker (sans le mot « docker »). Les valeurs
// sensibles sont masquées dès l'analyse.
func Parse(args []string) Command {
	args = redact.RedactArgs(args)
	cmd := Command{Family: FamilyDocker, Flags: map[string]string{}}

	i := skipFlags(args, 0, globalValueFlags, nil)
	if i >= len(args) {
		return cmd
	}
	word := args[i]
	i++

	if word == "compose" {
		cmd.Family = FamilyCompose
		i = skipFlags(args, i, composeGlobalValueFlags, func(name, value string) {
			if name == "f" || name == "file" {
				cmd.ComposeFiles = append(cmd.ComposeFiles, value)
			}
			cmd.Flags[name] = value
		})
		if i >= len(args) {
			return cmd
		}
		cmd.Subcommand = args[i]
		parseRest(&cmd, args[i+1:], set())
		return cmd
	}

	if managementCommands[word] && i < len(args) && !strings.HasPrefix(args[i], "-") {
		action := args[i]
		i++
		name := word + " " + action
		switch {
		case legacyNames[name] != "":
			word = legacyNames[name]
		case word == "container":
			word = action
		default:
			word = name
		}
	}
	cmd.Subcommand = word
	parseRest(&cmd, args[i:], valueFlags[word])
	cmd.Interactive = isInteractive(cmd)
	return cmd
}

// Image renvoie l'image visée par la commande, si la sous-commande en désigne une.
func (c Command) Image() string {
	switch c.Subcommand {
	case "run", "create", "pull", "push", "rmi":
		if len(c.Positionals) > 0 {
			return c.Positionals[0]
		}
	}
	return ""
}

// parseRest analyse les options et arguments positionnels qui suivent la sous-commande.
func parseRest(cmd *Command, args []string, values map[string]bool) {
	stop := stopAtFirstPositional[cmd.Subcommand]
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			cmd.Positionals = append(cmd.Positionals, args[i+1:]...)
			return
		}
		if !strings.HasPrefix(a, "-") || a == "-" {
			cmd.Positionals = append(cmd.Positionals, a)
			if stop {
				cmd.Positionals = append(cmd.Positionals, args[i+1:]...)
				return
			}
			continue
		}
		i += consumeFlag(args, i, values, func(name, value string) { cmd.Flags[name] = value })
	}
}

// skipFlags avance sur les options situées à partir de start et renvoie l'indice du
// premier argument qui n'est pas une option.
func skipFlags(args []string, start int, values map[string]bool, onFlag func(name, value string)) int {
	i := start
	for i < len(args) && strings.HasPrefix(args[i], "-") && args[i] != "-" {
		if onFlag == nil {
			onFlag = func(string, string) {}
		}
		i += consumeFlag(args, i, values, onFlag) + 1
	}
	return i
}

// consumeFlag traite l'option args[i] et renvoie le nombre d'arguments supplémentaires
// consommés (1 si la valeur est l'argument suivant, 0 sinon).
func consumeFlag(args []string, i int, values map[string]bool, onFlag func(name, value string)) int {
	a := args[i]
	if strings.HasPrefix(a, "--") {
		name, value, hasValue := strings.Cut(a[2:], "=")
		if hasValue {
			onFlag(name, value)
			return 0
		}
		if values[name] && i+1 < len(args) {
			onFlag(name, args[i+1])
			return 1
		}
		onFlag(name, "")
		return 0
	}
	// Options courtes, éventuellement groupées (-it) ; la dernière peut attendre une valeur.
	short := a[1:]
	for j, r := range short {
		name := string(r)
		if values[name] {
			if rest := short[j+1:]; rest != "" {
				onFlag(name, strings.TrimPrefix(rest, "="))
				return 0
			}
			if i+1 < len(args) {
				onFlag(name, args[i+1])
				return 1
			}
		}
		onFlag(name, "")
	}
	return 0
}

// isInteractive indique si la commande utilise le terminal de façon interactive.
func isInteractive(c Command) bool {
	_, i := c.Flags["i"]
	_, t := c.Flags["t"]
	_, interactive := c.Flags["interactive"]
	_, tty := c.Flags["tty"]
	switch c.Subcommand {
	case "attach":
		return true
	case "login":
		_, stdin := c.Flags["password-stdin"]
		_, p := c.Flags["p"]
		_, password := c.Flags["password"]
		return !stdin && !p && !password
	case "run", "create", "exec", "start":
		return (i || interactive) && (t || tty || c.Subcommand == "start")
	}
	return false
}

func set(items ...string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, it := range items {
		m[it] = true
	}
	return m
}
