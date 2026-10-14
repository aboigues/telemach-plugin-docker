package textdist

import "testing"

func TestDistance(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"nginx", "nginx", 0},
		{"ngnix", "nginx", 2},
		{"port", "ports", 1},
		{"service", "services", 1},
		{"", "abc", 3},
		{"créer", "creer", 1},
	}
	for _, tt := range tests {
		if got := Distance(tt.a, tt.b); got != tt.want {
			t.Errorf("Distance(%q, %q) = %d, attendu %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestClosest(t *testing.T) {
	images := []string{"nginx", "alpine", "node", "redis"}
	if got, ok := Closest("ngnix", images, 2); !ok || got != "nginx" {
		t.Errorf("Closest(ngnix) = %q, %v ; attendu nginx", got, ok)
	}
	if _, ok := Closest("postgresql-custom", images, 2); ok {
		t.Error("aucune suggestion attendue pour un mot éloigné")
	}
	if _, ok := Closest("nginx", images, 2); ok {
		t.Error("aucune suggestion attendue pour un mot déjà correct")
	}
}
