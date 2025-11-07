package loader

import (
	"fmt"
	"os"
	"path/filepath"
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
	files, err := os.ReadDir(l.directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration directory: %w", err)
	}

	var migrations []*VersionedMigration
	migrationPattern := regexp.MustCompile(`^V(.+?)__(.+)\.sql$`)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		matches := migrationPattern.FindStringSubmatch(filename)
		if len(matches) != 3 {
			continue
		}

		version := matches[1]
		description := matches[2]
		filePath := filepath.Join(l.directory, filename)

		// Validate version format
		if !IsValidVersion(version) {
			return nil, fmt.Errorf("invalid version format in file %s: %s (expected format: 1, 1.2, 1.2.3, etc.)", filename, version)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		checksum := CalculateChecksum(content)

		migration := &VersionedMigration{
			Version:     version,
			Description: description,
			Content:     string(content),
			FilePath:    filePath,
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
