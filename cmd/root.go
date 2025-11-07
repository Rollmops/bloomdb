package cmd

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"bloomdb/logger"

	"github.com/spf13/cobra"
)

var (
	dbConnStr           string
	migrationPath       string
	baselineVersion     string
	logLevel            string
	versionTableName    string
	postMigrationScript string
	globalSetup         *DatabaseSetup
	globalSetupMu       sync.RWMutex
)

var rootCmd = &cobra.Command{
	Use:   "bloomdb",
	Short: "BloomDB CLI tool",
	Long:  "A CLI tool for database migration management",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize logger with specified log level
		logger.Init(logger.LogLevel(logLevel))

		if dbConnStr == "" {
			dbConnStr = os.Getenv("BLOOMDB_CONNECT_STRING")
		}

		if dbConnStr == "" {
			logger.Fatal("connection string is required (use --conn or BLOOMDB_CONNECT_STRING env var)")
		}

		// Handle baseline version: environment -> default
		if envVersion := os.Getenv("BLOOMDB_BASELINE_VERSION"); envVersion != "" {
			baselineVersion = envVersion
		}

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
		logger.Debug("Closing previous global database connection")
		globalSetup.Database.Close()
	}

	globalSetup = setup
	logger.Debug("Global database setup registered for cleanup")
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
		logger.Infof("Received signal %v, shutting down gracefully...", sig)
		cleanupGlobalDatabase()
		os.Exit(0)
	}()

	// Also set up cleanup on normal exit
	defer cleanupGlobalDatabase()
}

// cleanupGlobalDatabase closes the global database connection if it exists
func cleanupGlobalDatabase() {
	globalSetupMu.Lock()
	defer globalSetupMu.Unlock()

	if globalSetup != nil && globalSetup.Database != nil {
		logger.Debug("Cleaning up global database connection")
		globalSetup.Database.Close()
		globalSetup.Database = nil
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

	// Add subcommands
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(repairCmd)
	rootCmd.AddCommand(baselineCmd)
	rootCmd.AddCommand(destroyCmd)
}
