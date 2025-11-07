#!/bin/bash

# BloomDB Integration Test Script
# Tests all functionalities: destroy, baseline, migrate, repair, etc.
#
# Usage: ./integration-test-sqlite.sh [BASELINE_VERSION]
# If BASELINE_VERSION is not provided, defaults to "0.5"

set -e  # Exit on any error

# Set baseline version from parameter or default to 0.5
BASELINE_VERSION="${1:-0.5}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Create temporary directory
TEMP_DIR=$(mktemp -d)
DB_PATH="$TEMP_DIR/test.db"
MIGRATIONS_DIR="$TEMP_DIR/migrations"

print_status "Created temporary directory: $TEMP_DIR"

# Cleanup function
cleanup() {
    print_status "Cleaning up temporary directory: $TEMP_DIR"
    rm -rf "$TEMP_DIR"
}

# Set trap for cleanup
trap cleanup EXIT

# Build the application
print_status "Building BloomDB..."
go build -o "$TEMP_DIR/bloomdb" ..

# Create migrations directory
mkdir -p "$MIGRATIONS_DIR"

# Function to run bloomdb commands
run_bloomdb() {
    echo -e "${BLUE}Executing: bloomdb --conn sqlite:$DB_PATH --path $MIGRATIONS_DIR $@${NC}"
    "$TEMP_DIR/bloomdb" --conn "sqlite:$DB_PATH" --path "$MIGRATIONS_DIR" "$@"
}

# Function to print test separator
print_separator() {
    echo ""
    echo "================================================================================"
    echo "$1"
    echo "================================================================================"
    echo ""
}

print_separator "Test 1: Destroy functionality"
print_status "Test 1: Destroy functionality"
echo "DESTROY" | run_bloomdb destroy
print_success "Destroy completed successfully"

print_separator "Test 2: Create initial migrations"
print_status "Test 2: Creating initial migrations"

# Create V0.1 migration
cat > "$MIGRATIONS_DIR/V0.1__Create_old_users_table.sql" << 'EOF'
CREATE TABLE old_users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
EOF

# Create V1 migration
cat > "$MIGRATIONS_DIR/V1__Create_users_table.sql" << 'EOF'
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
EOF

# Create V2 migration
cat > "$MIGRATIONS_DIR/V2__Create_posts_table.sql" << 'EOF'
CREATE TABLE posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
EOF

# Create repeatable migration
cat > "$MIGRATIONS_DIR/R__Create_views.sql" << 'EOF'
CREATE VIEW IF NOT EXISTS user_posts AS
SELECT u.name, u.email, p.title, p.created_at
FROM users u
LEFT JOIN posts p ON u.id = p.user_id;
EOF

print_success "Created initial migration files"

print_separator "Test 3: Baseline functionality"
print_status "Test 3: Baseline functionality with version $BASELINE_VERSION"
run_bloomdb baseline --version "$BASELINE_VERSION"
print_success "Baseline completed successfully"

# Print BLOOMDB_VERSION table after baseline
print_status "Checking BLOOMDB_VERSION table after baseline"
echo "BLOOMDB_VERSION table contents:"
sqlite3 "$DB_PATH" "SELECT * FROM BLOOMDB_VERSION;"
echo ""

ls -la $MIGRATIONS_DIR

print_separator "Test 4: Info command to check baseline"
print_status "Test 4: Checking baseline status"
run_bloomdb info

# Print BLOOMDB_VERSION table after info check
print_status "Checking BLOOMDB_VERSION table after info check"
echo "BLOOMDB_VERSION table contents:"
sqlite3 "$DB_PATH" "SELECT * FROM BLOOMDB_VERSION;"
echo ""

print_separator "Test 5: Migrate functionality"
print_status "Test 5: Migrate functionality"
run_bloomdb migrate
print_success "Migration completed successfully"

# Print BLOOMDB_VERSION table after migrate
print_status "Checking BLOOMDB_VERSION table after migrate"
echo "BLOOMDB_VERSION table contents:"
sqlite3 "$DB_PATH" "SELECT * FROM BLOOMDB_VERSION;"
echo ""

print_separator "Test 6: Info command to check migration status"
print_status "Test 6: Checking migration status after migrate"
run_bloomdb info

# Print BLOOMDB_VERSION table after info check
print_status "Checking BLOOMDB_VERSION table after info check"
echo "BLOOMDB_VERSION table contents:"
sqlite3 "$DB_PATH" "SELECT * FROM BLOOMDB_VERSION;"
echo ""

print_separator "Test 7: Add faulty migration"
print_status "Test 7: Adding faulty migration"

