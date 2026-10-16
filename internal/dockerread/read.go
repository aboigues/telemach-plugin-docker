package dockerread

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/aboigues/telemach-plugin-docker/internal/redact"
	"github.com/aboigues/telemach-plugin-docker/internal/sanitize"
)

// allowed est la liste fermée des sous-commandes autorisées (research R2) : toutes sont des
// lectures. Ajouter une entrée ici demande de mettre à jour FR-010 et ses tests.
var allowed = map[string]bool{"ps": true, "inspect": true, "logs": true, "events": true}

// DefaultTimeout est le délai maximal d'une lecture.
const DefaultTimeout = 5 * time.Second

var (
	// ErrNotAllowed signale une sous-commande hors de la liste fermée : rien n'est exécuté.
	ErrNotAllowed = errors.New("lecture du démon non autorisée")
	// ErrTimeout signale une lecture interrompue après le délai maximal.
	ErrTimeout = errors.New("le démon Docker ne répond pas dans le délai")
)

// CLIError est un échec de la CLI docker (démon injoignable, permission refusée…), avec sa
// sortie d'erreur nettoyée et masquée.
type CLIError struct {
	Code   int
	Stderr string
}

func (e *CLIError) Error() string { return fmt.Sprintf("docker (code %d) : %s", e.Code, e.Stderr) }

// Runner exécute les lectures avec la CLI docker de l'apprenant : même contexte, même hôte,
// mêmes certificats (Global : options globales reçues avant « telemach »).
type Runner struct {
	Bin     string // « docker » par défaut, cherché dans le PATH
	Global  []string
	Timeout time.Duration
}

// output exécute « docker <global> sub args… » sans shell et renvoie stdout.
func (r Runner) output(sub string, args ...string) ([]byte, error) {
	stdout, _, err := r.exec(sub, args...)
	return stdout, err
}

func (r Runner) exec(sub string, args ...string) (stdout, stderr []byte, err error) {
	if !allowed[sub] {
		return nil, nil, fmt.Errorf("%w : %s", ErrNotAllowed, sub)
	}
	bin := r.Bin
	if bin == "" {
		bin = "docker"
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	argv := append(append(append([]string{}, r.Global...), sub), args...)
	cmd := exec.CommandContext(ctx, bin, argv...) // #nosec G204 -- liste d'arguments, sous-commande de la liste fermée
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	cmd.WaitDelay = time.Second
	runErr := cmd.Run()
	if ctx.Err() != nil {
		return nil, nil, ErrTimeout
	}
	if runErr != nil {
		code := -1
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			code = ee.ExitCode()
		}
		return nil, nil, &CLIError{Code: code, Stderr: clean(strings.TrimSpace(errb.String()))}
	}
	return out.Bytes(), errb.Bytes(), nil
}

// ListCandidates liste les conteneurs arrêtés, en boucle de redémarrage ou créés sans avoir
// démarré, du plus récemment créé au plus ancien (ordre de « docker ps »).
func (r Runner) ListCandidates() ([]Candidate, error) {
	out, err := r.output("ps", "-a", "--no-trunc", "--filter", "status=exited", "--filter", "status=restarting",
		"--filter", "status=created", "--format", "{{json .}}")
	if err != nil {
		return nil, err
	}
	var cs []Candidate
	for _, line := range lines(out) {
		var p psLine
		if err := json.Unmarshal([]byte(line), &p); err != nil {
			return nil, fmt.Errorf("sortie de docker ps illisible : %w", err)
		}
		c := p.candidate()
		c.Name, c.Image = clean(c.Name), clean(c.Image)
		cs = append(cs, c)
	}
	return cs, nil
}

// Inspect décode l'état des conteneurs désignés (identifiants, préfixes ou noms).
func (r Runner) Inspect(ids ...string) ([]Container, error) {
	out, err := r.output("inspect", ids...)
	if err != nil {
		return nil, err
	}
	return DecodeInspect(out)
}

// DecodeInspect décode une sortie JSON de « docker inspect » (variables réduites aux noms).
func DecodeInspect(data []byte) ([]Container, error) {
	var raw []inspectJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("sortie de docker inspect illisible : %w", err)
	}
	cs := make([]Container, len(raw))
	for i, j := range raw {
		cs[i] = j.container()
	}
	return cs, nil
}

// Logs lit les n dernières lignes du journal (sorties standard et d'erreur du conteneur).
// Un journal illisible (pilote de journalisation) n'est pas une erreur : Unavailable.
func (r Runner) Logs(id string, n int) (LogExcerpt, error) {
	stdout, stderr, err := r.exec("logs", "--tail", strconv.Itoa(n), id)
	var ce *CLIError
	if errors.As(err, &ce) {
		return LogExcerpt{Unavailable: true, Reason: ce.Stderr}, nil
	}
	if err != nil {
		return LogExcerpt{}, err
	}
	// La CLI recopie le journal du conteneur sur ses deux sorties : l'ordre exact entre les
	// deux est perdu, stdout d'abord.
	var ex LogExcerpt
	for _, l := range append(lines(stdout), lines(stderr)...) {
		ex.Lines = append(ex.Lines, clean(l))
	}
	if len(ex.Lines) > n {
		ex.Lines = ex.Lines[len(ex.Lines)-n:]
	}
	return ex, nil
}

// Events lit les événements de création et d'arrêt de conteneurs entre since et until
// (secondes depuis l'époque, décimales permises) ; « --until » rend la main aussitôt.
func (r Runner) Events(since, until string) ([]Event, error) {
	out, err := r.output("events", "--since", since, "--until", until, "--filter", "type=container",
		"--filter", "event=create", "--filter", "event=die", "--format", "{{json .}}")
	if err != nil {
		return nil, err
	}
	var evs []Event
	for _, line := range lines(out) {
		var j eventJSON
		if err := json.Unmarshal([]byte(line), &j); err != nil {
			return nil, fmt.Errorf("sortie de docker events illisible : %w", err)
		}
		evs = append(evs, j.event())
	}
	return evs, nil
}

func lines(b []byte) []string {
	var out []string
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		if l := sc.Text(); strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

// clean nettoie (séquences de contrôle) puis masque (secrets) une chaîne venue du démon.
func clean(s string) string { return redact.Redact(sanitize.Clean(s)) }

// cleanArgs nettoie et masque une commande, puis la joint pour l'affichage.
func cleanArgs(args []string) string {
	c := make([]string, len(args))
	for i, a := range args {
		c[i] = sanitize.Clean(a)
	}
	return strings.Join(redact.RedactArgs(c), " ")
}
