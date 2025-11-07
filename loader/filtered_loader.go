package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// FilterMode represents the type of filtering to apply
type FilterMode int

const (
	// NoFilter means no filtering is applied
	NoFilter FilterMode = iota
	// HardFilter means only files with the specified filter are collected
	HardFilter
	// SoftFilter means prefer files with filter, fallback to non-filtered
	SoftFilter
)

// FilterConfig holds the filter configuration from environment variables
type FilterConfig struct {
	Mode   FilterMode
	Filter string
}

// MigrationFile represents a parsed migration filename
type MigrationFile struct {
	FullPath     string
	Filename     string
	Version      string // Empty for repeatable migrations
	Description  string
	Filter       string // Empty if no filter
	IsRepeatable bool
}

// GetFilterConfig reads filter configuration from environment variables
func GetFilterConfig() FilterConfig {
	hardFilter := os.Getenv("BLOOMDB_FILTER_HARD")
	softFilter := os.Getenv("BLOOMDB_FILTER_SOFT")

	if hardFilter != "" {
		return FilterConfig{
			Mode:   HardFilter,
			Filter: hardFilter,
		}
	}

	if softFilter != "" {
		return FilterConfig{
			Mode:   SoftFilter,
			Filter: softFilter,
		}
	}

	return FilterConfig{
		Mode:   NoFilter,
		Filter: "",
	}
}

// ParseMigrationFilename parses a migration filename and extracts its components
func ParseMigrationFilename(filename string) (*MigrationFile, error) {
	// Pattern for versioned migrations: V<version>__<description>[.<filter>].sql
	versionedPattern := regexp.MustCompile(`^V(.+?)__(.+?)(?:\.([^.]+))?\.sql$`)

	// Pattern for repeatable migrations: R__<description>[.<filter>].sql
	repeatablePattern := regexp.MustCompile(`^R__(.+?)(?:\.([^.]+))?\.sql$`)

	// Try versioned pattern first
	if matches := versionedPattern.FindStringSubmatch(filename); len(matches) >= 3 {
		version := matches[1]
		description := matches[2]
		filter := ""
		if len(matches) == 4 && matches[3] != "" {
			filter = matches[3]
		}

		// Validate version format
		if !IsValidVersion(version) {
			return nil, fmt.Errorf("invalid version format in file %s: %s (expected format: 1, 1.2, 1.2.3, etc.)", filename, version)
		}

		return &MigrationFile{
			Filename:     filename,
			Version:      version,
			Description:  description,
			Filter:       filter,
			IsRepeatable: false,
		}, nil
	}

	// Try repeatable pattern
	if matches := repeatablePattern.FindStringSubmatch(filename); len(matches) >= 2 {
		description := matches[1]
		filter := ""
		if len(matches) == 3 && matches[2] != "" {
			filter = matches[2]
		}

		return &MigrationFile{
			Filename:     filename,
			Version:      "",
			Description:  description,
			Filter:       filter,
			IsRepeatable: true,
		}, nil
	}

	return nil, fmt.Errorf("filename does not match migration pattern: %s", filename)
}

// CollectFilteredMigrationFiles collects migration files based on the filter configuration
func CollectFilteredMigrationFiles(directory string, filterConfig FilterConfig) ([]*MigrationFile, error) {
	files, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration directory: %w", err)
	}

	// Pattern to check if file looks like a migration (V* or R__*.sql)
	migrationLikePattern := regexp.MustCompile(`^(V.+?__.+|R__.+)\.sql$`)

	// Parse all valid migration files
	var allFiles []*MigrationFile
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()

		// Check if this looks like a migration file
		if !migrationLikePattern.MatchString(filename) {
			// Not a migration file, skip silently
			continue
		}

		// This looks like a migration file, so parse it and report errors
		migFile, err := ParseMigrationFilename(filename)
		if err != nil {
			// Return error for files that look like migrations but are invalid
			return nil, err
		}

		migFile.FullPath = filepath.Join(directory, filename)
		allFiles = append(allFiles, migFile)
	}

	// Apply filter based on mode
	switch filterConfig.Mode {
	case NoFilter:
		// Return all files without filters, ignore files with filters
		return filterFilesWithoutFilter(allFiles), nil

	case HardFilter:
		// Return only files with the specified filter
		return filterFilesHard(allFiles, filterConfig.Filter), nil

	case SoftFilter:
		// Return files with filter, fallback to non-filtered for missing versions
		return filterFilesSoft(allFiles, filterConfig.Filter), nil

	default:
		return nil, fmt.Errorf("unknown filter mode: %d", filterConfig.Mode)
	}
}

// filterFilesWithoutFilter returns only files without any filter
func filterFilesWithoutFilter(files []*MigrationFile) []*MigrationFile {
	var result []*MigrationFile
	for _, file := range files {
		if file.Filter == "" {
			result = append(result, file)
		}
	}
	return result
}

// filterFilesHard returns only files with the specified filter
func filterFilesHard(files []*MigrationFile, filter string) []*MigrationFile {
	var result []*MigrationFile
	for _, file := range files {
		if file.Filter == filter {
			result = append(result, file)
		}
	}
	return result
}

// filterFilesSoft returns files with filter, falling back to non-filtered files
// when a filtered version is not available.
// For versioned migrations, grouping is by version only (not version+description).
// For repeatable migrations, grouping is by description only.
func filterFilesSoft(files []*MigrationFile, filter string) []*MigrationFile {
	// Group files by appropriate key
	type migKey struct {
		version      string // Used for versioned migrations
		description  string // Used for repeatable migrations
		isRepeatable bool
	}

	filesByKey := make(map[migKey][]*MigrationFile)
	for _, file := range files {
		var key migKey
		if file.IsRepeatable {
			// For repeatable migrations, group by description only
			key = migKey{
				version:      "",
				description:  file.Description,
				isRepeatable: true,
			}
		} else {
			// For versioned migrations, group by version only (not description)
			key = migKey{
				version:      file.Version,
				description:  "",
				isRepeatable: false,
			}
		}
		filesByKey[key] = append(filesByKey[key], file)
	}

	// For each key, prefer filtered version, fallback to non-filtered
	var result []*MigrationFile
	for _, fileGroup := range filesByKey {
		var filteredFile *MigrationFile
		var nonFilteredFile *MigrationFile

		for _, file := range fileGroup {
			if file.Filter == filter {
				filteredFile = file
			} else if file.Filter == "" {
				nonFilteredFile = file
			}
		}

		// Prefer filtered, fallback to non-filtered
		if filteredFile != nil {
			result = append(result, filteredFile)
		} else if nonFilteredFile != nil {
			result = append(result, nonFilteredFile)
		}
	}

	return result
}
