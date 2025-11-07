package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFilterConfig_NoFilter(t *testing.T) {
	os.Unsetenv("BLOOMDB_FILTER_HARD")
	os.Unsetenv("BLOOMDB_FILTER_SOFT")

	config := GetFilterConfig()

	assert.Equal(t, NoFilter, config.Mode)
	assert.Equal(t, "", config.Filter)
}

func TestGetFilterConfig_HardFilter(t *testing.T) {
	os.Setenv("BLOOMDB_FILTER_HARD", "postgres")
	defer os.Unsetenv("BLOOMDB_FILTER_HARD")
	os.Unsetenv("BLOOMDB_FILTER_SOFT")

	config := GetFilterConfig()

	assert.Equal(t, HardFilter, config.Mode)
	assert.Equal(t, "postgres", config.Filter)
}

func TestGetFilterConfig_SoftFilter(t *testing.T) {
	os.Unsetenv("BLOOMDB_FILTER_HARD")
	os.Setenv("BLOOMDB_FILTER_SOFT", "sqlite")
	defer os.Unsetenv("BLOOMDB_FILTER_SOFT")

	config := GetFilterConfig()

	assert.Equal(t, SoftFilter, config.Mode)
	assert.Equal(t, "sqlite", config.Filter)
}

func TestGetFilterConfig_HardFilterTakesPrecedence(t *testing.T) {
	os.Setenv("BLOOMDB_FILTER_HARD", "postgres")
	os.Setenv("BLOOMDB_FILTER_SOFT", "sqlite")
	defer os.Unsetenv("BLOOMDB_FILTER_HARD")
	defer os.Unsetenv("BLOOMDB_FILTER_SOFT")

	config := GetFilterConfig()

	assert.Equal(t, HardFilter, config.Mode)
	assert.Equal(t, "postgres", config.Filter)
}

func TestParseMigrationFilename_VersionedNoFilter(t *testing.T) {
	file, err := ParseMigrationFilename("V1.0__create_users.sql")

	require.NoError(t, err)
	assert.Equal(t, "V1.0__create_users.sql", file.Filename)
	assert.Equal(t, "1.0", file.Version)
	assert.Equal(t, "create_users", file.Description)
	assert.Equal(t, "", file.Filter)
	assert.False(t, file.IsRepeatable)
}

func TestParseMigrationFilename_VersionedWithFilter(t *testing.T) {
	file, err := ParseMigrationFilename("V1.0__create_users.postgres.sql")

	require.NoError(t, err)
	assert.Equal(t, "V1.0__create_users.postgres.sql", file.Filename)
	assert.Equal(t, "1.0", file.Version)
	assert.Equal(t, "create_users", file.Description)
	assert.Equal(t, "postgres", file.Filter)
	assert.False(t, file.IsRepeatable)
}

func TestParseMigrationFilename_RepeatableNoFilter(t *testing.T) {
	file, err := ParseMigrationFilename("R__create_views.sql")

	require.NoError(t, err)
	assert.Equal(t, "R__create_views.sql", file.Filename)
	assert.Equal(t, "", file.Version)
	assert.Equal(t, "create_views", file.Description)
	assert.Equal(t, "", file.Filter)
	assert.True(t, file.IsRepeatable)
}

func TestParseMigrationFilename_RepeatableWithFilter(t *testing.T) {
	file, err := ParseMigrationFilename("R__create_views.oracle.sql")

	require.NoError(t, err)
	assert.Equal(t, "R__create_views.oracle.sql", file.Filename)
	assert.Equal(t, "", file.Version)
	assert.Equal(t, "create_views", file.Description)
	assert.Equal(t, "oracle", file.Filter)
	assert.True(t, file.IsRepeatable)
}

func TestParseMigrationFilename_InvalidFormat(t *testing.T) {
	invalidFiles := []string{
		"invalid.sql",
		"V1.0_missing_double_underscore.sql",
		"R_missing_double_underscore.sql",
		"V__missing_version.sql",
		"notamigration.txt",
	}

	for _, filename := range invalidFiles {
		t.Run(filename, func(t *testing.T) {
			_, err := ParseMigrationFilename(filename)
			assert.Error(t, err, "Expected error for %s", filename)
		})
	}
}

