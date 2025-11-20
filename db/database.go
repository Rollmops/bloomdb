package db

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type DatabaseType string

const (
	SQLite     DatabaseType = "sqlite"
	PostgreSQL DatabaseType = "postgresql"
	Oracle     DatabaseType = "oracle"
	MySQL      DatabaseType = "mysql"
)

type Database interface {
	Connect(connectionString string) error
	Close() error
	Ping() error
	GetDB() *sql.DB
	TableExists(tableName string) (bool, error)
	CreateMigrationTable(tableName string) error
	InsertBaselineRecord(tableName, version string) error
	GetMigrationRecords(tableName string) ([]MigrationRecord, error)
	InsertMigrationRecord(tableName string, record MigrationRecord) error
	UpdateMigrationRecord(tableName string, installedRank int, version, description string, checksum int64) error
	UpdateMigrationRecordFull(tableName string, record MigrationRecord) error
	DeleteFailedMigrationRecords(tableName string) error
	ExecuteMigration(content string) error
	DestroyAllObjects() error
	GetDatabaseObjects() ([]DatabaseObject, error)
}

// DatabaseObject represents a database object with its type and name
type DatabaseObject struct {
	Type string `json:"type"` // table, view, index, etc.
	Name string `json:"name"` // object name
}

// MigrationRecord represents a record in the migration table
type MigrationRecord struct {
	InstalledRank int     `json:"installed_rank"`
	Version       *string `json:"version"`
	Description   string  `json:"description"`
	Type          string  `json:"type"`
	Script        string  `json:"script"`
	Checksum      *int64  `json:"checksum"`
	InstalledBy   string  `json:"installed_by"`
	InstalledOn   string  `json:"installed_on"`
	ExecutionTime int     `json:"execution_time"`
	Success       int     `json:"success"`
}

type Config struct {
	Type             DatabaseType
	ConnectionString string
}

// versionToInt converts a version string to an integer for installed rank
// Takes the first part of the version (before the dot) and converts to int
func versionToInt(version string) int {
	parts := strings.Split(version, ".")
	if len(parts) == 0 {
		return 0
	}
	if num, err := strconv.Atoi(parts[0]); err == nil {
		return num
	}
	return 0
}

// logSQL logs SQL statements when verbose mode is enabled
func logSQL(query string, args ...interface{}) {
	// Check if verbose mode is enabled via environment variable
	verbose := os.Getenv("BLOOMDB_VERBOSE")
	if verbose == "" {
		return
	}

	// Format the SQL for display
	displayQuery := strings.TrimSpace(query)

	// If there are arguments, show them
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "[SQL] %s\n[ARGS] %v\n", displayQuery, args)
	} else {
		fmt.Fprintf(os.Stderr, "[SQL] %s\n", displayQuery)
	}
}
