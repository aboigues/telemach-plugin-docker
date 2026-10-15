package lint

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// corpusCase est une ligne d'un fichier testdata/cmdline/*.tsv (format du README).
type corpusCase struct {
	file, line string
	want       []string
	vars, dirs map[string]bool
	args       []string
}

// splitShell découpe une ligne comme le ferait le shell (guillemets simples et doubles,
// sans expansion).
func splitShell(t *testing.T, s string) []string {
	t.Helper()
	var out []string
	var cur strings.Builder
	inWord, quote := false, rune(0)
	for _, r := range s {
		switch {
		case quote != 0 && r == quote:
			quote = 0
		case quote != 0:
			cur.WriteRune(r)
		case r == '\'' || r == '"':
			quote, inWord = r, true
		case r == ' ' || r == '\t':
			if inWord {
				out = append(out, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	if quote != 0 {
		t.Fatalf("guillemet non fermé : %q", s)
	}
	if inWord {
		out = append(out, cur.String())
	}
	return out
}

func set(field string) map[string]bool {
	m := map[string]bool{}
	if field == "-" {
		return m
	}
	for _, v := range strings.Split(field, ",") {
		m[v] = true
	}
	return m
}

func loadCorpus(t *testing.T) []corpusCase {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "testdata", "cmdline", "*.tsv"))
	if err != nil || len(files) == 0 {
		t.Fatalf("corpus introuvable : %v", err)
	}
	var cases []corpusCase
	for _, path := range files {
		f, err := os.Open(path) // #nosec G304 -- fichiers du corpus
		if err != nil {
			t.Fatal(err)
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := sc.Text()
			if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
				continue
			}
			fields := strings.SplitN(line, "\t", 4)
			if len(fields) != 4 {
				t.Fatalf("%s : ligne mal formée : %q", path, line)
			}
			var want []string
			if fields[0] != "-" {
				want = strings.Split(fields[0], ",")
				sort.Strings(want)
			}
			cases = append(cases, corpusCase{
				file: filepath.Base(path), line: line, want: want,
				vars: set(fields[1]), dirs: set(fields[2]), args: splitShell(t, fields[3]),
			})
		}
		f.Close() //nolint:errcheck,gosec
	}
	return cases
}

// TestCorpus vérifie SC-001 (toutes les lignes fautives détectées) et SC-002 (aucun faux
// positif sur les lignes légitimes).
func TestCorpus(t *testing.T) {
	cases := loadCorpus(t)
	var faulty, detected, legit, silent int
	for _, c := range cases {
		env := Env{HasVar: func(n string) bool { return c.vars[n] }, DirExists: func(n string) bool { return c.dirs[n] }}
		var got []string
		for _, f := range Analyze(c.args, env) {
			got = append(got, f.EntryID)
		}
		sort.Strings(got)
		ok := strings.Join(got, ",") == strings.Join(c.want, ",")
		if len(c.want) > 0 {
			faulty++
			if ok {
				detected++
			}
		} else {
			legit++
			if ok {
				silent++
			}
		}
		if !ok {
			t.Errorf("%s : docker %s\n    obtenu  %v\n    attendu %v", c.file, strings.Join(c.args, " "), got, c.want)
		}
	}
	t.Logf("corpus : %d/%d lignes fautives détectées, %d/%d lignes légitimes silencieuses", detected, faulty, silent, legit)
}
