package printer

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_DefaultHumanPrinter(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("BLOOMDB_PRINTER")
	os.Unsetenv("BLOOMDB_VERBOSE")

	printer := New()

	assert.NotNil(t, printer)
	// Should return HumanPrinter by default
	_, ok := printer.(*HumanPrinter)
	assert.True(t, ok, "Default printer should be HumanPrinter")
}

func TestNew_HumanPrinterExplicit(t *testing.T) {
	// Set explicit human printer
	os.Setenv("BLOOMDB_PRINTER", "human")
	defer os.Unsetenv("BLOOMDB_PRINTER")

	os.Unsetenv("BLOOMDB_VERBOSE")

	printer := New()

	assert.NotNil(t, printer)
	_, ok := printer.(*HumanPrinter)
	assert.True(t, ok, "Should return HumanPrinter when explicitly set")
}

func TestNew_TestPrinter(t *testing.T) {
	os.Setenv("BLOOMDB_PRINTER", "test")
	defer os.Unsetenv("BLOOMDB_PRINTER")

	os.Unsetenv("BLOOMDB_VERBOSE")

	printer := New()

	assert.NotNil(t, printer)
	_, ok := printer.(*TestPrinter)
	assert.True(t, ok, "Should return TestPrinter when BLOOMDB_PRINTER=test")
}

func TestNew_VerboseMode(t *testing.T) {
	os.Unsetenv("BLOOMDB_PRINTER")
	os.Setenv("BLOOMDB_VERBOSE", "1")
	defer os.Unsetenv("BLOOMDB_VERBOSE")

	printer := New()

	assert.NotNil(t, printer)
	humanPrinter, ok := printer.(*HumanPrinter)
	assert.True(t, ok)
	assert.True(t, humanPrinter.verbose, "Verbose mode should be enabled")
}

func TestNew_VerboseWithTest(t *testing.T) {
	os.Setenv("BLOOMDB_PRINTER", "test")
	defer os.Unsetenv("BLOOMDB_PRINTER")

	os.Setenv("BLOOMDB_VERBOSE", "true")
	defer os.Unsetenv("BLOOMDB_VERBOSE")

	printer := New()

	assert.NotNil(t, printer)
	testPrinter, ok := printer.(*TestPrinter)
	assert.True(t, ok)
	assert.True(t, testPrinter.verbose, "Verbose mode should be enabled for Test printer")
}

func TestNew_NonVerboseMode(t *testing.T) {
	os.Unsetenv("BLOOMDB_PRINTER")
	os.Unsetenv("BLOOMDB_VERBOSE")

	printer := New()

	assert.NotNil(t, printer)
	humanPrinter, ok := printer.(*HumanPrinter)
	assert.True(t, ok)
	assert.False(t, humanPrinter.verbose, "Verbose mode should be disabled by default")
}

func TestNew_UnknownPrinterTypeFallsBackToHuman(t *testing.T) {
	os.Setenv("BLOOMDB_PRINTER", "unknown-type")
	defer os.Unsetenv("BLOOMDB_PRINTER")

	os.Unsetenv("BLOOMDB_VERBOSE")

	printer := New()

	assert.NotNil(t, printer)
	_, ok := printer.(*HumanPrinter)
	assert.True(t, ok, "Should fall back to HumanPrinter for unknown types")
}

func TestNewWithType_HumanPrinter(t *testing.T) {
	printer := NewWithType("human", false)

	assert.NotNil(t, printer)
	_, ok := printer.(*HumanPrinter)
	assert.True(t, ok, "Should return HumanPrinter")

	humanPrinter := printer.(*HumanPrinter)
	assert.False(t, humanPrinter.verbose)
}

func TestNewWithType_HumanPrinterVerbose(t *testing.T) {
	printer := NewWithType("human", true)

	assert.NotNil(t, printer)
	humanPrinter, ok := printer.(*HumanPrinter)
	assert.True(t, ok)
	assert.True(t, humanPrinter.verbose, "Verbose should be enabled")
}

func TestNewWithType_TestPrinter(t *testing.T) {
	printer := NewWithType("test", false)

	assert.NotNil(t, printer)
	_, ok := printer.(*TestPrinter)
	assert.True(t, ok, "Should return TestPrinter")

	testPrinter := printer.(*TestPrinter)
	assert.False(t, testPrinter.verbose)
}

func TestNewWithType_TestPrinterVerbose(t *testing.T) {
	printer := NewWithType("test", true)

	assert.NotNil(t, printer)
	testPrinter, ok := printer.(*TestPrinter)
	assert.True(t, ok)
	assert.True(t, testPrinter.verbose, "Verbose should be enabled")
}

func TestNewWithType_UnknownTypeFallsBackToHuman(t *testing.T) {
	printer := NewWithType("xml", false)

	assert.NotNil(t, printer)
	_, ok := printer.(*HumanPrinter)
	assert.True(t, ok, "Should fall back to HumanPrinter for unknown types")
}

func TestNewWithType_EmptyStringFallsBackToHuman(t *testing.T) {
	printer := NewWithType("", false)

	assert.NotNil(t, printer)
	_, ok := printer.(*HumanPrinter)
	assert.True(t, ok, "Should fall back to HumanPrinter for empty string")
}

func TestNewWithType_CaseSensitive(t *testing.T) {
	// Test that "JSON" (uppercase) falls back to human since the check is case-sensitive
	printer := NewWithType("JSON", false)

	assert.NotNil(t, printer)
	_, ok := printer.(*HumanPrinter)
	assert.True(t, ok, "Should be case-sensitive and fall back to HumanPrinter")
}

func TestNew_EnvironmentIsolation(t *testing.T) {
	// Test 1: Test printer
	os.Setenv("BLOOMDB_PRINTER", "test")
	printer1 := New()
	_, ok := printer1.(*TestPrinter)
	assert.True(t, ok)

	// Test 2: Change to human printer
	os.Setenv("BLOOMDB_PRINTER", "human")
	printer2 := New()
	_, ok = printer2.(*HumanPrinter)
	assert.True(t, ok)

	// Clean up
	os.Unsetenv("BLOOMDB_PRINTER")
}

func TestNew_VerboseValueVariations(t *testing.T) {
	tests := []struct {
		name            string
		verboseValue    string
		shouldBeVerbose bool
	}{
		{"Empty string", "", false},
		{"Value '1'", "1", true},
		{"Value 'true'", "true", true},
		{"Value 'false'", "false", true}, // Any non-empty value enables verbose
		{"Value 'yes'", "yes", true},
		{"Value '0'", "0", true}, // Any non-empty value enables verbose
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("BLOOMDB_PRINTER")

			if tt.verboseValue == "" {
				os.Unsetenv("BLOOMDB_VERBOSE")
			} else {
				os.Setenv("BLOOMDB_VERBOSE", tt.verboseValue)
				defer os.Unsetenv("BLOOMDB_VERBOSE")
			}

			printer := New()
			humanPrinter, ok := printer.(*HumanPrinter)
			assert.True(t, ok)
			assert.Equal(t, tt.shouldBeVerbose, humanPrinter.verbose)
		})
	}
}
