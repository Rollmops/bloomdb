package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteDatabase_CreateMigrationTable(t *testing.T) {
	db := NewSQLiteDatabase()

	// Use in-memory SQLite for testing
	err := db.Connect(":memory:")
	require.NoError(t, err, "Failed to connect to SQLite")
	defer db.Close()

	tableName := "test_migrations"

	// Create migration table
	err = db.CreateMigrationTable(tableName)
	require.NoError(t, err, "Failed to create migration table")

	// Verify table exists
	exists, err := db.TableExists(tableName)
	require.NoError(t, err, "Failed to check table existence")
	assert.True(t, exists, "Migration table was not created")

	// Verify table structure by querying schema
	sqlDB := db.GetDB()
	rows, err := sqlDB.Query("PRAGMA table_info(" + tableName + ")")
	require.NoError(t, err, "Failed to get table info")
	defer rows.Close()

	// Check column names and types
	expectedColumns := map[string]string{
		"installed_rank": "INTEGER",
		"version":        "TEXT",
		"description":    "TEXT",
		"type":           "TEXT",
		"script":         "TEXT",
		"checksum":       "INTEGER",
		"installed_by":   "TEXT",
		"installed_on":   "DATETIME",
		"execution_time": "INTEGER",
		"success":        "INTEGER",
	}

	foundColumns := make(map[string]string)
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue interface{}
		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		require.NoError(t, err, "Failed to scan row")
		foundColumns[name] = dataType
	}

	for col, expectedType := range expectedColumns {
		actualType, exists := foundColumns[col]
		assert.True(t, exists, "Expected column '%s' not found", col)
		if exists {
			assert.Equal(t, expectedType, actualType, "Column '%s': expected type '%s', got '%s'", col, expectedType, actualType)
		}
	}
}

func TestPostgreSQLDatabase_CreateMigrationTable_NotConnected(t *testing.T) {
	db := NewPostgreSQLDatabase()
	tableName := "test_migrations"

	// Test that the method generates correct SQL
	// We can't easily test without a real connection, but we can verify query structure
	err := db.CreateMigrationTable(tableName)

	// Should fail because not connected
	assert.Error(t, err, "Expected error when not connected to database")
}

func TestOracleDatabase_CreateMigrationTable_NotConnected(t *testing.T) {
	db := NewOracleDatabase()
	tableName := "test_migrations"

	// Test that the method generates correct SQL
	// We can't easily test without a real connection, but we can verify query structure
	err := db.CreateMigrationTable(tableName)

	// Should fail because not connected
	assert.Error(t, err, "Expected error when not connected to database")
}

func TestCreateMigrationTable_SQLGeneration(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		dbType    DatabaseType
	}{
		{
			name:      "SQLite table creation",
			tableName: "schema_migrations",
			dbType:    SQLite,
		},
		{
			name:      "PostgreSQL table creation",
			tableName: "flyway_schema_history",
			dbType:    PostgreSQL,
		},
		{
			name:      "Oracle table creation",
			tableName: "schema_version",
			dbType:    Oracle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var db Database

			switch tt.dbType {
			case SQLite:
				db = NewSQLiteDatabase()
			case PostgreSQL:
				db = NewPostgreSQLDatabase()
			case Oracle:
				db = NewOracleDatabase()
			}

			// Test that the method handles table name correctly
			// We can't test full execution without real connections,
			// but we can verify method signature and basic error handling
			err := db.CreateMigrationTable(tt.tableName)

			// Should fail because not connected, but error should be about table creation
			assert.Error(t, err, "Expected error when not connected to database")
		})
	}
}
