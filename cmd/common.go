package cmd

import (
	"fmt"
	"os"

	"bloomdb/db"
)

// DatabaseSetup holds the common database connection and configuration
type DatabaseSetup struct {
	Database  db.Database
	ConnStr   string
	DBType    db.DatabaseType
	TableName string
}

// SetupDatabase performs the common database setup steps used across commands
// Returns DatabaseSetup on success, exits on error
func SetupDatabase() *DatabaseSetup {
	// Validate connection string
	if dbConnStr == "" {
		PrintError("connection string is required")
		os.Exit(1)
	}

	var database db.Database
	var connStr string
	var dbType db.DatabaseType
	var tableName string

	// Use defer to ensure cleanup on any error during setup
	defer func() {
		if r := recover(); r != nil {
			PrintError(fmt.Sprintf("Panic during database setup: %v", r))
			if database != nil {
				database.Close()
			}
			panic(r) // Re-panic after cleanup
		}
	}()

	// Create database instance
	database, err := db.NewDatabaseFromConnectionString(dbConnStr)
	if err != nil {
		PrintError(fmt.Sprintf("Error creating database: %v", err))
		os.Exit(1)
	}

	// Extract connection string
	connStr, extractErr := db.ExtractConnectionString(dbConnStr)
	if extractErr != nil {
		if database != nil {
			database.Close()
		}
		PrintError(fmt.Sprintf("Error extracting connection string: %v", extractErr))
		os.Exit(1)
	}

	// Connect to database
	err = database.Connect(connStr)
	if err != nil {
		if database != nil {
			database.Close()
		}
		PrintError(fmt.Sprintf("Error connecting to database: %v", err))
		os.Exit(1)
	}

	// Test connection
	err = database.Ping()
	if err != nil {
		if database != nil {
			database.Close()
		}
		PrintError(fmt.Sprintf("Error pinging database: %v", err))
		os.Exit(1)
	}
	PrintInfo("Database connection test successful")

	// Get database type
	dbType, parseErr := db.ParseDatabaseType(dbConnStr)
	if parseErr != nil {
		if database != nil {
			database.Close()
		}
		PrintError(fmt.Sprintf("Error parsing database type: %v", parseErr))
		os.Exit(1)
	}

	// Get table name from command configuration
	tableName = GetVersionTableName()

	setup := &DatabaseSetup{
		Database:  database,
		ConnStr:   connStr,
		DBType:    dbType,
		TableName: tableName,
	}

	// Register this setup for global cleanup
	SetGlobalDatabaseSetup(setup)

	return setup
}

func (ds *DatabaseSetup) CreateMigrationTable() error {
	err := ds.Database.CreateMigrationTable(ds.TableName)
	if err != nil {
		PrintError(fmt.Sprintf("Failed to create migration table %s: %v", ds.TableName, err))
		return fmt.Errorf("failed to create migration table %s: %w", ds.TableName, err)
	}
	return nil
}

// InsertBaselineRecord inserts a baseline record into the migration table
func (ds *DatabaseSetup) InsertBaselineRecord(version string) error {
	err := ds.Database.InsertBaselineRecord(ds.TableName, version)
	if err != nil {
		PrintError(fmt.Sprintf("Failed to insert baseline record: %v", err))
		return fmt.Errorf("failed to insert baseline record: %w", err)
	}
	return nil
}

// GetMigrationRecords retrieves all migration records from the database
func (ds *DatabaseSetup) GetMigrationRecords() ([]db.MigrationRecord, error) {
	return ds.Database.GetMigrationRecords(ds.TableName)
}

// InsertMigrationRecord inserts a migration record into the database
func (ds *DatabaseSetup) InsertMigrationRecord(record db.MigrationRecord) error {
	return ds.Database.InsertMigrationRecord(ds.TableName, record)
}

// DeleteFailedMigrationRecords removes all unsuccessful migration records from the version table
func (ds *DatabaseSetup) UpdateMigrationRecord(installedRank int, version, description string, checksum int64) error {
	return ds.Database.UpdateMigrationRecord(ds.TableName, installedRank, version, description, checksum)
}

func (ds *DatabaseSetup) UpdateMigrationRecordFull(record db.MigrationRecord) error {
	return ds.Database.UpdateMigrationRecordFull(ds.TableName, record)
}

