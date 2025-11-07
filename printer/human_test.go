package printer

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"bloomdb/db"
	"github.com/stretchr/testify/assert"
)

// captureOutput captures stdout during test execution
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestHumanPrinter_PrintOutput(t *testing.T) {
	tests := []struct {
		name     string
		level    OutputLevel
		message  string
		args     []interface{}
		verbose  bool
		contains []string
	}{
		{
			name:     "success message",
			level:    LevelSuccess,
			message:  "Migration completed",
			verbose:  false,
			contains: []string{"✓", "Migration completed"},
		},
		{
			name:     "warning message",
			level:    LevelWarning,
			message:  "Deprecated feature",
			verbose:  false,
			contains: []string{"⚠", "Deprecated feature"},
		},
		{
			name:     "error message",
			level:    LevelError,
			message:  "Connection failed",
			verbose:  false,
			contains: []string{"✗", "Connection failed"},
		},
		{
			name:     "info message in verbose mode",
			level:    LevelInfo,
			message:  "Debug information",
			verbose:  true,
			contains: []string{"ℹ", "Debug information"},
		},
		{
			name:     "info message not in verbose mode (should not print)",
			level:    LevelInfo,
			message:  "Debug information",
			verbose:  false,
			contains: []string{}, // Empty - should not print
		},
		{
			name:     "message with formatting",
			level:    LevelSuccess,
			message:  "Applied %d migrations",
			args:     []interface{}{5},
			verbose:  false,
			contains: []string{"✓", "Applied 5 migrations"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &HumanPrinter{verbose: tt.verbose}
			output := captureOutput(func() {
				p.PrintOutput(tt.level, tt.message, tt.args...)
			})

			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
			}

			// For info messages in non-verbose mode, output should be empty
			if tt.level == LevelInfo && !tt.verbose {
				assert.Empty(t, output)
			}
		})
	}
}

func TestHumanPrinter_PrintSuccess(t *testing.T) {
	p := &HumanPrinter{verbose: false}
	output := captureOutput(func() {
		p.PrintSuccess("Operation successful")
	})

	assert.Contains(t, output, "✓")
	assert.Contains(t, output, "Operation successful")
}

func TestHumanPrinter_PrintWarning(t *testing.T) {
	p := &HumanPrinter{verbose: false}
	output := captureOutput(func() {
		p.PrintWarning("Please review")
	})

	assert.Contains(t, output, "⚠")
	assert.Contains(t, output, "Please review")
}

func TestHumanPrinter_PrintError(t *testing.T) {
	p := &HumanPrinter{verbose: false}
	output := captureOutput(func() {
		p.PrintError("Something went wrong")
	})

	assert.Contains(t, output, "✗")
	assert.Contains(t, output, "Something went wrong")
}

func TestHumanPrinter_PrintInfo(t *testing.T) {
	t.Run("verbose mode", func(t *testing.T) {
		p := &HumanPrinter{verbose: true}
		output := captureOutput(func() {
			p.PrintInfo("Detailed info")
		})

		assert.Contains(t, output, "ℹ")
		assert.Contains(t, output, "Detailed info")
	})

	t.Run("non-verbose mode", func(t *testing.T) {
		p := &HumanPrinter{verbose: false}
		output := captureOutput(func() {
			p.PrintInfo("Detailed info")
		})

		assert.Empty(t, output)
	})
}

