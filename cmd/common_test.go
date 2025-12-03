package cmd

import (
	"bloomdb/db"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindBaselineVersion(t *testing.T) {
	tests := []struct {
		name     string
		records  []db.MigrationRecord
		expected string
	}{
		{
			name:     "No records",
			records:  []db.MigrationRecord{},
			expected: "",
		},
		{
			name: "Single baseline record",
			records: []db.MigrationRecord{
				{Version: stringPtr("1"), Type: "BASELINE"},
			},
			expected: "1",
		},
		{
			name: "Multiple records with baseline",
			records: []db.MigrationRecord{
				{Version: stringPtr("1"), Type: "BASELINE"},
				{Version: stringPtr("2"), Type: "versioned"},
				{Version: stringPtr("3"), Type: "versioned"},
			},
			expected: "1",
		},
		{
			name: "No baseline record",
			records: []db.MigrationRecord{
				{Version: stringPtr("1"), Type: "versioned"},
				{Version: stringPtr("2"), Type: "versioned"},
			},
			expected: "",
		},
		{
			name: "Baseline with nil version",
			records: []db.MigrationRecord{
				{Version: nil, Type: "BASELINE"},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindBaselineVersion(tt.records)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateNextRank(t *testing.T) {
	tests := []struct {
		name     string
		records  []db.MigrationRecord
		expected int
	}{
		{
			name:     "No records",
			records:  []db.MigrationRecord{},
			expected: 1,
		},
		{
			name: "Single record",
			records: []db.MigrationRecord{
				{InstalledRank: 1},
			},
			expected: 2,
		},
		{
			name: "Multiple records",
			records: []db.MigrationRecord{
				{InstalledRank: 1},
				{InstalledRank: 2},
				{InstalledRank: 5},
				{InstalledRank: 3},
			},
			expected: 6,
		},
		{
			name: "Non-sequential ranks",
			records: []db.MigrationRecord{
				{InstalledRank: 10},
				{InstalledRank: 20},
				{InstalledRank: 15},
			},
			expected: 21,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateNextRank(tt.records)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
