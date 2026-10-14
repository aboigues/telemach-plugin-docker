package redact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRedactCorpus(t *testing.T) {
	inputs, err := filepath.Glob("../../testdata/secrets/*.txt")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, in := range inputs {
		if strings.HasSuffix(in, ".expected.txt") {
			continue
		}
		count++
		name := strings.TrimSuffix(filepath.Base(in), ".txt")
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(in)
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(strings.TrimSuffix(in, ".txt") + ".expected.txt")
			if err != nil {
				t.Fatalf("fichier attendu manquant : %v", err)
			}
			if diff := cmp.Diff(string(want), Redact(string(raw))); diff != "" {
				t.Errorf("Redact (-attendu +obtenu) :\n%s", diff)
			}
		})
	}
	if count < 15 {
		t.Fatalf("le corpus doit contenir au moins 15 cas, %d trouvés", count)
	}
}

func TestRedactArgs(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"option -e séparée", []string{"run", "-e", "DB_PASSWORD=x", "alpine"}, []string{"run", "-e", "DB_PASSWORD=" + Mask, "alpine"}},
		{"option --env=", []string{"run", "--env=API_TOKEN=x", "nginx"}, []string{"run", "--env=API_TOKEN=" + Mask, "nginx"}},
		{"login -p", []string{"login", "-u", "bob", "-p", "secret"}, []string{"login", "-u", "bob", "-p", Mask}},
		{"login --password=", []string{"login", "--password=secret"}, []string{"login", "--password=" + Mask}},
		{"variable non sensible", []string{"run", "-e", "TZ=UTC", "alpine"}, []string{"run", "-e", "TZ=UTC", "alpine"}},
		{"-p de publication de port conservé", []string{"run", "-p", "8080:80", "nginx"}, []string{"run", "-p", "8080:80", "nginx"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, RedactArgs(tt.in)); diff != "" {
				t.Errorf("RedactArgs (-attendu +obtenu) :\n%s", diff)
			}
		})
	}
}

func FuzzRedact(f *testing.F) {
	for _, seed := range []string{"", "A_PASSWORD=x", "postgres://u:p@h", "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", "KEY: \"v\""} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		once := Redact(in)
		if twice := Redact(once); twice != once {
			t.Fatalf("Redact n'est pas idempotent :\n1: %q\n2: %q", once, twice)
		}
	})
}
