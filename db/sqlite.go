package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteDatabase struct {
	db *sql.DB
}

func NewSQLiteDatabase() *SQLiteDatabase {
	return &SQLiteDatabase{}
}

func (s *SQLiteDatabase) Connect(connectionString string) error {
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		return fmt.Errorf("failed to connect to SQLite: %w", err)
	}
	s.db = db
	return nil
}

func (s *SQLiteDatabase) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *SQLiteDatabase) Ping() error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}
	return s.db.Ping()
}

func (s *SQLiteDatabase) GetDB() *sql.DB {
	return s.db
}

func (s *SQLiteDatabase) TableExists(tableName string) (bool, error) {
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
	var result string
	err := s.db.QueryRow(query, tableName).Scan(&result)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return false, nil
		}
		return false, fmt.Errorf("error checking table existence: %w", err)
	}
	return true, nil
}

func (s *SQLiteDatabase) CreateMigrationTable(tableName string) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		CREATE TABLE %s (
			"installed rank" INTEGER,
			"version" TEXT,
			"description" TEXT,
			"type" TEXT,
			"script" TEXT,
			"checksum" INTEGER,
			"installed by" TEXT,
			"installed on" DATETIME,
			"execution time" INTEGER,
			"success" INTEGER
		)
	`, tableName)

	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create migration table %s: %w", tableName, err)
	}

	return nil
}

func (s *SQLiteDatabase) InsertBaselineRecord(tableName, version string) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		INSERT INTO %s ("installed rank", "version", "description", "type", "script", "checksum", "installed by", "installed on", "execution time", "success")
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), ?, ?)
	`, tableName)

	// Convert version to integer for installed rank
	installedRank := versionToInt(version)
	_, err := s.db.Exec(query, installedRank, version, "<< Baseline >>", "BASELINE", "<< Baseline >>", nil, "bloomdb", 0, 1)
	if err != nil {
		return fmt.Errorf("failed to insert baseline record: %w", err)
	}

	return nil
}

func (s *SQLiteDatabase) GetMigrationRecords(tableName string) ([]MigrationRecord, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		SELECT "installed rank", "version", "description", "type", "script", "checksum", "installed by", "installed on", "execution time", "success"
		FROM %s 
		ORDER BY "installed rank"
	`, tableName)

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query migration records: %w", err)
	}
	defer rows.Close()

	var records []MigrationRecord
	for rows.Next() {
		var record MigrationRecord
		err := rows.Scan(&record.InstalledRank, &record.Version, &record.Description, &record.Type, &record.Script, &record.Checksum, &record.InstalledBy, &record.InstalledOn, &record.ExecutionTime, &record.Success)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration record: %w", err)
		}
		records = append(records, record)
	}

	return records, nil
}

func (s *SQLiteDatabase) InsertMigrationRecord(tableName string, record MigrationRecord) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		INSERT INTO %s ("installed rank", "version", "description", "type", "script", "checksum", "installed by", "installed on", "execution time", "success")
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), ?, ?)
	`, tableName)

	_, err := s.db.Exec(query, record.InstalledRank, record.Version, record.Description, record.Type, record.Script, record.Checksum, record.InstalledBy, record.ExecutionTime, record.Success)
	if err != nil {
		return fmt.Errorf("failed to insert migration record: %w", err)
	}

	return nil
}

func (s *SQLiteDatabase) ExecuteMigration(content string) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	_, err := s.db.Exec(content)
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}

func (s *SQLiteDatabase) GetDatabaseObjects() ([]DatabaseObject, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	var objects []DatabaseObject

	// Get tables
	tableQuery := "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"
	rows, err := s.db.Query(tableQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		objects = append(objects, DatabaseObject{Type: "table", Name: tableName})
	}

	// Get views
	viewQuery := "SELECT name FROM sqlite_master WHERE type='view'"
	rows, err = s.db.Query(viewQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query views: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var viewName string
		if err := rows.Scan(&viewName); err != nil {
			return nil, fmt.Errorf("failed to scan view name: %w", err)
		}
		objects = append(objects, DatabaseObject{Type: "view", Name: viewName})
	}

	// Get indexes
	indexQuery := "SELECT name, tbl_name FROM sqlite_master WHERE type='index' AND name NOT LIKE 'sqlite_%'"
	rows, err = s.db.Query(indexQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var indexName, tableName string
		if err := rows.Scan(&indexName, &tableName); err != nil {
			return nil, fmt.Errorf("failed to scan index name: %w", err)
		}
		objects = append(objects, DatabaseObject{Type: "index", Name: indexName})
	}

	return objects, nil
}

func (s *SQLiteDatabase) DestroyAllObjects() error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	// Get all table names
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"
	rows, err := s.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	// Drop all tables
	for _, table := range tables {
		dropQuery := fmt.Sprintf("DROP TABLE IF EXISTS %s", table)
		if _, err := s.db.Exec(dropQuery); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	return nil
}
