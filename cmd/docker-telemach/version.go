package main

import (
	"runtime/debug"
	"strings"
)

const (
	// websiteURL est le site du projet, affiché par --version.
	websiteURL = "https://www.telemach-learning.fr"
	// licenseID est l'identifiant SPDX de la licence du projet (fichier LICENSE).
	licenseID = "Apache-2.0"
)

// resolveVersion renvoie la version injectée à la compilation ; à défaut, celle que Go
// inscrit dans le binaire (tag git ou pseudo-version lors d'un « go build » depuis un clone).
func resolveVersion(injected string, readBuildInfo func() (*debug.BuildInfo, bool)) string {
	if injected != "dev" {
		return injected
	}
	if info, ok := readBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return strings.TrimPrefix(v, "v")
		}
	}
	return injected
}

// versionTemplate est le texte affiché par « docker telemach --version ».
const versionTemplate = `TELEMACH PLUGIN DOCKER {{.Version}}
Licence ` + licenseID + ` — ` + websiteURL + `
`
