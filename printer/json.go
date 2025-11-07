package printer

import (
	"encoding/json"
	"fmt"
	"time"

	"bloomdb/db"
)

// JSONPrinter implements Printer for machine-readable JSON output
type JSONPrinter struct {
	verbose bool
}

// NewJSONPrinter creates a new JSON console printer
func NewJSONPrinter(verbose bool) *JSONPrinter {
	return &JSONPrinter{
		verbose: verbose,
	}
}

// JSONOutput represents structured JSON output
type JSONOutput struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

func (p *JSONPrinter) outputJSON(level string, message string, data map[string]interface{}) {
	output := JSONOutput{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Data:      data,
	}
	jsonBytes, _ := json.Marshal(output)
	fmt.Println(string(jsonBytes))
}

// PrintOutput prints formatted JSON output
func (p *JSONPrinter) PrintOutput(level OutputLevel, message string, args ...interface{}) {
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}

	var levelStr string
	switch level {
	case LevelSuccess:
		levelStr = "success"
	case LevelWarning:
		levelStr = "warning"
	case LevelError:
		levelStr = "error"
	case LevelInfo:
		levelStr = "info"
	default:
		levelStr = "unknown"
	}

	// Only print if not info level or if verbose mode is enabled
	if level != LevelInfo || p.verbose {
		p.outputJSON(levelStr, message, nil)
	}
}

// PrintSuccess prints a success message
func (p *JSONPrinter) PrintSuccess(message string, args ...interface{}) {
	p.PrintOutput(LevelSuccess, message, args...)
}

// PrintWarning prints a warning message
func (p *JSONPrinter) PrintWarning(message string, args ...interface{}) {
	p.PrintOutput(LevelWarning, message, args...)
}

// PrintError prints an error message
func (p *JSONPrinter) PrintError(message string, args ...interface{}) {
	p.PrintOutput(LevelError, message, args...)
}

// PrintInfo prints an info message (only in verbose mode)
func (p *JSONPrinter) PrintInfo(message string, args ...interface{}) {
	p.PrintOutput(LevelInfo, message, args...)
}

// PrintSeparator prints a separator event
func (p *JSONPrinter) PrintSeparator(title string) {
	data := map[string]interface{}{"type": "separator"}
	if title != "" {
		data["title"] = title
	}
	p.outputJSON("info", "separator", data)
}

// PrintCommand prints a command execution event
func (p *JSONPrinter) PrintCommand(cmd string) {
	p.outputJSON("info", "executing command", map[string]interface{}{
		"command": cmd,
	})
}

// PrintSection prints a section header event
func (p *JSONPrinter) PrintSection(title string) {
	p.outputJSON("info", "section start", map[string]interface{}{
		"title": title,
	})
}

// PrintSectionEnd prints a section footer event
func (p *JSONPrinter) PrintSectionEnd() {
	p.outputJSON("info", "section end", nil)
}

// PrintMigration prints migration information
func (p *JSONPrinter) PrintMigration(version, description, status string) {
	p.outputJSON("info", "migration", map[string]interface{}{
		"version":     version,
		"description": description,
		"status":      status,
	})
}

// PrintObject prints database object information
func (p *JSONPrinter) PrintObject(objType, name string) {
	p.outputJSON("info", "object", map[string]interface{}{
		"type": objType,
		"name": name,
	})
}

// DisplayMigrationTable prints migration table as JSON array
func (p *JSONPrinter) DisplayMigrationTable(dbType db.DatabaseType, tableName string, statuses []MigrationStatus) {
	output := JSONOutput{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     "info",
		Message:   "migration table",
		Data: map[string]interface{}{
			"database_type": dbType,
			"table_name":    tableName,
			"migrations":    statuses,
		},
	}
	jsonBytes, _ := json.Marshal(output)
	fmt.Println(string(jsonBytes))
}
