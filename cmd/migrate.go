package cmd

import (
	"bloomdb/db"
	"bloomdb/loader"
	"bloomdb/logger"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	logger.Info("Starting migration process")

	// Setup database connection
	setup := SetupDatabase()
	logger.Infof("Connected to %s database", setup.DBType)

	// Ensure migration table exists
	setup.EnsureTableExists()
	logger.Infof("Migration table '%s' ensured to exist", setup.TableName)

	// Get initial database state for tracking deleted objects
	initialObjects, err := setup.Database.GetDatabaseObjects()
	if err != nil {
		logger.Warnf("Failed to get initial database objects: %v", err)
		initialObjects = []db.DatabaseObject{} // Use empty slice as fallback
	}

	// Load migrations from filesystem
	logger.Infof("Loading migrations from path: %s", migrationPath)
	versionedLoader := loader.NewVersionedMigrationLoader(migrationPath)
	versionedMigrations, err := versionedLoader.LoadMigrations()
	if err != nil {
		logger.Errorf("Error loading versioned migrations: %v", err)
		return
	}

	repeatableLoader := loader.NewRepeatableMigrationLoader(migrationPath)
	repeatableMigrations, err := repeatableLoader.LoadRepeatableMigrations()
	if err != nil {
		logger.Errorf("Error loading repeatable migrations: %v", err)
		return
	}

	logger.Infof("Loaded %d versioned migrations and %d repeatable migrations", len(versionedMigrations), len(repeatableMigrations))

	// Get existing migration records from database
	existingRecords, err := setup.GetMigrationRecords()
	if err != nil {
		logger.Errorf("Error reading migration records: %v", err)
		return
	}

	// Find the greatest version in the database
	greatestVersion := findGreatestVersion(existingRecords)
	logger.Infof("Current greatest version in database: %s", greatestVersion)

	// Find pending versioned migrations (versions greater than greatest version)
	pendingMigrations := findPendingMigrations(versionedMigrations, greatestVersion)

	if len(pendingMigrations) == 0 {
		logger.Info("No pending versioned migrations to execute")
	} else {
		logger.Infof("Found %d pending versioned migrations", len(pendingMigrations))
		for _, migration := range pendingMigrations {
			logger.Infof("  - %s", migration)
		}

		// Execute pending versioned migrations
		logger.Debugf("Starting execution of %d pending migrations", len(pendingMigrations))
		for i, migration := range pendingMigrations {
			logger.Infof("Executing migration %d/%d: %s", i+1, len(pendingMigrations), migration)
			executionTime, err := executeVersionedMigration(setup, migration)
			if err != nil {
				logger.Errorf("Migration %s failed: %v", migration, err)
				fmt.Printf("✗ Migration %s failed: %v\n", migration, err)
				fmt.Printf("Migration process stopped due to failure at step %d/%d\n", i+1, len(pendingMigrations))
				return
			}
			fmt.Printf("✓ Successfully executed migration: %s (%dms)\n", migration, executionTime)
		}
		logger.Debug("Finished executing versioned migrations")
	}

	// Handle repeatable migrations
	if len(repeatableMigrations) > 0 {
		// Get updated migration records after versioned migrations
		updatedRecords, err := setup.GetMigrationRecords()
		if err != nil {
			logger.Errorf("Error reading updated migration records: %v", err)
			return
		}

		// Find repeatable migrations that need to be executed
		pendingRepeatable := findPendingRepeatableMigrations(repeatableMigrations, updatedRecords)

		if len(pendingRepeatable) == 0 {
			logger.Info("No repeatable migrations need to be executed")
		} else {
			logger.Infof("Found %d repeatable migrations to execute", len(pendingRepeatable))
			for _, migration := range pendingRepeatable {
				logger.Infof("  - %s", migration.Description)
			}

			// Execute pending repeatable migrations
			for i, migration := range pendingRepeatable {
				logger.Infof("Executing repeatable migration %d/%d: %s", i+1, len(pendingRepeatable), migration.Description)
				executionTime, err := executeRepeatableMigration(setup, migration)
				if err != nil {
					logger.Errorf("Repeatable migration %s failed: %v", migration.Description, err)
					fmt.Printf("✗ Repeatable migration %s failed: %v\n", migration.Description, err)
					fmt.Printf("Migration process stopped due to failure at step %d/%d\n", i+1, len(pendingRepeatable))
					return
				}
				fmt.Printf("✓ Successfully executed repeatable migration: %s (%dms)\n", migration.Description, executionTime)
			}
		}
	}

	logger.Infof("Migration process completed successfully - connected to %s database, table '%s' exists", setup.DBType, setup.TableName)

	// Execute post-migration script if it exists
	if err := executePostMigrationScript(setup, migrationPath, postMigrationScript, initialObjects); err != nil {
		logger.Errorf("Error executing post-migration script: %v", err)
		fmt.Printf("⚠ Post-migration script failed: %v\n", err)
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
	logger.Debugf("Executing versioned migration: %s", migration.Description)

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
	)

	// Get database objects before migration
	beforeObjects, err = setup.Database.GetDatabaseObjects()
	if err != nil {
		logger.Warnf("Failed to get database objects before migration: %v", err)
	}

	// Measure execution time
	startTime = time.Now()

	// Execute the migration SQL
	err = setup.ExecuteMigration(migration.Content)

	// Calculate execution time in milliseconds
	executionTime = time.Since(startTime).Milliseconds()

	// Get database objects after migration
	afterObjects, err = setup.Database.GetDatabaseObjects()
	if err != nil {
		logger.Warnf("Failed to get database objects after migration: %v", err)
	}

	// Find and print newly created and deleted objects
	if beforeObjects != nil && afterObjects != nil {
		createdObjects = findCreatedObjects(beforeObjects, afterObjects)
		deletedObjects = findDeletedObjects(beforeObjects, afterObjects)

		if len(createdObjects) > 0 {
			fmt.Printf("  Created objects:\n")
			for _, obj := range createdObjects {
				fmt.Printf("    - %s: %s\n", obj.Type, obj.Name)
			}
		}

		if len(deletedObjects) > 0 {
			fmt.Printf("  Deleted objects:\n")
			for _, obj := range deletedObjects {
				fmt.Printf("    - %s: %s\n", obj.Type, obj.Name)
			}
		}
	}

	// Determine the state based on execution result
	state = "success"
	if err != nil {
		state = "failed"
		logger.Warnf("Migration %s failed with state: %s", migration.Description, state)
	}

	// Measure execution time
	startTime = time.Now()

	// Execute the migration SQL
	err = setup.ExecuteMigration(migration.Content)

	// Calculate execution time in milliseconds
	executionTime = time.Since(startTime).Milliseconds()

	// Get database objects after migration
	afterObjects, err = setup.Database.GetDatabaseObjects()
	if err != nil {
		logger.Warnf("Failed to get database objects after migration: %v", err)
	}

	// Find and print newly created and deleted objects
	if beforeObjects != nil && afterObjects != nil {
		createdObjects = findCreatedObjects(beforeObjects, afterObjects)
		deletedObjects = findDeletedObjects(beforeObjects, afterObjects)
		if len(createdObjects) > 0 {
			fmt.Printf("  Created objects:\n")
			for _, obj := range createdObjects {
				fmt.Printf("    - %s: %s\n", obj.Type, obj.Name)
			}
		}
	}

	// Determine the state based on execution result
	state = "success"
	if err != nil {
		state = "failed"
		logger.Warnf("Migration %s failed with state: %s", migration.Description, state)
	}

	// Always record the migration (whether success or failed)
	successFlag = 0
	if state == "success" {
		successFlag = 1
	}
	// Calculate installed rank as integer (for simplicity, using version parsing)
	installedRank := 0
	if versionParts := strings.Split(migration.Version, "."); len(versionParts) > 0 {
		if major, err := strconv.Atoi(versionParts[0]); err == nil {
			installedRank = major * 1000 // Simple rank calculation
		}
	}
	record := db.MigrationRecord{
		InstalledRank: installedRank,
		Version:       &migration.Version,
		Description:   migration.Description,
		Type:          "versioned",
		Script:        migration.String(),
		Checksum:      &migration.Checksum,
		InstalledBy:   "bloomdb",
		ExecutionTime: int(executionTime),
		Success:       successFlag,
	}

	logger.Debugf("Recording migration %s with state: %s", migration.Description, state)
	recordErr := setup.InsertMigrationRecord(record)
	if recordErr != nil {
		logger.Errorf("Failed to record migration %s: %v", migration.Description, recordErr)
		// If we can't record the migration, that's a critical error
		if err != nil {
			return int(executionTime), fmt.Errorf("migration failed AND failed to record: %v (recording error: %v)", err, recordErr)
		}
		return int(executionTime), fmt.Errorf("failed to insert migration record: %w", recordErr)
	}

	// Return the original execution error if it failed
	if err != nil {
		return int(executionTime), fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	logger.Debugf("Successfully executed and recorded migration: %s", migration.Description)
	return int(executionTime), nil
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
	} else {
		// Look for default post-migration script files
		postScriptFiles := []string{
			"post_migration.sql",
			"post_migration.sql.tmpl",
			"post_migration.template",
		}

		for _, filename := range postScriptFiles {
			fullPath := filepath.Join(migrationPath, filename)
			if _, err := os.Stat(fullPath); err == nil {
				postScriptPath = fullPath
				break
			}
		}
	}

	if postScriptPath == "" {
		logger.Debug("No post-migration script found")
		return nil
	}

	logger.Infof("Found post-migration script: %s", filepath.Base(postScriptPath))

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
		logger.Info("Post-migration script rendered to empty SQL, skipping execution")
		return nil
	}

	logger.Infof("Executing post-migration script (%d characters)", len(renderedSQL))
	fmt.Printf("🔧 Executing post-migration script...\n")

	err = setup.ExecuteMigration(renderedSQL)
	if err != nil {
		return fmt.Errorf("failed to execute post-migration SQL: %w", err)
	}

	fmt.Printf("✓ Post-migration script executed successfully\n")
	logger.Info("Post-migration script executed successfully")

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
	logger.Debugf("Executing repeatable migration: %s", migration.Description)

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
	)

	// Get database objects before migration
	beforeObjects, err = setup.Database.GetDatabaseObjects()
	if err != nil {
		logger.Warnf("Failed to get database objects before migration: %v", err)
	}

	// Measure execution time
	startTime = time.Now()

	// Execute the migration SQL
	err = setup.ExecuteMigration(migration.Content)

	// Calculate execution time in milliseconds
	executionTime = time.Since(startTime).Milliseconds()

	// Get database objects after migration
	afterObjects, err = setup.Database.GetDatabaseObjects()
	if err != nil {
		logger.Warnf("Failed to get database objects after migration: %v", err)
	}

	// Find and print newly created and deleted objects
	if beforeObjects != nil && afterObjects != nil {
		createdObjects = findCreatedObjects(beforeObjects, afterObjects)
		deletedObjects = findDeletedObjects(beforeObjects, afterObjects)

		if len(createdObjects) > 0 {
			fmt.Printf("  Created objects:\n")
			for _, obj := range createdObjects {
				fmt.Printf("    - %s: %s\n", obj.Type, obj.Name)
			}
		}

		if len(deletedObjects) > 0 {
			fmt.Printf("  Deleted objects:\n")
			for _, obj := range deletedObjects {
				fmt.Printf("    - %s: %s\n", obj.Type, obj.Name)
			}
		}
	}

	// Determine the state based on execution result
	state = "success"
	if err != nil {
		state = "failed"
		logger.Warnf("Repeatable migration %s failed with state: %s", migration.Description, state)
	}

	// Always record the migration (whether success or failed)
	successFlag = 0
	if state == "success" {
		successFlag = 1
	}
	// Use a high rank number to ensure they appear after versioned migrations
	record := db.MigrationRecord{
		InstalledRank: 999999, // High rank for repeatable migrations
		Version:       nil,    // Empty version for repeatable migrations
		Description:   migration.Description,
		Type:          "repeatable",
		Script:        migration.String(),
		Checksum:      &migration.Checksum,
		InstalledBy:   "bloomdb",
		ExecutionTime: int(executionTime),
		Success:       successFlag,
	}

	logger.Debugf("Recording repeatable migration %s with state: %s", migration.Description, state)
	recordErr := setup.InsertMigrationRecord(record)
	if recordErr != nil {
		logger.Errorf("Failed to record repeatable migration %s: %v", migration.Description, recordErr)
		return int(executionTime), fmt.Errorf("failed to insert repeatable migration record: %w", recordErr)
	}

	// Return the original execution error if it failed
	if err != nil {
		return int(executionTime), fmt.Errorf("failed to execute repeatable migration SQL: %w", err)
	}

	logger.Debugf("Successfully executed and recorded repeatable migration: %s", migration.Description)
	return int(executionTime), nil
}
