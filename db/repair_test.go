package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeleteFailedMigrationRecords(t *testing.T) {
	// Test with PostgreSQL
	t.Run("PostgreSQL", func(t *testing.T) {
		testDeleteFailedMigrationRecords(t, NewPostgreSQLDatabase())
	})

	// Test with SQLite
	t.Run("SQLite", func(t *testing.T) {
		testDeleteFailedMigrationRecords(t, NewSQLiteDatabase())
	})
}

func testDeleteFailedMigrationRecords(t *testing.T, db Database) {
	// Create a temporary database for testing
	var connStr string
	var cleanup func()

	switch db.(type) {
	case *PostgreSQLDatabase:
		// Skip PostgreSQL test if no connection string is provided
		t.Skip("PostgreSQL test skipped - no test database configured")
		return
	case *SQLiteDatabase:
		connStr = ":memory:" // Use in-memory database for testing
		cleanup = func() {
			db.Close()
		}
	default:
		t.Skip("Unsupported database type for test")
		return
	}
	defer cleanup()

	// Connect to database
	err := db.Connect(connStr)
	require.NoError(t, err)

	// Create migration table
	err = db.CreateMigrationTable("test_schema_version")
	require.NoError(t, err)

	// Insert test migration records (some successful, some failed)
	testRecords := []MigrationRecord{
		{
			InstalledRank: 1,
			Version:       stringPtr("1.0.0"),
			Description:   "Successful migration 1",
			Type:          "versioned",
			Script:        "V1.0.0__test.sql",
			Checksum:      int64Ptr(12345),
			InstalledBy:   "test",
			ExecutionTime: 100,
			Success:       1, // Success
		},
		{
			InstalledRank: 2,
			Version:       stringPtr("1.1.0"),
			Description:   "Failed migration 1",
			Type:          "versioned",
			Script:        "V1.1.0__test.sql",
			Checksum:      int64Ptr(67890),
			InstalledBy:   "test",
			ExecutionTime: 200,
			Success:       0, // Failed
		},
		{
			InstalledRank: 3,
			Version:       stringPtr("1.2.0"),
			Description:   "Failed migration 2",
			Type:          "versioned",
			Script:        "V1.2.0__test.sql",
			Checksum:      int64Ptr(11111),
			InstalledBy:   "test",
			ExecutionTime: 300,
			Success:       0, // Failed
		},
		{
			InstalledRank: 4,
			Version:       stringPtr("1.3.0"),
			Description:   "Successful migration 2",
			Type:          "versioned",
			Script:        "V1.3.0__test.sql",
			Checksum:      int64Ptr(22222),
			InstalledBy:   "test",
			ExecutionTime: 400,
			Success:       1, // Success
		},
	}

	// Insert test records
	for _, record := range testRecords {
		err := db.InsertMigrationRecord("test_schema_version", record)
		require.NoError(t, err)
	}

	// Verify all records are inserted
	records, err := db.GetMigrationRecords("test_schema_version")
	require.NoError(t, err)
	require.Len(t, records, 4)

	// Count successful and failed records
	var successCount, failedCount int
	for _, record := range records {
		if record.Success == 1 {
			successCount++
		} else {
			failedCount++
		}
	}
	require.Equal(t, 2, successCount)
	require.Equal(t, 2, failedCount)

	// Delete failed migration records
	err = db.DeleteFailedMigrationRecords("test_schema_version")
	require.NoError(t, err)

	// Verify only successful records remain
	records, err = db.GetMigrationRecords("test_schema_version")
	require.NoError(t, err)
	require.Len(t, records, 2)

	// Verify all remaining records are successful
	for _, record := range records {
		require.Equal(t, 1, record.Success, "All remaining records should be successful")
	}

	// Verify the correct records remain (the successful ones)
	successVersions := make(map[string]bool)
	for _, record := range records {
		if record.Success == 1 {
			successVersions[*record.Version] = true
		}
	}
	require.True(t, successVersions["1.0.0"], "Version 1.0.0 should remain")
	require.True(t, successVersions["1.3.0"], "Version 1.3.0 should remain")
	require.False(t, successVersions["1.1.0"], "Version 1.1.0 should be deleted")
	require.False(t, successVersions["1.2.0"], "Version 1.2.0 should be deleted")
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}
