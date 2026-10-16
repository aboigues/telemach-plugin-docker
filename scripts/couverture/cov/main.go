// cov mesure la couverture du catalogue TELEMACH par rapport aux messages d'erreur du code
// source de Docker (étude docs/couverture-catalogue.md ; module à part, hors du plugin).
//
//	cov extract <étiquette> <racine>...   → TSV : étiquette, fichier:ligne, appel, format
//	cov match <dossier du catalogue>      ← TSV sur stdin, → TSV + entrées reconnues
//	cov text <dossier du catalogue>       ← un texte par ligne (\n échappés), → entrées
package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage : cov extract <étiquette> <racine>... | cov match <catalogue> | cov text <catalogue>")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "extract":
		for _, root := range os.Args[3:] {
			extract(os.Args[2], root)
		}
	case "match":
		match(load(os.Args[2]), false)
	case "text":
		match(load(os.Args[2]), true)
	default:
		os.Exit(2)
	}
}

// skipDir écarte tests, outillage et documentation : seuls restent les messages qu'un
// utilisateur peut voir.
var skipDir = regexp.MustCompile(`^(testdata|testutil|testutils|integration|integration-cli|e2e|hack|contrib|docs|man|fixtures|internal/test.*|test|tests|mocks?|example|examples)$`)

func extract(label, root string) {
	fset := token.NewFileSet()
	// #nosec G703 -- outil d'étude local : les racines sont choisies par qui lance l'outil.
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() {
			if path != root && (skipDir.MatchString(d.Name()) || d.Name() == "vendor" || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil
		}
		// Plages des appels errdefs.<Classe>(…) : la classe dit si l'erreur vient de la requête
		// de l'utilisateur (paramètre invalide, introuvable, conflit…) ou du système.
		type span struct {
			from, to token.Pos
			class    string
		}
		var spans []span
		ast.Inspect(f, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok {
				if sel, ok := c.Fun.(*ast.SelectorExpr); ok {
					if id, ok := sel.X.(*ast.Ident); ok && (id.Name == "errdefs" || id.Name == "cerrdefs") && len(c.Args) > 0 && !strings.HasPrefix(sel.Sel.Name, "Is") {
						spans = append(spans, span{c.Lparen, c.Rparen, sel.Sel.Name})
					}
				}
			}
			return true
		})
		classOf := func(pos token.Pos) string {
			best := "-"
			var size token.Pos = -1
			for _, sp := range spans {
				if pos > sp.from && pos < sp.to && (size < 0 || sp.to-sp.from < size) {
					best, size = sp.class, sp.to-sp.from
				}
			}
			return best
		}
		emit := func(pos token.Pos, kind, s string) {
			class := classOf(pos)
			if strings.HasPrefix(kind, "types.") {
				class = strings.TrimSuffix(strings.TrimPrefix(kind, "types."), "Errorf")
			}
			kind += "@" + class
			if meaningful(s) {
				fmt.Printf("%s\t%s:%d\t%s\t%s\n", label, rel, fset.Position(pos).Line, kind, escape(s))
			}
		}
		// Méthodes Error() : leur fmt.Sprintf ou leur chaîne retournée est le message.
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || fd.Name.Name != "Error" || fd.Body == nil {
				continue
			}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				switch v := n.(type) {
				case *ast.CallExpr:
					if sel, ok := v.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Sprintf" && len(v.Args) > 0 {
						if s, ok := literal(v.Args[0]); ok {
							emit(v.Pos(), "Error().Sprintf", s)
						}
					}
				case *ast.ReturnStmt:
					for _, r := range v.Results {
						if s, ok := literal(r); ok {
							emit(r.Pos(), "Error().return", s)
						}
					}
				}
				return true
			})
		}
		ast.Inspect(f, func(n ast.Node) bool {
			// Constantes et champs typés erreur : errX conflictError = "…", message: "…".
			if vs, ok := n.(*ast.ValueSpec); ok {
				if id, ok := vs.Type.(*ast.Ident); ok && strings.HasSuffix(strings.ToLower(id.Name), "error") {
					for _, v := range vs.Values {
						if s, ok := literal(v); ok {
							emit(v.Pos(), "typed-const", s)
						}
					}
				}
				return true
			}
			if kv, ok := n.(*ast.KeyValueExpr); ok {
				if id, ok := kv.Key.(*ast.Ident); ok && (id.Name == "message" || id.Name == "Message" || id.Name == "msg") {
					if s, ok := literal(kv.Value); ok {
						emit(kv.Pos(), "field", s)
					}
				}
				return true
			}
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			idx := argIndex(pkg.Name, sel.Sel.Name)
			if idx < 0 || idx >= len(call.Args) {
				return true
			}
			s, ok := literal(call.Args[idx])
			if !ok {
				return true
			}
			emit(call.Pos(), pkg.Name+"."+sel.Sel.Name, s)
			return true
		})
		return nil
	})
}

