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
			"installed rank" INTEGER,
			"version" VARCHAR(50),
			"description" TEXT,
			"type" VARCHAR(20),
			"script" VARCHAR(1000),
			"checksum" BIGINT,
			"installed by" VARCHAR(100),
			"installed on" TIMESTAMP,
			"execution time" INTEGER,
			"success" INTEGER
		)
	`, tableName)

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
		INSERT INTO %s ("installed rank", "version", "description", "type", "script", "checksum", "installed by", "installed on", "execution time", "success")
		VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, $8, $9)
	`, tableName)

	// Convert version to integer for installed rank
	installedRank := versionToInt(version)
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
		SELECT "installed rank", "version", "description", "type", "script", "checksum", "installed by", "installed on", "execution time", "success"
		FROM %s 
		ORDER BY "installed rank"
	`, tableName)

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
		INSERT INTO %s ("installed rank", "version", "description", "type", "script", "checksum", "installed by", "installed on", "execution time", "success")
		VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, $8, $9)
	`, tableName)

	_, err := p.db.Exec(query, record.InstalledRank, record.Version, record.Description, record.Type, record.Script, record.Checksum, record.InstalledBy, record.ExecutionTime, record.Success)
	if err != nil {
		return fmt.Errorf("failed to insert migration record: %w", err)
	}

	return nil
}

func (p *PostgreSQLDatabase) ExecuteMigration(content string) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	_, err := p.db.Exec(content)
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
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
		if _, err := p.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute destroy query: %w", err)
		}
	}

	return nil
}
