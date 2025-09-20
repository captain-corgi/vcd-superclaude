#!/bin/bash

# Database Migration Script
# Handles database schema migrations for Multi-Agent System

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DB_HOST=${DB_HOST:-"localhost"}
DB_PORT=${DB_PORT:-"5432"}
DB_NAME=${DB_NAME:-"multiagent"}
DB_USER=${DB_USER:-"postgres"}
DB_PASSWORD=${DB_PASSWORD:-"password"}
MIGRATIONS_DIR="./migrations"
BACKUP_DIR="./backups"

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

# Check if psql is available
check_psql() {
    if ! command -v psql &> /dev/null; then
        log "ERROR" "PostgreSQL client (psql) is not installed"
        exit 1
    fi
}

# Check database connectivity
check_database() {
    log "INFO" "Checking database connectivity..."

    if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "SELECT 1;" &> /dev/null; then
        log "SUCCESS" "Database connection successful"
    else
        log "ERROR" "Cannot connect to database"
        exit 1
    fi
}

# Create backup directory
create_backup_dir() {
    if [ ! -d "$BACKUP_DIR" ]; then
        mkdir -p "$BACKUP_DIR"
        log "INFO" "Created backup directory: $BACKUP_DIR"
    fi
}

# Create backup of database
create_backup() {
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="$BACKUP_DIR/db_backup_$timestamp.sql"

    log "INFO" "Creating database backup: $backup_file"

    PGPASSWORD=$DB_PASSWORD pg_dump -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME > "$backup_file"

    if [ $? -eq 0 ]; then
        log "SUCCESS" "Database backup created: $backup_file"
    else
        log "ERROR" "Failed to create database backup"
        exit 1
    fi
}

