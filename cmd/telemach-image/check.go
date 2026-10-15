package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// projectDir est le dossier du projet monté pour l'essai ; c'est aussi le WORKDIR de l'image.
const projectDir = "/projet"

// composeNames sont les fichiers cherchés par Docker Compose (et par le plugin) sans -f.
var composeNames = []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"}

// valueFlags sont les options de « check » suivies d'une valeur séparée.
var valueFlags = map[string]bool{"-f": true, "--file": true, "--color": true, "--width": true}

// checkPreflight contrôle le dossier visé par « check » avant de lancer le plugin : si aucun
// fichier Compose n'y est trouvé, l'image l'explique elle-même (montage du dossier) au lieu
// du message du plugin, qui ne sait pas qu'il tourne dans un conteneur. handled vaut vrai
// quand la commande a été traitée ici.
func (a *app) checkPreflight(args []string) (code int, handled bool) {
	dir := projectDir
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-f" || arg == "--file" || strings.HasPrefix(arg, "-f=") || strings.HasPrefix(arg, "--file="):
			return 0, false // fichiers explicites : le plugin les vérifie lui-même
		case valueFlags[arg]:
			i++
		case strings.HasPrefix(arg, "-"):
		default:
			dir = arg
		}
	}
	if hasComposeFile(dir) {
		return 0, false
	}
	theme, w := a.output(a.stderr)
	fmt.Fprint(w, noComposeText(theme, a.imageTag(), dir))
	return exitUsage, true
}

// hasComposeFile indique si dir est un dossier contenant un fichier Compose.
func hasComposeFile(dir string) bool {
	for _, name := range composeNames {
		// #nosec G703 -- simple lecture des métadonnées, dans le conteneur, du dossier que
		// l'apprenant a lui-même monté (en lecture seule).
		if fi, err := os.Stat(filepath.Join(dir, name)); err == nil && fi.Mode().IsRegular() {
			return true
		}
	}
	return false
}
