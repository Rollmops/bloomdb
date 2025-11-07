# BloomDB Development Guide

This guide contains comprehensive information for developers working on BloomDB, including setup, testing, debugging, and contribution guidelines.

## Prerequisites

- Go 1.25 or later
- Docker and Docker Compose (for containerized development)
- Git

## Development Setup

### Local Development

1. **Clone the repository:**
   ```bash
   git clone <repository-url>
   cd bloomdb
   ```

2. **Build the application:**
   ```bash
   go build
   ```

3. **Run tests to verify setup:**
   ```bash
   go test ./...
   ```

### Docker Development Environment

BloomDB includes a complete Docker development environment with PostgreSQL for testing and development.

> **💡 Quick Tip**: After changing Go code, rebuild with: `docker-compose build --no-cache bloom-app && docker-compose restart bloom-app`

#### Getting Started with Docker

1. **Start environment:**
   ```bash
   docker-compose up -d
   ```
   
   This starts:
   - PostgreSQL 15 database on port 5432
   - Bloom application container with migration files mounted

2. **Verify containers are running:**
   ```bash
   docker-compose ps
   ```

3. **Check database health:**
   ```bash
   docker exec bloom-postgres pg_isready -U bloom_user -d bloom_db
   ```

#### Container Configuration

##### Environment Variables

- `BLOOMDB_CONNECT_STRING`: PostgreSQL connection string (auto-set in docker-compose.yml)
  - Value: `postgres://bloom_user:bloom_password@postgres:5432/bloom_db?sslmode=disable`
  - Note: This is pre-configured, so you don't need to pass `--conn` parameter
- `BLOOMDB_TABLE_PREFIX`: Migration table name prefix (default: "BLOOM_")
- `BLOOMDB_TABLE_SUFFIX`: Migration table name suffix (default: "DEFAULT")
- `BLOOMDB_BASELINE_VERSION`: Baseline version (default: "1")

##### Database Connection

The PostgreSQL container is configured with:
- **Host**: postgres (internal Docker network)
- **Port**: 5432
- **Database**: bloom_db
- **User**: bloom_user
- **Password**: bloom_password

External connection (from host): `postgres://bloom_user:bloom_password@localhost:5432/bloom_db?sslmode=disable`

##### Volume Mounts

- `./migrations:/app/migrations`: Local migration files mounted in container
- `postgres_data`: PostgreSQL data persistence

#### Development Workflow

1. **Make changes to migration files** in local `migrations/` directory
2. **Test immediately** using Docker container
3. **Iterate quickly** without rebuilding containers
4. **Rebuild only when** changing Go code:

```bash
# Rebuild Bloom container with code changes
docker-compose build --no-cache bloom-app
docker-compose restart bloom-app
```

#### 🔄 Rebuilding Container After Code Changes

**When to rebuild:**
- Modified any Go source files (`*.go`)
- Added new commands or database drivers
- Changed dependencies in `go.mod`
- Modified Dockerfile

**When NOT to rebuild:**
- Only changed migration SQL files
- Modified configuration files
- Updated documentation

**Quick rebuild commands:**
```bash
# Fast rebuild (reuses cached layers)
docker-compose build bloom-app
docker-compose restart bloom-app

# Complete rebuild (no cache, for major changes)
docker-compose build --no-cache bloom-app
docker-compose restart bloom-app

# One-liner for development
docker-compose build --no-cache bloom-app && docker-compose restart bloom-app
```

**Verify rebuild worked:**
```bash
# Check container is running
docker-compose ps

# Test new functionality
docker exec bloom-app ./bloomdb --help
```

## Testing

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./db/...

# Integration tests
go test ./integration_test/

# Run single test
go test -run TestSpecificFunction ./cmd/

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...
```

### Test with Docker

```bash
# Start environment
docker-compose up -d

