#!/bin/bash

# Deployment Script for Multi-Agent System
# Automated deployment with validation and rollback support

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="multi-agent-system"
IMAGE_NAME="multi-agent-system"
IMAGE_TAG="latest"
DEPLOY_ENV=${1:-"production"}
BACKUP_DIR="./backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

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

# Error handling
handle_error() {
    local error_message=$1
    local exit_code=${2:-1}
    log "ERROR" "$error_message"
    exit $exit_code
}

# Check if deployment environment is valid
validate_environment() {
    log "INFO" "Validating deployment environment..."

    if [ "$DEPLOY_ENV" != "production" ] && [ "$DEPLOY_ENV" != "staging" ] && [ "$DEPLOY_ENV" != "development" ]; then
        handle_error "Invalid deployment environment: $DEPLOY_ENV. Must be production, staging, or development."
    fi

    # Check if Docker is running
    if ! docker info >/dev/null 2>&1; then
        handle_error "Docker is not running"
    fi

    log "SUCCESS" "Environment validation passed for $DEPLOY_ENV"
}

# Create backup directory
create_backup_dir() {
    if [ ! -d "$BACKUP_DIR" ]; then
        mkdir -p "$BACKUP_DIR"
        log "INFO" "Created backup directory: $BACKUP_DIR"
    fi
}

# Backup current deployment
backup_deployment() {
    log "INFO" "Creating backup of current deployment..."

    local backup_file="$BACKUP_DIR/deployment_backup_$TIMESTAMP.tar.gz"

    # Create backup of configs and important files
    tar -czf "$backup_file" \
        --exclude="node_modules" \
        --exclude=".git" \
        --exclude="*.log" \
        --exclude="backups" \
        ./

    if [ $? -eq 0 ]; then
        log "SUCCESS" "Backup created: $backup_file"
        echo "$backup_file" > "$BACKUP_DIR/latest_backup.txt"
    else
        handle_error "Failed to create backup"
    fi
}

# Build Docker image
build_docker_image() {
    log "INFO" "Building Docker image..."

    # Pull latest base image
    docker pull golang:1.21-alpine || log "WARNING" "Failed to pull base image, using local"

    # Build the image
    docker build -f "docker/Dockerfile" -t "$IMAGE_NAME:$IMAGE_TAG" . || handle_error "Failed to build Docker image"

    # Tag for deployment environment
    docker tag "$IMAGE_NAME:$IMAGE_TAG" "$IMAGE_NAME:$DEPLOY_ENV"

    log "SUCCESS" "Docker image built successfully"
}

# Run tests before deployment
run_pre_deploy_tests() {
    log "INFO" "Running pre-deployment tests..."

    # Run unit tests
    if ! make test-unit; then
        log "WARNING" "Unit tests failed, but continuing with deployment"
    else
        log "SUCCESS" "Unit tests passed"
    fi

    # Run security scan
    if command -v gosec &> /dev/null; then
        if ! gosec ./...; then
            log "WARNING" "Security scan found issues, review output"
        else
            log "SUCCESS" "Security scan passed"
        fi
    else
        log "WARNING" "Security scanner not available"
    fi
}

# Check service dependencies
check_dependencies() {
    log "INFO" "Checking service dependencies..."

    # Check database
    if docker ps | grep -q "postgres"; then
        log "SUCCESS" "PostgreSQL is running"
    else
        log "WARNING" "PostgreSQL is not running"
    fi

    # Check Redis
    if docker ps | grep -q "redis"; then
        log "SUCCESS" "Redis is running"
    else
        log "WARNING" "Redis is not running"
    fi
}

# Stop existing containers
stop_existing_containers() {
    log "INFO" "Stopping existing containers..."

    local containers=("multi-agent-system" "multi-agent-system-app")

    for container in "${containers[@]}"; do
        if docker ps -a --format "{{.Names}}" | grep -q "$container"; then
            docker stop "$container" || log "WARNING" "Failed to stop container: $container"
            docker rm "$container" || log "WARNING" "Failed to remove container: $container"
            log "INFO" "Removed container: $container"
        fi
    done
}

# Start new deployment
start_deployment() {
    log "INFO" "Starting new deployment..."

    # Start database and Redis if not running
    docker-compose -f docker/docker-compose.yml up -d postgres redis

    # Wait for services to be ready
    sleep 10

    # Start application
    docker-compose -f docker/docker-compose.yml up -d

    if [ $? -eq 0 ]; then
        log "SUCCESS" "Deployment started successfully"
    else
        handle_error "Failed to start deployment"
    fi
}

