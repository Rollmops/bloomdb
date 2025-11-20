package db

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLDatabase struct {
	db *sql.DB
}

func NewMySQLDatabase() *MySQLDatabase {
	return &MySQLDatabase{}
}

func (m *MySQLDatabase) Connect(connectionString string) error {
	// MySQL driver expects DSN without scheme
	// If we receive mysql:// or mysql:, it should have been stripped by ExtractConnectionString
	// But we double check here just in case
	dsn := connectionString
	if strings.HasPrefix(dsn, "mysql://") {
		dsn = strings.TrimPrefix(dsn, "mysql://")
	} else if strings.HasPrefix(dsn, "mysql:") {
		dsn = strings.TrimPrefix(dsn, "mysql:")
	}

	// Add parseTime=true if not present, as it's required for time.Time support
	if !strings.Contains(dsn, "parseTime=true") {
		if strings.Contains(dsn, "?") {
			dsn += "&parseTime=true"
		} else {
			dsn += "?parseTime=true"
		}
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}
	m.db = db
	return nil
}

func (m *MySQLDatabase) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

func (m *MySQLDatabase) Ping() error {
	if m.db == nil {
		return fmt.Errorf("database not connected")
	}
	return m.db.Ping()
}

func (m *MySQLDatabase) GetDB() *sql.DB {
	return m.db
}

func (m *MySQLDatabase) TableExists(tableName string) (bool, error) {
	if m.db == nil {
		return false, fmt.Errorf("database not connected")
	}

	query := "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?"
	logSQL(query, tableName)
	var count int
	err := m.db.QueryRow(query, tableName).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("error checking table existence: %w", err)
	}
	return count > 0, nil
}

func (m *MySQLDatabase) CreateMigrationTable(tableName string) error {
	if m.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		CREATE TABLE %s (
			installed_rank INT,
			version VARCHAR(50),
			description TEXT,
			type VARCHAR(20),
			script VARCHAR(1000),
			checksum BIGINT,
			installed_by VARCHAR(100),
			installed_on TIMESTAMP,
			execution_time INT,
			success TINYINT
		)
	`, tableName)

	logSQL(query)
	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create migration table %s: %w", tableName, err)
	}

	return nil
}

func (m *MySQLDatabase) InsertBaselineRecord(tableName, version string) error {
	if m.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (installed_rank, version, description, type, script, checksum, installed_by, installed_on, execution_time, success)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), ?, ?)
	`, tableName)

	// Convert version to integer for installed rank
	installedRank := versionToInt(version)
	logSQL(query, installedRank, version, "<< Baseline >>", "BASELINE", "<< Baseline >>", nil, "bloomdb", 0, 1)
	_, err := m.db.Exec(query, installedRank, version, "<< Baseline >>", "BASELINE", "<< Baseline >>", nil, "bloomdb", 0, 1)
	if err != nil {
		return fmt.Errorf("failed to insert baseline record: %w", err)
	}

	return nil
}

func (m *MySQLDatabase) GetMigrationRecords(tableName string) ([]MigrationRecord, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		SELECT installed_rank, version, description, type, script, checksum, installed_by, installed_on, execution_time, success
		FROM %s 
		ORDER BY installed_rank
	`, tableName)

	logSQL(query)
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query migration records: %w", err)
	}
	defer rows.Close()

	var records []MigrationRecord
	for rows.Next() {
		var record MigrationRecord
		// MySQL driver handles NULLs differently, need to be careful with pointers
		// For checksum and version which are pointers in MigrationRecord
		var version sql.NullString
		var checksum sql.NullInt64

		// We need to scan installed_on as string because MySQL driver returns []uint8 for TIMESTAMP by default unless parseTime=true is set
		// But we set parseTime=true in Connect, so it should return time.Time, but our struct has string
		// Let's scan into a temporary variable for the time
		var installedOn []uint8

		err := rows.Scan(&record.InstalledRank, &version, &record.Description, &record.Type, &record.Script, &checksum, &record.InstalledBy, &installedOn, &record.ExecutionTime, &record.Success)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration record: %w", err)
		}

		if version.Valid {
			v := version.String
			record.Version = &v
		}

		if checksum.Valid {
			c := checksum.Int64
			record.Checksum = &c
		}

		record.InstalledOn = string(installedOn)

		records = append(records, record)
	}

	return records, nil
}

func (m *MySQLDatabase) InsertMigrationRecord(tableName string, record MigrationRecord) error {
	if m.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (installed_rank, version, description, type, script, checksum, installed_by, installed_on, execution_time, success)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), ?, ?)
	`, tableName)

	logSQL(query, record.InstalledRank, record.Version, record.Description, record.Type, record.Script, record.Checksum, record.InstalledBy, record.ExecutionTime, record.Success)
	_, err := m.db.Exec(query, record.InstalledRank, record.Version, record.Description, record.Type, record.Script, record.Checksum, record.InstalledBy, record.ExecutionTime, record.Success)
	if err != nil {
		return fmt.Errorf("failed to insert migration record: %w", err)
	}

	return nil
}

