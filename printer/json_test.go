package printer

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"bloomdb/db"

	"github.com/stretchr/testify/assert"
)

// captureJSONOutput captures stdout and parses JSON output
func captureJSONOutput(t *testing.T, f func()) JSONOutput {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)

	var output JSONOutput
	err := json.Unmarshal(buf.Bytes(), &output)
	if err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, buf.String())
	}

	return output
}

func TestNewJSONPrinter(t *testing.T) {
	printer := NewJSONPrinter(true)
	assert.NotNil(t, printer)
	assert.True(t, printer.verbose)

	printer = NewJSONPrinter(false)
	assert.NotNil(t, printer)
	assert.False(t, printer.verbose)
}

func TestJSONPrinter_PrintOutput_Success(t *testing.T) {
	printer := NewJSONPrinter(false)
	output := captureJSONOutput(t, func() {
		printer.PrintOutput(LevelSuccess, "Test success message")
	})

	assert.Equal(t, "success", output.Level)
	assert.Equal(t, "Test success message", output.Message)
	assert.NotEmpty(t, output.Timestamp)

	// Verify timestamp is valid RFC3339
	_, err := time.Parse(time.RFC3339, output.Timestamp)
	assert.NoError(t, err)
}

func TestJSONPrinter_PrintOutput_Warning(t *testing.T) {
	printer := NewJSONPrinter(false)
	output := captureJSONOutput(t, func() {
		printer.PrintOutput(LevelWarning, "Test warning message")
	})

	assert.Equal(t, "warning", output.Level)
	assert.Equal(t, "Test warning message", output.Message)
}

func TestJSONPrinter_PrintOutput_Error(t *testing.T) {
	printer := NewJSONPrinter(false)
	output := captureJSONOutput(t, func() {
		printer.PrintOutput(LevelError, "Test error message")
	})

	assert.Equal(t, "error", output.Level)
	assert.Equal(t, "Test error message", output.Message)
}

func TestJSONPrinter_PrintOutput_Info_Verbose(t *testing.T) {
	printer := NewJSONPrinter(true)
	output := captureJSONOutput(t, func() {
		printer.PrintOutput(LevelInfo, "Test info message")
	})

	assert.Equal(t, "info", output.Level)
	assert.Equal(t, "Test info message", output.Message)
}

func TestJSONPrinter_PrintOutput_Info_NonVerbose(t *testing.T) {
	printer := NewJSONPrinter(false)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printer.PrintOutput(LevelInfo, "This should not appear")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Should produce no output in non-verbose mode
	assert.Empty(t, buf.String())
}

func TestJSONPrinter_PrintOutput_WithFormatting(t *testing.T) {
	printer := NewJSONPrinter(false)
	output := captureJSONOutput(t, func() {
		printer.PrintOutput(LevelSuccess, "User %s has %d items", "John", 42)
	})

	assert.Equal(t, "success", output.Level)
	assert.Equal(t, "User John has 42 items", output.Message)
}

func TestJSONPrinter_PrintSuccess(t *testing.T) {
	printer := NewJSONPrinter(false)
	output := captureJSONOutput(t, func() {
		printer.PrintSuccess("Operation completed")
	})

	assert.Equal(t, "success", output.Level)
	assert.Equal(t, "Operation completed", output.Message)
}

func TestJSONPrinter_PrintWarning(t *testing.T) {
	printer := NewJSONPrinter(false)
	output := captureJSONOutput(t, func() {
		printer.PrintWarning("Proceeding with caution")
	})

	assert.Equal(t, "warning", output.Level)
	assert.Equal(t, "Proceeding with caution", output.Message)
}

func TestJSONPrinter_PrintError(t *testing.T) {
	printer := NewJSONPrinter(false)
	output := captureJSONOutput(t, func() {
		printer.PrintError("Something went wrong")
	})

	assert.Equal(t, "error", output.Level)
	assert.Equal(t, "Something went wrong", output.Message)
}

func TestJSONPrinter_PrintInfo(t *testing.T) {
	printer := NewJSONPrinter(true)
	output := captureJSONOutput(t, func() {
		printer.PrintInfo("Additional information")
	})

	assert.Equal(t, "info", output.Level)
	assert.Equal(t, "Additional information", output.Message)
}

func TestJSONPrinter_PrintSeparator_WithTitle(t *testing.T) {
	printer := NewJSONPrinter(false)
	output := captureJSONOutput(t, func() {
		printer.PrintSeparator("Section Break")
	})

	assert.Equal(t, "info", output.Level)
	assert.Equal(t, "separator", output.Message)
	assert.NotNil(t, output.Data)
	assert.Equal(t, "separator", output.Data["type"])
	assert.Equal(t, "Section Break", output.Data["title"])
}

func TestJSONPrinter_PrintSeparator_WithoutTitle(t *testing.T) {
	printer := NewJSONPrinter(false)
	output := captureJSONOutput(t, func() {
		printer.PrintSeparator("")
	})

	assert.Equal(t, "info", output.Level)
	assert.Equal(t, "separator", output.Message)
	assert.NotNil(t, output.Data)
	assert.Equal(t, "separator", output.Data["type"])
	assert.Nil(t, output.Data["title"])
}

