package main

import (
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
	"github.com/aboigues/telemach-plugin-docker/internal/command"
	"github.com/aboigues/telemach-plugin-docker/internal/compose"
	"github.com/aboigues/telemach-plugin-docker/internal/diagnose"
	"github.com/aboigues/telemach-plugin-docker/internal/redact"
	"github.com/aboigues/telemach-plugin-docker/internal/render"
	"github.com/aboigues/telemach-plugin-docker/internal/sanitize"
	"github.com/aboigues/telemach-plugin-docker/internal/state"
)

// maxStderr est la taille maximale lue : au-delà, seule la fin du message est conservée.
const maxStderr = 1 << 20

var versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+([-+.][0-9A-Za-z.-]+)?$`)

type explainFlags struct {
	exitCode      int
	stderrFile    string
	interactive   bool
	dockerVersion string
}

func (a *app) explainCommand() *cobra.Command {
	var f explainFlags
	cmd := &cobra.Command{
		Use:   "explain --exit-code <n> [--stderr-file <chemin>] [--interactive] [--docker-version <x.y.z>] -- <args docker…>",
		Short: "Explique une commande docker en échec (appelée par l'intégration shell)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if f.exitCode < 1 || f.exitCode > 255 {
				fmt.Fprintf(a.stderr, "docker telemach : --exit-code doit être compris entre 1 et 255 (reçu %d)\n", f.exitCode)
				return exitError{code: 2}
			}
			return a.explain(f, args)
		},
	}
	cmd.Flags().IntVar(&f.exitCode, "exit-code", 0, "code de sortie de la commande docker (1 à 255)")
	cmd.Flags().StringVar(&f.stderrFile, "stderr-file", "", "fichier contenant la sortie d'erreur de la commande")
	cmd.Flags().BoolVar(&f.interactive, "interactive", false, "la commande était interactive (sortie d'erreur non capturée)")
	cmd.Flags().StringVar(&f.dockerVersion, "docker-version", "", "version du client Docker")
	_ = cmd.MarkFlagRequired("exit-code")
	return cmd
}

// explain enchaîne : lecture → nettoyage → masquage → analyse → diagnostic → rendu.
// Une explication est toujours produite (code 0), même pour une erreur non reconnue.
func (a *app) explain(f explainFlags, args []string) error {
	raw := ""
	if f.stderrFile != "" && !f.interactive {
		var err error
		if raw, err = readTail(f.stderrFile, maxStderr); err != nil {
			raw = ""
		}
	}
	stderr := redact.Redact(sanitize.Clean(raw))

	cmd := command.Parse(args)
	cat, err := catalog.Default()
	if err != nil {
		cat = nil // catalogue invalide : le repli « non reconnue » reste disponible
	}
	in := diagnose.Input{Command: cmd, ExitCode: f.exitCode, Stderr: stderr, Interactive: f.interactive || cmd.Interactive}
	d := diagnose.Diagnose(in, cat)

	now := a.now()
	repeated := a.store.IsRepeat(d.Fingerprint, now)

	dockerVersion := ""
	if versionPattern.MatchString(f.dockerVersion) {
		dockerVersion = f.dockerVersion
	}
	theme, w, width := a.output(a.stderr)
	opts := render.Options{
		Theme: theme, Width: width, DockerVersion: dockerVersion, Repeated: repeated,
		Interactive: in.Interactive, ExitCode: f.exitCode,
	}
	entryID, err := a.renderDiagnosis(w, in, d, stderr, opts)
	if err != nil {
		return err
	}
	_ = a.store.Save(state.LastDiagnosis{Fingerprint: d.Fingerprint, EntryID: entryID, Captures: d.Captures, RecordedAt: now})
	return nil
}

// renderDiagnosis affiche le diagnostic. Pour docker compose, les fichiers Compose sont
// d'abord analysés : un problème bloquant y est montré à la place de l'explication
// générique ; les simples remarques (variables, secrets) sont ajoutées après.
// Elle renvoie l'identifiant de l'entrée expliquée, mémorisé pour « deepen ».
func (a *app) renderDiagnosis(w io.Writer, in diagnose.Input, d diagnose.Diagnosis, stderr string, opts render.Options) (string, error) {
	entryID := ""
	if d.Entry != nil {
		entryID = d.Entry.ID
	}
	fmt.Fprintln(w)
	if in.Command.Family != command.FamilyCompose {
		return entryID, render.RenderExplanation(w, d, stderr, opts)
	}
	dir := in.Command.Flags["project-directory"]
	if dir == "" {
		if wd, err := a.getwd(); err == nil {
			dir = wd
		}
	}
	issues := compose.Analyze(dir, in.Command.ComposeFiles)
	cat, _ := catalog.Default()
	if blocking(issues) {
		return "compose." + string(issues[0].Kind), compose.RenderIssues(w, dir, nil, issues, cat, opts)
	}
	if err := render.RenderExplanation(w, d, stderr, opts); err != nil {
		return entryID, err
	}
	if len(issues) > 0 {
		fmt.Fprintln(w)
		return entryID, compose.RenderIssues(w, dir, nil, issues, cat, opts)
	}
	return entryID, nil
}

func blocking(issues []compose.Issue) bool {
	for _, is := range issues {
		if is.Blocking {
			return true
		}
	}
	return false
}

// readTail lit au plus limit octets, en gardant la fin du fichier, sans le modifier.
func readTail(path string, limit int64) (string, error) {
	f, err := os.Open(path) //nolint:gosec // chemin fourni par l'intégration shell
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	if info.Size() > limit {
		if _, err := f.Seek(info.Size()-limit, io.SeekStart); err != nil {
			return "", err
		}
	}
	data, err := io.ReadAll(io.LimitReader(f, limit))
	return string(data), err
}
