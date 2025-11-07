package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	files, err := os.ReadDir(r.directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration directory: %w", err)
	}

	var migrations []*RepeatableMigration
	migrationPattern := regexp.MustCompile(`^R__(.+)\.sql$`)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		matches := migrationPattern.FindStringSubmatch(filename)
		if len(matches) != 2 {
			continue
		}

		description := matches[1]
		filePath := filepath.Join(r.directory, filename)

		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		checksum := CalculateChecksum(content)

		migration := &RepeatableMigration{
			Description: description,
			Content:     string(content),
			FilePath:    filePath,
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
