# BloomDB CLI - Database Migration Tool

BloomDB is a powerful database migration tool that supports multiple database backends including PostgreSQL, SQLite, and Oracle. It provides versioned and repeatable migrations with proper tracking and error handling.

## Features

- **Multi-database support**: PostgreSQL, SQLite, Oracle
- **Versioned migrations**: Ordered migration execution with version tracking
- **Repeatable migrations**: Re-applyable migrations for triggers, views, etc.
- **Error handling**: Failed migrations stop execution and aren't recorded
- **Automatic re-execution**: Repeatable migrations automatically re-run when content changes
- **Baseline support**: Initialize existing databases with migration tracking
- **Execution timing**: Track and display migration execution times
- **Version validation**: Ensures migration files follow proper version format

## Quick Start

### Prerequisites

- Go 1.25 or later
- Database access (PostgreSQL, SQLite, or Oracle)

### Installation

```bash
# Build from source
git clone <repository-url>
cd bloomdb
go build

# Or download pre-built binary (when available)
```

### Basic Usage

1. **Initialize migration tracking for existing database:**
    ```bash
    ./bloomdb baseline "your_connection_string"
    ```

2. **Apply pending migrations:**
    ```bash
    ./bloomdb migrate "your_connection_string" --path ./migrations
    ```

3. **Check migration status:**
    ```bash
    ./bloomdb info "your_connection_string"
    ```

### Connection Strings

**PostgreSQL:**
```
postgres://user:password@localhost:5432/database?sslmode=disable
```

**SQLite:**
```
sqlite:/path/to/database.db
```

**Oracle:**
```
oracle://user:password@localhost:1521/service_name
```

## Migration Files

### File Structure

Create a `migrations/` directory with your SQL files:

```
migrations/
├── V1__Create_users_table.sql
├── V2__Create_posts_table.sql
├── V3__Add_foreign_keys.sql
└── R__Create_updated_at_triggers.sql
```

### Versioned Migrations

- **Naming**: `V{version}__{description}.sql`
- **Format**: Version must be numeric (e.g., `1`, `1.2`, `1.2.3`)
- **Execution**: Run once in version order
- **Examples**:
  - `V1__Create_users_table.sql`
  - `V2.1__Add_email_column.sql`
  - `V3.0.5__Add_indexes.sql`

### Repeatable Migrations

- **Naming**: `R__{description}.sql`
- **Execution**: Re-run when content changes (different checksum)
- **Use cases**: Triggers, views, functions, stored procedures
- **Examples**:
  - `R__Create_user_triggers.sql`
  - `R__Update_product_views.sql`
  - `R__Refresh_materialized_views.sql`

## Commands

### baseline

Initialize migration tracking for existing databases:

```bash
./bloomdb baseline "your_connection_string" [flags]

Flags:
  --path string          Directory containing migration files (default: ".")
  --table-name string    Migration table name (default: "BLOOMDB_VERSION")
```

### migrate

Apply pending migrations:

```bash
./bloomdb migrate "your_connection_string" [flags]

Flags:
  --path string          Directory containing migration files (default: ".")
  --table-name string    Migration table name (default: "BLOOMDB_VERSION")
  --post-migration-script string   Path to post-migration SQL script (env: BLOOMDB_POST_MIGRATION_SCRIPT)
```

### info

Display migration status and database information:

```bash
./bloomdb info "your_connection_string" [flags]

Flags:
  --path string          Directory containing migration files (default: ".")
  --table-name string    Migration table name (default: "BLOOMDB_VERSION")
```

### repair

Repair migration records (for manual recovery):

```bash
./bloomdb repair [flags]

Flags:
  --table-name string    Migration table name (default: "BLOOMDB_VERSION")
```

### destroy

Remove all database objects (use with caution):

```bash
./bloomdb destroy [flags]

Flags:
  --table-name string    Migration table name (default: "BLOOMDB_VERSION")
```

## Environment Variables

