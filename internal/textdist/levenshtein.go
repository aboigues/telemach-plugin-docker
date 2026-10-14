package textdist

// Distance renvoie la distance de Levenshtein entre a et b, calculée sur les runes.
func Distance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 {
		return len(rb)
	}
	if len(rb) == 0 {
		return len(ra)
	}
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}

// Closest renvoie le candidat le plus proche de word si sa distance est comprise entre
// 1 et maxDist. En cas d'égalité, le premier candidat de la liste l'emporte. Un candidat
// identique à word n'est jamais proposé : il n'y a rien à corriger.
func Closest(word string, candidates []string, maxDist int) (string, bool) {
	best, bestDist := "", maxDist+1
	for _, c := range candidates {
		d := Distance(word, c)
		if d == 0 {
			return "", false
		}
		if d < bestDist {
			best, bestDist = c, d
		}
	}
	if best == "" {
		return "", false
	}
	return best, true
}
