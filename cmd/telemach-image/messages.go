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

// archLabel présente le système et l'architecture installés de façon lisible.
func archLabel(hostOS, arch string) string {
	names := map[string]string{
		"darwin/arm64": "macOS, Apple Silicon",
		"darwin/amd64": "macOS, Intel",
		"linux/amd64":  "Linux, x86-64",
		"linux/arm64":  "Linux, ARM64",
	}
	key := hostOS + "/" + arch
	if name, ok := names[key]; ok {
		return name + " — " + key
	}
	return key
}

func successText(t render.Theme, version, previous string, upToDate bool, hostOS, arch string) string {
	var b strings.Builder
	switch {
	case upToDate:
		fmt.Fprintf(&b, "%s\n", t.Success(render.SymbolSuccess+" TELEMACH "+version+" est déjà à jour dans ~/.docker/cli-plugins ("+archLabel(hostOS, arch)+")."))
		fmt.Fprintf(&b, "%s\n\n", indent("Le binaire a été réécrit à l'identique.", 2))
	default:
		fmt.Fprintf(&b, "%s\n", t.Success(render.SymbolSuccess+" TELEMACH "+version+" installé dans ~/.docker/cli-plugins ("+archLabel(hostOS, arch)+")."))
		if previous != "" {
			fmt.Fprintf(&b, "%s\n", indent("Version remplacée : "+previous+".", 2))
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "%s\n", t.Concept("📘 Ce qui vient de se passer"))
	fmt.Fprintf(&b, "%s\n\n", indent("Le conteneur a écrit dans un dossier de ta machine grâce au montage\n« -v ~/.docker/cli-plugins:/sortie » (bind mount), avec ton identité (--user).\nDocker découvre automatiquement les plugins placés dans ce dossier.", 3))
	if hostOS == "linux" && arch != "" {
		fmt.Fprintf(&b, "%s\n", indent(t.Dim("Sur un Mac, relance avec -e HOST_OS=darwin pour installer le binaire macOS."), 3))
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "%s\n", t.Label("➜ Étape suivante : active l'explication automatique des erreurs"))
	fmt.Fprintf(&b, "%s\n", indent(t.Fix(`echo 'eval "$(docker telemach shell-init zsh)"' >> ~/.zshrc`), 3))
	fmt.Fprintf(&b, "%s\n", indent(t.Dim(`(bash : echo 'eval "$(docker telemach shell-init bash)"' >> ~/.bashrc)`), 3))
	fmt.Fprintf(&b, "%s\n", indent("puis ouvre un nouveau terminal et vérifie : "+t.Fix("docker telemach --version"), 3))
	return b.String()
}

func refusalText(t render.Theme, kind refusalKind, tag, hostOS, detail string) string {
	var title, concept, explain string
	var fix []string
	install := installCommand(tag, hostOS)
	switch kind {
	case refuseNotMounted:
		title = "Aucun dossier de destination : /sortie n'est pas monté."
		concept = "montage (bind mount)"
		explain = "Un conteneur ne voit pas les fichiers de ta machine. L'option -v DOSSIER_MACHINE:/sortie\nlui donne accès à un dossier précis, ici celui des plugins Docker."
		fix = []string{mkdirCommand, install}
	case refuseRootOwned:
		title = "Le dossier des plugins appartient à root : TELEMACH ne peut pas y écrire."
		concept = "dossier créé par Docker"
		explain = "Quand le dossier monté n'existe pas, Docker le crée lui-même, avec le compte root.\nC'est pour cela que la commande commence par mkdir -p : le dossier t'appartient."
		fix = []string{`sudo chown "$(id -u):$(id -g)" ~/.docker/cli-plugins`, install}
	case refuseNotWritable:
		title = "Le dossier des plugins n'est pas accessible en écriture pour le conteneur."
		concept = "identité du processus (--user)"
		explain = "Le conteneur s'exécute sans privilèges, avec un utilisateur qui n'est pas toi.\nL'option --user \"$(id -u):$(id -g)\" lui donne ton identité : il peut alors écrire\ndans ton dossier, et le fichier installé t'appartient."
		fix = []string{install}
	case refuseNotRegular:
		title = "~/.docker/cli-plugins/docker-telemach existe déjà mais n'est pas un fichier ordinaire."
		concept = "fichier existant"
		explain = "Un lien symbolique ou un dossier porte déjà ce nom. Par sécurité, TELEMACH ne\nremplace pas ce qu'il ne reconnaît pas."
		fix = []string{"ls -l ~/.docker/cli-plugins/docker-telemach", "# s'il n'est plus utile : rm -r ~/.docker/cli-plugins/docker-telemach", install}
	case refuseForeign:
		title = "~/.docker/cli-plugins/docker-telemach existe déjà et n'est pas un binaire TELEMACH."
		concept = "fichier existant"
		explain = "Par sécurité, TELEMACH ne remplace pas un fichier qu'il ne reconnaît pas."
		fix = []string{"ls -l ~/.docker/cli-plugins/docker-telemach", "# s'il n'est plus utile : rm ~/.docker/cli-plugins/docker-telemach", install}
	case refuseCopyFailed:
		title = "La copie du plugin a échoué : " + detail
		concept = "installation interrompue"
		explain = "Rien n'a été laissé dans le dossier des plugins. Vérifie l'espace disque et relance."
		fix = []string{install}
	case refuseWindows:
		title = "Windows n'est pas pris en charge (HOST_OS=windows)."
		concept = "WSL2"
		explain = "Le plugin fonctionne sous Linux et macOS. Sous Windows, utilise-le dans WSL2 :\nlance la commande d'installation depuis ton terminal WSL2, sans HOST_OS."
		fix = []string{mkdirCommand, installCommand(tag, "linux")}
	case refuseUnknownOS:
		title = "Système inconnu : HOST_OS=" + hostOS + "."
		concept = "système cible"
		explain = "HOST_OS indique le système de ta machine : linux (par défaut) ou darwin (macOS)."
		fix = []string{installCommand(tag, "linux"), installCommand(tag, "darwin")}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", t.Anti(render.SymbolError+" "+title))
	fmt.Fprintf(&b, "%s\n", t.Concept(render.LabelConcept+" — "+concept))
	fmt.Fprintf(&b, "%s\n\n", indent(explain, 3))
	fmt.Fprintf(&b, "%s\n", t.Fix("💡 Correction"))
	fmt.Fprintf(&b, "%s\n", indent(strings.Join(fix, "\n"), 3))
	return b.String()
}

func installHelpText(t render.Theme, tag string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", t.Title("install — copie le plugin TELEMACH dans le dossier des plugins Docker de ta machine"))
	fmt.Fprintf(&b, "%s\n", t.Label("Linux et WSL2"))
	fmt.Fprintf(&b, "%s\n\n", indent(mkdirCommand+"\n"+installCommand(tag, "linux"), 3))
	fmt.Fprintf(&b, "%s\n", t.Label("macOS (Docker Desktop)"))
	fmt.Fprintf(&b, "%s\n\n", indent(mkdirCommand+"\n"+installCommand(tag, "darwin"), 3))
	fmt.Fprintf(&b, "%s\n", t.Label("Options"))
	fmt.Fprintf(&b, "%s\n", indent("-v ~/.docker/cli-plugins:/sortie   dossier de ta machine où copier le plugin (obligatoire)\n--user \"$(id -u):$(id -g)\"          le conteneur écrit avec ton identité\n-e HOST_OS=darwin                  installe le binaire macOS (linux par défaut)", 3))
	return b.String()
}

func noComposeText(t render.Theme, tag, dir string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", t.Anti(render.SymbolError+" Aucun fichier Compose trouvé dans "+dir+"."))
	fmt.Fprintf(&b, "%s\n", t.Concept(render.LabelConcept+" — montage (bind mount)"))
	fmt.Fprintf(&b, "%s\n\n", indent("Le conteneur ne voit que les dossiers montés avec -v. Lance la commande depuis le\ndossier de ton projet (celui qui contient compose.yaml) : -v \"$PWD\":/projet:ro le\nrend visible dans le conteneur, en lecture seule.", 3))
	fmt.Fprintf(&b, "%s\n", t.Fix("💡 Correction"))
	fmt.Fprintf(&b, "%s\n", indent("cd chemin/vers/ton/projet\n"+tryCommand(tag), 3))
	return b.String()
}
