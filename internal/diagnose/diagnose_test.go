package diagnose

import (
	"testing"
	"testing/fstest"

	"github.com/aboigues/telemach-plugin-docker/internal/catalog"
	"github.com/aboigues/telemach-plugin-docker/internal/command"
)

const testCatalog = `
- id: run.port-already-allocated
  category: network
  applies: [run]
  patterns: ['Bind for [0-9.]+:(?P<port>\d+) failed: port is already allocated']
  concept: {name: port publishing, definition: Relie un port de la machine à un port du conteneur.}
  summary: Le port {{.port}} de ta machine est déjà utilisé.
  cause: Un autre conteneur écoute déjà sur ce port.
  fixes: [{description: Utiliser un autre port., safe: true}]
  deepen: {explanation: Un port ne se publie qu'une fois., example: docker ps, exercise: Essaie deux ports.}
  doc_url: https://docs.docker.com/engine/network/
  tests: [x]
- id: compose.no-project
  category: compose
  applies: [compose]
  patterns: ['no configuration file provided']
  concept: {name: project, definition: Un projet est l'ensemble des services d'un fichier Compose.}
  summary: Aucun fichier Compose n'a été trouvé.
  cause: La commande est lancée hors d'un projet.
  fixes: [{description: Se placer dans le bon dossier., safe: true}]
  deepen: {explanation: Le nom du projet vient du dossier., example: docker compose ls, exercise: Lance ps dans le dossier.}
  doc_url: https://docs.docker.com/compose/
  tests: [x]
- id: runtime.exit-127
  category: runtime
  applies: [run, exec, start]
  exit_codes: [127]
  concept: {name: container command, definition: La commande lancée dans le conteneur.}
  summary: La commande est introuvable dans le conteneur.
  cause: Le programme demandé n'existe pas dans l'image.
  fixes: [{description: Vérifier le nom de la commande., safe: true}]
  deepen: {explanation: Le conteneur ne voit que les fichiers de l'image., example: docker run --rm alpine ls /bin, exercise: Liste /bin.}
  doc_url: https://docs.docker.com/reference/cli/docker/container/run/
  tests: [x]
`

func testCat(t *testing.T) *catalog.Catalog {
	t.Helper()
	c, err := catalog.Load(fstest.MapFS{"t.yaml": {Data: []byte(testCatalog)}})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestDiagnoseRecognized(t *testing.T) {
	in := Input{
		Command:  command.Parse([]string{"run", "-p", "8080:80", "nginx"}),
		ExitCode: 125,
		Stderr:   "docker: Error response from daemon: driver failed: Bind for 0.0.0.0:8080 failed: port is already allocated.",
	}
	d := Diagnose(in, testCat(t))
	if d.Confidence != Recognized || d.Entry == nil || d.Entry.ID != "run.port-already-allocated" {
		t.Fatalf("diagnostic inattendu : %+v", d)
	}
	if d.Captures["port"] != "8080" || d.Captures["image"] != "nginx" {
		t.Errorf("captures = %v", d.Captures)
	}
	if d.Origin != OriginDocker {
		t.Errorf("origine = %s, attendu docker", d.Origin)
	}
	if len(d.Fingerprint) != 64 {
		t.Errorf("empreinte invalide : %q", d.Fingerprint)
	}
}

func TestDiagnoseAppliesFilter(t *testing.T) {
	in := Input{Command: command.Parse([]string{"ps"}), ExitCode: 1, Stderr: "Bind for 0.0.0.0:8080 failed: port is already allocated"}
	if d := Diagnose(in, testCat(t)); d.Confidence != Unknown {
		t.Errorf("l'entrée ne s'applique qu'à run : %+v", d)
	}
}

func TestDiagnoseCompose(t *testing.T) {
	in := Input{Command: command.Parse([]string{"compose", "ps"}), ExitCode: 1, Stderr: "no configuration file provided: not found"}
	if d := Diagnose(in, testCat(t)); d.Entry == nil || d.Entry.ID != "compose.no-project" {
		t.Errorf("compose.no-project attendu : %+v", d)
	}
}

func TestDiagnoseUnknownHasNoEntry(t *testing.T) {
	in := Input{Command: command.Parse([]string{"pull", "x"}), ExitCode: 1, Stderr: "erreur totalement inventée"}
	d := Diagnose(in, testCat(t))
	if d.Confidence != Unknown || d.Entry != nil {
		t.Errorf("diagnostic inconnu attendu, sans entrée : %+v", d)
	}
}

func TestDiagnoseExitCodeOnly(t *testing.T) {
	in := Input{Command: command.Parse([]string{"run", "-it", "alpine", "foo"}), ExitCode: 127, Interactive: true}
	d := Diagnose(in, testCat(t))
	if d.Confidence != Partial || d.Entry == nil || d.Entry.ID != "runtime.exit-127" {
		t.Errorf("runtime.exit-127 attendu : %+v", d)
	}
	if d.Origin != OriginApplication {
		t.Errorf("origine = %s, attendu application", d.Origin)
	}
}

func TestFingerprintNormalization(t *testing.T) {
	cmd := command.Parse([]string{"rm", "web"})
	a := Fingerprint(cmd, "Error: container 3f2a9c1b7d4e5f6a is running at 2026-10-14T10:00:00Z")
	b := Fingerprint(cmd, "Error:  container 9e8d7c6b5a4f3e2d is running at 2026-10-14T11:22:33Z")
	if a != b {
		t.Error("les identifiants et horodatages doivent être neutralisés")
	}
	if a == Fingerprint(cmd, "autre erreur") {
		t.Error("deux erreurs différentes doivent avoir des empreintes différentes")
	}
}
