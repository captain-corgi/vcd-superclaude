#!/bin/bash

# Test Runner Script for Multi-Agent System
# Comprehensive testing with different test suites

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TEST_PATTERN=${1:-"Test"}
COVERAGE=${2:-false}
PARALLEL=${3:-true}
VERBOSE=${4:-false}
FAIL_FAST=${5:-false}

# Test directories
TEST_DIRS=(
    "pkg/config"
    "pkg/storage"
    "pkg/logger"
    "pkg/models"
    "internal/agents"
    "internal/communication"
    "internal/tools"
    "internal/web"
    "cmd/server"
)

# Logging function
log() {
    local level=$1
    local message=$2
    case $level in
        "INFO") echo -e "${BLUE}[INFO] $message${NC}" ;;
        "SUCCESS") echo -e "${GREEN}[SUCCESS] $message${NC}" ;;
        "WARNING") echo -e "${YELLOW}[WARNING] $message${NC}" ;;
        "ERROR") echo -e "${RED}[ERROR] $message${NC}" ;;
    esac
}

# Check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        log "ERROR" "Go is not installed"
        exit 1
    fi
    log "SUCCESS" "Go is installed: $(go version)"
}

# Check if testing tools are available
check_test_tools() {
    log "INFO" "Checking testing tools..."

    # Check ginkgo
    if command -v ginkgo &> /dev/null; then
        log "SUCCESS" "Ginkgo is available"
    else
        log "WARNING" "Ginkgo not found. Installing..."
        go install github.com/onsi/ginkgo/v2/ginkgo@latest
    fi

    # Check gomega
    if command -v gom &> /dev/null; then
        log "SUCCESS" "Gomega is available"
    else
        log "INFO" "Gomega not required for standard Go tests"
    fi
}

# Run Go tests
run_go_tests() {
    local test_type=$1
    local test_args=$2

    log "INFO" "Running Go $test_type tests..."

    local cmd="go test"

    # Add common test flags
    cmd="$cmd -v"

    if [ "$VERBOSE" = true ]; then
        cmd="$cmd -v"
    else
        cmd="$cmd -v"
    fi

    if [ "$FAIL_FAST" = true ]; then
        cmd="$cmd -failfast"
    fi

    # Add coverage if requested
    if [ "$COVERAGE" = true ]; then
        cmd="$cmd -cover"
        cmd="$cmd -coverprofile=coverage.out"
        cmd="$cmd -covermode=atomic"
    fi

    # Add parallel execution
    if [ "$PARALLEL" = true ]; then
        cmd="$cmd -parallel 4"
    fi

    # Add test pattern
    cmd="$cmd -run $TEST_PATTERN"

    # Add test directories
    for dir in "${TEST_DIRS[@]}"; do
        if [ -d "$dir" ]; then
            cmd="$cmd ./$dir"
        fi
    done

    # Execute tests
    log "INFO" "Running: $cmd"
    eval $cmd

    if [ $? -eq 0 ]; then
        log "SUCCESS" "Go $test_type tests passed"
    else
        log "ERROR" "Go $test_type tests failed"
        return 1
    fi
}

# Run integration tests
run_integration_tests() {
    log "INFO" "Running integration tests..."

    # Set up test database
    log "INFO" "Setting up test database..."

    # Check if Docker is available
    if command -v docker &> /dev/null; then
        log "INFO" "Using test containers..."

        # Start test containers
        docker-compose -f docker/docker-compose.test.yml up -d

        # Wait for services to be ready
        sleep 10

        # Run tests
        run_go_tests "integration" "-run TestIntegration"

        # Stop test containers
        docker-compose -f docker/docker-compose.test.yml down
    else
        log "WARNING" "Docker not available, running integration tests against local services"
        run_go_tests "integration" "-run TestIntegration"
    fi
}

# Run benchmark tests
run_benchmark_tests() {
    log "INFO" "Running benchmark tests..."

    local cmd="go test -bench=."

    if [ "$VERBOSE" = true ]; then
        cmd="$cmd -v"
    fi

    # Add benchmark directories
    for dir in "${TEST_DIRS[@]}"; do
        if [ -d "$dir" ]; then
            cmd="$cmd ./$dir"
        fi
    done

    log "INFO" "Running: $cmd"
    eval $cmd

    if [ $? -eq 0 ]; then
        log "SUCCESS" "Benchmark tests completed"
    else
        log "ERROR" "Benchmark tests failed"
        return 1
    fi
}

# Run stress tests
run_stress_tests() {
    log "INFO" "Running stress tests..."

    # Run tests with high concurrency
    local cmd="go test -parallel=10 -timeout=5m"

    if [ "$COVERAGE" = true ]; then
        cmd="$cmd -cover"
    fi

    # Add stress test pattern
    cmd="$cmd -run TestStress"

    # Add test directories
    for dir in "${TEST_DIRS[@]}"; do
        if [ -d "$dir" ]; then
            cmd="$cmd ./$dir"
        fi
    done

    log "INFO" "Running: $cmd"
    eval $cmd

    if [ $? -eq 0 ]; then
        log "SUCCESS" "Stress tests completed"
    else
        log "ERROR" "Stress tests failed"
        return 1
    fi
}

