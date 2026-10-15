package main

import "fmt"

// install copie le plugin dans le dossier monté de l'hôte (implémenté en US2).
func (a *app) install(_ []string) int {
	fmt.Fprintln(a.stderr, "telemach : install n'est pas encore disponible")
	return 2
}