func TestParseMigrationFilename_ComplexVersions(t *testing.T) {
	tests := []struct {
		filename    string
		version     string
		description string
		filter      string
	}{
		{"V1__simple.sql", "1", "simple", ""},
		{"V1.2__two_part.sql", "1.2", "two_part", ""},
		{"V1.2.3__three_part.sql", "1.2.3", "three_part", ""},
		{"V2.10.15__high_numbers.postgres.sql", "2.10.15", "high_numbers", "postgres"},
		{"V0.1__initial.sqlite.sql", "0.1", "initial", "sqlite"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			file, err := ParseMigrationFilename(tt.filename)
			require.NoError(t, err)
			assert.Equal(t, tt.version, file.Version)
			assert.Equal(t, tt.description, file.Description)
			assert.Equal(t, tt.filter, file.Filter)
		})
	}
}

func TestCollectFilteredMigrationFiles_NoFilter(t *testing.T) {
	tempDir := t.TempDir()

	files := map[string]string{
		"V1.0__create_users.sql":          "CREATE TABLE users;",
		"V1.0__create_users.postgres.sql": "CREATE TABLE users (id SERIAL);",
		"V2.0__add_column.sql":            "ALTER TABLE users ADD COLUMN email VARCHAR;",
		"R__create_views.sql":             "CREATE VIEW user_view AS SELECT * FROM users;",
		"R__create_views.oracle.sql":      "CREATE VIEW user_view AS SELECT * FROM users;",
	}

	for filename, content := range files {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	config := FilterConfig{Mode: NoFilter}
	collected, err := CollectFilteredMigrationFiles(tempDir, config)

	require.NoError(t, err)
	assert.Equal(t, 3, len(collected), "Should collect only non-filtered files")

	// Verify collected filenames
	filenames := make(map[string]bool)
	for _, file := range collected {
		filenames[file.Filename] = true
	}
	assert.True(t, filenames["V1.0__create_users.sql"])
	assert.True(t, filenames["V2.0__add_column.sql"])
	assert.True(t, filenames["R__create_views.sql"])
	assert.False(t, filenames["V1.0__create_users.postgres.sql"])
	assert.False(t, filenames["R__create_views.oracle.sql"])
}

func TestCollectFilteredMigrationFiles_HardFilter(t *testing.T) {
	tempDir := t.TempDir()

	files := map[string]string{
		"V1.0__create_users.sql":          "CREATE TABLE users;",
		"V1.0__create_users.postgres.sql": "CREATE TABLE users (id SERIAL);",
		"V2.0__add_column.postgres.sql":   "ALTER TABLE users ADD COLUMN email VARCHAR;",
		"R__create_views.sql":             "CREATE VIEW user_view AS SELECT * FROM users;",
		"R__create_views.postgres.sql":    "CREATE VIEW user_view AS SELECT * FROM users;",
	}

	for filename, content := range files {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	config := FilterConfig{Mode: HardFilter, Filter: "postgres"}
	collected, err := CollectFilteredMigrationFiles(tempDir, config)

	require.NoError(t, err)
	assert.Equal(t, 3, len(collected), "Should collect only postgres-filtered files")

	// Verify collected filenames
	filenames := make(map[string]bool)
	for _, file := range collected {
		filenames[file.Filename] = true
		assert.Equal(t, "postgres", file.Filter, "All files should have postgres filter")
	}
	assert.True(t, filenames["V1.0__create_users.postgres.sql"])
	assert.True(t, filenames["V2.0__add_column.postgres.sql"])
	assert.True(t, filenames["R__create_views.postgres.sql"])
}

func TestCollectFilteredMigrationFiles_SoftFilter_PreferFiltered(t *testing.T) {
	tempDir := t.TempDir()

	files := map[string]string{
		"V1.0__create_users.sql":          "CREATE TABLE users;",
		"V1.0__create_users.postgres.sql": "CREATE TABLE users (id SERIAL);",
		"V2.0__add_column.sql":            "ALTER TABLE users ADD COLUMN email VARCHAR;",
		"R__create_views.sql":             "CREATE VIEW user_view AS SELECT * FROM users;",
		"R__create_views.postgres.sql":    "CREATE VIEW user_view AS SELECT * FROM users;",
	}

	for filename, content := range files {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	config := FilterConfig{Mode: SoftFilter, Filter: "postgres"}
	collected, err := CollectFilteredMigrationFiles(tempDir, config)

	require.NoError(t, err)
	assert.Equal(t, 3, len(collected))

	// Verify that postgres versions are preferred when available
	filenames := make(map[string]bool)
	for _, file := range collected {
		filenames[file.Filename] = true
	}
	// V1.0 should use postgres version (available)
	assert.True(t, filenames["V1.0__create_users.postgres.sql"])
	assert.False(t, filenames["V1.0__create_users.sql"])

	// V2.0 should use non-filtered version (no postgres version available)
	assert.True(t, filenames["V2.0__add_column.sql"])

	// R should use postgres version (available)
	assert.True(t, filenames["R__create_views.postgres.sql"])
	assert.False(t, filenames["R__create_views.sql"])
}

func TestCollectFilteredMigrationFiles_SoftFilter_FallbackToNonFiltered(t *testing.T) {
	tempDir := t.TempDir()

	files := map[string]string{
		"V1.0__create_users.sql":     "CREATE TABLE users;",
		"V2.0__add_column.sql":       "ALTER TABLE users ADD COLUMN email VARCHAR;",
		"V2.0__add_column.mysql.sql": "ALTER TABLE users ADD COLUMN email VARCHAR(255);",
		"R__create_views.sql":        "CREATE VIEW user_view AS SELECT * FROM users;",
	}

	for filename, content := range files {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	config := FilterConfig{Mode: SoftFilter, Filter: "postgres"}
	collected, err := CollectFilteredMigrationFiles(tempDir, config)

	require.NoError(t, err)
	assert.Equal(t, 3, len(collected))

	// All should fallback to non-filtered since no postgres versions exist
	filenames := make(map[string]bool)
	for _, file := range collected {
		filenames[file.Filename] = true
		assert.Equal(t, "", file.Filter, "All files should have no filter (fallback)")
	}
	assert.True(t, filenames["V1.0__create_users.sql"])
	assert.True(t, filenames["V2.0__add_column.sql"])
	assert.True(t, filenames["R__create_views.sql"])
	assert.False(t, filenames["V2.0__add_column.mysql.sql"])
}

func TestCollectFilteredMigrationFiles_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	config := FilterConfig{Mode: NoFilter}
	collected, err := CollectFilteredMigrationFiles(tempDir, config)

	require.NoError(t, err)
	assert.Empty(t, collected)
}

func TestCollectFilteredMigrationFiles_NonExistentDirectory(t *testing.T) {
	config := FilterConfig{Mode: NoFilter}
	_, err := CollectFilteredMigrationFiles("/non/existent/directory", config)

	assert.Error(t, err)
}

func TestCollectFilteredMigrationFiles_IgnoresInvalidFiles(t *testing.T) {
	tempDir := t.TempDir()

	files := map[string]string{
		"V1.0__valid.sql":        "CREATE TABLE users;",
		"invalid.txt":            "Not a migration",
		"README.md":              "Documentation",
		"V__missing_version.sql": "Invalid version",
	}

	for filename, content := range files {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	config := FilterConfig{Mode: NoFilter}
	collected, err := CollectFilteredMigrationFiles(tempDir, config)

	require.NoError(t, err)
	assert.Equal(t, 1, len(collected), "Should only collect valid migration file")
	assert.Equal(t, "V1.0__valid.sql", collected[0].Filename)
}

func TestCollectFilteredMigrationFiles_MultipleFilters(t *testing.T) {
	tempDir := t.TempDir()

	files := map[string]string{
		"V1.0__create_users.sql":          "CREATE TABLE users;",
		"V1.0__create_users.postgres.sql": "CREATE TABLE users (id SERIAL);",
		"V1.0__create_users.mysql.sql":    "CREATE TABLE users (id INT AUTO_INCREMENT);",
		"V1.0__create_users.oracle.sql":   "CREATE TABLE users (id NUMBER GENERATED AS IDENTITY);",
	}

	for filename, content := range files {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Test with each filter
	filters := []string{"postgres", "mysql", "oracle"}
	for _, filter := range filters {
		t.Run(filter, func(t *testing.T) {
			config := FilterConfig{Mode: HardFilter, Filter: filter}
			collected, err := CollectFilteredMigrationFiles(tempDir, config)

			require.NoError(t, err)
			assert.Equal(t, 1, len(collected))
			assert.Equal(t, filter, collected[0].Filter)
			assert.Equal(t, "V1.0__create_users."+filter+".sql", collected[0].Filename)
		})
	}
}

func TestCollectFilteredMigrationFiles_FullPathSet(t *testing.T) {
	tempDir := t.TempDir()

	filename := "V1.0__test.sql"
	err := os.WriteFile(filepath.Join(tempDir, filename), []byte("CREATE TABLE test;"), 0644)
	require.NoError(t, err)

	config := FilterConfig{Mode: NoFilter}
	collected, err := CollectFilteredMigrationFiles(tempDir, config)

	require.NoError(t, err)
	assert.Equal(t, 1, len(collected))
	assert.Equal(t, filepath.Join(tempDir, filename), collected[0].FullPath)
	assert.NotEmpty(t, collected[0].FullPath)
}

func TestCollectFilteredMigrationFiles_SoftFilter_SameVersionDifferentDescriptions(t *testing.T) {
	tempDir := t.TempDir()

	// Same version (1.0) but different descriptions for different databases
	files := map[string]string{
		"V1.0__create_users_table.sql":                "CREATE TABLE users (id INTEGER);",
		"V1.0__create_users_with_serial.postgres.sql": "CREATE TABLE users (id SERIAL PRIMARY KEY);",
		"V1.0__create_users_with_identity.oracle.sql": "CREATE TABLE users (id NUMBER GENERATED AS IDENTITY);",
		"V2.0__add_email.sql":                         "ALTER TABLE users ADD COLUMN email TEXT;",
	}

	for filename, content := range files {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Test with postgres filter
	config := FilterConfig{Mode: SoftFilter, Filter: "postgres"}
	collected, err := CollectFilteredMigrationFiles(tempDir, config)

	require.NoError(t, err)
	assert.Equal(t, 2, len(collected), "Should collect V1.0 postgres version and V2.0 non-filtered")

	// Find the V1.0 migration
	var v1Migration *MigrationFile
	var v2Migration *MigrationFile
	for _, file := range collected {
		if file.Version == "1.0" {
			v1Migration = file
		} else if file.Version == "2.0" {
			v2Migration = file
		}
	}

	require.NotNil(t, v1Migration, "Should have found V1.0 migration")
	require.NotNil(t, v2Migration, "Should have found V2.0 migration")

	// V1.0 should use the postgres-filtered version (even though description differs)
	assert.Equal(t, "V1.0__create_users_with_serial.postgres.sql", v1Migration.Filename)
	assert.Equal(t, "postgres", v1Migration.Filter)
	assert.Equal(t, "create_users_with_serial", v1Migration.Description)

	// V2.0 should use the non-filtered version (no postgres version available)
	assert.Equal(t, "V2.0__add_email.sql", v2Migration.Filename)
	assert.Equal(t, "", v2Migration.Filter)
	assert.Equal(t, "add_email", v2Migration.Description)
}

func TestCollectFilteredMigrationFiles_HardFilter_SameVersionDifferentDescriptions(t *testing.T) {
	tempDir := t.TempDir()

	// Same version but different descriptions for different databases
	files := map[string]string{
		"V1.0__generic_users.sql":           "CREATE TABLE users (id INTEGER);",
		"V1.0__postgres_users.postgres.sql": "CREATE TABLE users (id SERIAL);",
		"V1.0__oracle_users.oracle.sql":     "CREATE TABLE users (id NUMBER);",
		"V2.0__add_column.postgres.sql":     "ALTER TABLE users ADD COLUMN email VARCHAR;",
	}

	for filename, content := range files {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Test with postgres filter
	config := FilterConfig{Mode: HardFilter, Filter: "postgres"}
	collected, err := CollectFilteredMigrationFiles(tempDir, config)

	require.NoError(t, err)
	assert.Equal(t, 2, len(collected), "Should only collect postgres-filtered files")

	// Verify only postgres versions collected
	filenames := make(map[string]bool)
	for _, file := range collected {
		filenames[file.Filename] = true
		assert.Equal(t, "postgres", file.Filter, "All files should have postgres filter")
	}
	assert.True(t, filenames["V1.0__postgres_users.postgres.sql"])
	assert.True(t, filenames["V2.0__add_column.postgres.sql"])
	assert.False(t, filenames["V1.0__generic_users.sql"])
	assert.False(t, filenames["V1.0__oracle_users.oracle.sql"])
}

func TestCollectFilteredMigrationFiles_RepeatableGroupsByDescription(t *testing.T) {
	tempDir := t.TempDir()

	// Repeatable migrations with same description should be grouped
	files := map[string]string{
		"R__create_views.sql":          "CREATE VIEW user_view AS SELECT * FROM users;",
		"R__create_views.postgres.sql": "CREATE VIEW user_view AS SELECT * FROM users WITH postgres features;",
		"R__other_views.sql":           "CREATE VIEW other_view AS SELECT * FROM orders;",
	}

	for filename, content := range files {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Test with soft filter
	config := FilterConfig{Mode: SoftFilter, Filter: "postgres"}
	collected, err := CollectFilteredMigrationFiles(tempDir, config)

	require.NoError(t, err)
	assert.Equal(t, 2, len(collected), "Should collect 2 repeatable migrations")

	// Verify that create_views uses postgres version
	filenames := make(map[string]bool)
	for _, file := range collected {
		filenames[file.Filename] = true
	}
	assert.True(t, filenames["R__create_views.postgres.sql"])
	assert.False(t, filenames["R__create_views.sql"], "Non-filtered version should not be collected when filtered version exists")
	assert.True(t, filenames["R__other_views.sql"])
}