# Generate test report
generate_test_report() {
    log "INFO" "Generating test report..."

    if [ "$COVERAGE" = true ] && [ -f "coverage.out" ]; then
        # Generate HTML coverage report
        go tool cover -html=coverage.out -o coverage.html
        log "SUCCESS" "HTML coverage report generated: coverage.html"

        # Generate coverage summary
        go tool cover -func=coverage.out | tail -1
        log "SUCCESS" "Coverage summary generated"
    fi
}

# Run linter
run_linter() {
    log "INFO" "Running linter..."

    if command -v golangci-lint &> /dev/null; then
        golangci-lint run
        log "SUCCESS" "Linter completed"
    else
        log "WARNING" "golangci-lint not available, skipping linting"
    fi
}

# Run security scan
run_security_scan() {
    log "INFO" "Running security scan..."

    if command -v gosec &> /dev/null; then
        gosec ./...
        log "SUCCESS" "Security scan completed"
    else
        log "WARNING" "gosec not available, skipping security scan"
    fi
}

# Test dependencies
test_dependencies() {
    log "INFO" "Testing dependencies..."

    # Test database connectivity
    if command -v psql &> /dev/null; then
        log "INFO" "Testing PostgreSQL connectivity..."
        if psql -h localhost -U postgres -d postgres -c "SELECT 1;" &> /dev/null; then
            log "SUCCESS" "PostgreSQL is accessible"
        else
            log "WARNING" "PostgreSQL is not accessible"
        fi
    else
        log "WARNING" "PostgreSQL client not available"
    fi

    # Test Redis connectivity
    if command -v redis-cli &> /dev/null; then
        log "INFO" "Testing Redis connectivity..."
        if redis-cli ping &> /dev/null; then
            log "SUCCESS" "Redis is accessible"
        else
            log "WARNING" "Redis is not accessible"
        fi
    else
        log "WARNING" "Redis CLI not available"
    fi
}

# Clean up test artifacts
cleanup_test_artifacts() {
    log "INFO" "Cleaning up test artifacts..."

    # Remove coverage files
    rm -f coverage.out coverage.html

    # Remove test binaries
    find . -name "test.*" -type f -delete

    log "SUCCESS" "Test artifacts cleaned up"
}

# Show test results summary
show_test_summary() {
    log "INFO" "Test Summary"
    log "INFO" "================"
    log "INFO" "Test Pattern: $TEST_PATTERN"
    log "INFO" "Coverage: $COVERAGE"
    log "INFO" "Parallel: $PARALLEL"
    log "INFO" "Verbose: $VERBOSE"
    log "INFO" "Fail Fast: $FAIL_FAST"
    log "INFO" "================"
}

# Main test function
main() {
    log "INFO" "Starting test suite for Multi-Agent System"
    log "INFO" "Timestamp: $(date)"
    echo ""

    # Check prerequisites
    check_go
    check_test_tools

    # Show test configuration
    show_test_summary

    # Test dependencies
    test_dependencies

    # Run tests based on configuration
    local tests_passed=0
    local total_tests=0

    # Unit tests
    total_tests=$((total_tests + 1))
    if run_go_tests "unit"; then
        tests_passed=$((tests_passed + 1))
    fi

    # Integration tests
    total_tests=$((total_tests + 1))
    if run_integration_tests; then
        tests_passed=$((tests_passed + 1))
    fi

    # Benchmark tests
    total_tests=$((total_tests + 1))
    if run_benchmark_tests; then
        tests_passed=$((tests_passed + 1))
    fi

    # Stress tests
    total_tests=$((total_tests + 1))
    if run_stress_tests; then
        tests_passed=$((tests_passed + 1))
    fi

    # Quality checks
    total_tests=$((total_tests + 1))
    if run_linter; then
        tests_passed=$((tests_passed + 1))
    fi

    total_tests=$((total_tests + 1))
    if run_security_scan; then
        tests_passed=$((tests_passed + 1))
    fi

    # Generate report
    generate_test_report

    echo ""
    log "INFO" "Test Results Summary"
    log "INFO" "==================="
    log "INFO" "Tests Passed: $tests_passed/$total_tests"
    log "INFO" "Success Rate: $((tests_passed * 100 / total_tests))%"

    if [ $tests_passed -eq $total_tests ]; then
        log "SUCCESS" "All tests passed!"
        exit 0
    else
        log "ERROR" "Some tests failed"
        exit 1
    fi
}

# Handle script arguments
main "$@"