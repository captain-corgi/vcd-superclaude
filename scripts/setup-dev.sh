#!/bin/bash

# Development Environment Setup Script
# Automated setup for local development environment

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="multi-agent-system"
GO_VERSION="1.21"
DB_NAME="multiagent"
DB_USER="postgres"
DB_PASSWORD="password"
REDIS_PORT="6379"
APP_PORT="8080"
DEPS_DIR="./deps"
BIN_DIR="./bin"

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

# Check if running on macOS
is_macos() {
    [[ "$(uname)" == "Darwin" ]]
}

# Check if running on Linux
is_linux() {
    [[ "$(uname)" == "Linux" ]]
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Install Go if not installed
install_go() {
    log "INFO" "Checking Go installation..."

    if command_exists go; then
        local installed_version=$(go version | awk '{print $3}' | sed 's/go//')
        log "SUCCESS" "Go $installed_version is installed"
        return 0
    fi

    log "INFO" "Installing Go $GO_VERSION..."

    if is_macos; then
        # Install Go using Homebrew
        if command_exists brew; then
            brew install go
            log "SUCCESS" "Go installed via Homebrew"
        else
            log "ERROR" "Homebrew not found. Please install Homebrew first."
            exit 1
        fi
    elif is_linux; then
        # Install Go using package manager
        if command_exists apt-get; then
            sudo apt-get update
            sudo apt-get install -y golang-go
            log "SUCCESS" "Go installed via apt-get"
        elif command_exists yum; then
            sudo yum install -y golang
            log "SUCCESS" "Go installed via yum"
        else
            log "ERROR" "No supported package manager found"
            exit 1
        fi
    else
        log "ERROR" "Unsupported operating system"
        exit 1
    fi

    # Add Go to PATH
    export PATH=$PATH:/usr/local/go/bin
}

# Install Docker if not installed
install_docker() {
    log "INFO" "Checking Docker installation..."

    if command_exists docker; then
        log "SUCCESS" "Docker is installed"
        return 0
    fi

    log "INFO" "Installing Docker..."

    if is_macos; then
        if command_exists brew; then
            brew install docker
            log "SUCCESS" "Docker installed via Homebrew"
        else
            log "ERROR" "Homebrew not found. Please install Homebrew first."
            exit 1
        fi
    elif is_linux; then
        # Install Docker using official script
        curl -fsSL https://get.docker.com -o get-docker.sh
        sh get-docker.sh
        log "SUCCESS" "Docker installed via official script"

        # Start Docker service
        sudo systemctl start docker
        sudo systemctl enable docker

        # Add user to docker group
        sudo usermod -aG docker $USER
        log "INFO" "User added to docker group. Please restart your terminal."
    else
        log "ERROR" "Unsupported operating system"
        exit 1
    fi
}

# Install development tools
install_dev_tools() {
    log "INFO" "Installing development tools..."

    # Install Git if not installed
    if ! command_exists git; then
        if is_macos; then
            brew install git
        elif is_linux; then
            sudo apt-get install -y git || sudo yum install -y git
        fi
        log "SUCCESS" "Git installed"
    fi

    # Install Make if not installed
    if ! command_exists make; then
        if is_macos; then
            brew install make
        elif is_linux; then
            sudo apt-get install -y build-essential || sudo yum install -y make
        fi
        log "SUCCESS" "Make installed"
    fi

    # Install golangci-lint if not installed
    if ! command_exists golangci-lint; then
        if is_macos; then
            brew install golangci-lint
        elif is_linux; then
            curl -sSf https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
        fi
        log "SUCCESS" "golangci-lint installed"
    fi

    # Install GoSec if not installed
    if ! command_exists gosec; then
        go install github.com/securego/gosec/v2/cmd/gosec@latest
        log "SUCCESS" "GoSec installed"
    fi

    log "SUCCESS" "All development tools installed"
}

# Create necessary directories
create_directories() {
    log "INFO" "Creating necessary directories..."

    # Create dependencies directory
    mkdir -p "$DEPS_DIR"

    # Create binary directory
    mkdir -p "$BIN_DIR"

    # Create logs directory
    mkdir -p logs

    # Create configs directory
    mkdir -p configs

    log "SUCCESS" "Directories created"
}

# Initialize Go module
init_go_module() {
    log "INFO" "Initializing Go module..."

    if [ ! -f "go.mod" ]; then
        go mod init github.com/anh-tuan-l/multi-agent-system
        go mod tidy
        log "SUCCESS" "Go module initialized"
    else
        log "INFO" "Go module already exists"
        go mod tidy
    fi
}

# Create environment file
create_env_file() {
    log "INFO" "Creating environment file..."

    if [ ! -f ".env" ]; then
        cat > .env << EOF
# Application Configuration
APP_NAME=multi-agent-system
APP_VERSION=1.0.0
APP_DEBUG=true
APP_PORT=$APP_PORT
APP_TIMEOUT=30s

# Database Configuration
DATABASE_URL=postgres://$DB_USER:$DB_PASSWORD@localhost:5432/$DB_NAME?sslmode=disable
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=$DB_NAME
DATABASE_USER=$DB_USER
DATABASE_PASSWORD=$DB_PASSWORD

# Redis Configuration
REDIS_URL=redis://localhost:$REDIS_PORT
REDIS_HOST=localhost
REDIS_PORT=$REDIS_PORT

# API Configuration
JIRA_CLIENT_ID=your-jira-client-id
JIRA_CLIENT_SECRET=your-jira-client-secret
SLACK_CLIENT_ID=your-slack-client-id
SLACK_CLIENT_SECRET=your-slack-client-secret
GITHUB_TOKEN=your-github-token

# Logging Configuration
LOG_LEVEL=debug
LOG_FORMAT=json

# Development Configuration
DEV_MODE=true
DEV_HOST=localhost
DEV_PORT=$APP_PORT
EOF

        log "SUCCESS" "Environment file created: .env"
    else
        log "INFO" "Environment file already exists"
    fi
}

# Create docker-compose file for local development
create_docker_compose_local() {
    log "INFO" "Creating local docker-compose file..."

    cat > docker-compose.local.yml << EOF
version: '3.8'

services:
  # PostgreSQL database
  postgres:
    image: postgres:15-alpine
    container_name: multi-agent-db-local
    environment:
      POSTGRES_DB: $DB_NAME
      POSTGRES_USER: $DB_USER
      POSTGRES_PASSWORD: $DB_PASSWORD
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./docker/init.sql:/docker-entrypoint-initdb.d/init.sql
    networks:
      - multi-agent-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $DB_USER"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Redis cache
  redis:
    image: redis:7-alpine
    container_name: multi-agent-redis-local
    ports:
      - "$REDIS_PORT:$REDIS_PORT"
    volumes:
      - redis_data:/data
    networks:
      - multi-agent-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 3

  # Application
  app:
    build:
      context: .
      dockerfile: docker/Dockerfile.dev
    container_name: multi-agent-app-local
    ports:
      - "$APP_PORT:8080"
      - "2345:2345" # Debug port
    environment:
      - APP_NAME=multi-agent-system
      - APP_VERSION=1.0.0
      - APP_DEBUG=true
      - APP_PORT=$APP_PORT
      - DATABASE_URL=postgres://$DB_USER:$DB_PASSWORD@postgres:5432/$DB_NAME?sslmode=disable
      - REDIS_URL=redis://redis:$REDIS_PORT
      - LOG_LEVEL=debug
    volumes:
      - .:/app:cached
      - /app/go.mod
      - /app/go.sum
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - multi-agent-network
    restart: unless-stopped
    profiles:
      - dev

networks:
  multi-agent-network:
    driver: bridge

volumes:
  postgres_data:
    driver: local
  redis_data:
    driver: local
EOF

    log "SUCCESS" "Local docker-compose file created: docker-compose.local.yml"
}

# Install Go dependencies
install_go_dependencies() {
    log "INFO" "Installing Go dependencies..."

    go mod tidy

    # Install additional development tools
    go install github.com/go-delve/delve/cmd/dlv@latest
    go install golang.org/x/tools/cmd/goimports@latest

    log "SUCCESS" "Go dependencies installed"
}

# Create development scripts
create_dev_scripts() {
    log "INFO" "Creating development scripts..."

    # Create start script
    cat > scripts/start.sh << 'EOF'
#!/bin/bash
echo "🚀 Starting Multi-Agent System..."
make dev
EOF

    # Create stop script
    cat > scripts/stop.sh << 'EOF'
#!/bin/bash
echo "🛑 Stopping Multi-Agent System..."
docker-compose -f docker-compose.local.yml down
EOF

    # Create restart script
    cat > scripts/restart.sh << 'EOF'
#!/bin/bash
echo "🔄 Restarting Multi-Agent System..."
./scripts/stop.sh
sleep 2
./scripts/start.sh
EOF

    # Make scripts executable
    chmod +x scripts/*.sh

    log "SUCCESS" "Development scripts created"
}

# Run database migrations
run_database_migrations() {
    log "INFO" "Running database migrations..."

    # Wait for database to be ready
    sleep 5

    # Create database if it doesn't exist
    if command_exists psql; then
        psql -h localhost -U $DB_USER -c "CREATE DATABASE $DB_NAME;" 2>/dev/null || log "INFO" "Database already exists"
        log "SUCCESS" "Database setup completed"
    else
        log "WARNING" "PostgreSQL client not available, skipping database setup"
    fi
}

# Verify installation
verify_installation() {
    log "INFO" "Verifying installation..."

    # Check Go
    if command_exists go; then
        log "SUCCESS" "Go is installed: $(go version)"
    else
        log "ERROR" "Go is not installed"
        return 1
    fi

    # Check Docker
    if command_exists docker; then
        log "SUCCESS" "Docker is installed: $(docker --version)"
    else
        log "ERROR" "Docker is not installed"
        return 1
    fi

    # Check Make
    if command_exists make; then
        log "SUCCESS" "Make is installed"
    else
        log "ERROR" "Make is not installed"
        return 1
    fi

    # Check project files
    if [ -f "go.mod" ]; then
        log "SUCCESS" "Go module initialized"
    else
        log "ERROR" "Go module not found"
        return 1
    fi

    if [ -f ".env" ]; then
        log "SUCCESS" "Environment file created"
    else
        log "ERROR" "Environment file not found"
        return 1
    fi

    log "SUCCESS" "All checks passed!"
}

# Main setup function
main() {
    log "INFO" "Starting development environment setup for $PROJECT_NAME"
    log "INFO" "Timestamp: $(date)"
    echo ""

    # Install prerequisites
    install_go
    install_docker
    install_dev_tools

    # Create project structure
    create_directories
    init_go_module
    create_env_file
    create_docker_compose_local

    # Install dependencies
    install_go_dependencies
    create_dev_scripts

    # Setup database
    run_database_migrations

    # Verify installation
    verify_installation

    echo ""
    log "SUCCESS" "Development environment setup completed!"
    echo ""
    log "INFO" "Next steps:"
    log "INFO" "  1. Start the application: ./scripts/start.sh"
    log "INFO" "  2. Access the application: http://localhost:$APP_PORT"
    log "INFO" "  3. View logs: docker-compose -f docker-compose.local.yml logs -f"
    log "INFO" "  4. Run tests: make test"
    log "INFO" "  5. Health check: ./scripts/health-check.sh"
    echo ""
    log "INFO" "Environment variables are configured in .env"
    log "INFO" "Development scripts are available in scripts/"
}