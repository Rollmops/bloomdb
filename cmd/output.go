package cmd

import (
	"bloomdb/db"
	"bloomdb/printer"
)

// Re-export types from printer package for backward compatibility
type OutputLevel = printer.OutputLevel
type MigrationStatus = printer.MigrationStatus
type Printer = printer.Printer

// Re-export constants from printer package
const (
	LevelSuccess = printer.LevelSuccess
	LevelWarning = printer.LevelWarning
	LevelError   = printer.LevelError
	LevelInfo    = printer.LevelInfo
)

// Global printer instance
var printerInstance printer.Printer

// InitPrinter initializes the global printer based on environment variables
func InitPrinter() {
	printerInstance = printer.New()
}

// Package-level print functions that delegate to global printer

// PrintOutput prints formatted output using the global printer
func PrintOutput(level OutputLevel, message string, args ...interface{}) {
	if printerInstance == nil {
		InitPrinter()
	}
	printerInstance.PrintOutput(level, message, args...)
}

// PrintSuccess prints a success message
func PrintSuccess(message string, args ...interface{}) {
	if printerInstance == nil {
		InitPrinter()
	}
	printerInstance.PrintSuccess(message, args...)
}

// PrintWarning prints a warning message
func PrintWarning(message string, args ...interface{}) {
	if printerInstance == nil {
		InitPrinter()
	}
	printerInstance.PrintWarning(message, args...)
}

// PrintError prints an error message
func PrintError(message string, args ...interface{}) {
	if printerInstance == nil {
		InitPrinter()
	}
	printerInstance.PrintError(message, args...)
}

// PrintInfo prints an info message (only in verbose mode)
func PrintInfo(message string, args ...interface{}) {
	if printerInstance == nil {
		InitPrinter()
	}
	printerInstance.PrintInfo(message, args...)
}

// PrintSeparator prints a beautiful separator line
func PrintSeparator(title string) {
	if printerInstance == nil {
		InitPrinter()
	}
	printerInstance.PrintSeparator(title)
}

// PrintCommand prints a command being executed
func PrintCommand(cmd string) {
	if printerInstance == nil {
		InitPrinter()
	}
	printerInstance.PrintCommand(cmd)
}

// PrintSection prints a section header
func PrintSection(title string) {
	if printerInstance == nil {
		InitPrinter()
	}
	printerInstance.PrintSection(title)
}

// PrintSectionEnd prints a section footer
func PrintSectionEnd() {
	if printerInstance == nil {
		InitPrinter()
	}
	printerInstance.PrintSectionEnd()
}

// PrintMigration prints migration information with proper formatting
func PrintMigration(version, description, status string) {
	if printerInstance == nil {
		InitPrinter()
	}
	printerInstance.PrintMigration(version, description, status)
}

// PrintObject prints database object information
func PrintObject(objType, name string) {
	if printerInstance == nil {
		InitPrinter()
	}
	printerInstance.PrintObject(objType, name)
}

// DisplayMigrationTable prints a formatted table of migration statuses
func DisplayMigrationTable(dbType db.DatabaseType, tableName string, statuses []MigrationStatus) {
	if printerInstance == nil {
		InitPrinter()
	}
	printerInstance.DisplayMigrationTable(dbType, tableName, statuses)
}
