package diagnose

// originFor détermine l'origine de l'erreur. Cette version fondamentale se contente du
// diagnostic par motif ; la logique des codes de sortie est dans exitcode.go.
func originFor(in Input, d Diagnosis) Origin {
	if o, ok := originFromExitCode(in); ok {
		return o
	}
	if d.Confidence == Recognized {
		return OriginDocker
	}
	return OriginUnknown
}

// enrich ajoute des captures calculées (suggestions, étape de build…).
func enrich(in Input, d *Diagnosis) {
	for _, f := range enrichers {
		f(in, d)
	}
}

// enrichers est complété par les autres fichiers du paquet.
var enrichers []func(Input, *Diagnosis)