func TestHumanPrinter_PrintSeparator(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		contains []string
	}{
		{
			name:     "separator with title",
			title:    "Database Info",
			contains: []string{"═", "Database Info"},
		},
		{
			name:     "separator without title",
			title:    "",
			contains: []string{"═"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &HumanPrinter{verbose: false}
			output := captureOutput(func() {
				p.PrintSeparator(tt.title)
			})

			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestHumanPrinter_PrintCommand(t *testing.T) {
	p := &HumanPrinter{verbose: false}
	output := captureOutput(func() {
		p.PrintCommand("bloomdb migrate")
	})

	assert.Contains(t, output, "➜")
	assert.Contains(t, output, "Executing:")
	assert.Contains(t, output, "bloomdb migrate")
}

func TestHumanPrinter_PrintSection(t *testing.T) {
	p := &HumanPrinter{verbose: false}
	output := captureOutput(func() {
		p.PrintSection("Migration Status")
	})

	assert.Contains(t, output, "┌─")
	assert.Contains(t, output, "Migration Status")
	assert.Contains(t, output, "│")
}

func TestHumanPrinter_PrintSectionEnd(t *testing.T) {
	p := &HumanPrinter{verbose: false}
	output := captureOutput(func() {
		p.PrintSectionEnd()
	})

	assert.Contains(t, output, "└─")
}

func TestHumanPrinter_PrintMigration(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		description string
		status      string
		contains    []string
	}{
		{
			name:        "success migration",
			version:     "V1",
			description: "create_users_table",
			status:      "success",
			contains:    []string{"✓", "V1", "create_users_table"},
		},
		{
			name:        "pending migration",
			version:     "V2",
			description: "add_indexes",
			status:      "pending",
			contains:    []string{"⏳", "V2", "add_indexes"},
		},
		{
			name:        "failed migration",
			version:     "V3",
			description: "update_schema",
			status:      "failed",
			contains:    []string{"✗", "V3", "update_schema"},
		},
		{
			name:        "baseline migration",
			version:     "V0",
			description: "baseline",
			status:      "baseline",
			contains:    []string{"📍", "V0", "baseline"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &HumanPrinter{verbose: false}
			output := captureOutput(func() {
				p.PrintMigration(tt.version, tt.description, tt.status)
			})

			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestHumanPrinter_PrintObject(t *testing.T) {
	p := &HumanPrinter{verbose: false}
	output := captureOutput(func() {
		p.PrintObject("table", "users")
	})

	assert.Contains(t, output, "table")
	assert.Contains(t, output, "users")
	assert.Contains(t, output, "•")
}

func TestHumanPrinter_DisplayMigrationTable(t *testing.T) {
	p := &HumanPrinter{verbose: false}
	statuses := []MigrationStatus{
		{
			Version:     "V0.1",
			Description: "create_user_table",
			Type:        "versioned",
			Status:      "success",
			InstalledOn: "2025-01-01 12:00:00",
		},
		{
			Version:     "V0.2",
			Description: "add_column_to_users",
			Type:        "versioned",
			Status:      "success",
			InstalledOn: "2025-01-02 12:00:00",
		},
		{
			Version:     "V1",
			Description: "create_test_tables",
			Type:        "versioned",
			Status:      "pending",
			InstalledOn: "",
		},
		{
			Version:     "R__Test_summary",
			Description: "Test_summary_views",
			Type:        "repeatable",
			Status:      "success",
			InstalledOn: "2025-01-03 12:00:00",
		},
	}

	output := captureOutput(func() {
		p.DisplayMigrationTable(db.PostgreSQL, "bloomdb_migrations", statuses)
	})

	// Check for header presence
	assert.Contains(t, output, "VERSION")
	assert.Contains(t, output, "DESCRIPTION")
	assert.Contains(t, output, "TYPE")
	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "INSTALLED ON")

	// Check for data rows
	assert.Contains(t, output, "V0.1")
	assert.Contains(t, output, "create user table") // Description with spaces
	assert.Contains(t, output, "V0.2")
	assert.Contains(t, output, "V1")
	assert.Contains(t, output, "R__Test_summary")
}

func TestColorize(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		colorType string
		contains  string
	}{
		{
			name:      "bold text",
			text:      "Hello",
			colorType: "bold",
			contains:  "\033[1m",
		},
		{
			name:      "red text",
			text:      "Error",
			colorType: "red",
			contains:  "\033[31m",
		},
		{
			name:      "bold+green text",
			text:      "Success",
			colorType: "bold+green",
			contains:  "\033[1;32m",
		},
		{
			name:      "cyan text",
			text:      "Info",
			colorType: "cyan",
			contains:  "\033[36m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := colorize(tt.text, tt.colorType)
			assert.Contains(t, result, tt.contains)
			assert.Contains(t, result, tt.text)
			assert.Contains(t, result, "\033[0m") // Reset code
		})
	}
}

func TestColorizeStatus(t *testing.T) {
	tests := []struct {
		status   string
		icon     string
		colorOpt string
	}{
		{status: "success", icon: "✓", colorOpt: "green"},
		{status: "pending", icon: "○", colorOpt: "yellow"},
		{status: "baseline", icon: "◉", colorOpt: "cyan"},
		{status: "below baseline", icon: "⊘", colorOpt: "dim"},
		{status: "failed", icon: "✗", colorOpt: "red"},
		{status: "missing", icon: "✗", colorOpt: "red"},
		{status: "checksum", icon: "⚠", colorOpt: "red"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := colorizeStatus(tt.status)
			assert.Contains(t, result, tt.icon)
			assert.Contains(t, result, tt.status)
		})
	}
}

func TestColorizeType(t *testing.T) {
	tests := []struct {
		migrationType string
		contains      string
	}{
		{migrationType: "versioned", contains: "versioned"},
		{migrationType: "repeatable", contains: "repeatable"},
	}

	for _, tt := range tests {
		t.Run(tt.migrationType, func(t *testing.T) {
			result := colorizeType(tt.migrationType)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestFormatDescription(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "create_user_table",
			expected: "create user table",
		},
		{
			input:    "add_column_to_users",
			expected: "add column to users",
		},
		{
			input:    "no_underscores_here",
			expected: "no underscores here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatDescription(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHumanPrinter_VerboseMode(t *testing.T) {
	t.Run("verbose enabled via constructor", func(t *testing.T) {
		p := &HumanPrinter{verbose: true}
		output := captureOutput(func() {
			p.PrintInfo("This should appear")
		})
		assert.NotEmpty(t, output)
		assert.Contains(t, output, "This should appear")
	})

	t.Run("verbose disabled via constructor", func(t *testing.T) {
		p := &HumanPrinter{verbose: false}
		output := captureOutput(func() {
			p.PrintInfo("This should not appear")
		})
		assert.Empty(t, output)
	})
}

func TestHumanPrinter_MultipleMessages(t *testing.T) {
	p := &HumanPrinter{verbose: true}
	output := captureOutput(func() {
		p.PrintSuccess("First message")
		p.PrintWarning("Second message")
		p.PrintError("Third message")
		p.PrintInfo("Fourth message")
	})

	// All messages should be present
	assert.Contains(t, output, "First message")
	assert.Contains(t, output, "Second message")
	assert.Contains(t, output, "Third message")
	assert.Contains(t, output, "Fourth message")

	// Count newlines to verify 4 separate messages
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, 4, len(lines))
}