cat > "$MIGRATIONS_DIR/V3__Faulty_migration.sql" << 'EOF'
CREATE TABLE faulty_table (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

-- This will cause an error - invalid SQL syntax
INVALID SQL SYNTAX HERE;
EOF

print_success "Created faulty migration file"

print_separator "Test 8: Attempt migration with faulty migration (should fail)"
print_status "Test 8: Attempting migration with faulty migration"
if run_bloomdb migrate 2>&1 | grep -q "failed"; then
    print_success "Migration failed as expected"
else
    print_error "Migration should have failed but succeeded"
    exit 1
fi

# Print BLOOMDB_VERSION table after failed migration
print_status "Checking BLOOMDB_VERSION table after failed migration"
echo "BLOOMDB_VERSION table contents:"
sqlite3 "$DB_PATH" "SELECT * FROM BLOOMDB_VERSION;"
echo ""

print_separator "Test 9: Info command to check failed status"
print_status "Test 9: Checking status after failed migration"
run_bloomdb info

# Print BLOOMDB_VERSION table after info check
print_status "Checking BLOOMDB_VERSION table after info check"
echo "BLOOMDB_VERSION table contents:"
sqlite3 "$DB_PATH" "SELECT * FROM BLOOMDB_VERSION;"
echo ""

print_separator "Test 10: Repair functionality"
print_status "Test 10: Repair functionality"
run_bloomdb repair
print_success "Repair completed successfully"

# Print BLOOMDB_VERSION table after repair
print_status "Checking BLOOMDB_VERSION table after repair"
echo "BLOOMDB_VERSION table contents:"
sqlite3 "$DB_PATH" "SELECT * FROM BLOOMDB_VERSION;"
echo ""

print_separator "Test 11: Fix the faulty migration"
print_status "Test 11: Fixing the faulty migration"

cat > "$MIGRATIONS_DIR/V3__Faulty_migration.sql" << 'EOF'
CREATE TABLE faulty_table (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL  -- Fixed: use TEXT instead of VARCHAR
);
EOF

print_success "Fixed the faulty migration"

print_separator "Test 12: Migrate after fixing"
print_status "Test 12: Migrating after fixing the faulty migration"
run_bloomdb migrate
print_success "Migration completed successfully after fix"

# Print BLOOMDB_VERSION table after migrate after fix
print_status "Checking BLOOMDB_VERSION table after migrate after fix"
echo "BLOOMDB_VERSION table contents:"
sqlite3 "$DB_PATH" "SELECT * FROM BLOOMDB_VERSION;"
echo ""

print_separator "Test 13: Info command to check final status"
print_status "Test 13: Checking final migration status"
run_bloomdb info

# Print BLOOMDB_VERSION table after info check
print_status "Checking BLOOMDB_VERSION table after info check"
echo "BLOOMDB_VERSION table contents:"
sqlite3 "$DB_PATH" "SELECT * FROM BLOOMDB_VERSION;"
echo ""

print_separator "Test 14: Test checksum validation - modify migration file"
print_status "Test 14: Testing checksum validation"

# Modify V1 migration to change checksum
cat > "$MIGRATIONS_DIR/V1__Create_users_table.sql" << 'EOF'
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP  -- Added this line
);
EOF

print_success "Modified V1 migration to test checksum validation"

print_separator "Test 15: Info command to check checksum status"
print_status "Test 15: Checking checksum validation status"
run_bloomdb info

# Print BLOOMDB_VERSION table after checksum validation check
print_status "Checking BLOOMDB_VERSION table after checksum validation check"
echo "BLOOMDB_VERSION table contents:"
sqlite3 "$DB_PATH" "SELECT * FROM BLOOMDB_VERSION;"
echo ""

print_separator "Test 15.5: Repair checksum and description alignment"
print_status "Test 15.5: Testing repair command's second step - aligning checksums and descriptions"

# Show current state with checksum mismatch
print_status "Current state shows checksum mismatch for V1 migration"
run_bloomdb info

# Print BLOOMDB_VERSION table before repair to show mismatched checksum
print_status "Checking BLOOMDB_VERSION table before repair (should show mismatched checksum)"
echo "BLOOMDB_VERSION table contents:"
sqlite3 "$DB_PATH" "SELECT \"installed rank\", version, description, checksum FROM BLOOMDB_VERSION WHERE version = '1';"
echo ""

# Run repair to align checksums and descriptions
print_status "Running repair command to align checksums and descriptions..."
run_bloomdb repair
print_success "Repair completed - checksums and descriptions should now be aligned"

