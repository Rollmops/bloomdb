package cmd

import (
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	dbConnStr           string
	migrationPath       string
	baselineVersion     string
	logLevel            string
	versionTableName    string
	postMigrationScript string
	verbose             bool
	globalSetup         *DatabaseSetup
	globalSetupMu       sync.RWMutex
)

var rootCmd = &cobra.Command{
	Use:   "bloomdb",
	Short: "BloomDB CLI tool",
	Long:  "A CLI tool for database migration management",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Set verbose flag in environment for printer
		if verbose {
			os.Setenv("BLOOMDB_VERBOSE", "true")
		}

		// Initialize printer based on BLOOMDB_PRINTER env var (json or human)
		InitPrinter()

		if dbConnStr == "" {
			dbConnStr = os.Getenv("BLOOMDB_CONNECT_STRING")
		}

		if dbConnStr == "" {
			// Try to get connection string from command
			if connectCmd := os.Getenv("BLOOMDB_CONNECT_STRING_CMD"); connectCmd != "" {
				output, err := exec.Command("sh", "-c", connectCmd).Output()
				if err != nil {
					PrintError("Failed to execute connection string command: " + err.Error())
					os.Exit(1)
				}
				dbConnStr = strings.TrimSpace(string(output))
				if dbConnStr == "" {
					PrintError("Connection string command returned empty output")
					os.Exit(1)
				}
			}
		}

		if dbConnStr == "" {
			PrintError("connection string is required (use --conn, BLOOMDB_CONNECT_STRING, or BLOOMDB_CONNECT_STRING_CMD env var)")
			os.Exit(1)
		}

		// Note: Baseline version resolution is now deferred until needed
		// to allow checking for existing baseline records first.
		// Use ResolveBaselineVersion() in commands that need baseline version.

		// Handle migration path: flag -> environment -> default
		// Note: Cobra sets the default, so we need to check if it was explicitly set
		if !cmd.Flags().Changed("path") {
			if envPath := os.Getenv("BLOOMDB_PATH"); envPath != "" {
				migrationPath = envPath
			}
		}

		// Handle version table name: flag -> environment -> default
		// Note: Cobra sets the default, so we need to check if it was explicitly set
		if !cmd.Flags().Changed("table-name") {
			if envTableName := os.Getenv("BLOOMDB_VERSION_TABLE_NAME"); envTableName != "" {
				versionTableName = envTableName
			}
		}

		// Handle post-migration script: flag -> environment -> default
		// Note: Cobra sets the default, so we need to check if it was explicitly set
		if !cmd.Flags().Changed("post-migration-script") {
			if envPostScript := os.Getenv("BLOOMDB_POST_MIGRATION_SCRIPT"); envPostScript != "" {
				postMigrationScript = envPostScript
			}
		}

		// Setup global database cleanup on program exit
		setupGlobalCleanup()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

// SetGlobalDatabaseSetup sets the global database setup for cleanup tracking
func SetGlobalDatabaseSetup(setup *DatabaseSetup) {
	globalSetupMu.Lock()
	defer globalSetupMu.Unlock()

	// Close any existing database setup
	if globalSetup != nil && globalSetup.Database != nil {
		globalSetup.Database.Close()
	}

	globalSetup = setup
}

// GetGlobalDatabaseSetup returns the current global database setup
func GetGlobalDatabaseSetup() *DatabaseSetup {
	globalSetupMu.RLock()
	defer globalSetupMu.RUnlock()
	return globalSetup
}

// setupGlobalCleanup sets up signal handlers for graceful shutdown
func setupGlobalCleanup() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-c
		PrintInfo("Received signal %v, shutting down gracefully...", sig)
		cleanupGlobalDatabase()
		os.Exit(0)
	}()

	// Register cleanup on exit
	defer cleanupGlobalDatabase()
}

// cleanupGlobalDatabase cleans up the global database connection
func cleanupGlobalDatabase() {
	globalSetupMu.Lock()
	defer globalSetupMu.Unlock()

	if globalSetup != nil && globalSetup.Database != nil {
		globalSetup.Database.Close()
		globalSetup = nil
	}
}

// GetBaselineVersion returns the resolved baseline version (flag -> env -> default)
func GetBaselineVersion() string {
	return baselineVersion
}

// GetMigrationPath returns the migration path
func GetMigrationPath() string {
	return migrationPath
}

// GetVersionTableName returns the version table name
func GetVersionTableName() string {
	return versionTableName
}

// GetPostMigrationScript returns the post-migration script path
func GetPostMigrationScript() string {
	return postMigrationScript
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbConnStr, "conn", "", "Database connection string (env: BLOOMDB_CONNECT_STRING)")
	rootCmd.PersistentFlags().StringVar(&migrationPath, "path", ".", "Directory containing migration files (env: BLOOMDB_PATH)")

	rootCmd.PersistentFlags().StringVar(&versionTableName, "table-name", "BLOOMDB_VERSION", "Version table name (env: BLOOMDB_VERSION_TABLE_NAME)")
	rootCmd.PersistentFlags().StringVar(&postMigrationScript, "post-migration-script", "", "Path to post-migration SQL script (env: BLOOMDB_POST_MIGRATION_SCRIPT)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "warn", "Log level (debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output (env: BLOOMDB_VERBOSE)")

	// Add subcommands
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(repairCmd)
	rootCmd.AddCommand(baselineCmd)
	rootCmd.AddCommand(destroyCmd)
}
