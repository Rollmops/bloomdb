package cmd

import (
	"fmt"
	"os"
	"strings"

	"bloomdb/db"
	"bloomdb/loader"
	"bloomdb/logger"

	"github.com/jedib0t/go-pretty/v6/table"
)

type InfoCommand struct{}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	Version     string
	Description string
	Type        string // "versioned" or "repeatable"
	Status      string // "baseline", "success", "pending", "below baseline"
	InstalledOn string
}

func (i *InfoCommand) Run() {
	logger.Info("Starting info command")

	// Get migration path from root command
	migrationPath := GetMigrationPath()

	// Setup database connection
	setup := SetupDatabase()
	logger.Infof("Connected to %s database", setup.DBType)

	// Ensure migration table exists
	setup.EnsureTableExists()
	logger.Debugf("Migration table '%s' ensured to exist", setup.TableName)

	// Load migrations from filesystem
	logger.Debugf("Loading migrations from path: %s", migrationPath)
	versionedLoader := loader.NewVersionedMigrationLoader(migrationPath)
	versionedMigrations, err := versionedLoader.LoadMigrations()
	if err != nil {
		logger.Errorf("Error loading versioned migrations: %v", err)
		return
	}

	repeatableLoader := loader.NewRepeatableMigrationLoader(migrationPath)
	repeatableMigrations, err := repeatableLoader.LoadRepeatableMigrations()
	if err != nil {
		logger.Errorf("Error loading repeatable migrations: %v", err)
		return
	}

	// Get existing migration records from database
	existingRecords, err := setup.GetMigrationRecords()
	if err != nil {
		logger.Errorf("Error reading migration records: %v", err)
		return
	}

	// Find baseline version
	baselineVersion := findBaselineVersion(existingRecords)

	// Build migration status list
	statuses := buildMigrationStatuses(versionedMigrations, repeatableMigrations, existingRecords, baselineVersion)

	// Display the table
	displayMigrationTable(setup.DBType, setup.TableName, statuses)
}

// findBaselineVersion returns the baseline version from records
func findBaselineVersion(records []db.MigrationRecord) string {
	for _, record := range records {
		if strings.ToLower(record.Type) == "baseline" && record.Version != nil {
			return *record.Version
		}
	}
	return ""
}

// buildMigrationStatuses creates a comprehensive list of migration statuses
func buildMigrationStatuses(versionedMigrations []*loader.VersionedMigration, repeatableMigrations []*loader.RepeatableMigration, records []db.MigrationRecord, baselineVersion string) []MigrationStatus {
	var statuses []MigrationStatus

	// Create a map of existing records for quick lookup
	recordMap := make(map[string]db.MigrationRecord)
	for _, record := range records {
		if record.Version != nil && *record.Version != "" {
			recordMap[*record.Version] = record
		} else {
			// For repeatable migrations, use description as key
			recordMap[record.Description] = record
		}
	}

	// Process versioned migrations
	for _, migration := range versionedMigrations {
		status := MigrationStatus{
			Version:     migration.Version,
			Description: migration.Description,
			Type:        "versioned",
		}

		// Check if below baseline first
		if baselineVersion != "" && loader.CompareVersions(migration.Version, baselineVersion) < 0 {
			status.Status = "below baseline"
		} else if record, exists := recordMap[migration.Version]; exists {
			// Only show as "success" if it was actually applied, not just baselined
			if record.Type == "baseline" {
				// If this version was baselined, it means the migration wasn't actually executed
				// So it should be considered as "pending" if there are newer migrations
				if migration.Version == baselineVersion {
					status.Status = "baseline"
				} else {
					status.Status = "pending"
				}
			} else {
				// Convert success flag to status string
				if record.Success == 1 {
					status.Status = "success"
				} else {
					status.Status = "failed"
				}
			}
			status.InstalledOn = record.InstalledOn
		} else {
			status.Status = "pending"
		}

		statuses = append(statuses, status)
	}

	// Process repeatable migrations
	for _, migration := range repeatableMigrations {
		status := MigrationStatus{
			Version:     "",
			Description: migration.Description,
			Type:        "repeatable",
		}

		if record, exists := recordMap[migration.Description]; exists {
			// Convert success flag to status string
			if record.Success == 1 {
				status.Status = "success"
			} else {
				status.Status = "failed"
			}
			status.InstalledOn = record.InstalledOn
		} else {
			status.Status = "pending"
		}

		statuses = append(statuses, status)
	}

	return statuses
}

