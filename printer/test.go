package printer

import (
	"fmt"

	"bloomdb/db"
)

// TestPrinter implements Printer for simple test-friendly output
// Format: LEVEL: message (one line per output)
type TestPrinter struct {
	verbose bool
}

// NewTestPrinter creates a new test printer
func NewTestPrinter(verbose bool) *TestPrinter {
	return &TestPrinter{
		verbose: verbose,
	}
}

func (p *TestPrinter) output(level string, message string) {
	fmt.Printf("%s: %s\n", level, message)
}

// PrintOutput prints formatted output with level prefix
func (p *TestPrinter) PrintOutput(level OutputLevel, message string, args ...interface{}) {
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}

	var levelStr string
	switch level {
	case LevelSuccess:
		levelStr = "SUCCESS"
	case LevelWarning:
		levelStr = "WARNING"
	case LevelError:
		levelStr = "ERROR"
	case LevelInfo:
		levelStr = "INFO"
	default:
		levelStr = "UNKNOWN"
	}

	// Only print if not info level or if verbose mode is enabled
	if level != LevelInfo || p.verbose {
		p.output(levelStr, message)
	}
}

// PrintSuccess prints a success message
func (p *TestPrinter) PrintSuccess(message string, args ...interface{}) {
	p.PrintOutput(LevelSuccess, message, args...)
}

// PrintWarning prints a warning message
func (p *TestPrinter) PrintWarning(message string, args ...interface{}) {
	p.PrintOutput(LevelWarning, message, args...)
}

// PrintError prints an error message
func (p *TestPrinter) PrintError(message string, args ...interface{}) {
	p.PrintOutput(LevelError, message, args...)
}

// PrintInfo prints an info message (only in verbose mode)
func (p *TestPrinter) PrintInfo(message string, args ...interface{}) {
	p.PrintOutput(LevelInfo, message, args...)
}

// PrintSeparator prints a separator (simplified for tests)
func (p *TestPrinter) PrintSeparator(title string) {
	if title != "" {
		p.output("INFO", fmt.Sprintf("separator: %s", title))
	} else {
		p.output("INFO", "separator")
	}
}

// PrintCommand prints a command execution
func (p *TestPrinter) PrintCommand(cmd string) {
	p.output("INFO", fmt.Sprintf("command: %s", cmd))
}

// PrintSection prints a section header
func (p *TestPrinter) PrintSection(title string) {
	p.output("INFO", fmt.Sprintf("section: %s", title))
}

// PrintSectionEnd prints a section footer
func (p *TestPrinter) PrintSectionEnd() {
	p.output("INFO", "section end")
}

// PrintMigration prints migration information
func (p *TestPrinter) PrintMigration(version, description, status string) {
	p.output("INFO", fmt.Sprintf("migration: version=%s description=%s status=%s", version, description, status))
}

// PrintObject prints database object information
func (p *TestPrinter) PrintObject(objType, name string) {
	p.output("INFO", fmt.Sprintf("object: type=%s name=%s", objType, name))
}

// DisplayMigrationTable prints migration table (simplified for tests)
func (p *TestPrinter) DisplayMigrationTable(dbType db.DatabaseType, tableName string, statuses []MigrationStatus) {
	p.output("INFO", fmt.Sprintf("migration_table: database=%s table=%s count=%d", dbType, tableName, len(statuses)))
	for _, status := range statuses {
		p.output("INFO", fmt.Sprintf("migration_row: version=%s description=%s type=%s status=%s",
			status.Version, status.Description, status.Type, status.Status))
	}
}
