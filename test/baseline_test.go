package test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBaseline_Version1 tests baselining with version 1
func TestBaseline_Version1(t *testing.T) {
	ctx := NewTestContext(t)

	// Verify table doesn't exist initially
	exists, err := ctx.TableExists("BLOOMDB_VERSION")
	require.NoError(t, err, "Failed to check table existence")
	assert.False(t, exists, "Expected BLOOMDB_VERSION table to not exist initially")

	// Run baseline command with version 1
	output, err := ctx.RunCommand("baseline", "--version", "1")
	require.NoError(t, err, "Baseline command failed")

	t.Logf("Baseline output:\n%s", output)

	// Verify success message in JSON output
	AssertSuccessMessage(t, output, "Baseline completed successfully")

	// Verify table now exists
	exists, err = ctx.TableExists("BLOOMDB_VERSION")
	require.NoError(t, err, "Failed to check table existence after baseline")
	assert.True(t, exists, "Expected BLOOMDB_VERSION table to exist after baseline")

	// Verify baseline record exists in database
	records, err := ctx.GetMigrationRecords()
	require.NoError(t, err, "Failed to get migration records")
	require.Len(t, records, 1, "Expected 1 baseline record")

	record := records[0]
	assert.Equal(t, "1", record.Version, "Expected version '1'")
	assert.Equal(t, "BASELINE", record.Type, "Expected type 'BASELINE'")
	assert.Equal(t, 1, record.Success, "Expected success 1")
	assert.Equal(t, "<< Baseline >>", record.Description, "Expected description '<< Baseline >>'")

	t.Logf("Baseline record: %+v", record)
}

// TestBaseline_Version0_2 tests baselining with version 0.2
func TestBaseline_Version0_2(t *testing.T) {
	ctx := NewTestContext(t)

	// Verify table doesn't exist initially
	exists, err := ctx.TableExists("BLOOMDB_VERSION")
	require.NoError(t, err, "Failed to check table existence")
	assert.False(t, exists, "Expected BLOOMDB_VERSION table to not exist initially")

	// Run baseline command with version 0.2
	output, err := ctx.RunCommand("baseline", "--version", "0.2")
	require.NoError(t, err, "Baseline command failed")

	t.Logf("Baseline output:\n%s", output)

	// Verify success message in JSON output
	AssertSuccessMessage(t, output, "Baseline completed successfully")

	// Verify table now exists
	exists, err = ctx.TableExists("BLOOMDB_VERSION")
	require.NoError(t, err, "Failed to check table existence after baseline")
	assert.True(t, exists, "Expected BLOOMDB_VERSION table to exist after baseline")

	// Verify baseline record exists in database
	records, err := ctx.GetMigrationRecords()
	require.NoError(t, err, "Failed to get migration records")
	require.Len(t, records, 1, "Expected 1 baseline record")

	record := records[0]
	assert.Equal(t, "0.2", record.Version, "Expected version '0.2'")
	assert.Equal(t, "BASELINE", record.Type, "Expected type 'BASELINE'")
	assert.Equal(t, 1, record.Success, "Expected success 1")
	assert.Equal(t, "<< Baseline >>", record.Description, "Expected description '<< Baseline >>'")

	t.Logf("Baseline record: %+v", record)
}

// TestBaseline_DuplicateBaseline tests that running baseline twice shows a warning
func TestBaseline_DuplicateBaseline(t *testing.T) {
	ctx := NewTestContext(t)

	// Run baseline first time
	output1, err := ctx.RunCommand("baseline", "--version", "1")
	require.NoError(t, err, "First baseline command failed")

	AssertSuccessMessage(t, output1, "Baseline completed successfully")

	// Run baseline second time with same version
	output2, err := ctx.RunCommand("baseline", "--version", "1")
	require.NoError(t, err, "Second baseline command failed")

	t.Logf("Second baseline output:\n%s", output2)

	// Should show success message about existing baseline (changed from warning)
	AssertSuccessMessage(t, output2, "Baseline already exists with version 1")

	// Verify still only one record in database
	records, err := ctx.GetMigrationRecords()
	require.NoError(t, err, "Failed to get migration records")
	assert.Len(t, records, 1, "Expected 1 baseline record after duplicate baseline")
}