You can use environment variables instead of command-line flags:

```bash
export BLOOMDB_CONNECT_STRING="postgres://user:pass@localhost/db"
export BLOOMDB_PATH="./migrations"
export BLOOMDB_VERSION_TABLE_NAME="my_migrations"
export BLOOMDB_BASELINE_VERSION="2"

./bloomdb migrate  # No need to specify connection string or path
```

## Migration Process

### What Happens During Migration

1. **Version Validation**: Checks that all versioned migrations have valid format
2. **Database Connection**: Connects to the specified database
3. **Table Check**: Ensures migration table exists
4. **Version Comparison**: Compares file versions with database records
5. **Execution**: Runs pending migrations in order
6. **Recording**: Stores migration records with execution time and status
7. **Error Handling**: Stops on first failure with detailed error message

### Migration Status Types

- **success**: Migration completed successfully
- **pending**: Migration not yet applied
- **failed**: Migration failed during execution
- **baseline**: Migration was baselined (skipped)
- **below baseline**: Migration version is below baseline version

## Examples

### Complete Workflow

```bash
# 1. Initialize existing database
./bloomdb baseline "postgres://user:pass@localhost/mydb"

# 2. Apply migrations
./bloomdb migrate "postgres://user:pass@localhost/mydb" --path ./migrations

# 3. Check status
./bloomdb info "postgres://user:pass@localhost/mydb" --path ./migrations
```

### Using Environment Variables

```bash
# Set up environment
export BLOOMDB_CONNECT_STRING="sqlite:///app/data.db"
export BLOOMDB_PATH="./database/migrations"

# Run commands without connection string
./bloomdb migrate
./bloomdb info
```

### Custom Migration Table

```bash
# Use custom table name
./bloomdb migrate --table-name "app_migrations"

# Or via environment
export BLOOMDB_VERSION_TABLE_NAME="app_migrations"
./bloomdb migrate
```

## Error Handling

### Migration Failures

When a migration fails:

1. **Execution stops**: No further migrations are executed
2. **Error recorded**: Failed migration is recorded with error details
3. **Detailed output**: Clear error message with step information
4. **Recovery possible**: Fix the issue and re-run to continue

### Common Issues

**Invalid version format:**
```
Error: invalid version format in file Vabc__migration.sql: abc (expected format: 1, 1.2, 1.2.3, etc.)
```

**Migration table not found:**
```
Error: Table 'BLOOMDB_VERSION' does not exist
Solution: Run baseline command first
```

**SQL syntax error:**
```
Error: Migration V1__Create_table.sql failed: syntax error at line 5
Solution: Fix SQL syntax and re-run
```

## Best Practices

### Migration Files

1. **Use descriptive names**: `V1__Create_users_table.sql` instead of `V1__table.sql`
2. **Keep migrations small**: One logical change per migration
3. **Test migrations**: Verify SQL syntax and logic
4. **Use transactions**: Wrap related statements in transactions
5. **Handle rollbacks**: Consider rollback scenarios

### Database Design

1. **Idempotent operations**: Design migrations to be re-runnable where possible
2. **Backward compatibility**: Consider impact on existing applications
3. **Performance**: Use appropriate indexes and batch operations
4. **Data integrity**: Include proper constraints and validations

### Version Management

1. **Semantic versioning**: Use consistent version numbering
2. **Sequential versions**: Don't skip version numbers
3. **Branch management**: Handle migration conflicts in feature branches

## Post-Migration Scripts

BloomDB supports post-migration SQL scripts that execute after all migrations complete successfully. These scripts support Go templating and have access to migration metadata including created and deleted database objects.

### Basic Usage

Post-migration scripts are automatically executed after successful migration completion. The system looks for these files in your migration directory:

1. `post_migration.sql`
2. `post_migration.sql.tmpl` 
3. `post_migration.template`

### Custom Script Path

You can specify a custom post-migration script path using CLI flag or environment variable:

```bash
# Using CLI flag
./bloomdb migrate --post-migration-script ./custom/post.sql

# Using environment variable
export BLOOMDB_POST_MIGRATION_SCRIPT="./custom/post.sql"
./bloomdb migrate

# Relative path (resolved from migration directory)
./bloomdb migrate --post-migration-script ../shared/post_migration.sql
```

### Template Variables

Post-migration scripts have access to the following variables in your Go templates:

| Variable | Type | Description |
|----------|------|-------------|
| `.CreatedObjects` | `[]DatabaseObject` | List of database objects created during migration |
| `.DeletedObjects` | `[]DatabaseObject` | List of database objects deleted during migration |
| `.MigrationPath` | `string` | Path to the migration directory |
| `.DatabaseType` | `string` | Database type (sqlite, postgresql, oracle) |
| `.TableName` | `string` | Name of the migration table |

Each `DatabaseObject` contains:
- `.Type` - Object type (table, view, index, etc.)
- `.Name` - Object name

### Example Post-Migration Script

Here's a comprehensive example that demonstrates all available features:

```sql
-- Post-migration script with Go templating (PostgreSQL compatible)
-- This script executes after all migrations are completed successfully

{{- if .CreatedObjects}}
-- Log all created objects
DO $$
BEGIN
    RAISE NOTICE 'Migration completed successfully!';
    RAISE NOTICE 'Created {{len .CreatedObjects}} database objects:';
    {{- range .CreatedObjects}}
    RAISE NOTICE '  - {{.Type}}: {{.Name}}';
    {{- end}}
END $$;

{{- if .DeletedObjects}}
-- Log all deleted objects
DO $$
BEGIN
    RAISE NOTICE 'Deleted {{len .DeletedObjects}} database objects:';
    {{- range .DeletedObjects}}
    RAISE NOTICE '  - {{.Type}}: {{.Name}}';
    {{- end}}
END $$;
{{- end}}

-- Create a summary table with migration information
CREATE TABLE IF NOT EXISTS migration_summary (
    id SERIAL PRIMARY KEY,
    migration_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    database_type VARCHAR(50) NOT NULL,
    total_objects INTEGER NOT NULL,
    notes TEXT
);

-- Insert summary record
INSERT INTO migration_summary (database_type, total_objects, notes)
VALUES (
    '{{.DatabaseType}}',
    {{len .CreatedObjects}},
    'Migration completed. Created objects: {{range .CreatedObjects}}{{.Type}}:{{.Name}} {{end}}{{if .DeletedObjects}}Deleted objects: {{range .DeletedObjects}}{{.Type}}:{{.Name}} {{end}}{{end}}'
);

-- Create a table to track created objects
CREATE TABLE IF NOT EXISTS created_objects_log (
    id SERIAL PRIMARY KEY,
    migration_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    object_type VARCHAR(50) NOT NULL,
    object_name VARCHAR(255) NOT NULL
);

-- Log each created object
{{- range .CreatedObjects}}
INSERT INTO created_objects_log (object_type, object_name)
VALUES ('{{.Type}}', '{{.Name}}');
{{- end}}

{{- else if .DeletedObjects}}
-- Only objects were deleted during migration
DO $$
BEGIN
    RAISE NOTICE 'Migration completed but objects were deleted';
    RAISE NOTICE 'Deleted {{len .DeletedObjects}} database objects:';
    {{- range .DeletedObjects}}
    RAISE NOTICE '  - {{.Type}}: {{.Name}}';
    {{- end}}
END $$;

INSERT INTO migration_summary (database_type, total_objects, notes)
VALUES (
    '{{.DatabaseType}}',
    0,
    'Migration completed. Deleted objects: {{range .DeletedObjects}}{{.Type}}:{{.Name}} {{end}}'
);

{{- else}}
-- No objects were created or deleted during migration
DO $$
BEGIN
    RAISE NOTICE 'Migration completed but no database objects were changed';
END $$;

INSERT INTO migration_summary (database_type, total_objects, notes)
VALUES (
    '{{.DatabaseType}}',
    0,
    'Migration completed but no database objects were changed'
);
{{- end}}

-- Example: Create documentation for created tables
{{- range .CreatedObjects}}
    {{- if eq .Type "table"}}
-- Documentation for table: {{.Name}}
COMMENT ON TABLE {{.Name}} IS 'Created by BloomDB migration process';
    {{- end}}
{{- end}}
```

