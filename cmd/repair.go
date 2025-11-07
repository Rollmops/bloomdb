package cmd

import (
	"bloomdb/loader"
	"fmt"
)

type RepairCommand struct{}

func (r *RepairCommand) Run() {
	setup := SetupDatabase()

	// Ensure migration table and baseline record exist (repair should only work on initialized databases)
	setup.EnsureTableAndBaselineExist()

	// Step 1: Remove all records from the version table that are not successful
	PrintInfo("Step 1: Removing failed migration records...")
	err := setup.DeleteFailedMigrationRecords()
	if err != nil {
		PrintError("Error removing failed migration records: %v", err)
		return
	}
	PrintSuccess("Failed migration records removed")

	// Step 2: Align checksums and descriptions of versioned migration files to existing entries
	PrintInfo("Step 2: Aligning checksums and descriptions...")
	err = alignMigrationChecksumsAndDescriptions(setup)
	if err != nil {
		PrintError("Error aligning migration checksums and descriptions: %v", err)
		return
	}
	PrintSuccess("Migration checksums and descriptions aligned")

	PrintSuccess("Repair completed successfully")
}

// alignMigrationChecksumsAndDescriptions updates migration records to match current files
func alignMigrationChecksumsAndDescriptions(setup *DatabaseSetup) error {
	// Get migration path
	migrationPath := GetMigrationPath()

	// Load versioned migrations from filesystem
	versionedLoader := loader.NewVersionedMigrationLoader(migrationPath)
	versionedMigrations, err := versionedLoader.LoadMigrations()
	if err != nil {
		return fmt.Errorf("error loading versioned migrations: %w", err)
	}

	// Get existing migration records from database
	records, err := setup.GetMigrationRecords()
	if err != nil {
		return fmt.Errorf("error reading migration records: %w", err)
	}

	// Find baseline version
	baselineVersion := FindBaselineVersion(records)

	// Create a map of versioned migrations for quick lookup
	migrationMap := make(map[string]*loader.VersionedMigration)
	for _, migration := range versionedMigrations {
		migrationMap[migration.Version] = migration
	}

	// Track updates
	updatesMade := 0

	// Process each record and align with file
	for _, record := range records {
		// Skip non-versioned records (repeatable migrations, baseline, etc.)
		if record.Version == nil || *record.Version == "" {
			continue
		}

		version := *record.Version

		// Skip baseline record and any records at or below baseline version
		if baselineVersion != "" && loader.CompareVersions(version, baselineVersion) <= 0 {
			continue
		}

		migration, exists := migrationMap[version]

		if !exists {
			// Migration file doesn't exist - this is a more serious issue
			PrintWarning("Migration file not found for version %s (description: %s)", version, record.Description)
			continue
		}

		// Check if description needs updating
		descriptionChanged := record.Description != migration.Description

		// Check if checksum needs updating
		checksumChanged := record.Checksum == nil || *record.Checksum != migration.Checksum

		if descriptionChanged || checksumChanged {
			PrintInfo("Updating migration record for version %s:", version)

			if descriptionChanged {
				PrintInfo("  Description: %s -> %s", record.Description, migration.Description)
			}

			if checksumChanged {
				oldChecksum := "nil"
				if record.Checksum != nil {
					oldChecksum = fmt.Sprintf("%d", *record.Checksum)
				}
				PrintInfo("  Checksum: %s -> %d", oldChecksum, migration.Checksum)
			}

			// Update the record
			err := setup.UpdateMigrationRecord(record.InstalledRank, version, migration.Description, migration.Checksum)
			if err != nil {
				return fmt.Errorf("failed to update migration record for version %s: %w", version, err)
			}

			updatesMade++
		}
	}

	if updatesMade == 0 {
		PrintInfo("No migration records needed updating")
	} else {
		PrintSuccess("Updated %d migration records", updatesMade)
	}

	return nil
}