// TestBaseline_DifferentVersion tests the behavior when trying to baseline with a different version
// Current behavior: Shows a warning with the existing baseline version, doesn't create a new record
// Note: This test documents current behavior. Consider if this should error instead.
func TestBaseline_DifferentVersion(t *testing.T) {
	ctx := NewTestContext(t)

	// Run baseline first time with version 1
	output1, err := ctx.RunCommand("baseline", "--version", "1")
	require.NoError(t, err, "First baseline command failed")

	AssertSuccessMessage(t, output1, "Baseline completed successfully")

	// Verify baseline record exists
	records, err := ctx.GetMigrationRecords()
	require.NoError(t, err, "Failed to get migration records")
	require.Len(t, records, 1, "Expected 1 baseline record")
	assert.Equal(t, "1", records[0].Version, "Expected version '1'")

	// Run baseline second time with different version
	output2, err := ctx.RunCommand("baseline", "--version", "2")
	require.NoError(t, err, "Second baseline command should not error (current behavior)")

	t.Logf("Second baseline output:\n%s", output2)

	// Changed behavior: Shows success with existing version (the resolved version takes precedence)
	// The CLI flag --version 2 is ignored because an existing baseline already exists in the DB
	AssertSuccessMessage(t, output2, "Baseline already exists with version 1")

	// Verify original baseline record remains unchanged (no new record created)
	records, err = ctx.GetMigrationRecords()
	require.NoError(t, err, "Failed to get migration records")
	require.Len(t, records, 1, "Expected 1 baseline record (no new record created)")
	assert.Equal(t, "1", records[0].Version, "Expected original version '1' to remain unchanged")
}

// TestBaselineVersionPriority_ExistingDBBaseline tests that existing baseline in DB takes priority
func TestBaselineVersionPriority_ExistingDBBaseline(t *testing.T) {
	ctx := NewTestContext(t)

	// Create initial baseline with version "1.0"
	output1, err := ctx.RunCommand("baseline", "--version", "1.0")
	require.NoError(t, err, "First baseline command failed")
	AssertSuccessMessage(t, output1, "Baseline completed successfully")

	// Verify initial baseline
	records, err := ctx.GetMigrationRecords()
	require.NoError(t, err, "Failed to get migration records")
	require.Len(t, records, 1, "Expected 1 baseline record")
	assert.Equal(t, "1.0", records[0].Version, "Expected version '1.0'")

	// Try to create baseline again with different CLI flag version "2.0"
	// The existing DB baseline should take priority over the CLI flag
	output2, err := ctx.RunCommand("baseline", "--version", "2.0")
	require.NoError(t, err, "Second baseline command should not error")

	t.Logf("Second baseline output:\n%s", output2)

	// Should use existing DB baseline version "1.0" (priority 1), not CLI flag "2.0"
	AssertInfoMessage(t, output2, "Using existing baseline version from database: 1.0")
	AssertSuccessMessage(t, output2, "Baseline already exists with version 1.0")

	// Verify no new record created
	records, err = ctx.GetMigrationRecords()
	require.NoError(t, err, "Failed to get migration records")
	require.Len(t, records, 1, "Expected 1 baseline record (no new record)")
	assert.Equal(t, "1.0", records[0].Version, "Expected original version '1.0' unchanged")
}

// TestBaselineVersionPriority_CLIFlag tests that CLI flag is used when no DB baseline exists
func TestBaselineVersionPriority_CLIFlag(t *testing.T) {
	ctx := NewTestContext(t)

	// Run baseline with CLI flag (no existing baseline)
	output, err := ctx.RunCommand("baseline", "--version", "3.5")
	require.NoError(t, err, "Baseline command failed")

	t.Logf("Baseline output:\n%s", output)

	// Should use CLI flag version
	AssertInfoMessage(t, output, "Using baseline version from CLI flag: 3.5")
	AssertSuccessMessage(t, output, "Baseline completed successfully")

	// Verify baseline record with CLI flag version
	records, err := ctx.GetMigrationRecords()
	require.NoError(t, err, "Failed to get migration records")
	require.Len(t, records, 1, "Expected 1 baseline record")
	assert.Equal(t, "3.5", records[0].Version, "Expected version '3.5' from CLI flag")
}

