package printer

import (
	"fmt"
	"os"
	"strings"

	"bloomdb/db"
	"github.com/jedib0t/go-pretty/v6/table"
)

// Color codes for terminal output
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorGray   = "\033[90m"
	ColorBold   = "\033[1m"
)

// HumanPrinter implements Printer for human-readable terminal output
type HumanPrinter struct {
	verbose bool
}

// NewHumanPrinter creates a new human-readable console printer
func NewHumanPrinter(verbose bool) *HumanPrinter {
	return &HumanPrinter{
		verbose: verbose,
	}
}

// PrintOutput prints formatted output with colors and icons
func (p *HumanPrinter) PrintOutput(level OutputLevel, message string, args ...interface{}) {
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}

	var icon, color string
	switch level {
	case LevelSuccess:
		icon = "✓"
		color = ColorGreen
	case LevelWarning:
		icon = "⚠"
		color = ColorYellow
	case LevelError:
		icon = "✗"
		color = ColorRed
	case LevelInfo:
		icon = "ℹ"
		color = ColorBlue
	default:
		icon = "•"
		color = ColorWhite
	}

	// Only print if not info level or if verbose mode is enabled
	if level != LevelInfo || p.verbose {
		fmt.Printf("%s%s%s%s%s %s\n", ColorBold, color, icon, ColorReset, ColorReset, message)
	}
}

// PrintSuccess prints a success message
func (p *HumanPrinter) PrintSuccess(message string, args ...interface{}) {
	p.PrintOutput(LevelSuccess, message, args...)
}

// PrintWarning prints a warning message
func (p *HumanPrinter) PrintWarning(message string, args ...interface{}) {
	p.PrintOutput(LevelWarning, message, args...)
}

// PrintError prints an error message
func (p *HumanPrinter) PrintError(message string, args ...interface{}) {
	p.PrintOutput(LevelError, message, args...)
}

// PrintInfo prints an info message (only in verbose mode)
func (p *HumanPrinter) PrintInfo(message string, args ...interface{}) {
	p.PrintOutput(LevelInfo, message, args...)
}

// PrintSeparator prints a beautiful separator line
func (p *HumanPrinter) PrintSeparator(title string) {
	if title != "" {
		title = fmt.Sprintf(" %s ", title)
	}
	line := strings.Repeat("═", 50)
	titleLen := len(title)
	if titleLen > 0 {
		half := (50 - titleLen) / 2
		line = strings.Repeat("═", half) + title + strings.Repeat("═", 50-titleLen-half)
	}
	fmt.Printf("%s%s%s%s\n", ColorCyan, ColorBold, line, ColorReset)
}

// PrintCommand prints a command being executed
func (p *HumanPrinter) PrintCommand(cmd string) {
	fmt.Printf("%s%s➜%s Executing: %s%s\n", ColorPurple, ColorBold, ColorReset, cmd, ColorReset)
}

// PrintSection prints a section header
func (p *HumanPrinter) PrintSection(title string) {
	fmt.Printf("\n%s%s┌─ %s%s\n", ColorBlue, ColorBold, title, ColorReset)
	fmt.Printf("%s%s│%s\n", ColorBlue, ColorBold, ColorReset)
}

// PrintSectionEnd prints a section footer
func (p *HumanPrinter) PrintSectionEnd() {
	fmt.Printf("%s%s└─%s\n", ColorBlue, ColorBold, ColorReset)
}

// PrintMigration prints migration information with proper formatting
func (p *HumanPrinter) PrintMigration(version, description, status string) {
	var statusColor, statusIcon string
	switch strings.ToLower(status) {
	case "success":
		statusColor = ColorGreen
		statusIcon = "✓"
	case "failed":
		statusColor = ColorRed
		statusIcon = "✗"
	case "pending":
		statusColor = ColorYellow
		statusIcon = "⏳"
	case "baseline":
		statusColor = ColorCyan
		statusIcon = "📍"
	default:
		statusColor = ColorGray
		statusIcon = "?"
	}

	fmt.Printf("  %s%s%s%s %s%-8s%s %s%s%s\n",
		ColorBold, statusColor, statusIcon, ColorReset,
		ColorGray, version, ColorReset,
		ColorWhite, description, ColorReset)
}

// PrintObject prints database object information
func (p *HumanPrinter) PrintObject(objType, name string) {
	fmt.Printf("    %s%s•%s %s%s: %s%s\n",
		ColorGreen, ColorBold, ColorReset,
		ColorCyan, objType, ColorReset,
		name)
}

