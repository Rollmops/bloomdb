package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_LoadMigrations(t *testing.T) {
	tempDir := t.TempDir()

	migrationFiles := map[string]string{
		"V1__create_users_table.sql":      "CREATE TABLE users (id INT PRIMARY KEY);",
		"V1.2.3__add_email_column.sql":    "ALTER TABLE users ADD COLUMN email VARCHAR(255);",
		"V2__create_posts_table.sql":      "CREATE TABLE posts (id INT PRIMARY KEY, user_id INT);",
		"V2.3__add_index.sql":             "CREATE INDEX idx_posts_user_id ON posts(user_id);",
		"V4.2.22.1__latest_migration.sql": "CREATE TABLE latest (id INT);",
		"invalid_file.txt":                "This should be ignored",
		"V__missing_version.sql":          "This should be ignored",
	}

	for filename, content := range migrationFiles {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	loader := NewVersionedMigrationLoader(tempDir)
	migrations, err := loader.LoadMigrations()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(migrations) != 5 {
		t.Errorf("Expected 5 migrations, got %d", len(migrations))
	}

	expectedVersions := []string{"1", "1.2.3", "2", "2.3", "4.2.22.1"}
	for i, migration := range migrations {
		if migration.Version != expectedVersions[i] {
			t.Errorf("Expected version %s at index %d, got %s", expectedVersions[i], i, migration.Version)
		}
	}

	expectedDescriptions := []string{"create_users_table", "add_email_column", "create_posts_table", "add_index", "latest_migration"}
	for i, migration := range migrations {
		if migration.Description != expectedDescriptions[i] {
			t.Errorf("Expected description '%s' at index %d, got '%s'", expectedDescriptions[i], i, migration.Description)
		}
	}
}

func TestLoader_LoadMigrations_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	loader := NewVersionedMigrationLoader(tempDir)
	migrations, err := loader.LoadMigrations()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(migrations) != 0 {
		t.Errorf("Expected 0 migrations, got %d", len(migrations))
	}
}

func TestLoader_LoadMigrations_NonExistentDirectory(t *testing.T) {
	loader := NewVersionedMigrationLoader("/non/existent/directory")
	_, err := loader.LoadMigrations()
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
}

func TestLoader_LoadMigrations_InvalidVersionFormat(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file with invalid version format
	invalidFile := filepath.Join(tempDir, "Vabc__invalid_version.sql")
	err := os.WriteFile(invalidFile, []byte("CREATE TABLE test (id INT);"), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid migration file: %v", err)
	}

	loader := NewVersionedMigrationLoader(tempDir)
	_, err = loader.LoadMigrations()
	if err == nil {
		t.Error("Expected error for invalid version format")
	}

	expectedError := "invalid version format in file Vabc__invalid_version.sql: abc (expected format: 1, 1.2, 1.2.3, etc.)"
	if err.Error() != expectedError {
		t.Errorf("Expected error %q, got %q", expectedError, err.Error())
	}
}

func TestLoader_GetMigrationByVersion(t *testing.T) {
	migrations := []*VersionedMigration{
		{Version: "1", Description: "first"},
		{Version: "2.1", Description: "second"},
		{Version: "3", Description: "third"},
	}

	loader := NewVersionedMigrationLoader(".")
	migration := loader.GetMigrationByVersion(migrations, "2.1")
	if migration == nil {
		t.Error("Expected to find migration with version 2.1")
	}

	if migration.Description != "second" {
		t.Errorf("Expected description 'second', got '%s'", migration.Description)
	}

	missing := loader.GetMigrationByVersion(migrations, "99")
	if missing != nil {
		t.Error("Expected nil for non-existent version")
	}
}

func TestLoader_GetLatestVersion(t *testing.T) {
	tests := []struct {
		name       string
		migrations []*VersionedMigration
		expected   string
	}{
		{
			name:       "Empty migrations",
			migrations: []*VersionedMigration{},
			expected:   "",
		},
		{
			name: "Single migration",
			migrations: []*VersionedMigration{
				{Version: "5", Description: "single"},
			},
			expected: "5",
		},
		{
			name: "Multiple migrations",
			migrations: []*VersionedMigration{
				{Version: "1", Description: "first"},
				{Version: "3.2", Description: "third"},
				{Version: "2", Description: "second"},
			},
			expected: "3.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewVersionedMigrationLoader(".")
			latest := loader.GetLatestVersion(tt.migrations)
			if latest != tt.expected {
				t.Errorf("Expected latest version %s, got %s", tt.expected, latest)
			}
		})
	}
}

func TestVersionedMigration_GetFileName(t *testing.T) {
	migration := &VersionedMigration{
		Version:     "1.2.3",
		Description: "create_table",
	}

	expected := "V1.2.3__create_table.sql"
	actual := migration.GetFileName()

	if actual != expected {
		t.Errorf("Expected filename '%s', got '%s'", expected, actual)
	}
}

func TestVersionedMigration_String(t *testing.T) {
	migration := &VersionedMigration{
		Version:     "10.5",
		Description: "add_column",
	}

	expected := "V10.5__add_column"
	actual := migration.String()

	if actual != expected {
		t.Errorf("Expected string '%s', got '%s'", expected, actual)
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{"Equal versions", "1.2.3", "1.2.3", 0},
		{"v1 less than v2", "1.2", "1.2.3", -1},
		{"v1 greater than v2", "2.0", "1.9.9", 1},
		{"Different lengths", "1.2", "1.2.0.0", 0},
		{"Single digit", "1", "1.0.0", 0},
		{"Complex versions", "4.2.22.1", "4.2.22", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("Expected compareVersions(%s, %s) = %d, got %d", tt.v1, tt.v2, tt.expected, result)
			}
		})
	}
}

func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		version  string
		expected bool
	}{
		// Valid versions
		{"1", true},
		{"2", true},
		{"10", true},
		{"1.2", true},
		{"2.3", true},
		{"10.20", true},
		{"1.2.3", true},
		{"2.3.4", true},
		{"10.20.30", true},
		{"1.2.3.4", true},
		{"1.2.3.4.5", true},
		{"0", true},
		{"0.1", true},
		{"0.0.1", true},

		// Invalid versions
		{"", false},
		{"abc", false},
		{"1.2.3abc", false},
		{"v1.2.3", false},
		{"1.2.", false},
		{".1.2", false},
		{"1..2", false},
		{"1.2.3.", false},
		{"1.2..3", false},
		{"1.2.3.4.", false},
		{"1.2.3a", false},
		{"a1.2.3", false},
		{"1.2.3-alpha", false},
		{"1.2.3+build", false},
		{"1.2.3.4.5.6.7.8.9.10", true}, // Many parts but still valid format
		{" 1.2.3", false},
		{"1.2.3 ", false},
		{"1. 2.3", false},
		{"1.2 .3", false},
		{"1.2.3 ", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := IsValidVersion(tt.version)
			if result != tt.expected {
				t.Errorf("Expected IsValidVersion(%q) = %v, got %v", tt.version, tt.expected, result)
			}
		})
	}
}
