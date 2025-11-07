package integration_test

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestBaselineCommand_InMemorySQLite tests the baseline command with an in-memory SQLite database
func TestBaselineCommand_InMemorySQLite(t *testing.T) {
	// Build the bloom binary first
	buildCmd := exec.Command("go", "build", "-o", "bloom-test", "..")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build bloom binary: %v", err)
	}
	defer os.Remove("bloom-test")

	// Test with in-memory SQLite
	connStr := "sqlite::memory:"

	// Run baseline command
	cmd := exec.Command("./bloom-test", "baseline", "--conn", connStr)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Baseline command failed: %v\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}

	// Check expected output
	output := stdout.String()
	expectedOutput := "baseline - connected to sqlite database, version: 1"

	if !strings.Contains(output, expectedOutput) {
		t.Errorf("Expected output to contain '%s', got: %s", expectedOutput, output)
	}

	// Verify no error output
	if stderr.Len() > 0 {
		t.Errorf("Unexpected error output: %s", stderr.String())
	}
}

// TestBaselineCommand_InvalidConnectionString tests error handling with invalid connection string
func TestBaselineCommand_InvalidConnectionString(t *testing.T) {
	// Build the bloom binary first
	buildCmd := exec.Command("go", "build", "-o", "bloom-test-invalid", "..")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build bloom binary: %v", err)
	}
	defer os.Remove("bloom-test-invalid")

	// Test with invalid connection string
	connStr := "invalid://connection/string"

	// Run baseline command
	cmd := exec.Command("./bloom-test-invalid", "baseline", "--conn", connStr)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Run()
	// Note: baseline command doesn't fail with invalid connection string, it prints error and continues

	// Check for error message in stdout (errors go to stdout, not stderr)
	output := stdout.String()
	expectedError := "Error creating database"
	if !strings.Contains(output, expectedError) {
		t.Errorf("Expected output to contain '%s', got: %s", expectedError, output)
	}
}

// TestBaselineCommand_NoConnectionString tests error handling when no connection string is provided
func TestBaselineCommand_NoConnectionString(t *testing.T) {
	// Build the bloom binary first
	buildCmd := exec.Command("go", "build", "-o", "bloom-test-noconn", "..")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build bloom binary: %v", err)
	}
	defer os.Remove("bloom-test-noconn")

	// Run baseline command without connection string
	cmd := exec.Command("./bloom-test-noconn", "baseline")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	// Command should fail when no connection string is provided
	if err == nil {
		t.Error("Expected baseline command to fail without connection string")
	}

	// Check for expected error message in stdout (errors go to stdout)
	output := stdout.String()
	expectedError := "connection string is required"
	if !strings.Contains(output, expectedError) {
		t.Errorf("Expected output to contain '%s', got: %s", expectedError, output)
	}
}

// TestBaselineCommand_EnvironmentVariable tests using environment variable for connection string
func TestBaselineCommand_EnvironmentVariable(t *testing.T) {
	// Build the bloom binary first
	buildCmd := exec.Command("go", "build", "-o", "bloom-test-env", "..")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build bloom binary: %v", err)
	}
	defer os.Remove("bloom-test-env")

	// Set environment variable
	connStr := "sqlite::memory:"
	os.Setenv("BLOOMDB_CONNECT_STRING", connStr)
	defer os.Unsetenv("BLOOMDB_CONNECT_STRING")

	// Run baseline command without --conn flag (should use env var)
	cmd := exec.Command("./bloom-test-env", "baseline")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(), "BLOOMDB_CONNECT_STRING=sqlite::memory:")

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Baseline command failed: %v\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}

	// Check expected output
	output := stdout.String()
	expectedOutput := "baseline - connected to sqlite database, version: 1"

	if !strings.Contains(output, expectedOutput) {
		t.Errorf("Expected output to contain '%s', got: %s", expectedOutput, output)
	}

	// Verify no error output
	if stderr.Len() > 0 {
		t.Errorf("Unexpected error output: %s", stderr.String())
	}
}

// TestBaselineCommand_VersionFlag tests baseline command with version flag
func TestBaselineCommand_VersionFlag(t *testing.T) {
	// Build the bloom binary first
	buildCmd := exec.Command("go", "build", "-o", "bloom-test-version", "..")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build bloom binary: %v", err)
	}
	defer os.Remove("bloom-test-version")

	// Run baseline command with version flag
	cmd := exec.Command("./bloom-test-version", "baseline", "--conn", "sqlite::memory:", "--baseline-version", "2.5")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Baseline command failed: %v\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}

	// Check expected output
	output := stdout.String()
	expectedOutput := "baseline - connected to sqlite database, version: 2.5"

	if !strings.Contains(output, expectedOutput) {
		t.Errorf("Expected output to contain '%s', got: %s", expectedOutput, output)
	}

	// Verify no error output
	if stderr.Len() > 0 {
		t.Errorf("Unexpected error output: %s", stderr.String())
	}
}

