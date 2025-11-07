# BloomDB CLI - Agent Guidelines

## Build/Lint/Test Commands
- `go build` - Build the application
- `go test ./...` - Run all tests
- `go test ./cmd/...` - Run cmd package tests
- `go test -run TestSpecificFunction ./cmd/` - Run single test
- `go mod tidy` - Clean up dependencies

## Code Style Guidelines

### Imports
- Group imports: stdlib, third-party, local packages
- Use absolute imports for local packages (e.g., "bloomdb/db")

### Naming Conventions
- PascalCase for exported types, functions, constants
- camelCase for unexported variables and functions
- Use descriptive names (e.g., `MigrateCommand`, `DatabaseType`)

### Error Handling
- Always handle errors immediately after function calls
- Use fmt.Printf for user-facing error messages
- Return wrapped errors with context using fmt.Errorf

### Types & Interfaces
- Define interfaces for database operations
- Use typed constants for database types
- Keep structs focused on single responsibility

### CLI Structure
- Use Cobra for CLI framework
- Separate command logic into structs with Run() methods
- Validate connection strings in PersistentPreRun
- Support environment variables for CLI flags (flag -> env -> default precedence)

### Database Pattern
- Extract database type from connection string prefix
- Use factory pattern for database creation
- Always defer database.Close() after successful connection