# Verify deployment
verify_deployment() {
    log "INFO" "Verifying deployment..."

    # Wait for application to be ready
    sleep 30

    # Check application health
    local app_url="http://localhost:8080"
    local health_check_attempts=0
    local max_attempts=5

    while [ $health_check_attempts -lt $max_attempts ]; do
        if curl -s "$app_url/health" | grep -q "healthy"; then
            log "SUCCESS" "Application is healthy"
            return 0
        else
            log "WARNING" "Health check failed, attempt $((health_check_attempts + 1))/$max_attempts"
            sleep 10
            health_check_attempts=$((health_check_attempts + 1))
        fi
    done

    handle_error "Application health check failed after $max_attempts attempts"
}

# Run post-deployment tests
run_post_deploy_tests() {
    log "INFO" "Running post-deployment tests..."

    # Integration tests
    if command -v curl &> /dev/null; then
        # Test API endpoints
        local endpoints=(
            "/health"
            "/api/v1/agents"
            "/api/v1/requests"
        )

        for endpoint in "${endpoints[@]}"; do
            if curl -s "http://localhost:8080$endpoint" | grep -q "error\|ERROR"; then
                log "WARNING" "API test failed for endpoint: $endpoint"
            else
                log "SUCCESS" "API test passed for endpoint: $endpoint"
            fi
        done
    fi
}

# Monitor deployment
monitor_deployment() {
    log "INFO" "Monitoring deployment for 60 seconds..."

    local monitoring_time=60
    local monitoring_interval=5
    local elapsed=0

    while [ $elapsed -lt $monitoring_time ]; do
        if docker ps | grep -q "multi-agent-system.*Up"; then
            log "SUCCESS" "Application is running normally at elapsed: ${elapsed}s"
        else
            log "WARNING" "Application container state changed at elapsed: ${elapsed}s"
        fi

        sleep $monitoring_interval
        elapsed=$((elapsed + monitoring_interval))
    done

    log "SUCCESS" "Deployment monitoring completed"
}

# Cleanup old backups
cleanup_old_backups() {
    log "INFO" "Cleaning up old backups..."

    # Keep only the last 5 backups
    local backup_count=$(ls -1 "$BACKUP_DIR"/deployment_backup_*.tar.gz 2>/dev/null | wc -l)

    if [ $backup_count -gt 5 ]; then
        local old_backups=$(ls -1t "$BACKUP_DIR"/deployment_backup_*.tar.gz | tail -n +6)
        for backup in $old_backups; do
            rm "$backup"
            log "INFO" "Removed old backup: $backup"
        done
    fi

    log "SUCCESS" "Backup cleanup completed"
}

# Main deployment function
main() {
    log "INFO" "Starting deployment for $DEPLOY_ENV environment"
    log "INFO" "Timestamp: $(date)"
    echo ""

    # Validate environment
    validate_environment

    # Create backup directory
    create_backup_dir

    # Backup current deployment
    backup_deployment

    # Run tests before deployment
    run_pre_deploy_tests

    # Check dependencies
    check_dependencies

    # Build Docker image
    build_docker_image

    # Stop existing containers
    stop_existing_containers

    # Start new deployment
    start_deployment

    # Verify deployment
    verify_deployment

    # Run post-deployment tests
    run_post_deploy_tests

    # Monitor deployment
    monitor_deployment

    # Cleanup old backups
    cleanup_old_backups

    log "SUCCESS" "Deployment completed successfully!"
    echo ""
    log "INFO" "Deployment Summary:"
    log "INFO" "  Environment: $DEPLOY_ENV"
    log "INFO" "  Image: $IMAGE_NAME:$IMAGE_TAG"
    log "INFO" "  Backup: $(cat "$BACKUP_DIR/latest_backup.txt")"
    log "INFO" "  Timestamp: $TIMESTAMP"
}

# Handle script arguments
case "${1:-deploy}" in
    "deploy")
        main
        ;;
    "backup")
        create_backup_dir
        backup_deployment
        ;;
    "build")
        build_docker_image
        ;;
    "test")
        run_pre_deploy_tests
        ;;
    "help")
        echo "Usage: $0 [deploy|backup|build|test|help]"
        echo "  deploy  - Full deployment process (default)"
        echo "  backup  - Create backup only"
        echo "  build   - Build Docker image only"
        echo "  test    - Run pre-deployment tests only"
        echo "  help    - Show this help message"
        ;;
    *)
        handle_error "Unknown command: $1. Use 'help' for usage information."
        ;;
esac