package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSubdirectoryMigrationIntegration tests the full subdirectory migration workflow
func TestSubdirectoryMigrationIntegration(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()

	// Create subdirectory structure for multi-tenant scenario
	tenantADir := filepath.Join(tmpDir, "tenant-a")
	tenantBDir := filepath.Join(tmpDir, "tenant-b")

	err := os.Mkdir(tenantADir, 0755)
	require.NoError(t, err)
	err = os.Mkdir(tenantBDir, 0755)
	require.NoError(t, err)

	// Create migrations for tenant-a
	err = os.WriteFile(filepath.Join(tenantADir, "V1__create_users.sql"), []byte("CREATE TABLE users (id INT);"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tenantADir, "V2__add_email.sql"), []byte("ALTER TABLE users ADD COLUMN email VARCHAR(255);"), 0644)
	require.NoError(t, err)

	// Create migrations for tenant-b
	err = os.WriteFile(filepath.Join(tenantBDir, "V1__create_users.sql"), []byte("CREATE TABLE users (id INT);"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tenantBDir, "V2__add_name.sql"), []byte("ALTER TABLE users ADD COLUMN name VARCHAR(255);"), 0644)
	require.NoError(t, err)

	// Detect migration directories
	dirs, err := DetectMigrationDirectories(tmpDir)
	require.NoError(t, err)

	// Should detect both subdirectories
	assert.Len(t, dirs, 2)

	// Verify each directory configuration
	dirMap := make(map[string]MigrationDirectory)
	for _, dir := range dirs {
		dirMap[dir.Name] = dir
	}

	// Check tenant-a
	tenantA, exists := dirMap["tenant-a"]
	assert.True(t, exists, "tenant-a should be detected")
	assert.Equal(t, "BLOOMDB_TENANT_A", tenantA.VersionTable)
	assert.Equal(t, tenantADir, tenantA.Path)
	assert.True(t, tenantA.IsSubdirectory)

	// Check tenant-b
	tenantB, exists := dirMap["tenant-b"]
	assert.True(t, exists, "tenant-b should be detected")
	assert.Equal(t, "BLOOMDB_TENANT_B", tenantB.VersionTable)
	assert.Equal(t, tenantBDir, tenantB.Path)
	assert.True(t, tenantB.IsSubdirectory)

	// Load migrations for tenant-a
	loaderA := NewVersionedMigrationLoader(tenantA.Path)
	migrationsA, err := loaderA.LoadMigrations()
	require.NoError(t, err)
	assert.Len(t, migrationsA, 2)
	assert.Equal(t, "1", migrationsA[0].Version)
	assert.Equal(t, "2", migrationsA[1].Version)

	// Load migrations for tenant-b
	loaderB := NewVersionedMigrationLoader(tenantB.Path)
	migrationsB, err := loaderB.LoadMigrations()
	require.NoError(t, err)
	assert.Len(t, migrationsB, 2)
	assert.Equal(t, "1", migrationsB[0].Version)
	assert.Equal(t, "2", migrationsB[1].Version)

	// Verify migrations have different descriptions (showing they're independent)
	assert.NotEqual(t, migrationsA[1].Description, migrationsB[1].Description)
}

// TestSubdirectoryWithRootMigrations tests that root migrations take precedence
func TestSubdirectoryWithRootMigrations(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a migration at root level
	err := os.WriteFile(filepath.Join(tmpDir, "V1__root.sql"), []byte("SELECT 1;"), 0644)
	require.NoError(t, err)

	// Create subdirectory with migrations
	subdir := filepath.Join(tmpDir, "tenant-a")
	err = os.Mkdir(subdir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subdir, "V1__sub.sql"), []byte("SELECT 2;"), 0644)
	require.NoError(t, err)

	// Detect migration directories
	dirs, err := DetectMigrationDirectories(tmpDir)
	require.NoError(t, err)

	// Should only return root directory (root takes precedence)
	assert.Len(t, dirs, 1)
	assert.Equal(t, tmpDir, dirs[0].Path)
	assert.False(t, dirs[0].IsSubdirectory)
	assert.Equal(t, "", dirs[0].VersionTable)
}
