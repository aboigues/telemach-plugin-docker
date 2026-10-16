package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Signaux en attente d'affichage à l'invite du shell (feature 004, data-model.md
// § PendingSignal) : un fichier par signal, « pending-<pid du shell>-<horodatage> », écrit par
// « docker telemach watch » et affiché puis supprimé par le crochet d'invite de ce shell.

const pendingPrefix = "pending-"

// PendingDir renvoie le dossier des signaux : $XDG_RUNTIME_DIR/telemach (mémoire, effacé à la
// déconnexion), sinon le sous-dossier « pending » du dossier d'état.
func PendingDir(getenv func(string) string, stateDir string) string {
	if d := getenv("XDG_RUNTIME_DIR"); d != "" {
		return filepath.Join(d, "telemach")
	}
	return filepath.Join(stateDir, "pending")
}

// Alive indique si le processus pid existe encore.
func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// WritePending écrit un signal pour le shell pid, de façon atomique (fichier temporaire puis
// renommage : le crochet d'invite ne lit jamais un fichier à moitié écrit), dans un dossier
// 0700 et un fichier 0600. Si le shell n'existe plus, rien n'est écrit.
func WritePending(dir string, pid int, text string, now time.Time) error {
	if !Alive(pid) {
		return nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) //nolint:errcheck // déjà renommé si tout s'est bien passé
	if _, err := tmp.WriteString(strings.TrimRight(text, "\n") + "\n"); err != nil {
		tmp.Close() //nolint:errcheck,gosec
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	name := fmt.Sprintf("%s%d-%d", pendingPrefix, pid, now.UnixNano())
	return os.Rename(tmp.Name(), filepath.Join(dir, name))
}

// CleanStalePending supprime les signaux des shells qui n'existent plus, et les fichiers
// temporaires abandonnés (appelé par shell-init, au chargement de l'intégration).
func CleanStalePending(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		switch {
		case strings.HasPrefix(name, ".tmp-"):
			if info, err := e.Info(); err == nil && time.Since(info.ModTime()) > time.Minute {
				_ = os.Remove(filepath.Join(dir, name))
			}
		case strings.HasPrefix(name, pendingPrefix):
			pidStr, _, _ := strings.Cut(strings.TrimPrefix(name, pendingPrefix), "-")
			if pid, err := strconv.Atoi(pidStr); err != nil || !Alive(pid) {
				_ = os.Remove(filepath.Join(dir, name))
			}
		}
	}
}
