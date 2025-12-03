package cmd

import (
	"bloomdb/loader"
)

type BaselineCommand struct{}

func (b *BaselineCommand) Run() {
	// Initialize printer first to ensure verbose output works
	InitPrinter()

	// Detect migration directories (root or subdirectories)
	migrationDirs, err := loader.DetectMigrationDirectories(migrationPath)
	if err != nil {
		PrintError("Error detecting migration directories: %v", err)
		return
	}

	// Process each migration directory
	for _, migDir := range migrationDirs {
		if migDir.IsSubdirectory {
			PrintInfo("Processing subdirectory: %s (table: %s)", migDir.Name, migDir.VersionTable)
		} else {
			PrintInfo("Processing migration directory: %s", migDir.Path)
		}

		// Process baseline for this directory
		err := b.processBaselineDirectory(migDir)
		if err != nil {
			PrintError("Error processing baseline for directory %s: %v", migDir.Path, err)
			return
		}
	}

	PrintSuccess("All migration directories baselined successfully")
}

func (b *BaselineCommand) processBaselineDirectory(migDir loader.MigrationDirectory) error {
	// Setup database connection with appropriate table name
	var setup *DatabaseSetup
	if migDir.VersionTable != "" {
		setup = SetupDatabaseWithTableName(migDir.VersionTable)
	} else {
		setup = SetupDatabase()
	}

	// Resolve baseline version with correct priority:
	// 1. Existing baseline in DB, 2. CLI flag, 3. Env var, 4. Default
	version := ResolveBaselineVersion(setup, baselineVersion)

	// Check if migration table exists
	tableExists, err := setup.Database.TableExists(setup.TableName)
	if err != nil {
		PrintError("Error checking table existence: " + err.Error())
		return err
	}

	if tableExists {
		// Table exists, check if baseline record already exists
		baselineExists, existingBaselineVersion, err := setup.CheckBaselineRecordExists()
		if err != nil {
			PrintError("Error checking baseline record: " + err.Error())
			return err
		}

		if baselineExists {
			// Baseline already exists - the resolved version IS the existing one
			// (ResolveBaselineVersion already returned the existing version)
			PrintSuccess("Baseline already exists with version " + existingBaselineVersion)
			return nil
		}

		PrintInfo("Migration table '" + setup.TableName + "' exists but no baseline record found")
	} else {
		// Table doesn't exist, create it
		PrintInfo("Migration table '" + setup.TableName + "' does not exist, creating it")
		err := setup.CreateMigrationTable()
		if err != nil {
			PrintError("Error creating migration table: " + err.Error())
			return err
		}
	}

	// Insert baseline record
	err = setup.InsertBaselineRecord(version)
	if err != nil {
		PrintError("Error inserting baseline record: " + err.Error())
		return err
	}

	PrintSuccess("Baseline completed successfully for directory: %s", migDir.Path)
	return nil
}
