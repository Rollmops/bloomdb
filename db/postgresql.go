package db

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

type PostgreSQLDatabase struct {
	db *sql.DB
}

func NewPostgreSQLDatabase() *PostgreSQLDatabase {
	return &PostgreSQLDatabase{}
}

func (p *PostgreSQLDatabase) Connect(connectionString string) error {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	p.db = db
	return nil
}

func (p *PostgreSQLDatabase) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

func (p *PostgreSQLDatabase) Ping() error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}
	return p.db.Ping()
}

func (p *PostgreSQLDatabase) GetDB() *sql.DB {
	return p.db
}

func (p *PostgreSQLDatabase) TableExists(tableName string) (bool, error) {
	if p.db == nil {
		return false, fmt.Errorf("database not connected")
	}

	// PostgreSQL stores table names in lowercase in information_schema
	query := "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1"
	logSQL(query, strings.ToLower(tableName))
	var result string
	err := p.db.QueryRow(query, strings.ToLower(tableName)).Scan(&result)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return false, nil
		}
		return false, fmt.Errorf("error checking table existence: %w", err)
	}
	return true, nil
}

func (p *PostgreSQLDatabase) CreateMigrationTable(tableName string) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		CREATE TABLE %s (
			installed_rank INTEGER,
			version VARCHAR(50),
			description TEXT,
			type VARCHAR(20),
			script VARCHAR(1000),
			checksum BIGINT,
			installed_by VARCHAR(100),
			installed_on TIMESTAMP,
			execution_time INTEGER,
			success INTEGER
		)
	`, tableName)

	logSQL(query)
	_, err := p.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create migration table %s: %w", tableName, err)
	}

	return nil
}

func (p *PostgreSQLDatabase) InsertBaselineRecord(tableName, version string) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (installed_rank, version, description, type, script, checksum, installed_by, installed_on, execution_time, success)
		VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, $8, $9)
	`, tableName)

	// Convert version to integer for installed rank
	installedRank := versionToInt(version)
	logSQL(query, installedRank, version, "<< Baseline >>", "BASELINE", "<< Baseline >>", nil, "bloomdb", 0, 1)
	_, err := p.db.Exec(query, installedRank, version, "<< Baseline >>", "BASELINE", "<< Baseline >>", nil, "bloomdb", 0, 1)
	if err != nil {
		return fmt.Errorf("failed to insert baseline record: %w", err)
	}

	return nil
}

func (p *PostgreSQLDatabase) GetMigrationRecords(tableName string) ([]MigrationRecord, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		SELECT installed_rank, version, description, type, script, checksum, installed_by, installed_on, execution_time, success
		FROM %s 
		ORDER BY installed_rank
	`, tableName)

	logSQL(query)
	rows, err := p.db.Query(query)
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

func (p *PostgreSQLDatabase) InsertMigrationRecord(tableName string, record MigrationRecord) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (installed_rank, version, description, type, script, checksum, installed_by, installed_on, execution_time, success)
		VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, $8, $9)
	`, tableName)

	logSQL(query, record.InstalledRank, record.Version, record.Description, record.Type, record.Script, record.Checksum, record.InstalledBy, record.ExecutionTime, record.Success)
	_, err := p.db.Exec(query, record.InstalledRank, record.Version, record.Description, record.Type, record.Script, record.Checksum, record.InstalledBy, record.ExecutionTime, record.Success)
	if err != nil {
		return fmt.Errorf("failed to insert migration record: %w", err)
	}

	return nil
}

