package catalog

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/aboigues/telemach-plugin-docker/internal/redact"
)

// Options interdites dans un exemple ou une correction sûre (FR-010, principe III).
var forbidden = []*regexp.Regexp{
	regexp.MustCompile(`--privileged`),
	regexp.MustCompile(`chmod\s+666`),
	regexp.MustCompile(`(?:--user[ =]|-u\s+)(?:root|0)(?::|\s|$)`),
	regexp.MustCompile(`--net(?:work)?[ =]host\b`),
}

var (
	runLine    = regexp.MustCompile(`docker (?:run|create)\b(.*)`)
	pullLine   = regexp.MustCompile(`docker (?:pull|image (?:inspect|ls))((?:\s+[^\s'«»]+)+)`)
	yamlImg    = regexp.MustCompile(`(?m)^\s*image:\s*(\S+)`)
	fromLine   = regexp.MustCompile(`FROM\s+(\S+)`)
	userFlag   = regexp.MustCompile(`(?:--user[ =]|-u\s+)(?:[1-9]|"?\$\(id -u\))`)
	userDocker = regexp.MustCompile(`USER\s+[1-9]`)
)

// Options de docker run qui attendent une valeur (pour trouver le nom de l'image).
var runValueFlags = map[string]bool{
	"-p": true, "--publish": true, "-v": true, "--volume": true, "--name": true, "-e": true, "--env": true,
	"--user": true, "-u": true, "--network": true, "--mount": true, "--entrypoint": true, "--memory": true,
	"--platform": true, "-w": true, "--workdir": true,
}

type allowed struct{ needsUser bool }

func loadAllowlist(t *testing.T) map[string]allowed {
	t.Helper()
	f, err := os.Open(filepath.Join("data", "example-images.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close() //nolint:errcheck
	out := map[string]allowed{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line, _, _ := strings.Cut(sc.Text(), "#")
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		if fields[1] != "rootless" && fields[1] != "needs-user" {
			t.Fatalf("mode inconnu pour %s : %s", fields[0], fields[1])
		}
		out[fields[0]] = allowed{needsUser: fields[1] == "needs-user"}
	}
	return out
}

// imageName sépare le nom et le tag d'une référence d'image.
func imageName(ref string) (name, tag string) {
	ref = strings.Trim(ref, `"'`)
	if i := strings.LastIndex(ref, ":"); i > strings.LastIndex(ref, "/") {
		return ref[:i], ref[i+1:]
	}
	return ref, ""
}

// runImage extrait l'image d'une ligne « docker run … ».
func runImage(args string) string {
	fields := strings.Fields(args)
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if strings.HasPrefix(f, "-") {
			if runValueFlags[f] {
				i++
			}
			continue
		}
		return f
	}
	return ""
}

// TestExamplesAreSafe applique la constitution v1.1.0 aux exemples produits par le
// catalogue. Les valeurs issues de la commande de l'apprenant ({{.image}}…) sont exclues :
// elles reprennent sa commande et ne sont pas des exemples produits.
func TestExamplesAreSafe(t *testing.T) {
	c, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	allow := loadAllowlist(t)
	for _, e := range c.Entries {
		texts := []string{e.Deepen.Example, e.Deepen.Exercise}
		for _, f := range e.Fixes {
			if f.Safe {
				texts = append(texts, f.Command)
			}
		}
		for _, raw := range texts {
			text := Render(raw, map[string]string{})
			for _, re := range forbidden {
				if re.MatchString(text) {
					t.Errorf("%s : option interdite %q dans un exemple ou une correction sûre :\n%s", e.ID, re, text)
				}
			}
			if redact.Redact(text) != text {
				t.Errorf("%s : secret littéral dans l'exemple :\n%s", e.ID, text)
			}
			checkImages(t, e.ID, raw, text, allow)
		}
	}
}

func checkImages(t *testing.T, id, raw, text string, allow map[string]allowed) {
	t.Helper()
	type ref struct {
		image      string
		hasUser    bool
		fromDocker bool
	}
	var refs []ref
	for _, line := range strings.Split(text, "\n") {
		if m := runLine.FindStringSubmatch(line); m != nil {
			img := runImage(m[1])
			if img != "" && !strings.Contains(img, "…") && !strings.Contains(img, "{{") {
				refs = append(refs, ref{image: img, hasUser: userFlag.MatchString(m[1])})
			}
		}
		if m := pullLine.FindStringSubmatch(line); m != nil {
			for _, tok := range strings.Fields(m[1]) {
				if !strings.HasPrefix(tok, "-") {
					refs = append(refs, ref{image: tok, hasUser: true})
					break
				}
			}
		}
		for _, m := range fromLine.FindAllStringSubmatch(line, -1) {
			refs = append(refs, ref{image: m[1], hasUser: userDocker.MatchString(text), fromDocker: true})
		}
	}
	for _, m := range yamlImg.FindAllStringSubmatch(text, -1) {
		refs = append(refs, ref{image: m[1], hasUser: true})
	}
	for _, r := range refs {
		if strings.Contains(raw, "{{.image}}") && r.image == "…" {
			continue
		}
		name, tag := imageName(r.image)
		// Un nom en majuscules (« Alpine ») n'apparaît que pour illustrer l'erreur de format.
		a, ok := allow[strings.ToLower(name)]
		if !ok {
			t.Errorf("%s : image %q absente de example-images.txt", id, name)
			continue
		}
		if a.needsUser && !r.hasUser {
			t.Errorf("%s : l'image %q tourne en root par défaut ; l'exemple doit préciser --user (ou USER) :\n%s", id, name, text)
		}
		if tag == "" || tag == "latest" {
			// Constitution v1.1.0 : avertissement non bloquant.
			t.Logf("⚠ %s : image %q sans tag explicite (latest implicite)", id, r.image)
		}
	}
}
