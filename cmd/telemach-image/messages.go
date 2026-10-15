package main

import (
	"fmt"
	"strings"

	"github.com/aboigues/telemach-plugin-docker/internal/render"
)

// imageRef renvoie la référence complète de l'image, avec un tag explicite (principe III).
func imageRef(tag string) string { return "telemachlearning/telemach:" + tag }

// Commandes canoniques (contracts/image-cli.md), à copier telles quelles.
func tryCommand(tag string) string {
	return `docker run --rm -t -v "$PWD":/projet:ro ` + imageRef(tag) + " check /projet"
}

func installCommand(tag, hostOS string) string {
	env := ""
	if hostOS == "darwin" {
		env = "-e HOST_OS=darwin "
	}
	return `docker run --rm --user "$(id -u):$(id -g)" ` + env + "-v ~/.docker/cli-plugins:/sortie " + imageRef(tag) + " install"
}

const mkdirCommand = "mkdir -p ~/.docker/cli-plugins"

// indent décale chaque ligne de n espaces.
func indent(s string, n int) string {
	pad := strings.Repeat(" ", n)
	return pad + strings.ReplaceAll(s, "\n", "\n"+pad)
}

func helpText(t render.Theme, tag string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", t.Title("TELEMACH — explique en français les erreurs Docker et Docker Compose"))
	fmt.Fprintf(&b, "%s\n\n", t.Dim("Image "+imageRef(tag)+" : essayer TELEMACH ou l'installer sur ta machine."))

	fmt.Fprintf(&b, "%s\n", t.Label("▶ Essayer sans rien installer (vérifier le fichier Compose du dossier courant)"))
	fmt.Fprintf(&b, "%s\n\n", indent(t.Fix(tryCommand(tag)), 3))

	fmt.Fprintf(&b, "%s\n", t.Label("▶ Installer le plugin — Linux et WSL2"))
	fmt.Fprintf(&b, "%s\n\n", indent(t.Fix(mkdirCommand+"\n"+installCommand(tag, "linux")), 3))

	fmt.Fprintf(&b, "%s\n", t.Label("▶ Installer le plugin — macOS (Docker Desktop)"))
	fmt.Fprintf(&b, "%s\n\n", indent(t.Fix(mkdirCommand+"\n"+installCommand(tag, "darwin")), 3))

	fmt.Fprintf(&b, "%s\n", t.Concept(render.LabelConcept+" — montage (bind mount)"))
	fmt.Fprintf(&b, "%s\n", indent("L'option -v DOSSIER_MACHINE:DOSSIER_CONTENEUR donne au conteneur accès à un dossier\nprécis de ta machine ; « :ro » le rend accessible en lecture seule.", 3))
	return b.String()
}
