package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepeatableMigrationLoader_LoadRepeatableMigrations(t *testing.T) {
	tempDir := t.TempDir()

	migrationFiles := map[string]string{
		"R__create_users_table.sql":   "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(255));",
		"R__add_indexes.sql":          "CREATE INDEX idx_users_name ON users(name);",
		"R__update_triggers.sql":      "CREATE TRIGGER update_timestamp BEFORE UPDATE ON users SET updated_at = NOW();",
		"R__create_views.sql":         "CREATE VIEW user_view AS SELECT id, name FROM users;",
		"invalid_file.txt":            "This should be ignored",
		"V1__versioned_migration.sql": "This should be ignored",
		"R__missing_extension":        "This should be ignored",
		"__missing_description.sql":   "This should be ignored",
	}

	for filename, content := range migrationFiles {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err, "Failed to create test file %s", filename)
	}

	loader := NewRepeatableMigrationLoader(tempDir)
	migrations, err := loader.LoadRepeatableMigrations()
	require.NoError(t, err, "Expected no error, got %v", err)

	assert.Equal(t, 4, len(migrations), "Expected 4 migrations, got %d", len(migrations))

	expectedDescriptions := []string{"create_users_table", "add_indexes", "update_triggers", "create_views"}
	foundDescriptions := make(map[string]bool)
	for _, migration := range migrations {
		foundDescriptions[migration.Description] = true

		assert.NotEmpty(t, migration.Content, "Expected non-empty content for migration %s", migration.Description)
		assert.NotZero(t, migration.Checksum, "Expected non-zero checksum for migration %s", migration.Description)
		assert.NotEmpty(t, migration.FilePath, "Expected non-empty file path for migration %s", migration.Description)
	}

	for _, expectedDesc := range expectedDescriptions {
		assert.True(t, foundDescriptions[expectedDesc], "Expected to find migration with description '%s'", expectedDesc)
	}
}

func TestRepeatableMigrationLoader_LoadRepeatableMigrations_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	loader := NewRepeatableMigrationLoader(tempDir)
	migrations, err := loader.LoadRepeatableMigrations()
	require.NoError(t, err, "Expected no error, got %v", err)
	assert.Empty(t, migrations, "Expected 0 migrations, got %d", len(migrations))
}

func TestRepeatableMigrationLoader_LoadRepeatableMigrations_NonExistentDirectory(t *testing.T) {
	loader := NewRepeatableMigrationLoader("/non/existent/directory")
	_, err := loader.LoadRepeatableMigrations()
	assert.Error(t, err, "Expected error for non-existent directory")
}

func TestRepeatableMigration_GetFileName(t *testing.T) {
	migration := &RepeatableMigration{
		Description: "create_table",
	}

	expected := "R__create_table.sql"
	actual := migration.GetFileName()
	assert.Equal(t, expected, actual, "Expected filename '%s', got '%s'", expected, actual)
}

func TestRepeatableMigration_String(t *testing.T) {
	migration := &RepeatableMigration{
		Description: "add_column",
	}

	expected := "R__add_column"
	actual := migration.String()
	assert.Equal(t, expected, actual, "Expected string '%s', got '%s'", expected, actual)
}

func TestRepeatableMigrationLoader_FilePatternMatching(t *testing.T) {
	tempDir := t.TempDir()

	testCases := map[string]bool{
		"R__valid.sql":              true,  // Should match
		"R__another_test.sql":       true,  // Should match
		"R__complex-name_123.sql":   true,  // Should match
		"V1__versioned.sql":         false, // Should not match
		"r__lowercase.sql":          false, // Should not match (case sensitive)
		"R__no_extension":           false, // Should not match
		"R_.sql":                    false, // Should not match (missing description)
		"R___double_underscore.sql": true,  // Should match (empty description between underscores)
		"__missing_prefix.sql":      false, // Should not match
	}

	for filename := range testCases {
		content := "CREATE TABLE test (id INT);"
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err, "Failed to create test file %s", filename)
	}

	loader := NewRepeatableMigrationLoader(tempDir)
	migrations, err := loader.LoadRepeatableMigrations()
	require.NoError(t, err, "Expected no error, got %v", err)

	matchedFiles := make(map[string]bool)
	for _, migration := range migrations {
		matchedFiles[migration.GetFileName()] = true
	}

	for filename, expectedMatch := range testCases {
		isMatched := matchedFiles[filename]
		if expectedMatch && !isMatched {
			t.Errorf("Expected file '%s' to be loaded but it wasn't", filename)
		}
		if !expectedMatch && isMatched {
			t.Errorf("Expected file '%s' to be ignored but it was loaded", filename)
		}
	}
}
