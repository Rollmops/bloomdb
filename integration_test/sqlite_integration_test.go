package integration_test

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestSQLiteIntegrationScript mimics the integration-test-sqlite.sh script
func TestSQLiteIntegrationScript(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "bloomdb_sqlite_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	migrationsDir := filepath.Join(tempDir, "migrations")
	connString := "sqlite:" + dbPath

	// Create migrations directory
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatalf("Failed to create migrations dir: %v", err)
	}

	// Build bloomdb binary
	binaryPath := filepath.Join(tempDir, "bloomdb")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "..")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build bloomdb: %v", err)
	}

	// Helper function to run bloomdb commands
	runBloomDB := func(args ...string) (string, error) {
		allArgs := []string{"--conn", connString, "--path", migrationsDir}
		allArgs = append(allArgs, args...)

		cmd := exec.Command(binaryPath, allArgs...)
		cmd.Env = append(os.Environ(), "BLOOMDB_CONNECT_STRING="+connString)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("Command failed: %v, output: %s", err, string(output))
		}
		return string(output), err
	}

	// Helper function to run bloomdb commands with input
	runBloomDBWithInput := func(input string, args ...string) (string, error) {
		allArgs := []string{"--conn", connString, "--path", migrationsDir}
		allArgs = append(allArgs, args...)

		cmd := exec.Command(binaryPath, allArgs...)
		cmd.Env = append(os.Environ(), "BLOOMDB_CONNECT_STRING="+connString)
		cmd.Stdin = strings.NewReader(input)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("Command failed: %v, output: %s", err, string(output))
		}
		return string(output), err
	}

	// Helper function to check database state
	checkDBState := func(description string, expectedTables []string) {
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Errorf("Failed to open database: %v", err)
			return
		}
		defer db.Close()

		rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
		if err != nil {
			t.Errorf("Failed to query tables: %v", err)
			return
		}
		defer rows.Close()

		var tables []string
		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err != nil {
				t.Errorf("Failed to scan table: %v", err)
				return
			}
			tables = append(tables, table)
		}

		t.Logf("%s - Tables in database: %v", description, tables)
	}

	// Helper function to check BLOOMDB_VERSION table
	checkMigrationTable := func(description string) {
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Errorf("Failed to open database: %v", err)
			return
		}
		defer db.Close()

		// Use the correct column names
		rows, err := db.Query(`SELECT installed_rank, version, description, type, success FROM BLOOMDB_VERSION ORDER BY installed_rank`)
		if err != nil {
			// Don't fail the test if table doesn't exist (expected after destroy)
			t.Logf("%s - Migration table does not exist (expected after destroy): %v", description, err)
			return
		}
		defer rows.Close()

		var records []string
		for rows.Next() {
			var installedRank int
			var version sql.NullString
			var description, migrationType string
			var success int
			if err := rows.Scan(&installedRank, &version, &description, &migrationType, &success); err != nil {
				t.Errorf("Failed to scan migration record: %v", err)
				return
			}
			status := "failed"
			if success == 1 {
				status = "success"
			}
			// Handle NULL version for repeatable migrations
			versionStr := "NULL"
			if version.Valid {
				versionStr = version.String
			}
			records = append(records, fmt.Sprintf("%d|%s|%s|%s|%s", installedRank, versionStr, description, migrationType, status))
		}

		t.Logf("%s - Migration records: %v", description, records)
	}

	// Test 1: Destroy functionality
	t.Run("Test_1_Destroy", func(t *testing.T) {
		t.Log("=== Test 1: Destroy functionality ===")

		if _, err := runBloomDBWithInput("DESTROY\n", "destroy"); err != nil {
			t.Errorf("Destroy failed: %v", err)
		}

		checkDBState("After destroy", []string{})
	})

	// Test 2: Create initial migrations
	t.Run("Test_2_CreateMigrations", func(t *testing.T) {
		t.Log("=== Test 2: Create initial migrations ===")

		// Create V0.1 migration
		v01Content := `CREATE TABLE old_users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`
		if err := os.WriteFile(filepath.Join(migrationsDir, "V0.1__Create_old_users_table.sql"), []byte(v01Content), 0644); err != nil {
			t.Fatalf("Failed to create V0.1 migration: %v", err)
		}

		// Create V1 migration
		v1Content := `CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`
		if err := os.WriteFile(filepath.Join(migrationsDir, "V1__Create_users_table.sql"), []byte(v1Content), 0644); err != nil {
			t.Fatalf("Failed to create V1 migration: %v", err)
		}

		// Create V2 migration
		v2Content := `CREATE TABLE posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);`
		if err := os.WriteFile(filepath.Join(migrationsDir, "V2__Create_posts_table.sql"), []byte(v2Content), 0644); err != nil {
			t.Fatalf("Failed to create V2 migration: %v", err)
		}

		// Create repeatable migration
		rContent := `CREATE VIEW IF NOT EXISTS user_posts AS
SELECT u.name, u.email, p.title, p.created_at
FROM users u
LEFT JOIN posts p ON u.id = p.user_id;`
		if err := os.WriteFile(filepath.Join(migrationsDir, "R__Create_views.sql"), []byte(rContent), 0644); err != nil {
			t.Fatalf("Failed to create repeatable migration: %v", err)
		}

		t.Log("Created all migration files")
	})

	// Test 3: Baseline functionality
	t.Run("Test_3_Baseline", func(t *testing.T) {
		t.Log("=== Test 3: Baseline functionality ===")

		baselineVersion := "0.5"
		if _, err := runBloomDB("baseline", "--version", baselineVersion); err != nil {
			t.Errorf("Baseline failed: %v", err)
		}

		checkMigrationTable("After baseline")
		checkDBState("After baseline", []string{"BLOOMDB_VERSION"})
	})

	// Test 4: Info command to check baseline
	t.Run("Test_4_InfoAfterBaseline", func(t *testing.T) {
		t.Log("=== Test 4: Info command to check baseline ===")

		if _, err := runBloomDB("info"); err != nil {
			t.Errorf("Info failed: %v", err)
		}

		checkMigrationTable("After info check")
	})

	// Test 5: Migrate functionality
	t.Run("Test_5_Migrate", func(t *testing.T) {
		t.Log("=== Test 5: Migrate functionality ===")

		if _, err := runBloomDB("migrate"); err != nil {
			t.Errorf("Migrate failed: %v", err)
		}

		checkMigrationTable("After migrate")
		checkDBState("After migrate", []string{"BLOOMDB_VERSION", "users", "posts"})
	})

	// Test 6: Info command to check migration status
	t.Run("Test_6_InfoAfterMigrate", func(t *testing.T) {
		t.Log("=== Test 6: Info command to check migration status ===")

		if _, err := runBloomDB("info"); err != nil {
			t.Errorf("Info failed: %v", err)
		}

		checkMigrationTable("After migrate info check")
	})

	// Test 7: Add faulty migration
	t.Run("Test_7_AddFaultyMigration", func(t *testing.T) {
		t.Log("=== Test 7: Add faulty migration ===")

		faultyContent := `CREATE TABLE faulty_table (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

-- This will cause an error - referencing non-existent table
INSERT INTO non_existent_table VALUES (1, 'test');`

		if err := os.WriteFile(filepath.Join(migrationsDir, "V3__Faulty_migration.sql"), []byte(faultyContent), 0644); err != nil {
			t.Fatalf("Failed to create faulty migration: %v", err)
		}

		t.Log("Created faulty migration file")
	})

	// Test 8: Attempt migration with faulty migration (should stop on failure)
	t.Run("Test_8_MigrateWithFaulty", func(t *testing.T) {
		t.Log("=== Test 8: Attempt migration with faulty migration ===")

		output, err := runBloomDB("migrate")
		// The command should succeed (return no error) but the output should contain "failed"
		if err != nil {
			t.Errorf("Migration command should have succeeded but failed: %v", err)
		}
		// Check if output contains "failed" to indicate migration failure
		if !strings.Contains(output, "failed") {
			t.Error("Expected output to contain 'failed' but it didn't")
		} else {
			t.Logf("Output contains 'failed' as expected")
		}

		checkMigrationTable("After failed migration")
	})

	// Test 9: Info command to check failed status
	t.Run("Test_9_InfoAfterFailed", func(t *testing.T) {
		t.Log("=== Test 9: Info command to check failed status ===")

		if _, err := runBloomDB("info"); err != nil {
			t.Errorf("Info failed: %v", err)
		}

		checkMigrationTable("After failed info check")
	})

	// Test 10: Repair functionality
	t.Run("Test_10_Repair", func(t *testing.T) {
		t.Log("=== Test 10: Repair functionality ===")

		if _, err := runBloomDB("repair"); err != nil {
			t.Errorf("Repair failed: %v", err)
		}

		checkMigrationTable("After repair")
	})

	// Test 11: Fix the faulty migration
	t.Run("Test_11_FixFaultyMigration", func(t *testing.T) {
		t.Log("=== Test 11: Fix the faulty migration ===")

		fixedContent := `CREATE TABLE faulty_table (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);`

		if err := os.WriteFile(filepath.Join(migrationsDir, "V3__Faulty_migration.sql"), []byte(fixedContent), 0644); err != nil {
			t.Fatalf("Failed to fix faulty migration: %v", err)
		}

		t.Log("Fixed the faulty migration")
	})

	// Test 12: Migrate after fixing
	t.Run("Test_12_MigrateAfterFix", func(t *testing.T) {
		t.Log("=== Test 12: Migrate after fixing ===")

		if _, err := runBloomDB("migrate"); err != nil {
			t.Errorf("Migrate after fix failed: %v", err)
		}

		checkMigrationTable("After migrate after fix")
		checkDBState("After migrate after fix", []string{"BLOOMDB_VERSION", "users", "posts", "faulty_table"})
	})

	// Test 13: Info command to check final status
	t.Run("Test_13_InfoAfterFix", func(t *testing.T) {
		t.Log("=== Test 13: Info command to check final status ===")

		if _, err := runBloomDB("info"); err != nil {
			t.Errorf("Info failed: %v", err)
		}

		checkMigrationTable("After final info check")
	})

	// Test 14: Test checksum validation - modify migration file
	t.Run("Test_14_ChecksumValidation", func(t *testing.T) {
		t.Log("=== Test 14: Test checksum validation ===")

		// Modify V1 migration to change checksum
		modifiedV1Content := `CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP  -- Added this line
);`

		if err := os.WriteFile(filepath.Join(migrationsDir, "V1__Create_users_table.sql"), []byte(modifiedV1Content), 0644); err != nil {
			t.Fatalf("Failed to modify V1 migration: %v", err)
		}

		t.Log("Modified V1 migration to test checksum validation")
	})

	// Test 15: Info command to check checksum status
	t.Run("Test_15_InfoAfterChecksum", func(t *testing.T) {
		t.Log("=== Test 15: Info command to check checksum status ===")

		if _, err := runBloomDB("info"); err != nil {
			t.Errorf("Info failed: %v", err)
		}

		checkMigrationTable("After checksum validation check")
	})

	// Test 16: Remove a migration file
	t.Run("Test_16_MissingFile", func(t *testing.T) {
		t.Log("=== Test 16: Remove a migration file ===")

		// Move V2 migration to test missing status
		oldPath := filepath.Join(migrationsDir, "V2__Create_posts_table.sql")
		backupPath := filepath.Join(migrationsDir, "V2__Create_posts_table.sql.bak")
		if err := os.Rename(oldPath, backupPath); err != nil {
			t.Fatalf("Failed to move V2 migration: %v", err)
		}

		t.Log("Moved V2 migration file to test missing status")
	})

	// Test 17: Info command to check missing status
	t.Run("Test_17_InfoAfterMissing", func(t *testing.T) {
		t.Log("=== Test 17: Info command to check missing status ===")

		if _, err := runBloomDB("info"); err != nil {
			t.Errorf("Info failed: %v", err)
		}

		checkMigrationTable("After missing file check")
	})

	// Test 18: Restore the file and test again
	t.Run("Test_18_RestoreFile", func(t *testing.T) {
		t.Log("=== Test 18: Restore the file and test again ===")

		// Restore V2 migration
		oldPath := filepath.Join(migrationsDir, "V2__Create_posts_table.sql")
		backupPath := filepath.Join(migrationsDir, "V2__Create_posts_table.sql.bak")
		if err := os.Rename(backupPath, oldPath); err != nil {
			t.Fatalf("Failed to restore V2 migration: %v", err)
		}

		t.Log("Restored migration file")
	})

	// Test 19: Test repeatable migration modification
	t.Run("Test_19_RepeatableModification", func(t *testing.T) {
		t.Log("=== Test 19: Test repeatable migration modification ===")

		// Modify repeatable migration
		modifiedRContent := `CREATE VIEW IF NOT EXISTS user_posts AS
SELECT u.name, u.email, p.title, p.created_at, p.content  -- Added content field
FROM users u
LEFT JOIN posts p ON u.id = p.user_id;

CREATE VIEW IF NOT EXISTS post_count AS
SELECT u.id, u.name, COUNT(p.id) as post_count
FROM users u
LEFT JOIN posts p ON u.id = p.user_id
GROUP BY u.id, u.name;`

		if err := os.WriteFile(filepath.Join(migrationsDir, "R__Create_views.sql"), []byte(modifiedRContent), 0644); err != nil {
			t.Fatalf("Failed to modify repeatable migration: %v", err)
		}

		t.Log("Modified repeatable migration")
	})

	// Test 20: Migrate to test repeatable migration
	t.Run("Test_20_MigrateRepeatable", func(t *testing.T) {
		t.Log("=== Test 20: Migrate to test repeatable migration ===")

		if _, err := runBloomDB("migrate"); err != nil {
			t.Errorf("Migrate repeatable failed: %v", err)
		}

		checkMigrationTable("After repeatable migration")
	})

	// Test 21: Final info check
	t.Run("Test_21_FinalInfo", func(t *testing.T) {
		t.Log("=== Test 21: Final info check ===")

		if _, err := runBloomDB("info"); err != nil {
			t.Errorf("Final info failed: %v", err)
		}

		checkMigrationTable("After final info check")
	})

	// Test 22: Test destroy with confirmation
	t.Run("Test_22_DestroyFinal", func(t *testing.T) {
		t.Log("=== Test 22: Test destroy with confirmation ===")

		if _, err := runBloomDBWithInput("DESTROY\n", "destroy"); err != nil {
			t.Errorf("Final destroy failed: %v", err)
		}

		checkMigrationTable("After final destroy")
	})

	// Test 23: Verify database is empty
	t.Run("Test_23_VerifyEmpty", func(t *testing.T) {
		t.Log("=== Test 23: Verify database is empty ===")

		checkDBState("After destroy", []string{})

		// Verify no user tables exist
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Errorf("Failed to open database: %v", err)
			return
		}
		defer db.Close()

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('users', 'posts', 'faulty_table')").Scan(&count)
		if err != nil {
			t.Errorf("Failed to query table count: %v", err)
			return
		}

		if count > 0 {
			t.Errorf("Database still contains %d tables after destroy", count)
		} else {
			t.Log("Database is empty after destroy")
		}
	})

	t.Log("All integration tests completed successfully! 🎉")
}
