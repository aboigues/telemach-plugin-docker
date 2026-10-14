package main

import "github.com/spf13/cobra"

// deepenCommand approfondit le dernier diagnostic (complétée en US3).
func (a *app) deepenCommand() *cobra.Command {
	return &cobra.Command{Use: "deepen", Short: "Approfondit le dernier diagnostic", Hidden: true}
}
