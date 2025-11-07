#!/bin/bash

# BloomDB Oracle Integration Test Runner
# This script starts an Oracle container and runs the integration tests

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
    if docker ps -q -f name=bloomdb-oracle-test | grep -q .; then
        docker stop bloomdb-oracle-test
    fi
    docker rm -f bloomdb-oracle-test 2>/dev/null || true
    docker-compose -f docker-compose.oracle.yml down -v 2>/dev/null || true
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

print_warning "Oracle setup can take 5-10 minutes to complete. Please be patient."
print_status "Starting Oracle container..."
$COMPOSE_CMD -f docker-compose.oracle.yml up -d

print_status "Waiting for Oracle to be ready (this can take several minutes)..."
# Wait for the container to be healthy - Oracle takes much longer to start
for i in {1..60}; do
    if $COMPOSE_CMD -f docker-compose.oracle.yml ps | grep -q "healthy"; then
        print_success "Oracle is ready!"
        break
    fi
    if [ $i -eq 60 ]; then
        print_error "Oracle failed to start within 10 minutes"
        $COMPOSE_CMD -f docker-compose.oracle.yml logs
        exit 1
    fi
    echo "Waiting for Oracle... ($i/60) - this is normal for Oracle startup"
    sleep 10
done

# Additional wait to ensure the bloomdb user is created
print_status "Waiting for bloomdb user setup to complete..."
sleep 30

# Test the bloomdb user connection
for i in {1..12}; do
    if docker exec bloomdb-oracle-test bash -c "echo 'SELECT 1 FROM DUAL;' | sqlplus -s bloomdb/bloomdb@XEPDB1" | grep -q "1"; then
        print_success "BloomDB user is ready!"
        break
    fi
    if [ $i -eq 12 ]; then
        print_error "BloomDB user setup failed"
        docker exec bloomdb-oracle-test bash -c "echo 'SELECT username FROM all_users;' | sqlplus -s sys/Oracle123456@XEPDB1 as sysdba"
        exit 1
    fi
    echo "Waiting for bloomdb user... ($i/12)"
    sleep 5
done

# Get the baseline version from command line argument or use default
BASELINE_VERSION="${1:-0.5}"

print_status "Running Oracle integration tests with baseline version: $BASELINE_VERSION"
echo ""

# Make the test script executable
chmod +x integration-test-oracle.sh

# Run the integration test
if ./integration-test-oracle.sh "$BASELINE_VERSION"; then
    print_success "All Oracle integration tests passed! 🎉"
else
    print_error "Oracle integration tests failed!"
    exit 1
fi