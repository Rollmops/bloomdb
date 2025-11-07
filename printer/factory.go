package printer

import "os"

// New creates a new Printer based on environment variables
// Reads BLOOMDB_PRINTER for printer type (test/human, default: human)
// Reads BLOOMDB_VERBOSE for verbose mode
func New() Printer {
	printerType := os.Getenv("BLOOMDB_PRINTER")
	verbose := os.Getenv("BLOOMDB_VERBOSE") != ""

	switch printerType {
	case "test":
		return NewTestPrinter(verbose)
	default:
		return NewHumanPrinter(verbose)
	}
}

// NewWithType creates a new Printer of the specified type
func NewWithType(printerType string, verbose bool) Printer {
	switch printerType {
	case "test":
		return NewTestPrinter(verbose)
	default:
		return NewHumanPrinter(verbose)
	}
}