# Run migrations
run_migrations() {
    local direction=$1
    local timestamp=$(date +%Y%m%d_%H%M%S)

    log "INFO" "Running $direction migrations..."

    # Create migrations directory if it doesn't exist
    mkdir -p "$MIGRATIONS_DIR"

    if [ "$direction" = "up" ]; then
        # Create initial migration if no migrations exist
        if [ ! -f "$MIGRATIONS_DIR/001_initial_schema.sql" ]; then
            cat > "$MIGRATIONS_DIR/001_initial_schema.sql" << 'EOF'
-- Multi-Agent System Database Schema

-- Create tables
CREATE TABLE IF NOT EXISTS requests (
    id VARCHAR(255) PRIMARY KEY,
    type VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    metadata JSONB,
    timestamp TIMESTAMP NOT NULL,
    source_agent VARCHAR(100) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending'
);

CREATE TABLE IF NOT EXISTS responses (
    id VARCHAR(255) PRIMARY KEY,
    request_id VARCHAR(255) NOT NULL,
    success BOOLEAN NOT NULL,
    data JSONB,
    error TEXT,
    metadata JSONB,
    timestamp TIMESTAMP NOT NULL,
    target_agent VARCHAR(100) NOT NULL,
    FOREIGN KEY (request_id) REFERENCES requests(id)
);

CREATE TABLE IF NOT EXISTS messages (
    id VARCHAR(255) PRIMARY KEY,
    type VARCHAR(50) NOT NULL,
    content JSONB NOT NULL,
    from_agent VARCHAR(100) NOT NULL,
    to_agent VARCHAR(100) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    correlation_id VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS agent_metrics (
    id SERIAL PRIMARY KEY,
    agent_id VARCHAR(100) NOT NULL,
    requests_processed BIGINT DEFAULT 0,
    requests_success BIGINT DEFAULT 0,
    requests_failed BIGINT DEFAULT 0,
    average_response_time INTERVAL,
    error_rate FLOAT,
    uptime INTERVAL,
    memory_usage BIGINT,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_requests_timestamp ON requests(timestamp);
CREATE INDEX IF NOT EXISTS idx_requests_status ON requests(status);
CREATE INDEX IF NOT EXISTS idx_responses_timestamp ON responses(timestamp);
CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_agent_metrics_agent_id ON agent_metrics(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_metrics_timestamp ON agent_metrics(timestamp);

-- Create migration history table
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    description TEXT
);

-- Record this migration
INSERT INTO schema_migrations (version, description) VALUES ('001', 'Initial schema creation')
ON CONFLICT (version) DO NOTHING;

EOF
        fi

        # Run up migrations
        for migration in "$MIGRATIONS_DIR"/*.sql; do
            if [ -f "$migration" ]; then
                local migration_name=$(basename "$migration")
                log "INFO" "Running migration: $migration_name"

                # Execute migration SQL
                PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$migration"

                if [ $? -eq 0 ]; then
                    log "SUCCESS" "Migration completed: $migration_name"
                else
                    log "ERROR" "Migration failed: $migration_name"
                    exit 1
                fi
            fi
        done

        log "SUCCESS" "All up migrations completed"

    elif [ "$direction" = "down" ]; then
        log "WARNING" "Down migrations are destructive and will remove data"
        read -p "Are you sure you want to run down migrations? (y/N): " -n 1 -r
        echo

        if [[ $REPLY =~ ^[Yy]$ ]]; then
            # Run down migrations in reverse order
            for migration in $(ls -r "$MIGRATIONS_DIR"/*.sql); do
                if [ -f "$migration" ]; then
                    local migration_name=$(basename "$migration")
                    log "INFO" "Running down migration: $migration_name"

                    # Create down migration SQL (reverse of up)
                    # Note: This is a simplified example. In production, you'd have proper down migrations
                    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME << EOF
-- Down migration for $migration_name
-- This is a placeholder - implement proper down logic for each migration
BEGIN;
-- Add your down migration logic here
COMMIT;
EOF

                    if [ $? -eq 0 ]; then
                        log "SUCCESS" "Down migration completed: $migration_name"
                    else
                        log "ERROR" "Down migration failed: $migration_name"
                        exit 1
                    fi
                fi
            done

            log "SUCCESS" "All down migrations completed"
        else
            log "INFO" "Down migrations cancelled"
        fi
    fi
}

# Show migration status
show_migration_status() {
    log "INFO" "Showing migration status..."

    if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "\dt schema_migrations" &> /dev/null; then
        log "SUCCESS" "Migration history table exists"
        PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT version, applied_at, description FROM schema_migrations ORDER BY version;"
    else
        log "WARNING" "Migration history table does not exist"
    fi
}

# Show database info
show_database_info() {
    log "INFO" "Database information:"

    echo "Database: $DB_NAME"
    echo "Host: $DB_HOST:$DB_PORT"
    echo "User: $DB_USER"
    echo ""

    # Show table counts
    log "INFO" "Table counts:"
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
    SELECT
        schemaname,
        tablename,
        n_tup_ins as inserts,
        n_tup_upd as updates,
        n_tup_del as deletes,
        n_live_tup as live_tuples,
        n_dead_tup as dead_tuples
    FROM pg_stat_user_tables
    ORDER BY tablename;
    "

    echo ""
    log "INFO" "Database size:"
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
    SELECT
        pg_size_pretty(pg_database_size('$DB_NAME')) as database_size,
        pg_size_pretty(pg_total_relation_size('requests')) as requests_table_size,
        pg_size_pretty(pg_total_relation_size('responses')) as responses_table_size,
        pg_size_pretty(pg_total_relation_size('messages')) as messages_table_size,
        pg_size_pretty(pg_total_relation_size('agent_metrics')) as agent_metrics_table_size
    ;"
}

# Clean up old backups
cleanup_backups() {
    log "INFO" "Cleaning up old backups..."

    # Keep only the last 10 backups
    local backup_count=$(ls -1 "$BACKUP_DIR"/db_backup_*.sql 2>/dev/null | wc -l)

    if [ $backup_count -gt 10 ]; then
        local old_backups=$(ls -1t "$BACKUP_DIR"/db_backup_*.sql | tail -n +11)
        for backup in $old_backups; do
            rm "$backup"
            log "INFO" "Removed old backup: $backup"
        done
    fi

    log "SUCCESS" "Backup cleanup completed"
}

# Main function
main() {
    case "${1:-status}" in
        "up")
            log "INFO" "Running database migrations up..."
            check_psql
            create_backup_dir
            create_backup
            check_database
            run_migrations "up"
            show_migration_status
            ;;
        "down")
            log "INFO" "Running database migrations down..."
            check_psql
            create_backup_dir
            create_backup
            check_database
            run_migrations "down"
            show_migration_status
            ;;
        "status")
            log "INFO" "Showing database migration status..."
            check_psql
            check_database
            show_migration_status
            show_database_info
            ;;
        "backup")
            log "INFO" "Creating database backup..."
            check_psql
            check_database
            create_backup_dir
            create_backup
            ;;
        "restore")
            log "INFO" "Restoring database from backup..."
            check_psql
            check_database
            # Implementation for restore functionality
            log "WARNING" "Restore functionality not yet implemented"
            ;;
        "clean")
            log "INFO" "Cleaning up old backups..."
            cleanup_backups
            ;;
        "help")
            echo "Usage: $0 [up|down|status|backup|restore|clean|help]"
            echo "  up      - Run all pending migrations"
            echo "  down    - Run down migrations (destructive)"
            echo "  status  - Show migration status and database info"
            echo "  backup  - Create database backup"
            echo "  restore - Restore database from backup"
            echo "  clean   - Clean up old backups"
            echo "  help    - Show this help message"
            ;;
        *)
            log "ERROR" "Unknown command: $1"
            echo "Use '$0 help' for usage information"
            exit 1
            ;;
    esac
}

# Handle script arguments
main "$@"