// displayMigrationTable prints a formatted table of migration statuses
func displayMigrationTable(dbType db.DatabaseType, tableName string, statuses []MigrationStatus) {
	logger.Infof("Displaying migration table for %s database, table '%s'", dbType, tableName)

	// Print header with styling
	fmt.Printf("\n%s %s %s\n",
		colorize("┌─", "cyan"),
		colorize(fmt.Sprintf("Migration Status for %s Database", strings.ToUpper(string(dbType))), "bold"),
		colorize("─┐", "cyan"))
	fmt.Printf("%s Table: %s %s\n",
		colorize("│", "cyan"),
		colorize(tableName, "yellow"),
		colorize("│", "cyan"))
	fmt.Printf("%s\n", colorize("└─────────────────────────────────────────┘", "cyan"))
	fmt.Println()

	// Create a new table
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.Style().Options.SeparateRows = false
	t.Style().Options.SeparateColumns = true
	t.Style().Options.SeparateHeader = true

	// Add header
	t.AppendHeader(table.Row{
		colorize("VERSION", "bold+blue"),
		colorize("DESCRIPTION", "bold+blue"),
		colorize("TYPE", "bold+blue"),
		colorize("STATUS", "bold+blue"),
		colorize("INSTALLED ON", "bold+blue"),
	})

	// Add rows
	for _, status := range statuses {
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
			status.Description,
			typeColored,
			statusColored,
			installedOn,
		})
	}

	// Render the table
	t.Render()

	// Print summary with styling
	fmt.Println()
	printSummary(statuses)
}

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

	// Handle multiple color types (e.g., "bold+blue")
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

// printSummary prints a summary of migration statuses
func printSummary(statuses []MigrationStatus) {
	var baselineCount, successCount, pendingCount, belowBaselineCount, failedCount int

	for _, status := range statuses {
		switch status.Status {
		case "baseline":
			baselineCount++
		case "success":
			successCount++
		case "pending":
			pendingCount++
		case "below baseline":
			belowBaselineCount++
		case "failed":
			failedCount++
		}
	}

	// Create summary table with go-pretty
	summaryTable := table.NewWriter()
	summaryTable.SetOutputMirror(os.Stdout)
	summaryTable.SetStyle(table.StyleRounded)
	summaryTable.Style().Options.SeparateRows = false
	summaryTable.Style().Options.SeparateColumns = true
	summaryTable.Style().Options.SeparateHeader = true

	// Add header
	summaryTable.AppendHeader(table.Row{
		colorize("Status", "bold+blue"),
		colorize("Count", "bold+blue"),
	})

	// Add rows for each status type
	if baselineCount > 0 {
		summaryTable.AppendRow(table.Row{
			colorize("◉ Baseline", "cyan"),
			baselineCount,
		})
	}
	if successCount > 0 {
		summaryTable.AppendRow(table.Row{
			colorize("✓ Success", "bold+green"),
			successCount,
		})
	}
	if pendingCount > 0 {
		summaryTable.AppendRow(table.Row{
			colorize("○ Pending", "yellow"),
			pendingCount,
		})
	}
	if belowBaselineCount > 0 {
		summaryTable.AppendRow(table.Row{
			colorize("⊘ Below Baseline", "dim"),
			belowBaselineCount,
		})
	}
	if failedCount > 0 {
		summaryTable.AppendRow(table.Row{
			colorize("✗ Failed", "bold+red"),
			failedCount,
		})
	}

	// Add title row
	titleTable := table.NewWriter()
	titleTable.SetOutputMirror(os.Stdout)
	titleTable.SetStyle(table.StyleRounded)
	titleTable.Style().Options.SeparateRows = false
	titleTable.Style().Options.SeparateColumns = false
	titleTable.Style().Options.SeparateHeader = false

	titleTable.AppendRow(table.Row{
		colorize("Migration Summary", "bold+white"),
	})
	titleTable.Render()

	// Render the summary table
	summaryTable.Render()

	// Log the summary for debugging
	summary := fmt.Sprintf("Summary: %d baseline, %d success, %d pending, %d below baseline, %d failed",
		baselineCount, successCount, pendingCount, belowBaselineCount, failedCount)
	logger.Infof("Migration summary: %s", summary)
}