# Run integration tests against container
go test ./integration_test/ -conn "postgres://bloom_user:bloom_password@localhost:5432/bloom_db?sslmode=disable"
```

### Test Structure

- **Unit tests**: Located in each package alongside the source code
- **Integration tests**: Located in `integration_test/` package
- **Database tests**: Test each database driver (SQLite, PostgreSQL, Oracle)

### Writing Tests

Follow these guidelines when writing tests:

1. **Use table-driven tests** for multiple test cases
2. **Test both success and failure scenarios**
3. **Use subtests** with descriptive names
4. **Mock external dependencies** where appropriate
5. **Clean up resources** in test teardown

Example test structure:
```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    InputType
        expected ExpectedType
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    validInput,
            expected: expectedOutput,
            wantErr:  false,
        },
        {
            name:     "invalid input",
            input:    invalidInput,
            expected: ExpectedType{},
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Function() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !tt.wantErr && result != tt.expected {
                t.Errorf("Function() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

## Build and Development Commands

### Build Commands

```bash
# Build application
go build

# Build for different platforms
GOOS=linux GOARCH=amd64 go build -o bloomdb-linux
GOOS=windows GOARCH=amd64 go build -o bloomdb.exe
GOOS=darwin GOARCH=amd64 go build -o bloomdb-macos

# Build with debug information
go build -gcflags="all=-N -l"

# Build optimized binary
go build -ldflags="-s -w"
```

### Development Commands

```bash
# Format code
go fmt ./...

# Run linter (if golangci-lint is installed)
golangci-lint run

# Tidy dependencies
go mod tidy

# Download dependencies
go mod download

# Update dependencies
go get -u ./...
```

## Code Style and Guidelines

See [AGENTS.md](./AGENTS.md) for comprehensive coding guidelines used by AI agents and developers.

### Key Points:

- **Imports**: Group imports (stdlib, third-party, local packages)
- **Naming**: PascalCase for exported, camelCase for unexported
- **Error handling**: Always handle errors immediately
- **Interfaces**: Define interfaces for database operations
- **CLI structure**: Use Cobra framework with proper command separation

## Architecture

### Project Structure

```
bloomdb/
├── cmd/                    # CLI commands and Cobra integration
│   ├── root.go            # Root command and global flags
│   ├── migrate.go         # Migrate command implementation
│   ├── baseline.go        # Baseline command implementation
│   ├── info.go            # Info command implementation
│   ├── repair.go          # Repair command implementation
│   ├── destroy.go         # Destroy command implementation
│   └── common.go          # Shared database setup logic
├── db/                    # Database drivers and interfaces
│   ├── database.go        # Database interface and types
│   ├── factory.go         # Database factory for connection strings
│   ├── sqlite.go          # SQLite driver implementation
│   ├── postgresql.go      # PostgreSQL driver implementation
│   ├── oracle.go          # Oracle driver implementation
│   └── migration_table_test.go  # Database schema tests
├── loader/                # Migration file loading and parsing
│   ├── versioned_migrations_loader.go    # Versioned migration loader
│   ├── repeatable_migration_loader.go    # Repeatable migration loader
│   ├── hash.go            # Checksum calculation
│   └── parser.go          # Migration file parsing
├── logger/                # Logging utilities
│   └── logger.go          # Logger implementation
├── integration_test/      # End-to-end tests
├── migrations/            # Sample migration files
├── docker-compose.yml     # Docker development environment
├── Dockerfile            # Container build configuration
├── go.mod                # Go module definition
└── main.go               # Application entry point
```

### Key Components

1. **CLI Layer** (`cmd/`): Command-line interface using Cobra
2. **Database Layer** (`db/`): Database abstraction and drivers
3. **Migration Layer** (`loader/`): File loading and parsing
4. **Utility Layer** (`logger/`): Shared utilities

### Data Flow

1. **Command Execution**: CLI command → Database setup → Migration loading
2. **Migration Processing**: File parsing → Version comparison → Database execution
3. **Error Handling**: Immediate error propagation with user-friendly messages

## Debugging

### Local Debugging

```bash
# Build with debug symbols
go build -gcflags="all=-N -l"

# Use delve debugger
dlv debug ./bloomdb

# Or run with delve
dlv run -- migrate "connection_string"
```

### Docker Debugging

```bash
# Check container logs
docker-compose logs bloom-app
docker-compose logs postgres

# Enter container for debugging
docker-compose exec bloom-app sh
docker-compose exec postgres bash

# Check database connection
docker-compose exec bloom-app ./bloomdb info
```

### Common Issues and Solutions

#### Build Issues

```bash
# Clean build
go clean -cache
go build

# Rebuild without cache
docker-compose build --no-cache

# Check build logs
docker-compose build bloom-app
```

#### Database Connection Issues

```bash
# Check PostgreSQL health
docker exec bloom-postgres pg_isready -U bloom_user -d bloom_db

# Test connection manually
docker-compose exec postgres psql -U bloom_user -d bloom_db -c "SELECT 1;"

# Check network connectivity
docker network ls
docker network inspect bloomdb_default
```

#### Migration Issues

```bash
# Check migration table
docker exec bloom-postgres psql -U bloom_user -d bloom_db -c "SELECT * FROM bloom_default ORDER BY \"installed rank\";"

# Verify migration files
docker-compose exec bloom-app ls -la /app/migrations/

# Test specific migration
docker-compose exec bloom-app ./bloomdb migrate --path migrations
```

## Performance Considerations

### Migration Performance

- **Batch operations**: Group related SQL statements
- **Index creation**: Create indexes after data insertion
- **Transaction management**: Use appropriate transaction boundaries
- **Memory usage**: Process large migrations in chunks

### Application Performance

- **Connection pooling**: Reuse database connections
- **Concurrent processing**: Process independent migrations in parallel where safe
- **Caching**: Cache migration file contents and checksums

## Contributing

### Development Workflow

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/new-feature`
3. **Make changes** following the coding guidelines
4. **Add tests** for new functionality
5. **Run all tests**: `go test ./...`
6. **Update documentation** if needed
7. **Commit changes**: `git commit -m "Add new feature"`
8. **Push to fork**: `git push origin feature/new-feature`
9. **Create pull request**

### Pull Request Guidelines

1. **Clear description**: Explain what changes were made and why
2. **Test coverage**: Ensure new code is properly tested
3. **Documentation**: Update relevant documentation
4. **Backward compatibility**: Maintain API compatibility
5. **Code review**: Respond to review feedback promptly

### Code Review Process

1. **Automated checks**: Tests must pass, code must be formatted
2. **Manual review**: Architecture, logic, and style review
3. **Integration testing**: Verify changes work in Docker environment
4. **Approval**: At least one maintainer approval required

## Release Process

### Versioning

- **Semantic versioning**: Use MAJOR.MINOR.PATCH format
- **Changelog**: Maintain CHANGELOG.md with version notes
- **Tagging**: Create Git tags for releases

### Release Checklist

1. **Update version** in appropriate files
2. **Update CHANGELOG.md**
3. **Run full test suite**
4. **Build release binaries**
5. **Create Git tag**
6. **Create GitHub release**
7. **Update documentation**

## Troubleshooting

### Common Docker Issues

**Container won't start:**
```bash
# Check logs
docker-compose logs bloom-app
docker-compose logs postgres

# Clean rebuild
docker-compose down
docker system prune -f
docker-compose up -d
```

**Migration table not found:**
- Ensure `baseline` command was run first
- Check table name with correct case sensitivity
- Verify connection string is correct

**Permission issues:**
- Containers run as non-root user `bloom`
- Migration files should be readable by user 1001

### Database Connection Issues

**PostgreSQL connection refused:**
- Ensure PostgreSQL container is healthy: `docker-compose ps`
- Check network: `docker network ls`
- Verify connection string format

**Migration failures:**
- Check SQL syntax in migration files
- Verify database permissions
- Review error messages for specific issues

### Reset Environment

```bash
# Stop and remove containers
docker-compose down -v

# Remove all images
docker system prune -a

# Start fresh
docker-compose up -d
```

## Production Considerations

For production deployments:

1. **Security**: Change default passwords and use environment files
2. **Persistence**: Configure proper volume mounts for data
3. **Networking**: Set up proper network isolation
4. **Monitoring**: Add health checks and monitoring
5. **Performance**: Tune database and application settings
6. **Backups**: Implement regular backup strategies

## Additional Resources

- [Go Documentation](https://golang.org/doc/)
- [Cobra CLI Framework](https://github.com/spf13/cobra)
- [Docker Documentation](https://docs.docker.com/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [SQLite Documentation](https://sqlite.org/docs.html)
- [Oracle Documentation](https://docs.oracle.com/en/database/)