func (m *MySQLDatabase) UpdateMigrationRecord(tableName string, installedRank int, version, description string, checksum int64) error {
	if m.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		UPDATE %s 
		SET description = ?, checksum = ?
		WHERE installed_rank = ? AND version = ?
	`, tableName)

	logSQL(query, description, checksum, installedRank, version)
	_, err := m.db.Exec(query, description, checksum, installedRank, version)
	if err != nil {
		return fmt.Errorf("failed to update migration record: %w", err)
	}

	return nil
}

func (m *MySQLDatabase) UpdateMigrationRecordFull(tableName string, record MigrationRecord) error {
	if m.db == nil {
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
	_, err := m.db.Exec(query,
		record.InstalledRank, versionPtr, record.Description, record.Type, record.Script,
		record.Checksum, record.InstalledBy, record.InstalledOn, record.ExecutionTime, record.Success,
		record.InstalledRank, versionPtr, versionPtr)
	if err != nil {
		return fmt.Errorf("failed to update migration record: %w", err)
	}

	return nil
}

func (m *MySQLDatabase) DeleteFailedMigrationRecords(tableName string) error {
	if m.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		DELETE FROM %s 
		WHERE success != 1
	`, tableName)

	logSQL(query)
	result, err := m.db.Exec(query)
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

func (m *MySQLDatabase) ExecuteMigration(content string) error {
	if m.db == nil {
		return fmt.Errorf("database not connected")
	}

	// Parse the SQL content into individual statements
	statements := ParseSQLStatements(content)

	// Execute each statement individually
	for i, statement := range statements {
		logSQL(statement)
		_, err := m.db.Exec(statement)
		if err != nil {
			return fmt.Errorf("failed to execute statement %d: %w", i+1, err)
		}
	}

	return nil
}

func (m *MySQLDatabase) DestroyAllObjects() error {
	if m.db == nil {
		return fmt.Errorf("database not connected")
	}

	// Disable foreign key checks to allow dropping tables in any order
	_, err := m.db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	if err != nil {
		return fmt.Errorf("failed to disable foreign key checks: %w", err)
	}
	defer m.db.Exec("SET FOREIGN_KEY_CHECKS = 1")

	// Get all tables
	rows, err := m.db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE()")
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
		query := fmt.Sprintf("DROP TABLE IF EXISTS `%s`", table)
		logSQL(query)
		if _, err := m.db.Exec(query); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	// Get all views
	rows, err = m.db.Query("SELECT table_name FROM information_schema.views WHERE table_schema = DATABASE()")
	if err != nil {
		return fmt.Errorf("failed to query views: %w", err)
	}
	defer rows.Close()

	var views []string
	for rows.Next() {
		var viewName string
		if err := rows.Scan(&viewName); err != nil {
			return fmt.Errorf("failed to scan view name: %w", err)
		}
		views = append(views, viewName)
	}

	// Drop all views
	for _, view := range views {
		query := fmt.Sprintf("DROP VIEW IF EXISTS `%s`", view)
		logSQL(query)
		if _, err := m.db.Exec(query); err != nil {
			return fmt.Errorf("failed to drop view %s: %w", view, err)
		}
	}

	return nil
}

func (m *MySQLDatabase) GetDatabaseObjects() ([]DatabaseObject, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	var objects []DatabaseObject

	// Get tables
	tableQuery := "SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'"
	logSQL(tableQuery)
	rows, err := m.db.Query(tableQuery)
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
	viewQuery := "SELECT table_name FROM information_schema.views WHERE table_schema = DATABASE()"
	logSQL(viewQuery)
	rows, err = m.db.Query(viewQuery)
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

	return objects, nil
}
