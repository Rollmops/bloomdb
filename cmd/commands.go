package cmd

import (
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	Long:  "Apply all pending database migrations",
	Run: func(cmd *cobra.Command, args []string) {
		migrate := &MigrateCommand{}
		migrate.Run()
	},
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show migration information",
	Long:  "Display current migration status and information",
	Run: func(cmd *cobra.Command, args []string) {
		info := &InfoCommand{}
		info.Run()
	},
}

var repairCmd = &cobra.Command{
	Use:   "repair",
	Short: "Repair migration state",
	Long:  "Fix inconsistent migration state",
	Run: func(cmd *cobra.Command, args []string) {
		repair := &RepairCommand{}
		repair.Run()
	},
}

var baselineCmd = &cobra.Command{
	Use:   "baseline",
	Short: "Baseline database",
	Long:  "Mark all migrations as applied without running them",
	Run: func(cmd *cobra.Command, args []string) {
		baseline := &BaselineCommand{}
		baseline.Run()
	},
}

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy all database objects",
	Long:  "Remove all database objects (tables, views, indexes, etc.) - DANGEROUS OPERATION",
	Run: func(cmd *cobra.Command, args []string) {
		destroy := &DestroyCommand{}
		destroy.Run()
	},
}
