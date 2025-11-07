#!/bin/bash

# BloomDB PostgreSQL Integration Test Runner
# This script starts a PostgreSQL container and runs the integration tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[SETUP]${NC} $1"
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

# Function to cleanup Docker container
cleanup() {
    print_status "Cleaning up Docker container..."
    if docker ps -q -f name=bloomdb-postgres-test | grep -q .; then
        docker stop bloomdb-postgres-test
    fi
    docker rm -f bloomdb-postgres-test 2>/dev/null || true
    docker-compose -f docker-compose.test.yml down -v 2>/dev/null || true
    print_success "Cleanup completed"
}

# Set trap for cleanup on script exit
trap cleanup EXIT

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker and try again."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    print_error "docker-compose is not available. Please install docker-compose."
    exit 1
fi

# Use docker-compose or docker compose based on availability
COMPOSE_CMD="docker-compose"
if ! command -v docker-compose &> /dev/null; then
    COMPOSE_CMD="docker compose"
fi

print_status "Starting PostgreSQL container..."
$COMPOSE_CMD -f docker-compose.test.yml up -d

print_status "Waiting for PostgreSQL to be ready..."
# Wait for the container to be healthy
for i in {1..30}; do
    if $COMPOSE_CMD -f docker-compose.test.yml ps | grep -q "healthy"; then
        print_success "PostgreSQL is ready!"
        break
    fi
    if [ $i -eq 30 ]; then
        print_error "PostgreSQL failed to start within 30 seconds"
        $COMPOSE_CMD -f docker-compose.test.yml logs
        exit 1
    fi
    echo "Waiting for PostgreSQL... ($i/30)"
    sleep 1
done

# Get the baseline version from command line argument or use default
BASELINE_VERSION="${1:-0.5}"

print_status "Running PostgreSQL integration tests with baseline version: $BASELINE_VERSION"
echo ""

# Make the test script executable
chmod +x integration-test-postgresql.sh

# Run the integration test
if ./integration-test-postgresql.sh "$BASELINE_VERSION"; then
    print_success "All PostgreSQL integration tests passed! 🎉"
else
    print_error "PostgreSQL integration tests failed!"
    exit 1
fi