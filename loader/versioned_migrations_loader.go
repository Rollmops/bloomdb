package loader

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type VersionedMigration struct {
	Version     string
	Description string
	Content     string
	FilePath    string
	Checksum    int64
}

type VersionedMigrationLoader struct {
	directory string
}

func NewVersionedMigrationLoader(directory string) *VersionedMigrationLoader {
	return &VersionedMigrationLoader{
		directory: directory,
	}
}

// IsValidVersion checks if the version string has a valid format (e.g., 1.2.3, 2.2, 1)
func IsValidVersion(version string) bool {
	if version == "" {
		return false
	}

	// Version pattern: one or more numeric parts separated by dots
	versionPattern := regexp.MustCompile(`^\d+(\.\d+)*$`)
	return versionPattern.MatchString(version)
}

func (l *VersionedMigrationLoader) LoadMigrations() ([]*VersionedMigration, error) {
	// Get filter configuration from environment
	filterConfig := GetFilterConfig()

	// Collect filtered migration files
	migrationFiles, err := CollectFilteredMigrationFiles(l.directory, filterConfig)
	if err != nil {
		return nil, err
	}

	// Filter for versioned migrations only
	var migrations []*VersionedMigration
	for _, mf := range migrationFiles {
		if mf.IsRepeatable {
			continue // Skip repeatable migrations
		}

		content, err := os.ReadFile(mf.FullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", mf.Filename, err)
		}

		checksum := CalculateChecksum(content)

		migration := &VersionedMigration{
			Version:     mf.Version,
			Description: mf.Description,
			Content:     string(content),
			FilePath:    mf.FullPath,
			Checksum:    checksum,
		}

		migrations = append(migrations, migration)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return CompareVersions(migrations[i].Version, migrations[j].Version) < 0
	})

	return migrations, nil
}

func (l *VersionedMigrationLoader) GetMigrationByVersion(migrations []*VersionedMigration, version string) *VersionedMigration {
	for _, migration := range migrations {
		if migration.Version == version {
			return migration
		}
	}
	return nil
}

func (l *VersionedMigrationLoader) GetLatestVersion(migrations []*VersionedMigration) string {
	if len(migrations) == 0 {
		return ""
	}

	latest := migrations[0].Version
	for _, migration := range migrations {
		if CompareVersions(migration.Version, latest) > 0 {
			latest = migration.Version
		}
	}
	return latest
}

func CompareVersions(v1, v2 string) int {
	v1Parts := strings.Split(v1, ".")
	v2Parts := strings.Split(v2, ".")

	maxLen := len(v1Parts)
	if len(v2Parts) > maxLen {
		maxLen = len(v2Parts)
	}

	for i := 0; i < maxLen; i++ {
		var v1Num, v2Num int

		if i < len(v1Parts) {
			if num, err := strconv.Atoi(v1Parts[i]); err == nil {
				v1Num = num
			}
		}

		if i < len(v2Parts) {
			if num, err := strconv.Atoi(v2Parts[i]); err == nil {
				v2Num = num
			}
		}

		if v1Num < v2Num {
			return -1
		}
		if v1Num > v2Num {
			return 1
		}
	}

	return 0
}

func (m *VersionedMigration) GetFileName() string {
	return fmt.Sprintf("V%s__%s.sql", m.Version, m.Description)
}

func (m *VersionedMigration) String() string {
	return fmt.Sprintf("V%s__%s", m.Version, m.Description)
}
