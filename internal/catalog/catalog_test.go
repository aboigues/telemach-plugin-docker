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

// cmdlineEntry est une entrée minimale de piège de ligne de commande (feature 003).
var cmdlineEntry = strings.NewReplacer(
	"id: run.port-already-allocated", "id: cmdline.option-after-image",
	"category: network", "category: cmdline",
	"patterns: ['port is already allocated']", "fallback: cmdline\n  severity: warning",
).Replace(validEntry)

func TestCmdlineEntries(t *testing.T) {
	if err := load(t, cmdlineEntry); err != nil {
		t.Fatalf("entrée cmdline valide refusée : %v", err)
	}
	info := strings.Replace(cmdlineEntry, "severity: warning", "severity: info", 1)
	if err := load(t, info); err != nil {
		t.Fatalf("entrée cmdline « info » refusée : %v", err)
	}
	tests := []struct{ name, from, to, wantErr string }{
		{"sans niveau", "\n  severity: warning", "", "severity"},
		{"niveau inconnu", "severity: warning", "severity: grave", "severity"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := load(t, strings.Replace(cmdlineEntry, tt.from, tt.to, 1))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("erreur contenant %q attendue, obtenu %v", tt.wantErr, err)
			}
		})
	}
	t.Run("niveau hors cmdline", func(t *testing.T) {
		err := load(t, strings.Replace(validEntry, "category: network", "category: network\n  severity: info", 1))
		if err == nil || !strings.Contains(err.Error(), "severity") {
			t.Errorf("severity doit être refusée hors cmdline, obtenu %v", err)
		}
	})
}

// exitedEntry est une entrée minimale de cause d'arrêt d'un conteneur (feature 004).
var exitedEntry = strings.NewReplacer(
	"id: run.port-already-allocated", "id: exited.address-in-use",
	"category: network", "category: exited",
	"patterns: ['port is already allocated']", "fallback: exited\n  patterns: ['address already in use']",
).Replace(validEntry)

func TestExitedEntries(t *testing.T) {
	if err := load(t, exitedEntry); err != nil {
		t.Fatalf("entrée exited à motifs refusée : %v", err)
	}
	byCode := strings.Replace(exitedEntry, "patterns: ['address already in use']", "exit_codes: [137]", 1)
	if err := load(t, byCode); err != nil {
		t.Fatalf("entrée exited par code refusée : %v", err)
	}
	bare := strings.Replace(exitedEntry, "\n  patterns: ['address already in use']", "", 1)
	if err := load(t, bare); err != nil {
		t.Fatalf("entrée exited sans motif ni code (cause inconnue) refusée : %v", err)
	}
	if err := load(t, strings.Replace(exitedEntry, "fallback: exited", "fallback: cmdline\n  severity: info", 1)); err == nil {
		t.Error("catégorie exited avec un autre fallback : refus attendu")
	}
}

// Une entrée exited ne s'applique jamais au message d'erreur d'une commande, même si son motif
// correspond (le journal d'un conteneur et la sortie d'erreur de docker sont distincts).
func TestExitedEntriesIgnoredForCommandErrors(t *testing.T) {
	c, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range c.Entries {
		if e.Fallback == "exited" && e.Category != "exited" {
			t.Errorf("%s : fallback exited hors de la catégorie exited", e.ID)
		}
		if e.Category == "exited" && !strings.HasPrefix(e.ID, "exited.") {
			t.Errorf("%s : identifiant exited.* attendu", e.ID)
		}
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
			if e.Fallback == "cmdline" {
				// Piège de ligne de commande : le cas de test est un fichier de
				// testdata/cmdline/ dont au moins une ligne attend cette entrée.
				data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "cmdline", name+".tsv"))
				if err != nil {
					t.Errorf("%s : corpus introuvable : %v", e.ID, err)
				} else if !strings.Contains(string(data), e.ID) {
					t.Errorf("%s : aucune ligne de %s.tsv n'attend cette entrée", e.ID, name)
				}
				continue
			}
			if e.Fallback == "exited" {
				// Cause d'arrêt d'un conteneur : le cas de test est un dossier de
				// testdata/exited/ dont expected.yaml attend cette entrée.
				data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "exited", name, "expected.yaml"))
				if err != nil {
					t.Errorf("%s : cas exited introuvable : %v", e.ID, err)
				} else if !strings.Contains(string(data), "entry_id: "+e.ID) {
					t.Errorf("%s : %s/expected.yaml n'attend pas cette entrée", e.ID, name)
				}
				continue
			}
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