func (p *PostgreSQLDatabase) UpdateMigrationRecord(tableName string, installedRank int, version, description string, checksum int64) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		UPDATE %s 
		SET description = $1, checksum = $2
		WHERE installed_rank = $3 AND version = $4
	`, tableName)

	logSQL(query, description, checksum, installedRank, version)
	_, err := p.db.Exec(query, description, checksum, installedRank, version)
	if err != nil {
		return fmt.Errorf("failed to update migration record: %w", err)
	}

	return nil
}

func (p *PostgreSQLDatabase) UpdateMigrationRecordFull(tableName string, record MigrationRecord) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		UPDATE %s 
		SET installed_rank = $1, version = $2, description = $3, type = $4, script = $5, checksum = $6, 
			installed_by = $7, installed_on = $8, execution_time = $9, success = $10
		WHERE installed_rank = $11 AND (version = $12 OR (version IS NULL AND $12 IS NULL))
	`, tableName)

	var versionPtr *string
	if record.Version != nil {
		versionPtr = record.Version
	}

	logSQL(query, record.InstalledRank, versionPtr, record.Description, record.Type, record.Script, record.Checksum, record.InstalledBy, record.InstalledOn, record.ExecutionTime, record.Success, record.InstalledRank, versionPtr)
	_, err := p.db.Exec(query,
		record.InstalledRank, versionPtr, record.Description, record.Type, record.Script,
		record.Checksum, record.InstalledBy, record.InstalledOn, record.ExecutionTime, record.Success,
		record.InstalledRank, versionPtr)
	if err != nil {
		return fmt.Errorf("failed to update migration record: %w", err)
	}

	return nil
}

func (p *PostgreSQLDatabase) DeleteFailedMigrationRecords(tableName string) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		DELETE FROM %s 
		WHERE success != 1
	`, tableName)

	logSQL(query)
	result, err := p.db.Exec(query)
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

func (p *PostgreSQLDatabase) ExecuteMigration(content string) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	// Parse the SQL content into individual statements
	statements := ParseSQLStatements(content)

	// Execute each statement individually
	for i, statement := range statements {
		logSQL(statement)
		_, err := p.db.Exec(statement)
		if err != nil {
			return fmt.Errorf("failed to execute statement %d: %w", i+1, err)
		}
	}

	return nil
}

func (p *PostgreSQLDatabase) GetDatabaseObjects() ([]DatabaseObject, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	var objects []DatabaseObject

	// Get tables
	tableQuery := `
		SELECT tablename 
		FROM pg_tables 
		WHERE schemaname = 'public'
	`
	logSQL(tableQuery)
	rows, err := p.db.Query(tableQuery)
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
	viewQuery := `
		SELECT viewname 
		FROM pg_views 
		WHERE schemaname = 'public'
	`
	logSQL(viewQuery)
	rows, err = p.db.Query(viewQuery)
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
	indexQuery := `
		SELECT indexname, tablename 
		FROM pg_indexes 
		WHERE schemaname = 'public'
	`
	logSQL(indexQuery)
	rows, err = p.db.Query(indexQuery)
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

	// Get sequences
	sequenceQuery := `
		SELECT sequencename 
		FROM pg_sequences 
		WHERE schemaname = 'public'
	`
	logSQL(sequenceQuery)
	rows, err = p.db.Query(sequenceQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query sequences: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var sequenceName string
		if err := rows.Scan(&sequenceName); err != nil {
			return nil, fmt.Errorf("failed to scan sequence name: %w", err)
		}
		objects = append(objects, DatabaseObject{Type: "sequence", Name: sequenceName})
	}

	return objects, nil
}

func (p *PostgreSQLDatabase) DestroyAllObjects() error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	// Drop all tables, views, and other objects in the correct order
	queries := []string{
		// Drop all functions
		`DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE`,

		// Drop all tables
		`DO $$
		DECLARE
			tbl RECORD;
		BEGIN
			FOR tbl IN
				SELECT tablename FROM pg_tables WHERE schemaname = 'public'
			LOOP
				EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(tbl.tablename) || ' CASCADE';
			END LOOP;
		END $$;`,

		// Drop all views
		`DO $$
		DECLARE
			view RECORD;
		BEGIN
			FOR view IN
				SELECT viewname FROM pg_views WHERE schemaname = 'public'
			LOOP
				EXECUTE 'DROP VIEW IF EXISTS ' || quote_ident(view.viewname) || ' CASCADE';
			END LOOP;
		END $$;`,

		// Drop all sequences
		`DO $$
		DECLARE
			seq RECORD;
		BEGIN
			FOR seq IN
				SELECT sequencename FROM pg_sequences WHERE schemaname = 'public'
			LOOP
				EXECUTE 'DROP SEQUENCE IF EXISTS ' || quote_ident(seq.sequencename) || ' CASCADE';
			END LOOP;
		END $$;`,
	}

	for _, query := range queries {
		logSQL(query)
		if _, err := p.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute destroy query: %w", err)
		}
	}

	return nil
}
