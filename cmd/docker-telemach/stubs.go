package main

import "github.com/spf13/cobra"

// checkCommand vérifie un fichier Compose avant exécution (complétée en US2).
func (a *app) checkCommand() *cobra.Command {
	return &cobra.Command{Use: "check", Short: "Vérifie un fichier Compose avant de le lancer", Hidden: true}
}

// deepenCommand approfondit le dernier diagnostic (complétée en US3).
func (a *app) deepenCommand() *cobra.Command {
	return &cobra.Command{Use: "deepen", Short: "Approfondit le dernier diagnostic", Hidden: true}
}
