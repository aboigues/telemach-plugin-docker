package plugin

import (
	"encoding/json"
	"io"
)

// MetadataSubcommand est la sous-commande appelée par la CLI Docker pour découvrir le plugin.
const MetadataSubcommand = "docker-cli-plugin-metadata"

// metadata reprend le format attendu par la CLI Docker (schéma 0.1.0).
type metadata struct {
	SchemaVersion    string
	Vendor           string
	Version          string
	ShortDescription string
}

// WriteMetadata écrit les métadonnées du plugin au format JSON attendu par Docker.
func WriteMetadata(w io.Writer, version string) error {
	return json.NewEncoder(w).Encode(metadata{
		SchemaVersion:    "0.1.0",
		Vendor:           "TELEMACH",
		Version:          version,
		ShortDescription: "Explique en français les erreurs Docker et Docker Compose",
	})
}
