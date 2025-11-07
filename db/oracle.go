package db

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/sijms/go-ora/v2"
)

type OracleDatabase struct {
	db *sql.DB
}

func NewOracleDatabase() *OracleDatabase {
	return &OracleDatabase{}
}

func (o *OracleDatabase) Connect(connectionString string) error {
	db, err := sql.Open("oracle", connectionString)
	if err != nil {
		return fmt.Errorf("failed to connect to Oracle: %w", err)
	}
	o.db = db
	return nil
}

func (o *OracleDatabase) Close() error {
	if o.db != nil {
		return o.db.Close()
	}
	return nil
}

func (o *OracleDatabase) Ping() error {
	if o.db == nil {
		return fmt.Errorf("database not connected")
	}
	return o.db.Ping()
}

func (o *OracleDatabase) GetDB() *sql.DB {
	return o.db
}

func (o *OracleDatabase) TableExists(tableName string) (bool, error) {
	query := "SELECT table_name FROM user_tables WHERE table_name = UPPER(?)"
	var result string
	err := o.db.QueryRow(query, strings.ToUpper(tableName)).Scan(&result)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return false, nil
		}
		return false, fmt.Errorf("error checking table existence: %w", err)
	}
	return true, nil
}

func (o *OracleDatabase) CreateMigrationTable(tableName string) error {
	if o.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		CREATE TABLE %s (
			"installed rank" NUMBER,
			"version" VARCHAR2(50),
			"description" VARCHAR2(4000),
			"type" VARCHAR2(20),
			"script" VARCHAR2(1000),
			"checksum" NUMBER,
			"installed by" VARCHAR2(100),
			"installed on" TIMESTAMP,
			"execution time" NUMBER,
			"success" NUMBER
		)
	`, tableName)

	_, err := o.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create migration table %s: %w", tableName, err)
	}

	return nil
}

func (o *OracleDatabase) InsertBaselineRecord(tableName, version string) error {
	if o.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		INSERT INTO %s ("installed rank", "version", "description", "type", "script", "checksum", "installed by", "installed on", "execution time", "success")
		VALUES (:1, :2, :3, :4, :5, :6, :7, CURRENT_TIMESTAMP, :8, :9)
	`, tableName)

	// Convert version to integer for installed rank
	installedRank := versionToInt(version)
	_, err := o.db.Exec(query, installedRank, version, "<< Baseline >>", "BASELINE", "<< Baseline >>", nil, "bloomdb", 0, 1)
	if err != nil {
		return fmt.Errorf("failed to insert baseline record: %w", err)
	}

	return nil
}

func (o *OracleDatabase) GetMigrationRecords(tableName string) ([]MigrationRecord, error) {
	if o.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		SELECT "installed rank", "version", "description", "type", "script", "checksum", "installed by", "installed on", "execution time", "success"
		FROM %s 
		ORDER BY "installed rank"
	`, tableName)

	rows, err := o.db.Query(query)
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

func (o *OracleDatabase) InsertMigrationRecord(tableName string, record MigrationRecord) error {
	if o.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := fmt.Sprintf(`
		INSERT INTO %s ("installed rank", "version", "description", "type", "script", "checksum", "installed by", "installed on", "execution time", "success")
		VALUES (:1, :2, :3, :4, :5, :6, :7, CURRENT_TIMESTAMP, :8, :9)
	`, tableName)

	_, err := o.db.Exec(query, record.InstalledRank, record.Version, record.Description, record.Type, record.Script, record.Checksum, record.InstalledBy, record.ExecutionTime, record.Success)
	if err != nil {
		return fmt.Errorf("failed to insert migration record: %w", err)
	}

	return nil
}

func (o *OracleDatabase) ExecuteMigration(content string) error {
	if o.db == nil {
		return fmt.Errorf("database not connected")
	}

	_, err := o.db.Exec(content)
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}

func (o *OracleDatabase) GetDatabaseObjects() ([]DatabaseObject, error) {
	if o.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	var objects []DatabaseObject

	// Get tables
	tableQuery := "SELECT table_name FROM user_tables"
	rows, err := o.db.Query(tableQuery)
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
	viewQuery := "SELECT view_name FROM user_views"
	rows, err = o.db.Query(viewQuery)
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
	indexQuery := "SELECT index_name, table_name FROM user_indexes"
	rows, err = o.db.Query(indexQuery)
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
	sequenceQuery := "SELECT sequence_name FROM user_sequences"
	rows, err = o.db.Query(sequenceQuery)
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

	// Get procedures
	procedureQuery := "SELECT object_name FROM user_procedures"
	rows, err = o.db.Query(procedureQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query procedures: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var procedureName string
		if err := rows.Scan(&procedureName); err != nil {
			return nil, fmt.Errorf("failed to scan procedure name: %w", err)
		}
		objects = append(objects, DatabaseObject{Type: "procedure", Name: procedureName})
	}

	// Get functions
	functionQuery := "SELECT object_name FROM user_objects WHERE object_type = 'FUNCTION'"
	rows, err = o.db.Query(functionQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query functions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var functionName string
		if err := rows.Scan(&functionName); err != nil {
			return nil, fmt.Errorf("failed to scan function name: %w", err)
		}
		objects = append(objects, DatabaseObject{Type: "function", Name: functionName})
	}

	return objects, nil
}

func (o *OracleDatabase) DestroyAllObjects() error {
	if o.db == nil {
		return fmt.Errorf("database not connected")
	}

	// Oracle-specific destroy logic
	queries := []string{
		// Drop all tables
		`BEGIN
			FOR tbl IN (SELECT table_name FROM user_tables) LOOP
				EXECUTE IMMEDIATE 'DROP TABLE ' || tbl.table_name || ' CASCADE CONSTRAINTS';
			END LOOP;
		END;`,

		// Drop all views
		`BEGIN
			FOR view IN (SELECT view_name FROM user_views) LOOP
				EXECUTE IMMEDIATE 'DROP VIEW ' || view.view_name;
			END LOOP;
		END;`,

		// Drop all sequences
		`BEGIN
			FOR seq IN (SELECT sequence_name FROM user_sequences) LOOP
				EXECUTE IMMEDIATE 'DROP SEQUENCE ' || seq.sequence_name;
			END LOOP;
		END;`,

		// Drop all procedures
		`BEGIN
			FOR proc IN (SELECT object_name FROM user_procedures) LOOP
				EXECUTE IMMEDIATE 'DROP PROCEDURE ' || proc.object_name;
			END LOOP;
		END;`,

		// Drop all functions
		`BEGIN
			FOR func IN (SELECT object_name FROM user_objects WHERE object_type = 'FUNCTION') LOOP
				EXECUTE IMMEDIATE 'DROP FUNCTION ' || func.object_name;
			END LOOP;
		END;`,
	}

	for _, query := range queries {
		if _, err := o.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute destroy query: %w", err)
		}
	}

	return nil
}
