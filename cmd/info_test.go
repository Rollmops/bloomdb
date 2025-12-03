package cmd

import (
	"bloomdb/db"
	"bloomdb/loader"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildMigrationStatuses(t *testing.T) {
	tests := []struct {
		name                string
		versionedMigs       []*loader.VersionedMigration
		repeatableMigs      []*loader.RepeatableMigration
		records             []db.MigrationRecord
		baselineVersion     string
		expectedCount       int
		expectedFirstStatus string
	}{
		{
			name:                "No migrations or records",
			versionedMigs:       []*loader.VersionedMigration{},
			repeatableMigs:      []*loader.RepeatableMigration{},
			records:             []db.MigrationRecord{},
			baselineVersion:     "",
			expectedCount:       0,
			expectedFirstStatus: "",
		},
		{
			name: "Pending versioned migration",
			versionedMigs: []*loader.VersionedMigration{
				{Version: "1.0", Description: "test", Checksum: 12345},
			},
			repeatableMigs:      []*loader.RepeatableMigration{},
			records:             []db.MigrationRecord{},
			baselineVersion:     "",
			expectedCount:       1,
			expectedFirstStatus: "pending",
		},
		{
			name: "Applied versioned migration",
			versionedMigs: []*loader.VersionedMigration{
				{Version: "1.0", Description: "test", Checksum: 12345},
			},
			repeatableMigs: []*loader.RepeatableMigration{},
			records: []db.MigrationRecord{
				{Version: stringPtr("1.0"), Description: "test", Type: "versioned", Checksum: int64Ptr(12345), Success: 1},
			},
			baselineVersion:     "",
			expectedCount:       1,
			expectedFirstStatus: "success",
		},
		{
			name: "Failed versioned migration",
			versionedMigs: []*loader.VersionedMigration{
				{Version: "1.0", Description: "test", Checksum: 12345},
			},
			repeatableMigs: []*loader.RepeatableMigration{},
			records: []db.MigrationRecord{
				{Version: stringPtr("1.0"), Description: "test", Type: "versioned", Checksum: int64Ptr(12345), Success: 0},
			},
			baselineVersion:     "",
			expectedCount:       1,
			expectedFirstStatus: "failed",
		},
		{
			name: "Checksum mismatch",
			versionedMigs: []*loader.VersionedMigration{
				{Version: "1.0", Description: "test", Checksum: 99999},
			},
			repeatableMigs: []*loader.RepeatableMigration{},
			records: []db.MigrationRecord{
				{Version: stringPtr("1.0"), Description: "test", Type: "versioned", Checksum: int64Ptr(12345), Success: 1},
			},
			baselineVersion:     "",
			expectedCount:       1,
			expectedFirstStatus: "checksum",
		},
		{
			name: "Below baseline",
			versionedMigs: []*loader.VersionedMigration{
				{Version: "1.0", Description: "test", Checksum: 12345},
			},
			repeatableMigs:      []*loader.RepeatableMigration{},
			records:             []db.MigrationRecord{},
			baselineVersion:     "2.0",
			expectedCount:       1,
			expectedFirstStatus: "below baseline",
		},
		{
			name: "At baseline version",
			versionedMigs: []*loader.VersionedMigration{
				{Version: "1.0", Description: "test", Checksum: 12345},
			},
			repeatableMigs:      []*loader.RepeatableMigration{},
			records:             []db.MigrationRecord{},
			baselineVersion:     "1.0",
			expectedCount:       1,
			expectedFirstStatus: "below baseline",
		},
		{
			name: "Pending repeatable migration",
			versionedMigs: []*loader.VersionedMigration{},
			repeatableMigs: []*loader.RepeatableMigration{
				{Description: "test_view", Checksum: 12345},
			},
			records:             []db.MigrationRecord{},
			baselineVersion:     "",
			expectedCount:       1,
			expectedFirstStatus: "pending",
		},
		{
			name: "Applied repeatable migration",
			versionedMigs: []*loader.VersionedMigration{},
			repeatableMigs: []*loader.RepeatableMigration{
				{Description: "test_view", Checksum: 12345},
			},
			records: []db.MigrationRecord{
				{Version: nil, Description: "test_view", Type: "repeatable", Checksum: int64Ptr(12345), Success: 1},
			},
			baselineVersion:     "",
			expectedCount:       1,
			expectedFirstStatus: "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildMigrationStatuses(tt.versionedMigs, tt.repeatableMigs, tt.records, tt.baselineVersion)
			assert.Equal(t, tt.expectedCount, len(result))
			if tt.expectedCount > 0 && tt.expectedFirstStatus != "" {
				assert.Equal(t, tt.expectedFirstStatus, result[0].Status)
			}
		})
	}
}

func TestValidateVersionedMigration(t *testing.T) {
	tests := []struct {
		name           string
		record         db.MigrationRecord
		migration      *loader.VersionedMigration
		expectedStatus string
	}{
		{
			name: "Checksum matches",
			record: db.MigrationRecord{
				Version:  stringPtr("1.0"),
				Checksum: int64Ptr(12345),
			},
			migration: &loader.VersionedMigration{
				Version:  "1.0",
				Checksum: 12345,
			},
			expectedStatus: "",
		},
		{
			name: "Checksum mismatch",
			record: db.MigrationRecord{
				Version:  stringPtr("1.0"),
				Checksum: int64Ptr(12345),
			},
			migration: &loader.VersionedMigration{
				Version:  "1.0",
				Checksum: 99999,
			},
			expectedStatus: "checksum",
		},
		{
			name: "Nil checksum in record",
			record: db.MigrationRecord{
				Version:  stringPtr("1.0"),
				Checksum: nil,
			},
			migration: &loader.VersionedMigration{
				Version:  "1.0",
				Checksum: 12345,
			},
			expectedStatus: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateVersionedMigration(tt.record, tt.migration)
			assert.Equal(t, tt.expectedStatus, result)
		})
	}
}

func TestValidateRepeatableMigration(t *testing.T) {
	tests := []struct {
		name           string
		record         db.MigrationRecord
		migration      *loader.RepeatableMigration
		expectedStatus string
	}{
		{
			name: "Checksum matches",
			record: db.MigrationRecord{
				Description: "test_view",
				Checksum:    int64Ptr(12345),
			},
			migration: &loader.RepeatableMigration{
				Description: "test_view",
				Checksum:    12345,
			},
			expectedStatus: "",
		},
		{
			name: "Checksum mismatch",
			record: db.MigrationRecord{
				Description: "test_view",
				Checksum:    int64Ptr(12345),
			},
			migration: &loader.RepeatableMigration{
				Description: "test_view",
				Checksum:    99999,
			},
			expectedStatus: "checksum",
		},
		{
			name: "Nil checksum in record",
			record: db.MigrationRecord{
				Description: "test_view",
				Checksum:    nil,
			},
			migration: &loader.RepeatableMigration{
				Description: "test_view",
				Checksum:    12345,
			},
			expectedStatus: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateRepeatableMigration(tt.record, tt.migration)
			assert.Equal(t, tt.expectedStatus, result)
		})
	}
}