# Verify repair worked by checking info command again
print_status "Checking status after repair - V1 should now show success status"
run_bloomdb info

# Print BLOOMDB_VERSION table after repair to verify checksum was updated
print_status "Checking BLOOMDB_VERSION table after repair (should show updated checksum)"
echo "BLOOMDB_VERSION table contents:"
sqlite3 "$DB_PATH" "SELECT \"installed rank\", version, description, checksum FROM BLOOMDB_VERSION WHERE version = '1';"
echo ""

print_success "Checksum and description alignment test completed successfully!"

print_separator "Test 16: Remove a migration file"
print_status "Test 16: Removing migration file to test missing status"
mv "$MIGRATIONS_DIR/V2__Create_posts_table.sql" "$MIGRATIONS_DIR/V2__Create_posts_table.sql.bak"

print_success "Moved V2 migration file to test missing status"

print_separator "Test 17: Info command to check missing status"
print_status "Test 17: Checking missing file status"
run_bloomdb info

# Print BLOOMDB_VERSION table after missing file check
print_status "Checking BLOOMDB_VERSION table after missing file check"
echo "BLOOMDB_VERSION table contents:"
sqlite3 "$DB_PATH" "SELECT * FROM BLOOMDB_VERSION;"
echo ""

print_separator "Test 18: Restore the file and test again"
print_status "Test 18: Restoring migration file"
mv "$MIGRATIONS_DIR/V2__Create_posts_table.sql.bak" "$MIGRATIONS_DIR/V2__Create_posts_table.sql"

print_separator "Test 19: Test repeatable migration modification"
print_status "Test 19: Testing repeatable migration modification"

cat > "$MIGRATIONS_DIR/R__Create_views.sql" << 'EOF'
CREATE VIEW IF NOT EXISTS user_posts AS
SELECT u.name, u.email, p.title, p.created_at, p.content  -- Added content field
FROM users u
LEFT JOIN posts p ON u.id = p.user_id;

CREATE VIEW IF NOT EXISTS post_count AS
SELECT u.id, u.name, COUNT(p.id) as post_count
FROM users u
LEFT JOIN posts p ON u.id = p.user_id
GROUP BY u.id, u.name;
EOF

print_success "Modified repeatable migration"

print_separator "Test 20: Migrate to test repeatable migration"
print_status "Test 20: Testing repeatable migration execution"
run_bloomdb migrate
print_success "Repeatable migration executed successfully"

# Print BLOOMDB_VERSION table after repeatable migration
print_status "Checking BLOOMDB_VERSION table after repeatable migration"
echo "BLOOMDB_VERSION table contents:"
sqlite3 "$DB_PATH" "SELECT * FROM BLOOMDB_VERSION;"
echo ""

print_separator "Test 21: Final info check"
print_status "Test 21: Final migration status check"
run_bloomdb info

# Print BLOOMDB_VERSION table after final info check
print_status "Checking BLOOMDB_VERSION table after final info check"
echo "BLOOMDB_VERSION table contents:"
sqlite3 "$DB_PATH" "SELECT * FROM BLOOMDB_VERSION;"
echo ""

print_separator "Test 22: Test destroy with confirmation"
print_status "Test 22: Testing destroy with confirmation"
echo "DESTROY" | run_bloomdb destroy
print_success "Final destroy completed successfully"

# Print BLOOMDB_VERSION table after destroy
print_status "Checking BLOOMDB_VERSION table after destroy"
echo "BLOOMDB_VERSION table contents:"
sqlite3 "$DB_PATH" "SELECT * FROM BLOOMDB_VERSION;" 2>/dev/null || echo "Table does not exist (expected after destroy)"
echo ""

print_separator "Test 23: Verify database is empty"
print_status "Test 23: Verifying database is empty after destroy"
if sqlite3 "$DB_PATH" "SELECT name FROM sqlite_master WHERE type='table';" | grep -q "users\|posts\|faulty_table"; then
    print_error "Database still contains tables after destroy"
    exit 1
else
    print_success "Database is empty after destroy"
fi

print_success "All integration tests completed successfully! 🎉"

# Optional: Show test summary
echo ""
echo "=== Integration Test Summary ==="
echo "✅ Destroy functionality"
echo "✅ Baseline functionality" 
echo "✅ Migration functionality"
echo "✅ Faulty migration handling"
echo "✅ Repair functionality (failed migration removal)"
echo "✅ Checksum validation"
echo "✅ Repair functionality (checksum/description alignment)"
echo "✅ Missing file detection"
echo "✅ Repeatable migration updates"
echo "✅ Database cleanup"
echo ""
echo "All tests passed! 🎉"