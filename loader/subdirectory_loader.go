package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MigrationDirectory represents a directory containing migrations
type MigrationDirectory struct {
	Path           string // Full path to the directory
	Name           string // Directory name (empty for root)
	VersionTable   string // Derived version table name
	IsSubdirectory bool   // True if this is a subdirectory
}

// DeriveVersionTableName converts a directory name to a version table name
// Rules: Uppercase, replace hyphens with underscores, prefix with "BLOOMDB_"
// Example: "tenant-a" -> "BLOOMDB_TENANT_A"
func DeriveVersionTableName(dirname string) string {
	if dirname == "" {
		return "" // Will use default table name from env/config
	}

	// Replace hyphens with underscores
	normalized := strings.ReplaceAll(dirname, "-", "_")

	// Convert to uppercase
	normalized = strings.ToUpper(normalized)

	// Add BLOOMDB_ prefix
	return "BLOOMDB_" + normalized
}

// DetectMigrationDirectories checks if the migration path contains subdirectories
// Returns a list of migration directories to process
func DetectMigrationDirectories(migrationPath string) ([]MigrationDirectory, error) {
	// Read the contents of the migration path
	entries, err := os.ReadDir(migrationPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration directory: %w", err)
	}

	// Check if there are any versioned or repeatable migrations at root level
	hasRootMigrations := false
	var subdirs []string

	for _, entry := range entries {
		if entry.IsDir() {
			subdirs = append(subdirs, entry.Name())
			continue
		}

		// Check if this file looks like a migration
		filename := entry.Name()
		if _, err := ParseMigrationFilename(filename); err == nil {
			hasRootMigrations = true
		}
	}

	// If there are migrations at root level, just return the root directory
	if hasRootMigrations {
		return []MigrationDirectory{
			{
				Path:           migrationPath,
				Name:           "",
				VersionTable:   "", // Will use default
				IsSubdirectory: false,
			},
		}, nil
	}

	// No root migrations, check subdirectories (depth 1 only)
	if len(subdirs) == 0 {
		// No subdirectories either, return root (even though it's empty)
		return []MigrationDirectory{
			{
				Path:           migrationPath,
				Name:           "",
				VersionTable:   "", // Will use default
				IsSubdirectory: false,
			},
		}, nil
	}

	// Process subdirectories
	var migrationDirs []MigrationDirectory
	for _, subdir := range subdirs {
		subdirPath := filepath.Join(migrationPath, subdir)

		// Check if this subdirectory contains any migrations
		subdirEntries, err := os.ReadDir(subdirPath)
		if err != nil {
			// Skip subdirectories that can't be read
			continue
		}

		hasMigrations := false
		for _, entry := range subdirEntries {
			if entry.IsDir() {
				continue
			}

			filename := entry.Name()
			if _, err := ParseMigrationFilename(filename); err == nil {
				hasMigrations = true
				break
			}
		}

		if hasMigrations {
			migrationDirs = append(migrationDirs, MigrationDirectory{
				Path:           subdirPath,
				Name:           subdir,
				VersionTable:   DeriveVersionTableName(subdir),
				IsSubdirectory: true,
			})
		}
	}

	// If no subdirectories with migrations found, return root
	if len(migrationDirs) == 0 {
		return []MigrationDirectory{
			{
				Path:           migrationPath,
				Name:           "",
				VersionTable:   "", // Will use default
				IsSubdirectory: false,
			},
		}, nil
	}

	return migrationDirs, nil
}