// DisplayMigrationTable prints a formatted table of migration statuses
func (p *HumanPrinter) DisplayMigrationTable(dbType db.DatabaseType, tableName string, statuses []MigrationStatus) {
	// Create a new table
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	// Customize the style to use thin separators
	style := table.StyleRounded
	style.Options.SeparateRows = false
	style.Options.SeparateColumns = true
	style.Options.SeparateHeader = true

	// Customize separator to be a thin line
	style.Box.MiddleSeparator = "─"
	style.Box.PaddingLeft = " "
	style.Box.PaddingRight = " "

	t.SetStyle(style)

	// Add header
	t.AppendHeader(table.Row{
		colorize("VERSION", "bold+blue"),
		colorize("DESCRIPTION", "bold+blue"),
		colorize("TYPE", "bold+blue"),
		colorize("STATUS", "bold+blue"),
		colorize("INSTALLED ON", "bold+blue"),
	})

	// Track if we've crossed the baseline boundary
	separatorAdded := false

	// Add rows
	for i, status := range statuses {
		// Check if we need to add a separator
		// Add separator after the last "below baseline" or "baseline" entry
		if !separatorAdded && i > 0 {
			prevStatus := statuses[i-1].Status
			currStatus := status.Status

			// If previous was below baseline/baseline and current is not
			if (prevStatus == "below baseline" || prevStatus == "baseline") &&
				(currStatus != "below baseline" && currStatus != "baseline") {
				t.AppendSeparator()
				separatorAdded = true
			}
		}

		version := status.Version
		if version == "" {
			version = colorize("─", "dim")
		} else {
			version = colorize(version, "cyan")
		}

		installedOn := status.InstalledOn
		if installedOn == "" {
			installedOn = colorize("─", "dim")
		}

		// Colorize status
		statusColored := colorizeStatus(status.Status)

		// Colorize type
		typeColored := colorizeType(status.Type)

		t.AppendRow(table.Row{
			version,
			formatDescription(status.Description),
			typeColored,
			statusColored,
			installedOn,
		})
	}

	// Render the table
	t.Render()
}

// Helper functions for colorization

// colorize adds ANSI color codes to text for terminal output
func colorize(text, colorType string) string {
	colors := map[string]string{
		"reset":        "\033[0m",
		"bold":         "\033[1m",
		"dim":          "\033[2m",
		"red":          "\033[31m",
		"green":        "\033[32m",
		"yellow":       "\033[33m",
		"blue":         "\033[34m",
		"magenta":      "\033[35m",
		"cyan":         "\033[36m",
		"white":        "\033[37m",
		"bold+red":     "\033[1;31m",
		"bold+green":   "\033[1;32m",
		"bold+yellow":  "\033[1;33m",
		"bold+blue":    "\033[1;34m",
		"bold+magenta": "\033[1;35m",
		"bold+cyan":    "\033[1;36m",
		"bold+white":   "\033[1;37m",
	}

	// Check if the exact color type exists first (e.g., "bold+blue")
	if code, exists := colors[colorType]; exists {
		return code + text + colors["reset"]
	}

	// Handle multiple color types by splitting (e.g., "bold+blue")
	parts := strings.Split(colorType, "+")
	var codes []string
	for _, part := range parts {
		if code, exists := colors[part]; exists {
			codes = append(codes, code)
		}
	}

	if len(codes) == 0 {
		return text
	}

	return strings.Join(codes, "") + text + colors["reset"]
}

// colorizeStatus adds appropriate colors based on migration status
func colorizeStatus(status string) string {
	switch status {
	case "success":
		return colorize("✓ "+status, "bold+green")
	case "pending":
		return colorize("○ "+status, "yellow")
	case "baseline":
		return colorize("◉ "+status, "cyan")
	case "below baseline":
		return colorize("⊘ "+status, "dim")
	case "failed":
		return colorize("✗ "+status, "bold+red")
	case "missing":
		return colorize("✗ "+status, "bold+red")
	case "checksum":
		return colorize("⚠ "+status, "bold+red")
	default:
		return status
	}
}

// colorizeType adds colors based on migration type
func colorizeType(migrationType string) string {
	switch migrationType {
	case "versioned":
		return colorize(migrationType, "blue")
	case "repeatable":
		return colorize(migrationType, "magenta")
	default:
		return migrationType
	}
}

// formatDescription replaces underscores with spaces in migration descriptions
func formatDescription(description string) string {
	return strings.ReplaceAll(description, "_", " ")
}