func (ds *DatabaseSetup) DeleteFailedMigrationRecords() error {
	return ds.Database.DeleteFailedMigrationRecords(ds.TableName)
}

// CheckBaselineRecordExists checks if a baseline record exists in the version table
func (ds *DatabaseSetup) CheckBaselineRecordExists() (bool, string, error) {
	records, err := ds.Database.GetMigrationRecords(ds.TableName)
	if err != nil {
		return false, "", fmt.Errorf("failed to get migration records: %w", err)
	}

	for _, record := range records {
		if record.Type == "BASELINE" && record.Version != nil {
			PrintInfo(fmt.Sprintf("Found baseline record for version: %s", *record.Version))
			return true, *record.Version, nil
		}
	}

	PrintInfo(fmt.Sprintf("No baseline record found in %s", ds.TableName))
	return false, "", nil
}

// EnsureTableAndBaselineExist checks if both the migration table and baseline record exist
func (ds *DatabaseSetup) EnsureTableAndBaselineExist() {
	tableExists, err := ds.Database.TableExists(ds.TableName)
	if err != nil {
		PrintError(fmt.Sprintf("Error checking table existence: %v", err))
		ds.Database.Close()
		os.Exit(1)
	}

	if !tableExists {
		PrintError(fmt.Sprintf("Migration table '%s' does not exist - have you run the baseline command?", ds.TableName))
		ds.Database.Close()
		os.Exit(1)
	}

	// Check for baseline record
	baselineExists, baselineVersion, err := ds.CheckBaselineRecordExists()
	if err != nil {
		PrintError(fmt.Sprintf("Error checking baseline record: %v", err))
		ds.Database.Close()
		os.Exit(1)
	}

	if !baselineExists {
		PrintError(fmt.Sprintf("No baseline record found in migration table '%s' - have you run the baseline command?", ds.TableName))
		ds.Database.Close()
		os.Exit(1)
	}

	PrintInfo(fmt.Sprintf("Migration table '%s' exists with baseline record version %s", ds.TableName, baselineVersion))
}

// ExecuteMigration executes a migration SQL script
func (ds *DatabaseSetup) ExecuteMigration(content string) error {
	return ds.Database.ExecuteMigration(content)
}

// Close closes the database connection
func (ds *DatabaseSetup) Close() {
	if ds.Database != nil {
		ds.Database.Close()
	}
}

// ResolveBaselineVersion determines the baseline version using the correct priority:
// 1. Existing baseline record in database (if table and record exist)
// 2. CLI flag value (if provided)
// 3. Environment variable (BLOOMDB_BASELINE_VERSION)
// 4. Default value ("1")
func ResolveBaselineVersion(setup *DatabaseSetup, flagValue string) string {
	// Priority 1: Check for existing baseline record in database
	tableExists, err := setup.Database.TableExists(setup.TableName)
	if err != nil {
		PrintWarning("Error checking table existence during baseline resolution: %v", err)
	} else if tableExists {
		// Table exists, check for baseline record
		baselineExists, existingVersion, err := setup.CheckBaselineRecordExists()
		if err != nil {
			PrintWarning("Error checking baseline record during resolution: %v", err)
		} else if baselineExists {
			PrintInfo("Using existing baseline version from database: " + existingVersion)
			return existingVersion
		}
	}

	// Priority 2: CLI flag
	if flagValue != "" {
		PrintInfo("Using baseline version from CLI flag: " + flagValue)
		return flagValue
	}

	// Priority 3: Environment variable
	if envVersion := os.Getenv("BLOOMDB_BASELINE_VERSION"); envVersion != "" {
		PrintInfo("Using baseline version from environment: " + envVersion)
		return envVersion
	}

	// Priority 4: Default value
	PrintInfo("Using default baseline version: 1")
	return "1"
}

// FindBaselineVersion returns the baseline version from migration records
func FindBaselineVersion(records []db.MigrationRecord) string {
	for _, record := range records {
		if record.Type == "BASELINE" && record.Version != nil {
			return *record.Version
		}
	}
	return ""
}

// CalculateNextRank finds the maximum installed rank and returns the next rank
func CalculateNextRank(records []db.MigrationRecord) int {
	maxRank := 0
	for _, record := range records {
		if record.InstalledRank > maxRank {
			maxRank = record.InstalledRank
		}
	}
	return maxRank + 1
}
