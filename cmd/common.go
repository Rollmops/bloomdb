package cmd

import (
	"fmt"
	"os"

	"bloomdb/db"
	"bloomdb/logger"
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
		logger.Fatal("connection string is required")
	}

	var database db.Database
	var connStr string
	var dbType db.DatabaseType
	var tableName string

	// Use defer to ensure cleanup on any error during setup
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("Panic during database setup: %v", r)
			if database != nil {
				database.Close()
			}
			panic(r) // Re-panic after cleanup
		}
	}()

	// Create database instance
	database, err := db.NewDatabaseFromConnectionString(dbConnStr)
	if err != nil {
		logger.Fatalf("Error creating database: %v", err)
	}

	// Extract connection string
	connStr, extractErr := db.ExtractConnectionString(dbConnStr)
	if extractErr != nil {
		if database != nil {
			database.Close()
		}
		logger.Fatalf("Error extracting connection string: %v", extractErr)
	}

	// Connect to database
	logger.Debugf("Connecting to database with connection string: %s", connStr)
	err = database.Connect(connStr)
	if err != nil {
		if database != nil {
			database.Close()
		}
		logger.Fatalf("Error connecting to database: %v", err)
	}

	// Test connection
	logger.Debug("Testing database connection")
	err = database.Ping()
	if err != nil {
		if database != nil {
			database.Close()
		}
		logger.Fatalf("Error pinging database: %v", err)
	}
	logger.Debug("Database connection test successful")

	// Get database type
	dbType, parseErr := db.ParseDatabaseType(dbConnStr)
	if parseErr != nil {
		if database != nil {
			database.Close()
		}
		logger.Fatalf("Error parsing database type: %v", parseErr)
	}
	logger.Infof("Detected database type: %s", dbType)

	// Get table name from command configuration
	tableName = GetVersionTableName()
	logger.Debugf("Migration table name: %s", tableName)

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

// EnsureTableExists checks if the migration table exists, exits with error if it doesn't
func (ds *DatabaseSetup) EnsureTableExists() {
	logger.Debugf("Checking if migration table '%s' exists", ds.TableName)
	tableExists, err := ds.Database.TableExists(ds.TableName)
	if err != nil {
		logger.Errorf("Error checking table existence: %v", err)
		ds.Database.Close()
		os.Exit(1)
	}

	if !tableExists {
		logger.Fatalf("Table '%s' does not exist", ds.TableName)
	}
	logger.Debugf("Migration table '%s' exists", ds.TableName)
}

// EnsureTableNotExists checks if the migration table doesn't exist, exits with error if it does
func (ds *DatabaseSetup) EnsureTableNotExists() {
	logger.Debugf("Checking if migration table '%s' does not exist", ds.TableName)
	tableExists, err := ds.Database.TableExists(ds.TableName)
	if err != nil {
		logger.Errorf("Error checking if table %s exists: %v", ds.TableName, err)
		ds.Database.Close()
		os.Exit(1)
	}

	if tableExists {
		logger.Fatalf("Migration table '%s' already exists - have you already run the baseline command?", ds.TableName)
	}
	logger.Debugf("Migration table '%s' does not exist as expected", ds.TableName)
}

// CreateMigrationTable creates the migration table
func (ds *DatabaseSetup) CreateMigrationTable() error {
	fmt.Printf("Creating migration table: %s\n", ds.TableName)
	logger.Infof("Creating migration table: %s", ds.TableName)
	err := ds.Database.CreateMigrationTable(ds.TableName)
	if err != nil {
		logger.Errorf("Failed to create migration table %s: %v", ds.TableName, err)
		return fmt.Errorf("failed to create migration table %s: %w", ds.TableName, err)
	}
	logger.Infof("Migration table %s created successfully", ds.TableName)
	return nil
}

// InsertBaselineRecord inserts a baseline record into the migration table
func (ds *DatabaseSetup) InsertBaselineRecord(version string) error {
	fmt.Printf("Inserting baseline record for version: %s\n", version)
	logger.Infof("Inserting baseline record for version: %s", version)
	err := ds.Database.InsertBaselineRecord(ds.TableName, version)
	if err != nil {
		logger.Errorf("Failed to insert baseline record: %v", err)
		return fmt.Errorf("failed to insert baseline record: %w", err)
	}
	logger.Info("Baseline record inserted successfully")
	return nil
}

// GetMigrationRecords retrieves all migration records from the database
func (ds *DatabaseSetup) GetMigrationRecords() ([]db.MigrationRecord, error) {
	logger.Debugf("Retrieving migration records from table: %s", ds.TableName)
	return ds.Database.GetMigrationRecords(ds.TableName)
}

// InsertMigrationRecord inserts a migration record into the database
func (ds *DatabaseSetup) InsertMigrationRecord(record db.MigrationRecord) error {
	logger.Debugf("Inserting migration record: %s (%s)", record.Description, record.Type)
	return ds.Database.InsertMigrationRecord(ds.TableName, record)
}

// ExecuteMigration executes a migration SQL script
func (ds *DatabaseSetup) ExecuteMigration(content string) error {
	logger.Debugf("Executing migration SQL script")
	return ds.Database.ExecuteMigration(content)
}

// Close closes the database connection
func (ds *DatabaseSetup) Close() {
	if ds.Database != nil {
		logger.Debug("Closing database connection")
		ds.Database.Close()
	}
}
