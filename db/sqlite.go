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
	logSQL(query, tableName)
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
			installed_rank INTEGER,
			version TEXT,
			description TEXT,
			type TEXT,
			script TEXT,
			checksum INTEGER,
			installed_by TEXT,
			installed_on DATETIME,
			execution_time INTEGER,
			success INTEGER
		)
	`, tableName)

	logSQL(query)
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
		INSERT INTO %s (installed_rank, version, description, type, script, checksum, installed_by, installed_on, execution_time, success)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), ?, ?)
	`, tableName)

	// Convert version to integer for installed rank
	installedRank := versionToInt(version)
	logSQL(query, installedRank, version, "<< Baseline >>", "BASELINE", "<< Baseline >>", nil, "bloomdb", 0, 1)
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
		SELECT installed_rank, version, description, type, script, checksum, installed_by, installed_on, execution_time, success
		FROM %s 
		ORDER BY installed_rank
	`, tableName)

	logSQL(query)
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
		INSERT INTO %s (installed_rank, version, description, type, script, checksum, installed_by, installed_on, execution_time, success)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), ?, ?)
	`, tableName)

	logSQL(query, record.InstalledRank, record.Version, record.Description, record.Type, record.Script, record.Checksum, record.InstalledBy, record.ExecutionTime, record.Success)
	_, err := s.db.Exec(query, record.InstalledRank, record.Version, record.Description, record.Type, record.Script, record.Checksum, record.InstalledBy, record.ExecutionTime, record.Success)
	if err != nil {
		return fmt.Errorf("failed to insert migration record: %w", err)
	}

	return nil
}

func (s *SQLiteDatabase) UpdateMigrationRecord(tableName string, installedRank int, version, description string, checksum int64) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		UPDATE %s 
		SET description = ?, checksum = ?
		WHERE installed_rank = ? AND version = ?
	`, tableName)

	logSQL(query, description, checksum, installedRank, version)
	_, err := s.db.Exec(query, description, checksum, installedRank, version)
	if err != nil {
		return fmt.Errorf("failed to update migration record: %w", err)
	}

	return nil
}

func (s *SQLiteDatabase) UpdateMigrationRecordFull(tableName string, record MigrationRecord) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		UPDATE %s 
		SET installed_rank = ?, version = ?, description = ?, type = ?, script = ?, checksum = ?, 
			installed_by = ?, installed_on = ?, execution_time = ?, success = ?
		WHERE installed_rank = ? AND (version = ? OR (version IS NULL AND ? IS NULL))
	`, tableName)

	var versionPtr *string
	if record.Version != nil {
		versionPtr = record.Version
	}

	logSQL(query, record.InstalledRank, versionPtr, record.Description, record.Type, record.Script, record.Checksum, record.InstalledBy, record.InstalledOn, record.ExecutionTime, record.Success, record.InstalledRank, versionPtr, versionPtr)
	_, err := s.db.Exec(query,
		record.InstalledRank, versionPtr, record.Description, record.Type, record.Script,
		record.Checksum, record.InstalledBy, record.InstalledOn, record.ExecutionTime, record.Success,
		record.InstalledRank, versionPtr, versionPtr)
	if err != nil {
		return fmt.Errorf("failed to update migration record: %w", err)
	}

	return nil
}

func (s *SQLiteDatabase) DeleteFailedMigrationRecords(tableName string) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		DELETE FROM %s 
		WHERE success != 1
	`, tableName)

	logSQL(query)
	result, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to delete failed migration records: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		fmt.Printf("Removed %d failed migration records from %s\n", rowsAffected, tableName)
	} else {
		fmt.Printf("No failed migration records found in %s\n", tableName)
	}

	return nil
}

func (s *SQLiteDatabase) ExecuteMigration(content string) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	// Parse the SQL content into individual statements
	statements := ParseSQLStatements(content)

	// Execute each statement individually
	for i, statement := range statements {
		logSQL(statement)
		_, err := s.db.Exec(statement)
		if err != nil {
			return fmt.Errorf("failed to execute statement %d: %w", i+1, err)
		}
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
	logSQL(tableQuery)
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
	logSQL(viewQuery)
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
	logSQL(indexQuery)
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
	logSQL(query)
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
		logSQL(dropQuery)
		if _, err := s.db.Exec(dropQuery); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	return nil
}
