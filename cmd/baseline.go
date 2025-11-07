package cmd

import (
	"fmt"

	"bloomdb/logger"
)

type BaselineCommand struct{}

func (b *BaselineCommand) Run() {
	logger.Info("Starting baseline command")

	// Get resolved baseline version from root command
	version := GetBaselineVersion()
	logger.Infof("Using baseline version: %s", version)

	// Setup database connection
	setup := SetupDatabase()
	logger.Infof("Connected to %s database", setup.DBType)

	// Ensure migration table doesn't exist
	logger.Debug("Ensuring migration table does not exist")
	setup.EnsureTableNotExists()

	// Create migration table
	logger.Info("Creating migration table")
	err := setup.CreateMigrationTable()
	if err != nil {
		logger.Errorf("Error creating migration table: %v", err)
		return
	}

	// Insert baseline record
	logger.Infof("Inserting baseline record for version: %s", version)
	err = setup.InsertBaselineRecord(version)
	if err != nil {
		logger.Errorf("Error inserting baseline record: %v", err)
		return
	}

	logger.Infof("Baseline completed successfully - connected to %s database, version: %s", setup.DBType, version)
	fmt.Printf("baseline - connected to %s database, version: %s\n", setup.DBType, version)
}
