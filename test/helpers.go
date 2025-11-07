package test

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestContext holds shared test configuration
type TestContext struct {
	DBPath        string
	MigrationPath string
	BloomDBBinary string
	T             *testing.T
}

// TestOutput represents a line of output from BloomDB commands
type TestOutput struct {
	Level   string
	Message string
}

// MigrationRecord represents a row in the migration table
type MigrationRecord struct {
	Version     string
	Description string
	Type        string
	Success     int
	InstalledOn string
	Checksum    *int64
}

// NewTestContext creates a new test context with temporary database
func NewTestContext(t *testing.T) *TestContext {
	t.Helper()

	// Create temporary directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Find the migration path relative to test directory
	migrationPath := filepath.Join("..", "migrations")

	// Build the binary if it doesn't exist
	binaryPath := filepath.Join("..", "bloomdb")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Logf("Building bloomdb binary...")
		cmd := exec.Command("go", "build", "-o", binaryPath, "..")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build bloomdb: %v\n%s", err, output)
		}
	}

	return &TestContext{
		DBPath:        dbPath,
		MigrationPath: migrationPath,
		BloomDBBinary: binaryPath,
		T:             t,
	}
}

// RunCommand executes a bloomdb command and returns the output
func (ctx *TestContext) RunCommand(args ...string) (string, error) {
	ctx.T.Helper()

	// Build command with connection string
	connStr := fmt.Sprintf("sqlite://%s", ctx.DBPath)
	fullArgs := append([]string{"--conn", connStr, "--path", ctx.MigrationPath}, args...)

	// Set environment for test output format
	cmd := exec.Command(ctx.BloomDBBinary, fullArgs...)
	cmd.Env = append(os.Environ(), "BLOOMDB_PRINTER=test", "BLOOMDB_VERBOSE=1")

	output, err := cmd.CombinedOutput()
	return string(output), err
}

// ParseTestOutput parses test format lines from bloomdb output
// Format: LEVEL: message
func ParseTestOutput(output string) ([]TestOutput, error) {
	var results []TestOutput
	lines := strings.Split(strings.TrimSpace(output), "\n")

	insideSQLBlock := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Track SQL blocks: start with [SQL] and end when we see [ARGS] or a LEVEL: line
		if strings.HasPrefix(line, "[SQL]") {
			insideSQLBlock = true
			continue
		}
		if strings.HasPrefix(line, "[ARGS]") {
			insideSQLBlock = false
			continue
		}

		// Check if this is a LEVEL: line (ends SQL block if we're inside one)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			// This is a potential LEVEL: line
			level := strings.TrimSpace(parts[0])
			// Check if it's a valid log level
			if level == "INFO" || level == "SUCCESS" || level == "ERROR" || level == "WARNING" || level == "DEBUG" {
				insideSQLBlock = false
				results = append(results, TestOutput{
					Level:   level,
					Message: strings.TrimSpace(parts[1]),
				})
				continue
			}
		}

		// Skip lines inside SQL blocks (multi-line SQL statements)
		if insideSQLBlock {
			continue
		}

		// Invalid format if we get here
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid output format, expected 'LEVEL: message', got: %s", line)
		}
	}

	return results, nil
}

// GetMigrationRecords queries the migration table and returns all records
func (ctx *TestContext) GetMigrationRecords() ([]MigrationRecord, error) {
	ctx.T.Helper()

	db, err := sql.Open("sqlite3", ctx.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT "version", "description", "type", "success", "installed on", "checksum"
		FROM BLOOMDB_VERSION
		ORDER BY "installed on"
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query migration table: %w", err)
	}
	defer rows.Close()

	var records []MigrationRecord
	for rows.Next() {
		var record MigrationRecord
		err := rows.Scan(
			&record.Version,
			&record.Description,
			&record.Type,
			&record.Success,
			&record.InstalledOn,
			&record.Checksum,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

// TableExists checks if a table exists in the database
func (ctx *TestContext) TableExists(tableName string) (bool, error) {
	ctx.T.Helper()

	db, err := sql.Open("sqlite3", ctx.DBPath)
	if err != nil {
		return false, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM sqlite_master
		WHERE type='table' AND name=?
	`, tableName).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check table existence: %w", err)
	}

	return count > 0, nil
}

// AssertOutputContains checks if output contains a message with given level
func AssertOutputContains(t *testing.T, output string, level string, messageSubstring string) {
	t.Helper()

	testOutput, err := ParseTestOutput(output)
	if err != nil {
		t.Fatalf("Failed to parse test output: %v\nOutput: %s", err, output)
	}

	for _, item := range testOutput {
		if item.Level == level && strings.Contains(item.Message, messageSubstring) {
			return
		}
	}

	t.Errorf("Expected output to contain level='%s' with message containing '%s'\nGot: %v",
		level, messageSubstring, testOutput)
}

// AssertSuccessMessage checks if the output contains a success message
func AssertSuccessMessage(t *testing.T, output string, messageSubstring string) {
	AssertOutputContains(t, output, "SUCCESS", messageSubstring)
}

// AssertErrorMessage checks if the output contains an error message
func AssertErrorMessage(t *testing.T, output string, messageSubstring string) {
	AssertOutputContains(t, output, "ERROR", messageSubstring)
}

// AssertWarningMessage checks if the output contains a warning message
func AssertWarningMessage(t *testing.T, output string, messageSubstring string) {
	AssertOutputContains(t, output, "WARNING", messageSubstring)
}

// AssertInfoMessage checks if the output contains an info message
func AssertInfoMessage(t *testing.T, output string, messageSubstring string) {
	AssertOutputContains(t, output, "INFO", messageSubstring)
}
