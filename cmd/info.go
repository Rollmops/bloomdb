package cmd

import (
	"bloomdb/db"
	"bloomdb/loader"
)

type InfoCommand struct{}

func (i *InfoCommand) Run() {
	// Get migration path from root command
	migrationPath := GetMigrationPath()

	// Setup database connection
	setup := SetupDatabase()

	// Ensure migration table and baseline record exist
	setup.EnsureTableAndBaselineExist()

	// Load migrations from filesystem
	versionedLoader := loader.NewVersionedMigrationLoader(migrationPath)
	versionedMigrations, err := versionedLoader.LoadMigrations()
	if err != nil {
		PrintError("Error loading versioned migrations: " + err.Error())
		return
	}

	repeatableLoader := loader.NewRepeatableMigrationLoader(migrationPath)
	repeatableMigrations, err := repeatableLoader.LoadRepeatableMigrations()
	if err != nil {
		PrintError("Error loading repeatable migrations: " + err.Error())
		return
	}

	// Get existing migration records from database
	existingRecords, err := setup.GetMigrationRecords()
	if err != nil {
		PrintError("Error reading migration records: " + err.Error())
		return
	}

	// Find baseline version
	baselineVersion := FindBaselineVersion(existingRecords)

	// Build migration status list
	statuses := buildMigrationStatuses(versionedMigrations, repeatableMigrations, existingRecords, baselineVersion)

	// Display the table
	DisplayMigrationTable(setup.DBType, setup.TableName, statuses)
}

// buildMigrationStatuses creates a comprehensive list of migration statuses
func buildMigrationStatuses(versionedMigrations []*loader.VersionedMigration, repeatableMigrations []*loader.RepeatableMigration, records []db.MigrationRecord, baselineVersion string) []MigrationStatus {
	var statuses []MigrationStatus

	// Create a map of existing records for quick lookup
	recordMap := make(map[string]db.MigrationRecord)
	for _, record := range records {
		if record.Version != nil && *record.Version != "" {
			recordMap[*record.Version] = record
		} else {
			// For repeatable migrations, use description as key
			recordMap[record.Description] = record
		}
	}

	// Create maps for file lookup and checksum validation
	versionedMigrationMap := make(map[string]*loader.VersionedMigration)
	for _, migration := range versionedMigrations {
		versionedMigrationMap[migration.Version] = migration
	}

	repeatableMigrationMap := make(map[string]*loader.RepeatableMigration)
	for _, migration := range repeatableMigrations {
		repeatableMigrationMap[migration.Description] = migration
	}

	// Process versioned migrations
	for _, migration := range versionedMigrations {
		status := MigrationStatus{
			Version:     migration.Version,
			Description: migration.Description,
			Type:        "versioned",
		}

		// Check if below or equal to baseline first
		if baselineVersion != "" && loader.CompareVersions(migration.Version, baselineVersion) <= 0 {
			status.Status = "below baseline"
		} else if record, exists := recordMap[migration.Version]; exists {
			// Check for checksum mismatch using actual file path
			if validationStatus := validateVersionedMigration(record, migration); validationStatus != "" {
				status.Status = validationStatus
			} else {
				// Only show as "success" if it was actually applied, not just baselined
				if record.Type == "baseline" {
					// If this version was baselined, it means the migration wasn't actually executed
					// So it should be considered as "pending" if there are newer migrations
					if migration.Version == baselineVersion {
						status.Status = "baseline"
					} else {
						status.Status = "pending"
					}
				} else {
					// Convert success flag to status string
					if record.Success == 1 {
						status.Status = "success"
					} else {
						status.Status = "failed"
					}
				}
			}
			status.InstalledOn = record.InstalledOn
		} else {
			status.Status = "pending"
		}

		statuses = append(statuses, status)
	}

	// Process database records that don't have corresponding files (missing files)
	for _, record := range records {
		// Skip non-versioned records (repeatable migrations, baseline, etc.)
		if record.Version == nil || *record.Version == "" {
			continue
		}

		version := *record.Version

		// Skip if already processed (file exists)
		if _, exists := versionedMigrationMap[version]; exists {
			continue
		}

		// Skip baseline and below-baseline records
		if baselineVersion != "" && loader.CompareVersions(version, baselineVersion) <= 0 {
			continue
		}

		// Create status for missing migration
		status := MigrationStatus{
			Version:     version,
			Description: record.Description,
			Type:        "versioned",
			Status:      "missing",
			InstalledOn: record.InstalledOn,
		}

		statuses = append(statuses, status)
	}

	// Process repeatable migrations
	for _, migration := range repeatableMigrations {
		status := MigrationStatus{
			Version:     "",
			Description: migration.Description,
			Type:        "repeatable",
		}

		if record, exists := recordMap[migration.Description]; exists {
			// Check for checksum mismatch using actual file path
			if validationStatus := validateRepeatableMigration(record, migration); validationStatus != "" {
				status.Status = validationStatus
			} else {
				// Convert success flag to status string
				if record.Success == 1 {
					status.Status = "success"
				} else {
					status.Status = "failed"
				}
			}
			status.InstalledOn = record.InstalledOn
		} else {
			status.Status = "pending"
		}

		statuses = append(statuses, status)
	}

	// Sort statuses by version (for versioned migrations) to ensure consistent order
	// Put versioned migrations first, then repeatable
	var versionedStatuses []MigrationStatus
	var repeatableStatuses []MigrationStatus

	for _, status := range statuses {
		if status.Type == "versioned" {
			versionedStatuses = append(versionedStatuses, status)
		} else {
			repeatableStatuses = append(repeatableStatuses, status)
		}
	}

	// Sort versioned migrations by version
	for i := 0; i < len(versionedStatuses)-1; i++ {
		for j := i + 1; j < len(versionedStatuses); j++ {
			if loader.CompareVersions(versionedStatuses[i].Version, versionedStatuses[j].Version) > 0 {
				versionedStatuses[i], versionedStatuses[j] = versionedStatuses[j], versionedStatuses[i]
			}
		}
	}

	// Combine sorted versioned and repeatable statuses
	sortedStatuses := append(versionedStatuses, repeatableStatuses...)

	return sortedStatuses
}

// validateVersionedMigration checks if the versioned migration checksum matches
func validateVersionedMigration(record db.MigrationRecord, migration *loader.VersionedMigration) string {
	// File existence is already guaranteed by the migration being loaded
	// Check if checksum matches
	if record.Checksum != nil && *record.Checksum != migration.Checksum {
		return "checksum"
	}

	return ""
}

// validateRepeatableMigration checks if the repeatable migration checksum matches
func validateRepeatableMigration(record db.MigrationRecord, migration *loader.RepeatableMigration) string {
	// File existence is already guaranteed by the migration being loaded
	// Check if checksum matches
	if record.Checksum != nil && *record.Checksum != migration.Checksum {
		return "checksum"
	}

	return ""
}
