# Integration Tests

This directory contains integration tests for BloomDB across different database systems.

## Available Tests

### SQLite Integration Tests
- **Script**: `integration-test-sqlite.sh`
- **Database**: SQLite (in-memory file)
- **Usage**: `./integration-test-sqlite.sh [BASELINE_VERSION]`
- **Startup**: Immediate (no container needed)

### PostgreSQL Integration Tests  
- **Script**: `run-postgresql-tests.sh` (wrapper) + `integration-test-postgresql.sh`
- **Database**: PostgreSQL 15 in Docker container
- **Usage**: `./run-postgresql-tests.sh [BASELINE_VERSION]`
- **Startup**: ~10 seconds

### Oracle Integration Tests
- **Script**: `run-oracle-tests.sh` (wrapper) + `integration-test-oracle.sh`
- **Database**: Oracle Database Free in Docker container
- **Usage**: `./run-oracle-tests.sh [BASELINE_VERSION]`
- **Startup**: 5-10 minutes (first time)

## Quick Start

```bash
# SQLite tests (fastest)
./integration-test-sqlite.sh

# PostgreSQL tests (medium speed)
./run-postgresql-tests.sh

# Oracle tests (slowest, but most comprehensive)
./run-oracle-tests.sh
```

## Test Coverage

All integration tests cover the same functionality:

✅ **Baseline functionality** - Tests below-baseline migration detection  
✅ **Migration execution** - Verifies successful migration runs  
✅ **Failed migration handling** - Tests error detection and recovery  
✅ **Repair functionality** - Validates failed migration cleanup  
✅ **Checksum validation** - Detects modified migration files  
✅ **Missing file detection** - Handles deleted migration files  
✅ **Repeatable migrations** - Tests repeatable migration updates  
✅ **Database cleanup** - Verifies destroy functionality  

## Baseline Version Parameter

All test scripts accept an optional baseline version parameter:

```bash
# Uses default baseline version (0.5)
./integration-test-sqlite.sh

# Uses custom baseline version (1.2)
./integration-test-sqlite.sh 1.2
```

The baseline version controls which migrations are considered "below baseline" and should be skipped during execution.

## Database-Specific Notes

### SQLite
- Uses file-based database in temporary directory
- No external dependencies
- Fastest for quick testing

### PostgreSQL  
- Requires Docker
- Uses standard PostgreSQL syntax
- Good balance of speed and features

### Oracle
- Requires Docker with 4GB+ RAM
- Uses Oracle-specific SQL syntax
- Most comprehensive enterprise testing
- See `README-Oracle.md` for detailed setup

## File Structure

```
integration-tests/
├── README.md                    # This file
├── integration-test-sqlite.sh   # SQLite test script
├── integration-test-postgresql.sh # PostgreSQL test script  
├── integration-test-oracle.sh     # Oracle test script
├── run-postgresql-tests.sh       # PostgreSQL wrapper script
├── run-oracle-tests.sh          # Oracle wrapper script
├── docker-compose.test.yml       # PostgreSQL Docker config
├── docker-compose.oracle.yml      # Oracle Docker config
├── oracle-setup/                # Oracle user setup scripts
│   └── 01-create-bloomdb-user.sh
└── README-Oracle.md            # Oracle-specific documentation
```

## Troubleshooting

### Docker Issues
```bash
# Check if Docker is running
docker info

# Check container status
docker-compose -f docker-compose.test.yml ps
docker-compose -f docker-compose.oracle.yml ps

# View logs
docker-compose -f docker-compose.test.yml logs
docker-compose -f docker-compose.oracle.yml logs
```

### Permission Issues
```bash
# Make scripts executable
chmod +x *.sh
chmod +x oracle-setup/*.sh
```

### Cleanup
```bash
# Stop and remove containers
docker-compose -f docker-compose.test.yml down -v
docker-compose -f docker-compose.oracle.yml down -v
```

## Contributing

When adding new database support:

1. Create `integration-test-{database}.sh` following the existing pattern
2. Create `docker-compose.{database}..yml` if containerized
3. Create `run-{database}-tests.sh` wrapper script if needed
4. Update this README with database-specific notes
5. Ensure all 23 test scenarios are covered

The test scripts are designed to be consistent across all databases while respecting each database's specific SQL syntax and connection requirements.