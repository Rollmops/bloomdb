package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitPrinter(t *testing.T) {
	// Reset global printer
	printerInstance = nil

	// Initialize printer
	InitPrinter()

	// Verify printer is initialized
	assert.NotNil(t, printerInstance, "Printer should be initialized")
}

func TestPrintFunctionsInitializePrinter(t *testing.T) {
	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "PrintSuccess",
			fn: func() {
				printerInstance = nil
				PrintSuccess("test")
				assert.NotNil(t, printerInstance)
			},
		},
		{
			name: "PrintWarning",
			fn: func() {
				printerInstance = nil
				PrintWarning("test")
				assert.NotNil(t, printerInstance)
			},
		},
		{
			name: "PrintError",
			fn: func() {
				printerInstance = nil
				PrintError("test")
				assert.NotNil(t, printerInstance)
			},
		},
		{
			name: "PrintInfo",
			fn: func() {
				printerInstance = nil
				PrintInfo("test")
				assert.NotNil(t, printerInstance)
			},
		},
		{
			name: "PrintCommand",
			fn: func() {
				printerInstance = nil
				PrintCommand("test")
				assert.NotNil(t, printerInstance)
			},
		},
		{
			name: "PrintSeparator",
			fn: func() {
				printerInstance = nil
				PrintSeparator("test")
				assert.NotNil(t, printerInstance)
			},
		},
		{
			name: "PrintSection",
			fn: func() {
				printerInstance = nil
				PrintSection("test")
				assert.NotNil(t, printerInstance)
			},
		},
		{
			name: "PrintSectionEnd",
			fn: func() {
				printerInstance = nil
				PrintSectionEnd()
				assert.NotNil(t, printerInstance)
			},
		},
		{
			name: "PrintMigration",
			fn: func() {
				printerInstance = nil
				PrintMigration("1.0", "test", "pending")
				assert.NotNil(t, printerInstance)
			},
		},
		{
			name: "PrintObject",
			fn: func() {
				printerInstance = nil
				PrintObject("table", "users")
				assert.NotNil(t, printerInstance)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn()
		})
	}
}
