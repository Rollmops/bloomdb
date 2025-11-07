# BloomDB CLI - Agent Guidelines

## Documentation Requirements
- Always write requirements/specifications in `specification.md`
- Always update `README.adoc` for end user documentation
- Remove outdated content from `README.adoc` when features change
- Document all new commands, flags, and environment variables

## Build/Lint/Test Commands
- **Build**: `go build` - Compile the application
- **Test All**: `go test ./...` - Run all tests in all packages
- **Test Package**: `go test ./cmd/...` - Run tests in specific package (cmd, db, loader, printer)
- **Test Single**: `go test -run TestSpecificFunction ./path/` - Run a specific test function
- **Verbose Tests**: `go test -v ./...` - Run tests with verbose output showing each test
- **Integration Tests**: See `integration-tests/` directory for SQLite, PostgreSQL, and Oracle tests
- **Dependencies**: `go mod tidy` - Clean up go.mod and go.sum
- **Format**: `gofmt -s -w .` - Format code with simplifications and write changes
- **Testify Required**: Always use github.com/stretchr/testify for assertions (assert, require)

## Environment Variables
- **BLOOMDB_FILTER_HARD**: Filter migrations by database type (e.g., `postgres`, `oracle`, `mysql`); only loads migrations matching the filter
- **BLOOMDB_FILTER_SOFT**: Same as HARD but falls back to non-filtered migrations if no filtered versions exist
- **Filter Priority**: BLOOMDB_FILTER_HARD takes precedence over BLOOMDB_FILTER_SOFT if both are set
- **Migration Naming**: Versioned `V<version>__<description>[.<filter>].sql`, Repeatable `R__<description>[.<filter>].sql`
- **Version Grouping**: Versioned migrations group by version number ONLY (not description); `V1.0__users.sql` and `V1.0__postgres.postgres.sql` = same version; repeatable migrations group by description

## Code Style Guidelines
- **Imports**: Group in order: stdlib → third-party → local (bloomdb/); use absolute imports (`"bloomdb/db"`)
- **Naming**: PascalCase for exported (MigrateCommand, TableExists), camelCase for unexported (dbConnStr, maxRank)
- **Error Handling**: Check errors immediately; wrap with `fmt.Errorf("context: %w", err)`; use Print* functions for user output
- **Types**: Interfaces for DB operations; typed constants for enums (DatabaseType); pointer fields for nullable values (*string, *int64)
- **CLI Pattern**: Cobra commands with Run() methods; flag → env → default precedence; validate in PersistentPreRun
- **Database Pattern**: Parse type from connection string prefix (postgres://, sqlite:, oracle://); factory pattern (NewDatabase); defer Close()
- **Testing**: testify/assert for simple checks, require for critical setup; table-driven tests with subtests using t.Run()
- **Comments**: Document exported functions/types; explain non-obvious logic; use TODO for future improvements
- **Variables**: Declare close to usage; use short names in small scopes (err, db); descriptive names for larger scopes (migrationPath)
- **JSON Tags**: Use snake_case for JSON field names (`json:"installed_rank"`)
