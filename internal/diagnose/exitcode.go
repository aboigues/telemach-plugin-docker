package diagnose

// originFromExitCode applique la convention des codes de sortie de docker run (research.md
// R3) : 125 = erreur de Docker, 126/127 = commande du conteneur non exécutable ou
// introuvable, 137/143 = conteneur arrêté par un signal, autre code = code de sortie du
// programme du conteneur. Pour les autres sous-commandes, le code n'indique pas l'origine.
func originFromExitCode(in Input) (Origin, bool) {
	switch in.Command.Subcommand {
	case "run", "start", "exec":
	default:
		return "", false
	}
	switch in.ExitCode {
	case 125:
		return OriginDocker, true
	case 126, 127, 137, 143:
		// La commande du conteneur n'a pas pu s'exécuter ou a été arrêtée : c'est le
		// conteneur qui est en cause, pas le démon Docker.
		return OriginApplication, true
	case 0:
		return "", false
	default:
		return OriginApplication, true
	}
}
