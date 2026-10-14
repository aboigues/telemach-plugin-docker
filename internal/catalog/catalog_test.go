package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// validEntry est une entrée minimale conforme, utilisée comme base des cas d'erreur.
const validEntry = `
- id: run.port-already-allocated
  category: network
  applies: [run]
  patterns: ['port is already allocated']
  concept:
    name: port publishing
    definition: Relie un port de la machine à un port du conteneur.
  summary: Le port {{.port}} de ta machine est déjà utilisé.
  cause: Un autre conteneur écoute déjà sur ce port.
  fixes:
    - description: Utiliser un autre port de la machine.
      command: docker run -p 8081:80 {{.image}}
      safe: true
  deepen:
    explanation: Un port de la machine ne peut être publié qu'une seule fois.
    example: docker ps --filter publish=8080
    exercise: Lance deux conteneurs sur deux ports différents.
  doc_url: https://docs.docker.com/engine/network/
  tests: [run-port-allocated]
`

func load(t *testing.T, content string) error {
	t.Helper()
	_, err := Load(fstest.MapFS{"x.yaml": {Data: []byte(content)}})
	return err
}

func TestLoadValidEntry(t *testing.T) {
	if err := load(t, validEntry); err != nil {
		t.Fatalf("entrée valide refusée : %v", err)
	}
}

func TestValidationRules(t *testing.T) {
	tests := []struct {
		name, from, to, wantErr string
	}{
		{"identifiant invalide", "id: run.port-already-allocated", "id: Run_Port", "identifiant invalide"},
		{"catégorie inconnue", "category: network", "category: réseau", "catégorie inconnue"},
		{"motif invalide", "'port is already allocated'", "'(['", "motif invalide"},
		{"résumé trop long", "summary: Le port {{.port}} de ta machine est déjà utilisé.", "summary: " + strings.Repeat("é", 161), "résumé trop long"},
		{"cause manquante", "cause: Un autre conteneur écoute déjà sur ce port.", "cause: ''", "cause obligatoire"},
		{"doc hors docs.docker.com", "https://docs.docker.com/engine/network/", "https://example.com/", "doc_url"},
		{"tests manquants", "tests: [run-port-allocated]", "tests: []", "cas de test"},
		{"texte en anglais", "cause: Un autre conteneur écoute déjà sur ce port.", "cause: Another container listens on this port.", "français"},
		{"champ inconnu", "doc_url:", "unknown_field: x\n  doc_url:", "unknown_field"},
		{"correction non sûre sans anti-pattern", "safe: true", "safe: false", "anti_pattern"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := strings.Replace(validEntry, tt.from, tt.to, 1)
			if content == validEntry {
				t.Fatalf("remplacement sans effet : %q", tt.from)
			}
			err := load(t, content)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("erreur contenant %q attendue, obtenu %v", tt.wantErr, err)
			}
		})
	}
}

func TestDuplicateID(t *testing.T) {
	if err := load(t, validEntry+strings.TrimPrefix(validEntry, "\n")); err == nil || !strings.Contains(err.Error(), "en double") {
		t.Errorf("identifiant en double non détecté : %v", err)
	}
}

func TestRender(t *testing.T) {
	if got := Render("Le port {{.port}} est pris.", map[string]string{"port": "8080"}); got != "Le port 8080 est pris." {
		t.Errorf("Render = %q", got)
	}
	if got := Render("Image {{.image}} absente.", nil); got != "Image … absente." {
		t.Errorf("Render sans capture = %q", got)
	}
}

// TestEmbeddedCatalog valide le catalogue livré et vérifie que chaque cas de test
// référencé existe dans testdata/errors/ et correspond à l'un des motifs de l'entrée.
func TestEmbeddedCatalog(t *testing.T) {
	c, err := Default()
	if err != nil {
		t.Fatalf("catalogue embarqué invalide :\n%v", err)
	}
	for _, e := range c.Entries {
		for _, name := range e.Tests {
			if e.Fallback == "compose-issue" {
				// Pédagogie d'un problème détecté par l'analyseur Compose : le cas de test
				// est un dossier de testdata/compose/.
				if _, err := os.Stat(filepath.Join("..", "..", "testdata", "compose", name, "expected.yaml")); err != nil {
					t.Errorf("%s : cas Compose introuvable : %v", e.ID, err)
				}
				continue
			}
			path := filepath.Join("..", "..", "testdata", "errors", name+".stderr.txt")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("%s : cas de test introuvable : %v", e.ID, err)
				continue
			}
			if len(e.Compiled()) == 0 {
				continue // entrée fondée sur le seul code de sortie
			}
			matched := false
			for _, re := range e.Compiled() {
				if re.Match(data) {
					matched = true
					break
				}
			}
			if !matched {
				t.Errorf("%s : aucun motif ne correspond à %s", e.ID, path)
			}
		}
	}
}
