# Migration File Filtering

- BloomDB supports filtering migrations by database type using optional filter suffixes in filenames
- Filter modes are controlled by environment variables:
  - `BLOOMDB_FILTER_HARD=<filter>`: Only load migrations with the specified filter (strict mode)
  - `BLOOMDB_FILTER_SOFT=<filter>`: Prefer filtered migrations, but fall back to non-filtered if none exist (compatibility mode)
  - No filter variables set: Only load migrations without filter suffixes (default behavior)
- Filter priority: BLOOMDB_FILTER_HARD takes precedence over BLOOMDB_FILTER_SOFT if both are set
- Migration file naming patterns:
  - Versioned migrations: `V<version>__<description>[.<filter>].sql`
    - Example without filter: `V1__create_users.sql`
    - Example with filter: `V1__create_users.postgres.sql`, `V1__create_users.oracle.sql`
  - Repeatable migrations: `R__<description>[.<filter>].sql`
    - Example without filter: `R__rebuild_views.sql`
    - Example with filter: `R__rebuild_views.mysql.sql`
- Filter behavior:
  - **No filter mode** (default): Files without filters are loaded; files with any filter are ignored
  - **Hard filter mode**: Only files matching the specified filter are loaded; non-filtered and other-filtered files are ignored
  - **Soft filter mode**: Files matching the filter are preferred; if no filtered version exists for a migration, the non-filtered version is used as fallback; other-filtered files are ignored
- Version identification for filtered migrations:
  - **Versioned migrations**: Identified by version number only (not description)
    - `V1.0__generic.sql` and `V1.0__postgres_specific.postgres.sql` are treated as the same version (1.0)
    - Different descriptions are allowed for different database-specific implementations of the same version
    - In the database, only the version number matters for identifying versioned migrations
  - **Repeatable migrations**: Identified by description
    - `R__views.sql` and `R__views_postgres.postgres.sql` are treated as different migrations
    - Must have the same description to be considered the same repeatable migration
- Filter format: Any alphanumeric identifier (e.g., `postgres`, `oracle`, `mysql`, `dev`, `prod`)
- Files that look like migrations but have invalid format (e.g., `Vabc__invalid.sql`) return errors
- Non-migration files in the migrations directory are silently ignored

# Subdirectory Migration Support

- BloomDB supports organizing migrations into subdirectories for multi-tenant or multi-schema scenarios
- Subdirectory detection algorithm:
  1. Check if the migration path contains any versioned or repeatable migrations at the root level
  2. If YES: Process only the root directory (ignore subdirectories)
  3. If NO: Iterate through subdirectories (depth 1 only) and process each as a separate migration directory
- Each subdirectory is treated as an independent migration directory with its own version table
- Version table naming for subdirectories:
  - Pattern: `BLOOMDB_<DIRNAME>` where DIRNAME is the subdirectory name
  - Directory name is uppercased
  - Hyphens (-) are replaced with underscores (_)
  - Examples:
    - `migrations/tenant-a/` → version table `BLOOMDB_TENANT_A`
    - `migrations/tenant_b/` → version table `BLOOMDB_TENANT_B`
    - `migrations/schema1/` → version table `BLOOMDB_SCHEMA1`
- All commands (baseline, migrate, info, repair) support subdirectory processing
- Subdirectories without any migration files are ignored
- When processing subdirectories, each directory is processed independently with its own:
  - Version table
  - Baseline record
  - Migration history
  - Checksum validation

# Version Table
 
- the tool uses a version table that is specified by the env variable BLOOMDB_VERSION_TABLE_NAME and defaults to BLOOMDB_VERSION
- for subdirectory migrations, the version table name is automatically derived from the subdirectory name (see Subdirectory Migration Support above)
- the table is created only when running the baseline command
- if the table is already there, skip creation
- when running the baseline command, baseline record will be created in the version table:

INSERT INTO <BLOOMDB_VERSION_TABLE_NAME>
("installed rank", "version", description, "type", script, checksum, "installed by", "installed on", "execution time", success)
VALUES(<version>, '<version>', '<< Baseline >>', 'BASELINE', '<< Baseline >>', NULL, '<user>', '<datetime>', 0, 1);

- if the entry already exists, print to the user that the baseline is already created
  - the success message shows the existing baseline version from the database
  - if the requested version differs from the existing baseline, the same success message is shown (no error)
  - no new baseline record is created when one already exists, regardless of version mismatch
- baseline version priority (in order of precedence):
  1. **Existing database baseline** (highest priority): If a baseline record exists in the database, always use that version regardless of other settings. This prevents accidental baseline version changes.
  2. **CLI flag** (`--baseline-version`): Explicit version passed on command line
  3. **Environment variable** (`BLOOMDB_BASELINE_VERSION`): Version set in environment
  4. **Default value**: Version "1" if nothing else is specified
- when resolving the baseline version, info messages are printed to indicate which source was used (only visible in verbose mode)
- all other commands than "baseline" should check the existing of the version table and the baseline record
  - if either the table is not there or the record is missing, print an error to the user, referring to the baseline command

# Output Formatting

- the tool supports three output formats: human-readable (default), test format, and JSON (deprecated)
- the output format is controlled by the BLOOMDB_PRINTER environment variable:
  - `BLOOMDB_PRINTER=human` (default): colorful terminal output with icons and tables
  - `BLOOMDB_PRINTER=test`: simple line-based format for integration testing
  - `BLOOMDB_PRINTER=json` (deprecated): replaced by test format
- all printers respect the BLOOMDB_VERBOSE environment variable for info-level messages
- Test output format (for integration testing):
  ```
  SUCCESS: message text
  WARNING: message text
  ERROR: message text
  INFO: message text
  ```
  - one line per output message
  - format: `LEVEL: message`
  - easy to parse with simple string operations
- migration tables are output with simplified format when using test printer
- all Print* functions automatically delegate to the configured printer

# Repair command
 
1. check if the baseline command was executed
2. remove the version table record with success = 0

# Migration Checksums

- BloomDB uses Flyway-compatible CRC32 checksums to detect modifications to applied migrations
- checksum calculation follows Flyway's algorithm:
  - uses CRC32 (IEEE polynomial)
  - normalizes line endings (\n, \r\n, \r all treated the same)
  - strips UTF-8 BOM from the first line if present
  - returns a signed 32-bit integer (-2,147,483,648 to 2,147,483,647)
- checksums are calculated when migrations are applied and stored in the version table
- checksums are validated by the info command:
  - if a migration file's checksum doesn't match the stored value, status shows "⚠ checksum"
  - checksum mismatches indicate the migration file has been modified after being applied
  - this helps detect accidental or unauthorized changes to applied migrations
- checksums are validated by the migrate command before applying new migrations:
  - if any applied migration's checksum doesn't match the current file, migration is blocked
  - displays detailed error messages showing which migrations have mismatches
  - shows expected vs found checksum values for troubleshooting
  - advises user to run repair command or restore original files
- baseline migrations have NULL checksum (no validation)

# Migrate command
 
- if the version table contains a record with success = 0 then do not continue migration but rather refer to the repair command
- validates checksums of all applied migrations before proceeding:
  - compares stored checksums with current file checksums
  - if any checksum mismatch is detected, migration is blocked
  - displays error messages indicating which migrations have been modified
  - user must run repair command to update checksums or restore original files