### Template Syntax Examples

**Conditional logic:**
```sql
{{- if .CreatedObjects}}
-- Objects were created
SELECT '{{len .CreatedObjects}} objects created';
{{- end}}

{{- if and .CreatedObjects .DeletedObjects}}
-- Both creation and deletion occurred
SELECT 'Schema changes detected';
{{- end}}
```

**Looping through objects:**
```sql
{{- range .CreatedObjects}}
-- Process each created object
{{- if eq .Type "table"}}
-- Handle table creation: {{.Name}}
{{- else if eq .Type "index"}}
-- Handle index creation: {{.Name}}
{{- end}}
{{- end}}
```

**Database-specific logic:**
```sql
{{- if eq .DatabaseType "postgresql"}}
-- PostgreSQL-specific code
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
{{- else if eq .DatabaseType "sqlite"}}
-- SQLite-specific code
PRAGMA foreign_keys = ON;
{{- end}}
```

### Use Cases

Post-migration scripts are perfect for:

1. **Documentation**: Auto-generate documentation for created objects
2. **Notifications**: Send alerts about schema changes
3. **Audit logging**: Record all changes in audit tables
4. **Data seeding**: Populate reference data after schema creation
5. **Performance optimization**: Create indexes after data loading
6. **Integration**: Trigger external processes or API calls
7. **Reporting**: Generate migration summary reports

### Best Practices

1. **Keep scripts idempotent**: Design scripts to run multiple times safely
2. **Use conditional logic**: Handle cases where no objects are created/deleted
3. **Database compatibility**: Use `{{.DatabaseType}}` for database-specific code
4. **Error handling**: Wrap operations in proper error handling
5. **Testing**: Test templates with different migration scenarios
6. **Documentation**: Document what your post-migration scripts do

### Error Handling

If a post-migration script fails:
- The error is logged but doesn't rollback successful migrations
- A warning message is displayed
- The migration process is still considered successful
- Fix the script and re-run to execute it properly

## Advanced Usage

### Docker Development

For development with Docker, see [DEVELOPMENT.md](./DEVELOPMENT.md) for complete setup instructions.

### Multiple Environments

```bash
# Development
./bloomdb migrate "$DEV_DB" --path ./migrations/dev

# Staging
./bloomdb migrate "$STAGING_DB" --path ./migrations/staging

# Production
./bloomdb migrate "$PROD_DB" --path ./migrations/prod
```

### Custom Configuration

```bash
# Custom baseline version for existing database
export BLOOMDB_BASELINE_VERSION="5.2"
./bloomdb baseline

# Custom migration directory
./bloomdb migrate --path ./sql/migrations

# Custom table name
./bloomdb migrate --table-name "schema_migrations"
```

## Troubleshooting

### Getting Help

```bash
# Show command help
./bloomdb --help
./bloomdb migrate --help

# Enable debug logging
./bloomdb migrate --log-level debug
```

### Common Problems

**Connection issues:**
- Verify connection string format
- Check database server is running
- Ensure network connectivity
- Validate credentials and permissions

**Migration issues:**
- Check SQL syntax in migration files
- Verify file naming conventions
- Ensure proper file permissions
- Review error messages for specific issues

**Performance issues:**
- Use appropriate indexes
- Batch large data operations
- Consider database-specific optimizations
- Monitor execution times

## Development

For development setup, testing, and contribution guidelines, see [DEVELOPMENT.md](./DEVELOPMENT.md).

## License

[Add your license information here]