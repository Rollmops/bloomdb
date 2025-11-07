package loader

import (
	"fmt"
	"os"
)

type RepeatableMigration struct {
	Description string
	Content     string
	FilePath    string
	Checksum    int64
}

type RepeatableMigrationLoader struct {
	directory string
}

func NewRepeatableMigrationLoader(directory string) *RepeatableMigrationLoader {
	return &RepeatableMigrationLoader{
		directory: directory,
	}
}

func (r *RepeatableMigrationLoader) LoadRepeatableMigrations() ([]*RepeatableMigration, error) {
	// Get filter configuration from environment
	filterConfig := GetFilterConfig()

	// Collect filtered migration files
	migrationFiles, err := CollectFilteredMigrationFiles(r.directory, filterConfig)
	if err != nil {
		return nil, err
	}

	// Filter for repeatable migrations only
	var migrations []*RepeatableMigration
	for _, mf := range migrationFiles {
		if !mf.IsRepeatable {
			continue // Skip versioned migrations
		}

		content, err := os.ReadFile(mf.FullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", mf.Filename, err)
		}

		checksum := CalculateChecksum(content)

		migration := &RepeatableMigration{
			Description: mf.Description,
			Content:     string(content),
			FilePath:    mf.FullPath,
			Checksum:    checksum,
		}

		migrations = append(migrations, migration)
	}

	return migrations, nil
}

func (r *RepeatableMigration) GetFileName() string {
	return fmt.Sprintf("R__%s.sql", r.Description)
}

func (r *RepeatableMigration) String() string {
	return fmt.Sprintf("R__%s", r.Description)
}
