package diagnose

// originFor détermine l'origine de l'erreur (la logique des codes de sortie est dans
// exitcode.go).
func originFor(in Input, d Diagnosis) Origin {
	// Un message de Docker reconnu (hors erreurs d'exécution) l'emporte sur la convention
	// des codes de sortie : « docker exec » renvoie 1 quand le conteneur est introuvable.
	if d.Confidence == Recognized && d.Entry != nil && d.Entry.Category != "runtime" {
		return OriginDocker
	}
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