// TestBaselineCommand_VersionEnvironmentVariable tests baseline command with version environment variable
func TestBaselineCommand_VersionEnvironmentVariable(t *testing.T) {
	// Build the bloom binary first
	buildCmd := exec.Command("go", "build", "-o", "bloom-test-version-env", "..")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build bloom binary: %v", err)
	}
	defer os.Remove("bloom-test-version-env")

	// Set version environment variable
	os.Setenv("BLOOMDB_BASELINE_VERSION", "3.1")
	defer os.Unsetenv("BLOOMDB_BASELINE_VERSION")

	// Run baseline command without version flag (should use env var)
	cmd := exec.Command("./bloom-test-version-env", "baseline", "--conn", "sqlite::memory:")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(), "BLOOMDB_BASELINE_VERSION=3.1")

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Baseline command failed: %v\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}

	// Check expected output
	output := stdout.String()
	expectedOutput := "baseline - connected to sqlite database, version: 3.1"

	if !strings.Contains(output, expectedOutput) {
		t.Errorf("Expected output to contain '%s', got: %s", expectedOutput, output)
	}

	// Verify no error output
	if stderr.Len() > 0 {
		t.Errorf("Unexpected error output: %s", stderr.String())
	}
}

// TestBaselineCommand_DefaultVersion tests baseline command with default version when neither flag nor env is set
func TestBaselineCommand_DefaultVersion(t *testing.T) {
	// Build the bloom binary first
	buildCmd := exec.Command("go", "build", "-o", "bloom-test-default", "..")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build bloom binary: %v", err)
	}
	defer os.Remove("bloom-test-default")

	// Ensure no version environment variable is set
	os.Unsetenv("BLOOM_BASELINE_VERSION")

	// Run baseline command without version flag (should use default)
	cmd := exec.Command("./bloom-test-default", "baseline", "--conn", "sqlite::memory:")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Baseline command failed: %v\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}

	// Check expected output
	output := stdout.String()
	expectedOutput := "baseline - connected to sqlite database, version: 1"

	if !strings.Contains(output, expectedOutput) {
		t.Errorf("Expected output to contain '%s', got: %s", expectedOutput, output)
	}

	// Verify no error output
	if stderr.Len() > 0 {
		t.Errorf("Unexpected error output: %s", stderr.String())
	}
}

// TestBaselineCommand_TableAlreadyExists tests error handling when migration table already exists
func TestBaselineCommand_TableAlreadyExists(t *testing.T) {
	// Build the bloom binary first
	buildCmd := exec.Command("go", "build", "-o", "bloom-test-existing", "..")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build bloom binary: %v", err)
	}
	defer os.Remove("bloom-test-existing")

	// Use a file-based database so table persists between runs
	dbFile := "test-existing.db"
	os.Remove(dbFile) // Clean up any existing file from previous test runs
	defer os.Remove(dbFile)
	connStr := "sqlite:" + dbFile

	// First run - should create table successfully
	cmd1 := exec.Command("./bloom-test-existing", "baseline", "--conn", connStr)
	var stdout1, stderr1 bytes.Buffer
	cmd1.Stdout = &stdout1
	cmd1.Stderr = &stderr1

	err := cmd1.Run()
	if err != nil {
		t.Fatalf("First baseline command failed: %v\nStdout: %s\nStderr: %s", err, stdout1.String(), stderr1.String())
	}

	// Verify first run succeeded
	output1 := stdout1.String()
	if !strings.Contains(output1, "Creating migration table: BLOOMDB_VERSION") {
		t.Errorf("Expected first run to create table, got: %s", output1)
	}

	// Second run - should fail with table already exists error
	cmd2 := exec.Command("./bloom-test-existing", "baseline", "--conn", connStr)
	var stdout2, stderr2 bytes.Buffer
	cmd2.Stdout = &stdout2
	cmd2.Stderr = &stderr2

	err = cmd2.Run()
	// Command should fail with exit status 1 when table already exists
	if err == nil {
		t.Fatal("Second baseline command should have failed when table already exists")
	}

	// Check for expected error message in both stdout and stderr
	combinedOutput := stdout2.String() + stderr2.String()
	expectedError := "Migration table 'BLOOMDB_VERSION' already exists - have you already run the baseline command?"
	if !strings.Contains(combinedOutput, expectedError) {
		t.Errorf("Expected output to contain '%s', got: %s", expectedError, combinedOutput)
	}

	// Verify it doesn't contain success message
	if strings.Contains(combinedOutput, "Migration table BLOOMDB_VERSION created successfully") {
		t.Error("Second run should not create table again")
	}
}
