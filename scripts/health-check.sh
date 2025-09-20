#!/bin/bash

# Health Check Script for Multi-Agent System
# Checks the health of all services and dependencies

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
APP_URL=${APP_URL:-"http://localhost:8080"}
DB_URL=${DB_URL:-"localhost:5432"}
REDIS_URL=${REDIS_URL:-"localhost:6379"}
TIMEOUT=10

# Function to print status
print_status() {
    local status=$1
    local message=$2
    case $status in
        "SUCCESS") echo -e "${GREEN}✅ $message${NC}" ;;
        "WARNING") echo -e "${YELLOW}⚠️  $message${NC}" ;;
        "ERROR") echo -e "${RED}❌ $message${NC}" ;;
    esac
}

# Function to check service availability
check_service() {
    local url=$1
    local name=$2
    local expected_code=${3:-200}

    if curl -s -o /dev/null -w "%{http_code}" --connect-timeout $TIMEOUT $url | grep -q "$expected_code"; then
        print_status "SUCCESS" "$name is healthy"
        return 0
    else
        print_status "ERROR" "$name is not available"
        return 1
    fi
}

# Function to check port availability
check_port() {
    local host=$1
    local port=$2
    local service=$3

    if nc -z $host $port >/dev/null 2>&1; then
        print_status "SUCCESS" "$service is listening on $host:$port"
        return 0
    else
        print_status "ERROR" "$service is not listening on $host:$port"
        return 1
    fi
}

# Function to check Docker services
check_docker_services() {
    echo "🐳 Checking Docker services..."

    # Check if Docker is running
    if ! docker info >/dev/null 2>&1; then
        print_status "ERROR" "Docker is not running"
        return 1
    fi
    print_status "SUCCESS" "Docker is running"

    # Check if services are running
    local services=("multi-agent-system" "postgres" "redis")
    for service in "${services[@]}"; do
        if docker ps | grep -q "$service"; then
            print_status "SUCCESS" "Docker container $service is running"
        else
            print_status "WARNING" "Docker container $service is not running"
        fi
    done
}

# Function to check database connectivity
check_database() {
    echo "📊 Checking database connectivity..."

    # Check PostgreSQL
    if command -v psql &> /dev/null; then
        if psql -h $DB_URL -U postgres -d multiagent -c "SELECT 1;" >/dev/null 2>&1; then
            print_status "SUCCESS" "PostgreSQL database is accessible"
        else
            print_status "ERROR" "PostgreSQL database is not accessible"
        fi
    else
        print_status "WARNING" "PostgreSQL client not available, skipping database check"
    fi
}

# Function to check Redis connectivity
check_redis() {
    echo "🔴 Checking Redis connectivity..."

    if command -v redis-cli &> /dev/null; then
        if redis-cli -h $REDIS_URL ping >/dev/null 2>&1; then
            print_status "SUCCESS" "Redis is accessible"
        else
            print_status "ERROR" "Redis is not accessible"
        fi
    else
        print_status "WARNING" "Redis CLI not available, skipping Redis check"
    fi
}

# Function to check application dependencies
check_dependencies() {
    echo "📦 Checking application dependencies..."

    # Check Go version
    local go_version=$(go version 2>/dev/null | awk '{print $3}' | sed 's/go//')
    if [ -n "$go_version" ]; then
        print_status "SUCCESS" "Go $go_version is installed"
    else
        print_status "ERROR" "Go is not installed"
        return 1
    fi

    # Check required tools
    local tools=("docker" "make" "curl")
    for tool in "${tools[@]}"; do
        if command -v $tool &> /dev/null; then
            print_status "SUCCESS" "$tool is available"
        else
            print_status "WARNING" "$tool is not available"
        fi
    done
}

# Function to check application health
check_app_health() {
    echo "🚀 Checking application health..."

    # Check if application is running
    if check_service $APP_URL "Application"; then
        # Check specific health endpoint
        if check_service "$APP_URL/health" "Health endpoint" 200; then
            print_status "SUCCESS" "Application health check passed"
        else
            print_status "WARNING" "Application is running but health endpoint failed"
        fi
    else
        print_status "WARNING" "Application is not running"
    fi
}

# Main health check
main() {
    echo "🔍 Multi-Agent System Health Check"
    echo "=================================="
    echo ""

    # Check system requirements
    check_dependencies
    echo ""

    # Check Docker services
    check_docker_services
    echo ""

    # Check database connectivity
    check_database
    echo ""

    # Check Redis connectivity
    check_redis
    echo ""

    # Check application health
    check_app_health
    echo ""

    # Summary
    echo "📋 Health Check Summary"
    echo "======================"

    # Calculate overall health
    local healthy_checks=0
    local total_checks=0

    # Count successful checks
    if [ $? -eq 0 ]; then
        healthy_checks=$((healthy_checks + 1))
    fi
    total_checks=$((total_checks + 1))

    echo "Completed $total_checks checks"

    if [ $healthy_checks -eq $total_checks ]; then
        print_status "SUCCESS" "All health checks passed!"
        exit 0
    else
        print_status "WARNING" "Some health checks failed. Please review the output above."
        exit 1
    fi
}

# Run main function
main "$@"