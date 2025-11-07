package cmd

import (
	"testing"
)

func TestMigrateCommand(t *testing.T) {
	if migrateCmd.Use != "migrate" {
		t.Errorf("Expected migrate command use 'migrate', got '%s'", migrateCmd.Use)
	}

	if migrateCmd.Short == "" {
		t.Error("Expected migrate command to have a short description")
	}

	if migrateCmd.Long == "" {
		t.Error("Expected migrate command to have a long description")
	}
}

func TestInfoCommand(t *testing.T) {
	if infoCmd.Use != "info" {
		t.Errorf("Expected info command use 'info', got '%s'", infoCmd.Use)
	}

	if infoCmd.Short == "" {
		t.Error("Expected info command to have a short description")
	}

	if infoCmd.Long == "" {
		t.Error("Expected info command to have a long description")
	}
}

func TestRepairCommand(t *testing.T) {
	if repairCmd.Use != "repair" {
		t.Errorf("Expected repair command use 'repair', got '%s'", repairCmd.Use)
	}

	if repairCmd.Short == "" {
		t.Error("Expected repair command to have a short description")
	}

	if repairCmd.Long == "" {
		t.Error("Expected repair command to have a long description")
	}
}

func TestBaselineCommand(t *testing.T) {
	if baselineCmd.Use != "baseline" {
		t.Errorf("Expected baseline command use 'baseline', got '%s'", baselineCmd.Use)
	}

	if baselineCmd.Short == "" {
		t.Error("Expected baseline command to have a short description")
	}

	if baselineCmd.Long == "" {
		t.Error("Expected baseline command to have a long description")
	}
}

func TestDestroyCommand(t *testing.T) {
	if destroyCmd.Use != "destroy" {
		t.Errorf("Expected destroy command use 'destroy', got '%s'", destroyCmd.Use)
	}

	if destroyCmd.Short == "" {
		t.Error("Expected destroy command to have a short description")
	}

	if destroyCmd.Long == "" {
		t.Error("Expected destroy command to have a long description")
	}
}

func TestRootCommand(t *testing.T) {
	if rootCmd.Use != "bloomdb" {
		t.Errorf("Expected root command use 'bloomdb', got '%s'", rootCmd.Use)
	}

	if rootCmd.Short == "" {
		t.Error("Expected root command to have a short description")
	}

	if rootCmd.Long == "" {
		t.Error("Expected root command to have a long description")
	}

	expectedCommands := []string{"migrate", "info", "repair", "baseline", "destroy"}
	for _, expected := range expectedCommands {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected root command to have '%s' subcommand", expected)
		}
	}
}