// TestBaselineVersionPriority_EnvVar tests that env var is used when no CLI flag provided
func TestBaselineVersionPriority_EnvVar(t *testing.T) {
	ctx := NewTestContext(t)

	// Set environment variable for baseline version
	oldEnv := os.Getenv("BLOOMDB_BASELINE_VERSION")
	os.Setenv("BLOOMDB_BASELINE_VERSION", "2.5")
	defer func() {
		if oldEnv != "" {
			os.Setenv("BLOOMDB_BASELINE_VERSION", oldEnv)
		} else {
			os.Unsetenv("BLOOMDB_BASELINE_VERSION")
		}
	}()

	// Run baseline WITHOUT --version flag (so env var should be used)
	output, err := ctx.RunCommand("baseline")
	require.NoError(t, err, "Baseline command failed")

	t.Logf("Baseline output:\n%s", output)

	// Should use environment variable version
	AssertInfoMessage(t, output, "Using baseline version from environment: 2.5")
	AssertSuccessMessage(t, output, "Baseline completed successfully")

	// Verify baseline record with env var version
	records, err := ctx.GetMigrationRecords()
	require.NoError(t, err, "Failed to get migration records")
	require.Len(t, records, 1, "Expected 1 baseline record")
	assert.Equal(t, "2.5", records[0].Version, "Expected version '2.5' from env var")
}

// TestBaselineVersionPriority_Default tests that default "1" is used when nothing else provided
func TestBaselineVersionPriority_Default(t *testing.T) {
	ctx := NewTestContext(t)

	// Ensure env var is not set
	oldEnv := os.Getenv("BLOOMDB_BASELINE_VERSION")
	os.Unsetenv("BLOOMDB_BASELINE_VERSION")
	defer func() {
		if oldEnv != "" {
			os.Setenv("BLOOMDB_BASELINE_VERSION", oldEnv)
		}
	}()

	// Run baseline WITHOUT --version flag and WITHOUT env var
	output, err := ctx.RunCommand("baseline")
	require.NoError(t, err, "Baseline command failed")

	t.Logf("Baseline output:\n%s", output)

	// Should use default version "1"
	AssertInfoMessage(t, output, "Using default baseline version: 1")
	AssertSuccessMessage(t, output, "Baseline completed successfully")

	// Verify baseline record with default version
	records, err := ctx.GetMigrationRecords()
	require.NoError(t, err, "Failed to get migration records")
	require.Len(t, records, 1, "Expected 1 baseline record")
	assert.Equal(t, "1", records[0].Version, "Expected default version '1'")
}

// TestBaselineVersionPriority_CLIOverridesEnv tests that CLI flag overrides env var
func TestBaselineVersionPriority_CLIOverridesEnv(t *testing.T) {
	ctx := NewTestContext(t)

	// Set environment variable
	oldEnv := os.Getenv("BLOOMDB_BASELINE_VERSION")
	os.Setenv("BLOOMDB_BASELINE_VERSION", "5.0")
	defer func() {
		if oldEnv != "" {
			os.Setenv("BLOOMDB_BASELINE_VERSION", oldEnv)
		} else {
			os.Unsetenv("BLOOMDB_BASELINE_VERSION")
		}
	}()

	// Run baseline WITH --version flag (should override env var)
	output, err := ctx.RunCommand("baseline", "--version", "6.0")
	require.NoError(t, err, "Baseline command failed")

	t.Logf("Baseline output:\n%s", output)

	// Should use CLI flag version (priority 2) over env var (priority 3)
	AssertInfoMessage(t, output, "Using baseline version from CLI flag: 6.0")
	AssertSuccessMessage(t, output, "Baseline completed successfully")

	// Verify baseline record with CLI flag version (not env var version)
	records, err := ctx.GetMigrationRecords()
	require.NoError(t, err, "Failed to get migration records")
	require.Len(t, records, 1, "Expected 1 baseline record")
	assert.Equal(t, "6.0", records[0].Version, "Expected version '6.0' from CLI flag (not '5.0' from env)")
}
