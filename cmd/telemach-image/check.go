package main

// checkPreflight contrôle le dossier visé par « check » avant de lancer le plugin
// (implémenté en US3). handled vaut vrai quand la commande a été traitée ici.
func (a *app) checkPreflight(_ []string) (code int, handled bool) {
	return 0, false
}
