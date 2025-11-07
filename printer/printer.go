package printer

import "bloomdb/db"

// OutputLevel represents the severity level of output messages
type OutputLevel int

const (
	LevelSuccess OutputLevel = iota
	LevelWarning
	LevelError
	LevelInfo
)

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	Version     string
	Description string
	Type        string // "versioned" or "repeatable"
	Status      string // "baseline", "success", "pending", "below baseline"
	InstalledOn string
}

// Printer interface defines all output methods for BloomDB CLI
type Printer interface {
	PrintOutput(level OutputLevel, message string, args ...interface{})
	PrintSuccess(message string, args ...interface{})
	PrintWarning(message string, args ...interface{})
	PrintError(message string, args ...interface{})
	PrintInfo(message string, args ...interface{})
	PrintSeparator(title string)
	PrintCommand(cmd string)
	PrintSection(title string)
	PrintSectionEnd()
	PrintMigration(version, description, status string)
	PrintObject(objType, name string)
	DisplayMigrationTable(dbType db.DatabaseType, tableName string, statuses []MigrationStatus)
}
