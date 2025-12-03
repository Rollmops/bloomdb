package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveVersionTableName(t *testing.T) {
	tests := []struct {
		name     string
		dirname  string
		expected string
	}{
		{
			name:     "simple name",
			dirname:  "tenant",
			expected: "BLOOMDB_TENANT",
		},
		{
			name:     "name with hyphen",
			dirname:  "tenant-a",
			expected: "BLOOMDB_TENANT_A",
		},
		{
			name:     "name with multiple hyphens",
			dirname:  "my-tenant-db",
			expected: "BLOOMDB_MY_TENANT_DB",
		},
		{
			name:     "name with underscore",
			dirname:  "tenant_a",
			expected: "BLOOMDB_TENANT_A",
		},
		{
			name:     "mixed case",
			dirname:  "TenantA",
			expected: "BLOOMDB_TENANTA",
		},
		{
			name:     "empty string",
			dirname:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeriveVersionTableName(tt.dirname)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectMigrationDirectories_RootMigrations(t *testing.T) {
	// Create temporary directory with migrations at root level
	tmpDir := t.TempDir()

	// Create a migration file at root
	err := os.WriteFile(filepath.Join(tmpDir, "V1__test.sql"), []byte("SELECT 1;"), 0644)
	require.NoError(t, err)

	// Create a subdirectory (should be ignored)
	subdir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subdir, 0755)
	require.NoError(t, err)

	// Detect migration directories
	dirs, err := DetectMigrationDirectories(tmpDir)
	require.NoError(t, err)

	// Should return only root directory
	assert.Len(t, dirs, 1)
	assert.Equal(t, tmpDir, dirs[0].Path)
	assert.Equal(t, "", dirs[0].Name)
	assert.Equal(t, "", dirs[0].VersionTable)
	assert.False(t, dirs[0].IsSubdirectory)
}

func TestDetectMigrationDirectories_Subdirectories(t *testing.T) {
	// Create temporary directory without root migrations
	tmpDir := t.TempDir()

	// Create subdirectory with migrations
	subdir1 := filepath.Join(tmpDir, "tenant-a")
	err := os.Mkdir(subdir1, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subdir1, "V1__test.sql"), []byte("SELECT 1;"), 0644)
	require.NoError(t, err)

	// Create another subdirectory with migrations
	subdir2 := filepath.Join(tmpDir, "tenant_b")
	err = os.Mkdir(subdir2, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subdir2, "R__view.sql"), []byte("CREATE VIEW;"), 0644)
	require.NoError(t, err)

	// Create empty subdirectory (should be ignored)
	emptySubdir := filepath.Join(tmpDir, "empty")
	err = os.Mkdir(emptySubdir, 0755)
	require.NoError(t, err)

	// Detect migration directories
	dirs, err := DetectMigrationDirectories(tmpDir)
	require.NoError(t, err)

	// Should return both subdirectories with migrations
	assert.Len(t, dirs, 2)

	// Check that both subdirectories are detected (order may vary)
	names := make(map[string]MigrationDirectory)
	for _, dir := range dirs {
		names[dir.Name] = dir
	}

	// Check tenant-a
	tenantA, exists := names["tenant-a"]
	assert.True(t, exists)
	assert.Equal(t, subdir1, tenantA.Path)
	assert.Equal(t, "BLOOMDB_TENANT_A", tenantA.VersionTable)
	assert.True(t, tenantA.IsSubdirectory)

	// Check tenant_b
	tenantB, exists := names["tenant_b"]
	assert.True(t, exists)
	assert.Equal(t, subdir2, tenantB.Path)
	assert.Equal(t, "BLOOMDB_TENANT_B", tenantB.VersionTable)
	assert.True(t, tenantB.IsSubdirectory)
}

func TestDetectMigrationDirectories_EmptyDirectory(t *testing.T) {
	// Create temporary empty directory
	tmpDir := t.TempDir()

	// Detect migration directories
	dirs, err := DetectMigrationDirectories(tmpDir)
	require.NoError(t, err)

	// Should return root directory even though it's empty
	assert.Len(t, dirs, 1)
	assert.Equal(t, tmpDir, dirs[0].Path)
	assert.Equal(t, "", dirs[0].Name)
	assert.Equal(t, "", dirs[0].VersionTable)
	assert.False(t, dirs[0].IsSubdirectory)
}

func TestDetectMigrationDirectories_SubdirectoriesWithNonMigrationFiles(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create subdirectory with non-migration files
	subdir := filepath.Join(tmpDir, "docs")
	err := os.Mkdir(subdir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subdir, "README.md"), []byte("# Docs"), 0644)
	require.NoError(t, err)

	// Detect migration directories
	dirs, err := DetectMigrationDirectories(tmpDir)
	require.NoError(t, err)

	// Should return root directory (no migration subdirectories found)
	assert.Len(t, dirs, 1)
	assert.Equal(t, tmpDir, dirs[0].Path)
	assert.Equal(t, "", dirs[0].Name)
	assert.False(t, dirs[0].IsSubdirectory)
}
