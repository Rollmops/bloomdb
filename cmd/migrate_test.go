package cmd

import (
	"bloomdb/db"
	"bloomdb/loader"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindGreatestVersion(t *testing.T) {
	tests := []struct {
		name     string
		records  []db.MigrationRecord
		expected string
	}{
		{
			name:     "Empty records",
			records:  []db.MigrationRecord{},
			expected: "",
		},
		{
			name: "Single versioned migration",
			records: []db.MigrationRecord{
				{Version: stringPtr("1.0"), Type: "versioned"},
			},
			expected: "1.0",
		},
		{
			name: "Multiple versioned migrations",
			records: []db.MigrationRecord{
				{Version: stringPtr("1.0"), Type: "versioned"},
				{Version: stringPtr("2.0"), Type: "versioned"},
				{Version: stringPtr("1.5"), Type: "versioned"},
			},
			expected: "2.0",
		},
		{
			name: "With baseline record",
			records: []db.MigrationRecord{
				{Version: stringPtr("1.0"), Type: "baseline"},
				{Version: stringPtr("1.5"), Type: "versioned"},
			},
			expected: "1.5",
		},
		{
			name: "Baseline is greatest",
			records: []db.MigrationRecord{
				{Version: stringPtr("5.0"), Type: "baseline"},
				{Version: stringPtr("1.0"), Type: "versioned"},
			},
			expected: "5.0",
		},
		{
			name: "Skip repeatable migrations",
			records: []db.MigrationRecord{
				{Version: stringPtr("1.0"), Type: "versioned"},
				{Version: nil, Type: "repeatable"},
				{Version: stringPtr(""), Type: "repeatable"},
			},
			expected: "1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findGreatestVersion(tt.records)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindPendingMigrations(t *testing.T) {
	tests := []struct {
		name            string
		migrations      []*loader.VersionedMigration
		greatestVersion string
		expectedCount   int
		expectedFirst   string
	}{
		{
			name:            "No migrations",
			migrations:      []*loader.VersionedMigration{},
			greatestVersion: "1.0",
			expectedCount:   0,
		},
		{
			name: "All migrations pending (no baseline)",
			migrations: []*loader.VersionedMigration{
				{Version: "1.0", Description: "First"},
				{Version: "2.0", Description: "Second"},
			},
			greatestVersion: "",
			expectedCount:   2,
			expectedFirst:   "1.0",
		},
		{
			name: "Some migrations pending",
			migrations: []*loader.VersionedMigration{
				{Version: "1.0", Description: "First"},
				{Version: "2.0", Description: "Second"},
				{Version: "3.0", Description: "Third"},
			},
			greatestVersion: "1.5",
			expectedCount:   2,
			expectedFirst:   "2.0",
		},
		{
			name: "No migrations pending",
			migrations: []*loader.VersionedMigration{
				{Version: "1.0", Description: "First"},
				{Version: "2.0", Description: "Second"},
			},
			greatestVersion: "3.0",
			expectedCount:   0,
		},
		{
			name: "Migration at exact version",
			migrations: []*loader.VersionedMigration{
				{Version: "1.0", Description: "First"},
				{Version: "2.0", Description: "Second"},
			},
			greatestVersion: "1.0",
			expectedCount:   1,
			expectedFirst:   "2.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findPendingMigrations(tt.migrations, tt.greatestVersion)
			assert.Equal(t, tt.expectedCount, len(result))
			if tt.expectedCount > 0 && tt.expectedFirst != "" {
				assert.Equal(t, tt.expectedFirst, result[0].Version)
			}
		})
	}
}

func TestFindPendingRepeatableMigrations(t *testing.T) {
	tests := []struct {
		name          string
		migrations    []*loader.RepeatableMigration
		records       []db.MigrationRecord
		expectedCount int
	}{
		{
			name:          "No migrations",
			migrations:    []*loader.RepeatableMigration{},
			records:       []db.MigrationRecord{},
			expectedCount: 0,
		},
		{
			name: "New repeatable migration",
			migrations: []*loader.RepeatableMigration{
				{Description: "test_view", Checksum: 12345},
			},
			records:       []db.MigrationRecord{},
			expectedCount: 1,
		},
		{
			name: "Existing migration with same checksum",
			migrations: []*loader.RepeatableMigration{
				{Description: "test_view", Checksum: 12345},
			},
			records: []db.MigrationRecord{
				{Version: nil, Description: "test_view", Type: "repeatable", Checksum: int64Ptr(12345)},
			},
			expectedCount: 0,
		},
		{
			name: "Existing migration with different checksum",
			migrations: []*loader.RepeatableMigration{
				{Description: "test_view", Checksum: 99999},
			},
			records: []db.MigrationRecord{
				{Version: nil, Description: "test_view", Type: "repeatable", Checksum: int64Ptr(12345)},
			},
			expectedCount: 1,
		},
		{
			name: "Multiple migrations, mixed states",
			migrations: []*loader.RepeatableMigration{
				{Description: "view1", Checksum: 11111},
				{Description: "view2", Checksum: 22222},
				{Description: "view3", Checksum: 33333},
			},
			records: []db.MigrationRecord{
				{Version: nil, Description: "view1", Type: "repeatable", Checksum: int64Ptr(11111)}, // Same checksum
				{Version: nil, Description: "view2", Type: "repeatable", Checksum: int64Ptr(99999)}, // Different checksum
				// view3 doesn't exist in records
			},
			expectedCount: 2, // view2 (changed) and view3 (new)
		},
		{
			name: "Skip baseline records",
			migrations: []*loader.RepeatableMigration{
				{Description: "test_view", Checksum: 12345},
			},
			records: []db.MigrationRecord{
				{Version: stringPtr("1"), Type: "baseline"},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findPendingRepeatableMigrations(tt.migrations, tt.records)
			assert.Equal(t, tt.expectedCount, len(result))
		})
	}
}

func TestValidateMigrationChecksums(t *testing.T) {
	tests := []struct {
		name               string
		versionedMigs      []*loader.VersionedMigration
		repeatableMigs     []*loader.RepeatableMigration
		records            []db.MigrationRecord
		expectedErrorCount int
	}{
		{
			name:               "No migrations",
			versionedMigs:      []*loader.VersionedMigration{},
			repeatableMigs:     []*loader.RepeatableMigration{},
			records:            []db.MigrationRecord{},
			expectedErrorCount: 0,
		},
		{
			name: "Versioned migration checksum matches",
			versionedMigs: []*loader.VersionedMigration{
				{Version: "1.0", Description: "test", Checksum: 12345},
			},
			repeatableMigs: []*loader.RepeatableMigration{},
			records: []db.MigrationRecord{
				{Version: stringPtr("1.0"), Description: "test", Type: "versioned", Checksum: int64Ptr(12345), Success: 1},
			},
			expectedErrorCount: 0,
		},
		{
			name: "Versioned migration checksum mismatch",
			versionedMigs: []*loader.VersionedMigration{
				{Version: "1.0", Description: "test", Checksum: 99999},
			},
			repeatableMigs: []*loader.RepeatableMigration{},
			records: []db.MigrationRecord{
				{Version: stringPtr("1.0"), Description: "test", Type: "versioned", Checksum: int64Ptr(12345), Success: 1},
			},
			expectedErrorCount: 1,
		},
		{
			name: "Repeatable migration checksum matches",
			versionedMigs: []*loader.VersionedMigration{},
			repeatableMigs: []*loader.RepeatableMigration{
				{Description: "test_view", Checksum: 12345},
			},
			records: []db.MigrationRecord{
				{Version: nil, Description: "test_view", Type: "repeatable", Checksum: int64Ptr(12345), Success: 1},
			},
			expectedErrorCount: 0,
		},
		{
			name: "Repeatable migration checksum mismatch",
			versionedMigs: []*loader.VersionedMigration{},
			repeatableMigs: []*loader.RepeatableMigration{
				{Description: "test_view", Checksum: 99999},
			},
			records: []db.MigrationRecord{
				{Version: nil, Description: "test_view", Type: "repeatable", Checksum: int64Ptr(12345), Success: 1},
			},
			expectedErrorCount: 1,
		},
		{
			name: "Skip baseline records",
			versionedMigs: []*loader.VersionedMigration{},
			repeatableMigs: []*loader.RepeatableMigration{},
			records: []db.MigrationRecord{
				{Version: stringPtr("1.0"), Type: "baseline", Checksum: nil},
			},
			expectedErrorCount: 0,
		},
		{
			name: "Skip failed migrations",
			versionedMigs: []*loader.VersionedMigration{
				{Version: "1.0", Description: "test", Checksum: 99999},
			},
			repeatableMigs: []*loader.RepeatableMigration{},
			records: []db.MigrationRecord{
				{Version: stringPtr("1.0"), Description: "test", Type: "versioned", Checksum: int64Ptr(12345), Success: 0},
			},
			expectedErrorCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validateMigrationChecksums(tt.versionedMigs, tt.repeatableMigs, tt.records)
			assert.Equal(t, tt.expectedErrorCount, len(errors))
		})
	}
}

func TestFindCreatedObjects(t *testing.T) {
	tests := []struct {
		name          string
		before        []db.DatabaseObject
		after         []db.DatabaseObject
		expectedCount int
		expectedName  string
	}{
		{
			name:          "No objects before or after",
			before:        []db.DatabaseObject{},
			after:         []db.DatabaseObject{},
			expectedCount: 0,
		},
		{
			name:   "New table created",
			before: []db.DatabaseObject{},
			after: []db.DatabaseObject{
				{Type: "table", Name: "users"},
			},
			expectedCount: 1,
			expectedName:  "users",
		},
		{
			name: "No new objects",
			before: []db.DatabaseObject{
				{Type: "table", Name: "users"},
			},
			after: []db.DatabaseObject{
				{Type: "table", Name: "users"},
			},
			expectedCount: 0,
		},
		{
			name: "Multiple new objects",
			before: []db.DatabaseObject{
				{Type: "table", Name: "users"},
			},
			after: []db.DatabaseObject{
				{Type: "table", Name: "users"},
				{Type: "table", Name: "posts"},
				{Type: "index", Name: "idx_posts"},
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findCreatedObjects(tt.before, tt.after)
			assert.Equal(t, tt.expectedCount, len(result))
			if tt.expectedCount > 0 && tt.expectedName != "" {
				assert.Equal(t, tt.expectedName, result[0].Name)
			}
		})
	}
}

func TestFindDeletedObjects(t *testing.T) {
	tests := []struct {
		name          string
		before        []db.DatabaseObject
		after         []db.DatabaseObject
		expectedCount int
		expectedName  string
	}{
		{
			name:          "No objects before or after",
			before:        []db.DatabaseObject{},
			after:         []db.DatabaseObject{},
			expectedCount: 0,
		},
		{
			name: "Table deleted",
			before: []db.DatabaseObject{
				{Type: "table", Name: "users"},
			},
			after:         []db.DatabaseObject{},
			expectedCount: 1,
			expectedName:  "users",
		},
		{
			name: "No objects deleted",
			before: []db.DatabaseObject{
				{Type: "table", Name: "users"},
			},
			after: []db.DatabaseObject{
				{Type: "table", Name: "users"},
			},
			expectedCount: 0,
		},
		{
			name: "Multiple objects deleted",
			before: []db.DatabaseObject{
				{Type: "table", Name: "users"},
				{Type: "table", Name: "posts"},
				{Type: "index", Name: "idx_posts"},
			},
			after: []db.DatabaseObject{
				{Type: "table", Name: "users"},
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findDeletedObjects(tt.before, tt.after)
			assert.Equal(t, tt.expectedCount, len(result))
			if tt.expectedCount > 0 && tt.expectedName != "" {
				found := false
				for _, obj := range result {
					if obj.Name == tt.expectedName {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected to find object %s in deleted objects", tt.expectedName)
			}
		})
	}
}

func int64Ptr(i int64) *int64 {
	return &i
}