func TestJSONPrinter_PrintCommand(t *testing.T) {
	printer := NewJSONPrinter(false)
	output := captureJSONOutput(t, func() {
		printer.PrintCommand("migrate up")
	})

	assert.Equal(t, "info", output.Level)
	assert.Equal(t, "executing command", output.Message)
	assert.NotNil(t, output.Data)
	assert.Equal(t, "migrate up", output.Data["command"])
}

func TestJSONPrinter_PrintSection(t *testing.T) {
	printer := NewJSONPrinter(false)
	output := captureJSONOutput(t, func() {
		printer.PrintSection("Migration Status")
	})

	assert.Equal(t, "info", output.Level)
	assert.Equal(t, "section start", output.Message)
	assert.NotNil(t, output.Data)
	assert.Equal(t, "Migration Status", output.Data["title"])
}

func TestJSONPrinter_PrintSectionEnd(t *testing.T) {
	printer := NewJSONPrinter(false)
	output := captureJSONOutput(t, func() {
		printer.PrintSectionEnd()
	})

	assert.Equal(t, "info", output.Level)
	assert.Equal(t, "section end", output.Message)
	assert.Nil(t, output.Data)
}

func TestJSONPrinter_PrintMigration(t *testing.T) {
	printer := NewJSONPrinter(false)
	output := captureJSONOutput(t, func() {
		printer.PrintMigration("V1.0.0", "Initial schema", "applied")
	})

	assert.Equal(t, "info", output.Level)
	assert.Equal(t, "migration", output.Message)
	assert.NotNil(t, output.Data)
	assert.Equal(t, "V1.0.0", output.Data["version"])
	assert.Equal(t, "Initial schema", output.Data["description"])
	assert.Equal(t, "applied", output.Data["status"])
}

func TestJSONPrinter_PrintObject(t *testing.T) {
	printer := NewJSONPrinter(false)
	output := captureJSONOutput(t, func() {
		printer.PrintObject("table", "users")
	})

	assert.Equal(t, "info", output.Level)
	assert.Equal(t, "object", output.Message)
	assert.NotNil(t, output.Data)
	assert.Equal(t, "table", output.Data["type"])
	assert.Equal(t, "users", output.Data["name"])
}

func TestJSONPrinter_DisplayMigrationTable(t *testing.T) {
	printer := NewJSONPrinter(false)

	statuses := []MigrationStatus{
		{
			Version:     "V1.0.0",
			Description: "Initial schema",
			Type:        "Versioned",
			Status:      "applied",
		},
		{
			Version:     "V1.1.0",
			Description: "Add users table",
			Type:        "Versioned",
			Status:      "pending",
		},
		{
			Version:     "R__seed_data",
			Description: "Seed reference data",
			Type:        "Repeatable",
			Status:      "applied",
		},
	}

	output := captureJSONOutput(t, func() {
		printer.DisplayMigrationTable(db.PostgreSQL, "schema_migrations", statuses)
	})

	assert.Equal(t, "info", output.Level)
	assert.Equal(t, "migration table", output.Message)
	assert.NotNil(t, output.Data)
	assert.Equal(t, string(db.PostgreSQL), output.Data["database_type"])
	assert.Equal(t, "schema_migrations", output.Data["table_name"])

	// Verify migrations array
	migrationsData, ok := output.Data["migrations"].([]interface{})
	assert.True(t, ok, "migrations should be an array")
	assert.Len(t, migrationsData, 3)

	// Verify first migration (JSON marshaling uses struct field names)
	firstMigration, ok := migrationsData[0].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "V1.0.0", firstMigration["Version"])
	assert.Equal(t, "Initial schema", firstMigration["Description"])
	assert.Equal(t, "Versioned", firstMigration["Type"])
	assert.Equal(t, "applied", firstMigration["Status"])
}

func TestJSONPrinter_DisplayMigrationTable_Empty(t *testing.T) {
	printer := NewJSONPrinter(false)

	statuses := []MigrationStatus{}

	output := captureJSONOutput(t, func() {
		printer.DisplayMigrationTable(db.SQLite, "migrations", statuses)
	})

	assert.Equal(t, "info", output.Level)
	assert.Equal(t, "migration table", output.Message)
	assert.NotNil(t, output.Data)
	assert.Equal(t, string(db.SQLite), output.Data["database_type"])

	migrationsData, ok := output.Data["migrations"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, migrationsData, 0)
}

func TestJSONPrinter_OutputFormat_ValidJSON(t *testing.T) {
	printer := NewJSONPrinter(false)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printer.PrintSuccess("Test message")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Verify output is valid JSON
	var result map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &result)
	assert.NoError(t, err, "Output should be valid JSON")

	// Verify required fields exist
	assert.Contains(t, result, "timestamp")
	assert.Contains(t, result, "level")
	assert.Contains(t, result, "message")
}

func TestJSONPrinter_MultipleOutputs(t *testing.T) {
	printer := NewJSONPrinter(true)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printer.PrintSuccess("First message")
	printer.PrintWarning("Second message")
	printer.PrintInfo("Third message")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 3, "Should have 3 JSON objects")

	// Verify each line is valid JSON
	for i, line := range lines {
		var output JSONOutput
		err := json.Unmarshal([]byte(line), &output)
		assert.NoError(t, err, "Line %d should be valid JSON", i+1)
	}
}