// argIndex donne la position du message dans les appels qui fabriquent une erreur.
func argIndex(pkg, fn string) int {
	switch pkg {
	case "errors", "fmt", "pkgerrors", "cerrdefs", "errdefs", "perrors":
		switch fn {
		case "New", "Errorf":
			return 0
		case "Wrap", "Wrapf":
			return 1
		}
	case "types":
		// libnetwork : types.InvalidParameterErrorf, types.NotFoundErrorf…
		if strings.HasSuffix(fn, "Errorf") {
			return 0
		}
	case "status", "grpcstatus":
		if fn == "Errorf" || fn == "Error" {
			return 1
		}
	}
	return -1
}

func literal(e ast.Expr) (string, bool) {
	switch v := e.(type) {
	case *ast.BasicLit:
		if v.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(v.Value)
		return s, err == nil
	case *ast.BinaryExpr:
		if v.Op != token.ADD {
			return "", false
		}
		// Une partie non littérale (« "No such container: " + id ») devient un verbe %v.
		a, ok1 := literal(v.X)
		b, ok2 := literal(v.Y)
		if !ok1 {
			a = "%v"
		}
		if !ok2 {
			b = "%v"
		}
		return a + b, ok1 || ok2
	case *ast.ParenExpr:
		return literal(v.X)
	}
	return "", false
}

var (
	verb    = regexp.MustCompile(`%[-+# 0]*(\d+|\*)?(\.(\d+|\*))?[a-zA-Z%]`)
	letters = regexp.MustCompile(`[A-Za-z]`)
)

// meaningful écarte les formats qui ne font qu'envelopper une autre erreur (« %s: %w »).
func meaningful(s string) bool {
	return len(letters.FindAllString(verb.ReplaceAllString(s, ""), -1)) >= 4
}

func escape(s string) string {
	return strings.NewReplacer("\\", "\\\\", "\t", "\\t", "\n", "\\n").Replace(s)
}

func unescape(s string) string {
	return strings.NewReplacer("\\\\", "\\", "\\t", "\t", "\\n", "\n").Replace(s)
}

type pattern struct {
	id string
	re *regexp.Regexp
}

func load(dir string) []pattern {
	files, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))
	sort.Strings(files)
	var ps []pattern
	for _, f := range files {
		data, err := os.ReadFile(f) // #nosec G304 G703 -- fichiers du catalogue, dossier passé en argument
		if err != nil {
			panic(err)
		}
		var entries []struct {
			ID       string   `yaml:"id"`
			Patterns []string `yaml:"patterns"`
		}
		if err := yaml.Unmarshal(data, &entries); err != nil {
			panic(fmt.Sprintf("%s : %v", f, err))
		}
		for _, e := range entries {
			for _, p := range e.Patterns {
				ps = append(ps, pattern{e.ID, regexp.MustCompile(p)})
			}
		}
	}
	if len(ps) == 0 {
		panic("aucun motif chargé")
	}
	fmt.Fprintf(os.Stderr, "%d motifs chargés\n", len(ps))
	return ps
}

// samples remplace les verbes de format par des valeurs plausibles : un même message est
// reconnu si l'une des variantes l'est.
var samples = []string{"x", "nginx", "nginx:latest", "0.0.0.0:8080", "/app/x", "8080", "web"}

func variants(format string) []string {
	var out []string
	for _, v := range samples {
		out = append(out, verb.ReplaceAllStringFunc(format, func(m string) string {
			switch m[len(m)-1] {
			case '%':
				return "%"
			case 'd':
				return "1"
			case 'q':
				return strconv.Quote(v)
			case 'w':
				// Erreur enveloppée : souvent une sentinelle (errdefs.ErrNotFound → « not found »).
				if v == "x" {
					return "not found"
				}
			}
			return v
		}))
	}
	return out
}

func match(ps []pattern, text bool) {
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		line := sc.Text()
		msg := line
		if !text {
			cols := strings.SplitN(line, "\t", 4)
			if len(cols) != 4 {
				continue
			}
			msg = cols[3]
		}
		var cands []string
		if text {
			cands = []string{unescape(msg)}
		} else {
			cands = variants(unescape(msg))
		}
		ids := map[string]bool{}
		for _, c := range cands {
			for _, p := range ps {
				if p.re.MatchString(c) {
					ids[p.id] = true
				}
			}
		}
		var list []string
		for id := range ids {
			list = append(list, id)
		}
		sort.Strings(list)
		res := "-"
		if len(list) > 0 {
			res = strings.Join(list, ",")
		}
		fmt.Printf("%s\t%s\n", line, res)
	}
}
