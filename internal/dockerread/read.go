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

// DefaultTimeout est le délai maximal d'une lecture : de quoi absorber un démon chargé, sans
// jamais laisser la commande bloquée (5 s étaient dépassés juste après la création de 200
// conteneurs, T031).
const DefaultTimeout = 10 * time.Second

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

// Bornes de « docker ps » : le démon met ~12 ms par conteneur renvoyé (mesuré avec 215
// conteneurs : 3 s sans borne), d'où une liste limitée aux plus récents (SC-004).
const (
	MaxStopped = 20 // conteneurs arrêtés ou en boucle de redémarrage
	MaxCreated = 10 // conteneurs créés sans avoir démarré (démarrage échoué possible)
)

// ListCandidates liste les conteneurs arrêtés ou en boucle de redémarrage, puis ceux créés
// sans avoir démarré, chacun du plus récemment créé au plus ancien, sans doublon.
func (r Runner) ListCandidates() ([]Candidate, error) {
	// Les deux lectures sont indépendantes : en parallèle, la plus courte ne s'ajoute pas.
	var created []Candidate
	var createdErr error
	done := make(chan struct{})
	go func() {
		created, createdErr = r.listPS("--filter", "status=created", "--last", strconv.Itoa(MaxCreated))
		close(done)
	}()
	stopped, err := r.ListStopped()
	<-done
	if err != nil {
		return nil, err
	}
	if createdErr != nil {
		return nil, createdErr
	}
	seen := map[string]bool{}
	var cs []Candidate
	for _, c := range append(stopped, created...) {
		if !seen[c.ID] {
			seen[c.ID] = true
			cs = append(cs, c)
		}
	}
	return cs, nil
}

// ListStopped liste seulement les conteneurs arrêtés ou en boucle de redémarrage, les plus
// récents d'abord (utilisé aussi par la complétion de « docker telemach ps <Tab> »).
func (r Runner) ListStopped() ([]Candidate, error) {
	return r.listPS("--filter", "status=exited", "--filter", "status=restarting", "--last", strconv.Itoa(MaxStopped))
}

func (r Runner) listPS(filters ...string) ([]Candidate, error) {
	out, err := r.output("ps", append(append([]string{"-a", "--no-trunc"}, filters...), "--format", "{{json .}}")...)
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
		e, err := DecodeEvent([]byte(line))
		if err != nil {
			return nil, err
		}
		evs = append(evs, e)
	}
	return evs, nil
}

// DecodeEvent décode une ligne JSON de « docker events ».
func DecodeEvent(line []byte) (Event, error) {
	var j eventJSON
	if err := json.Unmarshal(line, &j); err != nil {
		return Event{}, fmt.Errorf("sortie de docker events illisible : %w", err)
	}
	return j.event(), nil
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
