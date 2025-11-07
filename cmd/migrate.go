package cmd

import (
	"bloomdb/db"
	"bloomdb/loader"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

type MigrateCommand struct{}

// PostMigrationData holds data for post-migration template processing
type PostMigrationData struct {
	CreatedObjects []db.DatabaseObject
	DeletedObjects []db.DatabaseObject
	MigrationPath  string
	DatabaseType   db.DatabaseType
	TableName      string
}

func (m *MigrateCommand) Run() {
	// Setup database connection
	setup := SetupDatabase()

	// Ensure migration table and baseline record exist
	setup.EnsureTableAndBaselineExist()

	// Get initial database state for tracking deleted objects
	initialObjects, err := setup.Database.GetDatabaseObjects()
	if err != nil {
		initialObjects = []db.DatabaseObject{} // Use empty slice as fallback
	}

	// Load migrations from filesystem
	versionedLoader := loader.NewVersionedMigrationLoader(migrationPath)
	versionedMigrations, err := versionedLoader.LoadMigrations()
	if err != nil {
		PrintError("Error loading versioned migrations: %v", err)
		return
	}

	repeatableLoader := loader.NewRepeatableMigrationLoader(migrationPath)
	repeatableMigrations, err := repeatableLoader.LoadRepeatableMigrations()
	if err != nil {
		PrintError("Error loading repeatable migrations: %v", err)
		return
	}

	// Read existing migration records
	records, err := setup.GetMigrationRecords()
	if err != nil {
		PrintError("Error reading migration records: %v", err)
		return
	}

	// Check for failed migrations (success = 0)
	for _, record := range records {
		if record.Success == 0 {
			PrintError("Found failed migration: %s (version: %s)", record.Description, func() string {
				if record.Version != nil {
					return *record.Version
				}
				return "repeatable"
			}())
			PrintWarning("Please run the repair command to fix failed migrations before continuing.")
			return
		}
	}

	// Validate checksums of applied migrations
	checksumErrors := validateMigrationChecksums(versionedMigrations, repeatableMigrations, records)
	if len(checksumErrors) > 0 {
		PrintError("Checksum validation failed for %d migration(s):", len(checksumErrors))
		for _, errMsg := range checksumErrors {
			PrintError("  - %s", errMsg)
		}
		PrintWarning("Migration files have been modified after being applied.")
		PrintWarning("Please run the repair command to update checksums, or restore the original files.")
		return
	}

	// Find the greatest version in the database
	greatestVersion := findGreatestVersion(records)

	// Find pending versioned migrations (versions greater than greatest version)
	pendingMigrations := findPendingMigrations(versionedMigrations, greatestVersion)

	if len(pendingMigrations) == 0 {
		PrintInfo("No pending versioned migrations to execute")
	} else {
		PrintSuccess("Found %d pending versioned migrations", len(pendingMigrations))
		for _, migration := range pendingMigrations {
			PrintMigration(migration.Version, migration.Description, "pending")
		}

		// Execute pending versioned migrations
		for i, migration := range pendingMigrations {
			PrintCommand(fmt.Sprintf("Executing migration %d/%d: %s", i+1, len(pendingMigrations), migration))
			executionTime, err := executeVersionedMigration(setup, migration)
			if err != nil {
				PrintError("Migration %s failed: %v", migration, err)
				PrintError("Migration process stopped due to failure at step %d/%d", i+1, len(pendingMigrations))
				return
			}
			PrintSuccess("Successfully executed migration: %s (%dms)", migration, executionTime)
		}
	}

	// Handle repeatable migrations
	if len(repeatableMigrations) > 0 {
		// Get updated migration records after versioned migrations
		updatedRecords, err := setup.GetMigrationRecords()
		if err != nil {
			PrintError("Error reading updated migration records: %v", err)
			return
		}

		// Find repeatable migrations that need to be executed
		pendingRepeatable := findPendingRepeatableMigrations(repeatableMigrations, updatedRecords)

		if len(pendingRepeatable) == 0 {
			PrintInfo("No repeatable migrations need to be executed")
		} else {
			PrintSuccess("Found %d repeatable migrations to execute", len(pendingRepeatable))
			for _, migration := range pendingRepeatable {
				PrintMigration("", migration.Description, "pending")
			}

			// Execute pending repeatable migrations
			for i, migration := range pendingRepeatable {
				PrintCommand(fmt.Sprintf("Executing repeatable migration %d/%d: %s", i+1, len(pendingRepeatable), migration.Description))
				executionTime, err := executeRepeatableMigration(setup, migration)
				if err != nil {
					PrintError("Repeatable migration %s failed: %v", migration.Description, err)
					PrintError("Migration process stopped due to failure at step %d/%d", i+1, len(pendingRepeatable))
					return
				}
				PrintSuccess("Successfully executed repeatable migration: %s (%dms)", migration.Description, executionTime)
			}
		}
	}

	PrintSuccess("Migration process completed successfully")

	// Execute post-migration script if it exists
	if err := executePostMigrationScript(setup, migrationPath, postMigrationScript, initialObjects); err != nil {
		PrintWarning("Post-migration script failed: %v", err)
	}
}

// findGreatestVersion finds the greatest version among existing migration records
func findGreatestVersion(records []db.MigrationRecord) string {
	greatest := ""
	for _, record := range records {
		if record.Version == nil || *record.Version == "" {
			continue // Skip records without version (repeatable migrations)
		}
		// Include baseline records to establish starting version
		if record.Type == "baseline" {
			if greatest == "" || loader.CompareVersions(*record.Version, greatest) > 0 {
				greatest = *record.Version
			}
		} else if greatest == "" || loader.CompareVersions(*record.Version, greatest) > 0 {
			greatest = *record.Version
		}
	}
	return greatest
}

// findPendingMigrations returns versioned migrations with versions greater than the greatest version
func findPendingMigrations(migrations []*loader.VersionedMigration, greatestVersion string) []*loader.VersionedMigration {
	var pending []*loader.VersionedMigration
	for _, migration := range migrations {
		if greatestVersion == "" || loader.CompareVersions(migration.Version, greatestVersion) > 0 {
			pending = append(pending, migration)
		}
	}
	return pending
}

// executeVersionedMigration executes a versioned migration and records it
func executeVersionedMigration(setup *DatabaseSetup, migration *loader.VersionedMigration) (int, error) {
	// Get current migration records to find the next installed rank
	records, err := setup.GetMigrationRecords()
	if err != nil {
		return 0, fmt.Errorf("error reading migration records for rank calculation: %w", err)
	}

	// Find the maximum installed rank and increment by 1
	maxRank := 0
	for _, record := range records {
		if record.InstalledRank > maxRank {
			maxRank = record.InstalledRank
		}
	}
	nextRank := maxRank + 1

	record := db.MigrationRecord{
		InstalledRank: nextRank,
		Version:       &migration.Version,
		Description:   migration.Description,
		Type:          "versioned",
		Script:        migration.String(),
		Checksum:      &migration.Checksum,
		InstalledBy:   "bloomdb",
	}

	return executeMigrationCommon(setup, migration.Content, migration.Description, record)
}

// findCreatedObjects compares before and after object lists to find newly created objects
func findCreatedObjects(before, after []db.DatabaseObject) []db.DatabaseObject {
	// Create a map of before objects for quick lookup
	beforeMap := make(map[string]bool)
	for _, obj := range before {
		key := obj.Type + ":" + obj.Name
		beforeMap[key] = true
	}

	var created []db.DatabaseObject
	for _, obj := range after {
		key := obj.Type + ":" + obj.Name
		if !beforeMap[key] {
			created = append(created, obj)
		}
	}

	return created
}

// findDeletedObjects compares before and after object lists to find deleted objects
func findDeletedObjects(before, after []db.DatabaseObject) []db.DatabaseObject {
	// Create a map of after objects for quick lookup
	afterMap := make(map[string]bool)
	for _, obj := range after {
		key := obj.Type + ":" + obj.Name
		afterMap[key] = true
	}

	var deleted []db.DatabaseObject
	for _, obj := range before {
		key := obj.Type + ":" + obj.Name
		if !afterMap[key] {
			deleted = append(deleted, obj)
		}
	}

	return deleted
}

// executeMigrationCommon contains the shared logic for executing migrations
func executeMigrationCommon(setup *DatabaseSetup, content, description string, record db.MigrationRecord) (int, error) {
	var (
		beforeObjects  []db.DatabaseObject
		afterObjects   []db.DatabaseObject
		createdObjects []db.DatabaseObject
		deletedObjects []db.DatabaseObject
		startTime      time.Time
		executionTime  int64
		state          string
		successFlag    int
		err            error
		existingRecord *db.MigrationRecord
	)

	// Check if this is a repeatable migration that already exists
	if record.Type == "repeatable" {
		records, err := setup.GetMigrationRecords()
		if err != nil {
			return 0, fmt.Errorf("error reading migration records: %w", err)
		}

		for _, r := range records {
			if r.Type == "repeatable" && r.Description == record.Description {
				existingRecord = &r
				break
			}
		}
	}

	// Get database objects before migration
	beforeObjects, err = setup.Database.GetDatabaseObjects()

	// Measure execution time
	startTime = time.Now()

	// Execute migration SQL
	err = setup.ExecuteMigration(content)

	// Calculate execution time in milliseconds
	executionTime = time.Since(startTime).Milliseconds()

	// Get database objects after migration
	afterObjects, _ = setup.Database.GetDatabaseObjects()

	// Find and print newly created and deleted objects
	if beforeObjects != nil && afterObjects != nil {
		createdObjects = findCreatedObjects(beforeObjects, afterObjects)
		deletedObjects = findDeletedObjects(beforeObjects, afterObjects)

		if len(createdObjects) > 0 {
			PrintInfo("Created objects:")
			for _, obj := range createdObjects {
				PrintObject(obj.Type, obj.Name)
			}
		}

		if len(deletedObjects) > 0 {
			PrintWarning("Deleted objects:")
			for _, obj := range deletedObjects {
				PrintObject(obj.Type, obj.Name)
			}
		}
	}

	// Determine the state based on execution result
	state = "success"
	if err != nil {
		state = "failed"
	}

	// Always record the migration (whether success or failed)
	successFlag = 0
	if state == "success" {
		successFlag = 1
	}

	// Update the record with execution details
	record.ExecutionTime = int(executionTime)
	record.Success = successFlag

	var recordErr error
	if existingRecord != nil {
		// Update existing repeatable migration record
		recordErr = setup.UpdateMigrationRecordFull(record)
	} else {
		// Insert new migration record
		recordErr = setup.InsertMigrationRecord(record)
	}

	if recordErr != nil {
		// If we can't record the migration, that's a critical error
		if err != nil {
			return int(executionTime), fmt.Errorf("migration failed AND failed to record: %v (recording error: %v)", err, recordErr)
		}
		return int(executionTime), fmt.Errorf("failed to record migration: %w", recordErr)
	}

	// Return the original execution error if it failed
	if err != nil {
		return int(executionTime), fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	return int(executionTime), nil
}

// executePostMigrationScript looks for and executes a post-migration SQL script with Go templating
func executePostMigrationScript(setup *DatabaseSetup, migrationPath string, customScriptPath string, initialObjects []db.DatabaseObject) error {
	var postScriptPath string

	// If custom script path is provided, use it
	if customScriptPath != "" {
		// Check if it's an absolute path or relative to migration path
		if filepath.IsAbs(customScriptPath) {
			postScriptPath = customScriptPath
		} else {
			postScriptPath = filepath.Join(migrationPath, customScriptPath)
		}

		// Verify the custom script exists
		if _, err := os.Stat(postScriptPath); err != nil {
			return fmt.Errorf("custom post-migration script not found: %s", postScriptPath)
		}
	}

	if postScriptPath == "" {
		return nil
	}

	PrintInfo("Found post-migration script: %s", filepath.Base(postScriptPath))

	// Read the script content
	content, err := os.ReadFile(postScriptPath)
	if err != nil {
		return fmt.Errorf("failed to read post-migration script: %w", err)
	}

	// Get current database objects for template
	currentObjects, err := setup.Database.GetDatabaseObjects()
	if err != nil {
		return fmt.Errorf("failed to get current database objects: %w", err)
	}

	// Filter out migration table from created objects
	var createdObjects []db.DatabaseObject
	for _, obj := range currentObjects {
		if obj.Name != setup.TableName {
			createdObjects = append(createdObjects, obj)
		}
	}

	// Calculate deleted objects by comparing initial and current states
	deletedObjects := findDeletedObjects(initialObjects, currentObjects)

	// Prepare template data
	templateData := PostMigrationData{
		CreatedObjects: createdObjects,
		DeletedObjects: deletedObjects,
		MigrationPath:  migrationPath,
		DatabaseType:   setup.DBType,
		TableName:      setup.TableName,
	}

	// Parse and execute the template
	tmpl, err := template.New("postMigration").Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse post-migration template: %w", err)
	}

	var renderedScript bytes.Buffer
	if err := tmpl.Execute(&renderedScript, templateData); err != nil {
		return fmt.Errorf("failed to execute post-migration template: %w", err)
	}

	// Execute the rendered SQL
	renderedSQL := renderedScript.String()
	if strings.TrimSpace(renderedSQL) == "" {
		PrintInfo("Post-migration script rendered to empty SQL, skipping execution")
		return nil
	}

	PrintInfo("Executing post-migration script (%d characters)", len(renderedSQL))
	PrintCommand("🔧 Executing post-migration script...")

	err = setup.ExecuteMigration(renderedSQL)
	if err != nil {
		return fmt.Errorf("failed to execute post-migration SQL: %w", err)
	}

	PrintSuccess("Post-migration script executed successfully")

	return nil
}

// findPendingRepeatableMigrations returns repeatable migrations that need to be executed
// A repeatable migration needs to be executed if:
// 1. No record exists for it in the migration table
// 2. A record exists but the hash is different (content has changed)
func findPendingRepeatableMigrations(migrations []*loader.RepeatableMigration, records []db.MigrationRecord) []*loader.RepeatableMigration {
	var pending []*loader.RepeatableMigration

	// Create a map of existing repeatable migration records by description
	existingRecords := make(map[string]db.MigrationRecord)
	for _, record := range records {
		// Repeatable migrations have empty version (they're not versioned)
		if (record.Version == nil || *record.Version == "") && record.Type != "baseline" {
			existingRecords[record.Description] = record
		}
	}

	for _, migration := range migrations {
		existingRecord, exists := existingRecords[migration.Description]

		// Execute if no record exists or checksum has changed
		if !exists || (existingRecord.Checksum == nil || *existingRecord.Checksum != migration.Checksum) {
			pending = append(pending, migration)
		}
	}

	return pending
}

// executeRepeatableMigration executes a repeatable migration and records it
func executeRepeatableMigration(setup *DatabaseSetup, migration *loader.RepeatableMigration) (int, error) {
	// Get current migration records to find the next installed rank
	records, err := setup.GetMigrationRecords()
	if err != nil {
		return 0, fmt.Errorf("error reading migration records for rank calculation: %w", err)
	}

	// Find the maximum installed rank and increment by 1
	maxRank := 0
	for _, record := range records {
		if record.InstalledRank > maxRank {
			maxRank = record.InstalledRank
		}
	}
	nextRank := maxRank + 1

	record := db.MigrationRecord{
		InstalledRank: nextRank,
		Version:       nil, // Empty version for repeatable migrations
		Description:   migration.Description,
		Type:          "repeatable",
		Script:        migration.String(),
		Checksum:      &migration.Checksum,
		InstalledBy:   "bloomdb",
	}

	return executeMigrationCommon(setup, migration.Content, migration.Description, record)
}

// validateMigrationChecksums checks if any applied migrations have been modified
// Returns a slice of error messages for any checksum mismatches found
func validateMigrationChecksums(versionedMigrations []*loader.VersionedMigration, repeatableMigrations []*loader.RepeatableMigration, records []db.MigrationRecord) []string {
	var errors []string

	// Create maps for quick lookup
	versionedMap := make(map[string]*loader.VersionedMigration)
	for _, migration := range versionedMigrations {
		versionedMap[migration.Version] = migration
	}

	repeatableMap := make(map[string]*loader.RepeatableMigration)
	for _, migration := range repeatableMigrations {
		repeatableMap[migration.Description] = migration
	}

	// Check each applied migration record
	for _, record := range records {
		// Skip baseline records (they have NULL checksum)
		if record.Type == "baseline" || record.Checksum == nil {
			continue
		}

		// Skip failed migrations (they will be caught by the failed migration check)
		if record.Success == 0 {
			continue
		}

		// Check versioned migrations
		if record.Version != nil && *record.Version != "" {
			version := *record.Version
			if migration, exists := versionedMap[version]; exists {
				if *record.Checksum != migration.Checksum {
					errors = append(errors, fmt.Sprintf("V%s - %s (expected: %d, found: %d)", version, record.Description, *record.Checksum, migration.Checksum))
				}
			}
			// Note: If migration file is missing, it won't be in the map, but that's a different issue
			// The info command will show it as "missing"
		} else {
			// Check repeatable migrations
			if migration, exists := repeatableMap[record.Description]; exists {
				if *record.Checksum != migration.Checksum {
					errors = append(errors, fmt.Sprintf("R - %s (expected: %d, found: %d)", record.Description, *record.Checksum, migration.Checksum))
				}
			}
		}
	}

	return errors
}
