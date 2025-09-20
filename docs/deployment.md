# Deployment Guide

Comprehensive guide for deploying the multi-agent system using Docker Compose for local development and testing.

## 🏗️ Deployment Architecture

```mermaid
graph TB
    subgraph "Local Development Environment"
        HOST[Host Machine]
        DOCKER[Docker Engine]
        COMPOSE[Docker Compose]
    end

    subgraph "Application Services"
        APP[Multi-Agent App<br/>Go Application]
        WEB[Web Server<br/>Echo Framework]
        AGENTS[Agent Services<br/>Planner/Executor/Evaluator]
    end

    subgraph "Data Services"
        POSTGRES[(PostgreSQL<br/>Message Storage)]
        REDIS[(Redis<br/>Cache Layer)]
        SQLITE[(SQLite<br/>Local Development)]
    end

    subgraph "External Services"
        JIRA_EXT[Jira API]
        SLACK_EXT[Slack API]
        GITHUB_EXT[GitHub API]
    end

    HOST --> DOCKER
    DOCKER --> COMPOSE
    COMPOSE --> APP
    COMPOSE --> WEB
    COMPOSE --> POSTGRES
    COMPOSE --> REDIS
    COMPOSE --> SQLITE

    APP --> AGENTS
    AGENTS --> POSTGRES
    AGENTS --> REDIS
    AGENTS --> JIRA_EXT
    AGENTS --> SLACK_EXT
    AGENTS --> GITHUB_EXT
```

## 📁 Project Structure

```
multi-agent-system/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── agents/
│   ├── communication/
│   ├── tools/
│   └── web/
├── pkg/
│   ├── config/
│   ├── models/
│   └── storage/
├── configs/
│   ├── app.yaml
│   ├── database.yaml
│   └── logging.yaml
├── docker/
│   ├── Dockerfile
│   ├── docker-compose.yml
│   └── docker-compose.dev.yml
├── docs/
├── scripts/
│   ├── build.sh
│   ├── deploy.sh
│   └── test.sh
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## 🐳 Docker Configuration

### Dockerfile
```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 appgroup && adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .
COPY --from=builder /app/configs ./configs

# Create directories
RUN mkdir -p /app/logs /app/data && chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run application
CMD ["./main"]
```

### docker-compose.yml
```yaml
version: '3.8'

services:
  app:
    build:
      context: ..
      dockerfile: docker/Dockerfile
    container_name: multi-agent-app
    ports:
      - "8080:8080"
    environment:
      - APP_ENV=production
      - DATABASE_URL=postgres://postgres:password@postgres:5432/multiagent?sslmode=disable
      - REDIS_URL=redis://redis:6379
      - LOG_LEVEL=info
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    volumes:
      - ../logs:/app/logs
      - ../data:/app/data
    restart: unless-stopped
    networks:
      - multi-agent-network

  postgres:
    image: postgres:15-alpine
    container_name: multi-agent-postgres
    environment:
      - POSTGRES_DB=multiagent
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=password
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ../docker/init.sql:/docker-entrypoint-initdb.d/init.sql
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped
    networks:
      - multi-agent-network

  redis:
    image: redis:7-alpine
    container_name: multi-agent-redis
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 3
    restart: unless-stopped
    networks:
      - multi-agent-network

  nginx:
    image: nginx:alpine
    container_name: multi-agent-nginx
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ../docker/nginx.conf:/etc/nginx/nginx.conf
      - ../docker/ssl:/etc/nginx/ssl
    depends_on:
      - app
    restart: unless-stopped
    networks:
      - multi-agent-network

volumes:
  postgres_data:
  redis_data:

networks:
  multi-agent-network:
    driver: bridge
```

### docker-compose.dev.yml
```yaml
version: '3.8'

services:
  app:
    build:
      context: ..
      dockerfile: docker/Dockerfile.dev
    container_name: multi-agent-app-dev
    ports:
      - "8080:8080"
      - "2345:2345"  # Debug port
    environment:
      - APP_ENV=development
      - DATABASE_URL=postgres://postgres:password@postgres:5432/multiagent?sslmode=disable
      - REDIS_URL=redis://redis:6379
      - LOG_LEVEL=debug
      - DELVE_ENABLED=true
    volumes:
      - ..:/app
      - go_mod_cache:/go/pkg/mod
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - multi-agent-network

  postgres:
    image: postgres:15-alpine
    container_name: multi-agent-postgres-dev
    environment:
      - POSTGRES_DB=multiagent
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - multi-agent-network

  redis:
    image: redis:7-alpine
    container_name: multi-agent-redis-dev
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 3
    networks:
      - multi-agent-network

volumes:
  postgres_data:
  redis_data:
  go_mod_cache:

networks:
  multi-agent-network:
    driver: bridge
```

## ⚙️ Configuration

### Environment Variables
```bash
# .env.example

# Application
APP_ENV=development
APP_PORT=8080
APP_DEBUG=true

# Database
DATABASE_URL=postgres://postgres:password@localhost:5432/multiagent?sslmode=disable
DB_MAX_CONNECTIONS=25
DB_MAX_IDLE_CONNECTIONS=5
DB_CONNECTION_LIFETIME=5m

# Redis
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Logging
LOG_LEVEL=debug
LOG_FORMAT=json
LOG_OUTPUT=stdout

# API Credentials
JIRA_BASE_URL=https://your-domain.atlassian.net
JIRA_CLIENT_ID=your-client-id
JIRA_CLIENT_SECRET=your-client-secret

SLACK_CLIENT_ID=your-client-id
SLACK_CLIENT_SECRET=your-client-secret

GITHUB_TOKEN=your-personal-access-token

# Agent Configuration
PLANNER_ENABLED=true
PLANNER_MAX_REQUESTS=10
PLANNER_TIMEOUT=30s

EXECUTOR_ENABLED=true
EXECUTOR_RATE_LIMIT=100
EXECUTOR_RETRY_COUNT=3

EVALUATOR_ENABLED=true
EVALUATOR_CACHE_TTL=1h
EVALUATOR_MAX_REPORT_SIZE=10MB

# Security
JWT_SECRET=your-secret-key
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080
```

### Configuration Files

#### configs/app.yaml
```yaml
app:
  name: "multi-agent-system"
  version: "1.0.0"
  debug: true
  port: 8080
  timeout: 30s

agents:
  planner:
    enabled: true
    max_requests: 10
    timeout: 30s

  executor:
    enabled: true
    rate_limit: 100
    retry_count: 3
    tools:
      - jira
      - slack
      - github

  evaluator:
    enabled: true
    cache_ttl: 1h
    max_report_size: 10MB

communication:
  a2a:
    protocol: "trpc"
    timeout: 10s
    retries: 3

  streaming:
    enabled: true
    buffer_size: 1000

tools:
  jira:
    base_url: "${JIRA_BASE_URL}"
    rate_limit:
      requests_per_minute: 100
    cache:
      ttl: 5m

  slack:
    rate_limit:
      tier: "tier3"
    pagination:
      limit: 200

  github:
    rate_limit:
      requests_per_hour: 5000
      remaining_threshold: 100
    cache:
      ttl: 10m
```

#### configs/database.yaml
```yaml
database:
  driver: "postgres"
  url: "${DATABASE_URL}"
  max_connections: 25
  max_idle_connections: 5
  connection_lifetime: 5m
  migrations:
    enabled: true
    path: "migrations"

redis:
  url: "${REDIS_URL}"
  password: "${REDIS_PASSWORD}"
  db: 0
  pool_size: 10
  min_idle_connections: 5
```

## 🚀 Deployment Scripts

### Makefile
```makefile
.PHONY: help build run test clean deploy dev-setup

# Default target
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Development setup
dev-setup: ## Setup development environment
	@echo "Setting up development environment..."
	@cp .env.example .env
	@go mod download
	@go mod tidy
	@mkdir -p logs data
	@echo "Development environment setup complete!"

# Build application
build: ## Build the application
	@echo "Building application..."
	@go build -o bin/server ./cmd/server
	@echo "Build complete!"

# Run application
run: ## Run the application
	@echo "Running application..."
	@./bin/server

# Run tests
test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -cover ./...

# Clean build artifacts
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf logs/
	@rm -rf data/
	@echo "Clean complete!"

# Docker build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t multi-agent-system:latest -f docker/Dockerfile .
	@echo "Docker build complete!"

# Docker run
docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@docker run -p 8080:8080 --env-file .env multi-agent-system:latest

# Docker compose up
docker-up: ## Start Docker Compose
	@echo "Starting Docker Compose..."
	@docker-compose -f docker/docker-compose.yml up -d

# Docker compose down
docker-down: ## Stop Docker Compose
	@echo "Stopping Docker Compose..."
	@docker-compose -f docker/docker-compose.yml down

# Docker compose logs
docker-logs: ## Show Docker Compose logs
	@echo "Showing Docker Compose logs..."
	@docker-compose -f docker/docker-compose.yml logs -f

# Development environment
dev: ## Start development environment
	@echo "Starting development environment..."
	@docker-compose -f docker/docker-compose.dev.yml up -d

# Development environment logs
dev-logs: ## Show development environment logs
	@echo "Showing development environment logs..."
	@docker-compose -f docker/docker-compose.dev.yml logs -f

# Database migration
migrate-up: ## Run database migrations
	@echo "Running database migrations..."
	@go run cmd/migrate/main.go up

# Database migration down
migrate-down: ## Rollback database migrations
	@echo "Rolling back database migrations..."
	@go run cmd/migrate/main.go down

# Generate documentation
docs: ## Generate documentation
	@echo "Generating documentation..."
	@swag init -g ./cmd/server/main.go -o ./docs/api

# Lint code
lint: ## Lint code
	@echo "Linting code..."
	@golangci-lint run

# Format code
fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

# Security scan
security: ## Run security scan
	@echo "Running security scan..."
	@gosec ./...

# All checks
check: fmt lint test security ## Run all checks
	@echo "All checks complete!"
```

### Deployment Scripts

#### scripts/build.sh
```bash
#!/bin/bash

set -e

echo "🚀 Starting build process..."

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | xargs)
fi

# Create build directory
mkdir -p build

# Build application
echo "📦 Building Go application..."
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/server ./cmd/server

# Copy configuration files
echo "📋 Copying configuration files..."
cp -r configs build/

# Copy static files
echo "🎨 Copying static files..."
cp -r web/static build/

# Create package
echo "📦 Creating deployment package..."
tar -czf build/multi-agent-system.tar.gz -C build .

echo "✅ Build complete!"
```

#### scripts/deploy.sh
```bash
#!/bin/bash

set -e

echo "🚀 Starting deployment..."

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | xargs)
fi

# Build Docker image
echo "🐳 Building Docker image..."
docker build -t multi-agent-system:${VERSION:-latest} -f docker/Dockerfile .

# Stop existing containers
echo "🛑 Stopping existing containers..."
docker-compose down

# Start new containers
echo "🚀 Starting new containers..."
docker-compose up -d

# Wait for services to be ready
echo "⏳ Waiting for services to be ready..."
sleep 10

# Run health checks
echo "🏥 Running health checks..."
curl -f http://localhost:8080/health || exit 1

echo "✅ Deployment complete!"
```

#### scripts/test.sh
```bash
#!/bin/bash

set -e

echo "🧪 Running tests..."

# Run unit tests
echo "📝 Running unit tests..."
go test -v -cover ./...

# Run integration tests
echo "🔗 Running integration tests..."
go test -v -tags=integration ./...

# Run end-to-end tests
echo "🔄 Running end-to-end tests..."
go test -v -tags=e2e ./...

echo "✅ All tests passed!"
```

## 📊 Monitoring & Health Checks

### Health Check Endpoints
```go
// main.go
func main() {
    // Initialize application
    app := initializeApp()

    // Health check endpoint
    e.GET("/health", func(c echo.Context) error {
        return c.JSON(http.StatusOK, map[string]interface{}{
            "status": "healthy",
            "timestamp": time.Now().UTC(),
            "version": app.Version,
            "uptime": time.Since(app.StartTime).String(),
        })
    })

    // Detailed health check
    e.GET("/health/detailed", func(c echo.Context) error {
        health := app.GetDetailedHealth()
        return c.JSON(http.StatusOK, health)
    })

    // Readiness check
    e.GET("/ready", func(c echo.Context) error {
        if !app.IsReady() {
            return c.JSON(http.StatusServiceUnavailable, map[string]string{
                "status": "not ready",
            })
        }
        return c.JSON(http.StatusOK, map[string]string{
            "status": "ready",
        })
    })

    // Liveness check
    e.GET("/live", func(c echo.Context) error {
        if !app.IsAlive() {
            return c.JSON(http.StatusServiceUnavailable, map[string]string{
                "status": "not alive",
            })
        }
        return c.JSON(http.StatusOK, map[string]string{
            "status": "alive",
        })
    })
}
```

### Health Check Configuration
```yaml
health:
  endpoints:
    - path: "/health"
      interval: 30s
      timeout: 10s
      retries: 3

    - path: "/health/detailed"
      interval: 60s
      timeout: 30s
      retries: 2

  database:
    check_query: "SELECT 1"
    timeout: 5s

  redis:
    timeout: 3s

  external_apis:
    - name: "jira"
      url: "${JIRA_BASE_URL}/rest/api/3/myself"
      timeout: 10s
      interval: 300s

    - name: "slack"
      url: "https://slack.com/api/auth.test"
      timeout: 10s
      interval: 300s
```

## 🔒 Security Configuration

### Security Best Practices
1. **Non-root User**: Run containers as non-root user
2. **Environment Variables**: Store sensitive data in environment variables
3. **Network Security**: Use internal networks for service communication
4. **Resource Limits**: Set memory and CPU limits
5. **Health Checks**: Monitor application health
6. **Logging**: Enable structured logging
7. **Updates**: Regularly update base images

### Docker Security Configuration
```yaml
# Add to docker-compose.yml
services:
  app:
    # ... existing configuration
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
    security_opt:
      - no-new-privileges:true
    read_only: true
    tmpfs:
      - /tmp
      - /var/run
      - /var/cache/nginx
    cap_drop:
      - ALL
    cap_add:
      - CHOWN
      - SETGID
      - SETUID
```

## 📈 Performance Monitoring

### Monitoring Configuration
```yaml
monitoring:
  enabled: true
  metrics:
    - name: "requests_total"
      type: "counter"
      help: "Total number of requests"

    - name: "request_duration"
      type: "histogram"
      help: "Request duration in seconds"

    - name: "active_connections"
      type: "gauge"
      help: "Number of active connections"

  logging:
    level: "info"
    format: "json"
    output: "stdout"

  tracing:
    enabled: true
    service_name: "multi-agent-system"
    sample_rate: 0.1
```

## 🚨 Troubleshooting

### Common Issues

#### 1. Database Connection Issues
```bash
# Check PostgreSQL logs
docker-compose logs postgres

# Test database connection
docker-compose exec postgres psql -U postgres -d multiagent -c "SELECT 1"

# Reset database
docker-compose down -v
docker-compose up -d postgres
```

#### 2. Redis Connection Issues
```bash
# Check Redis logs
docker-compose logs redis

# Test Redis connection
docker-compose exec redis redis-cli ping

# Clear Redis cache
docker-compose exec redis redis-cli FLUSHALL
```

#### 3. Application Issues
```bash
# Check application logs
docker-compose logs app

# Restart application
docker-compose restart app

# Debug mode
docker-compose -f docker/docker-compose.dev.yml up --build
```

#### 4. Port Conflicts
```bash
# Check port usage
lsof -i :8080
lsof -i :5432
lsof -i :6379

# Change ports in docker-compose.yml
ports:
  - "8081:8080"  # Change host port
```

### Performance Optimization

#### Database Optimization
```yaml
# Add to postgres configuration
environment:
  - POSTGRES_SHARED_BUFFERS=256MB
  - POSTGRES_EFFECTIVE_CACHE_SIZE=1GB
  - POSTGRES_WORK_MEM=16MB
  - POSTGRES_MAINTENANCE_WORK_MEM=64MB
```

#### Redis Optimization
```yaml
# Add to redis configuration
command: redis-server --maxmemory 256mb --maxmemory-policy allkeys-lru
```

#### Application Optimization
```yaml
# Add to app environment
environment:
  - GOMAXPROCS=4
  - GOGC=100
```

## 🎯 Deployment Checklist

### Pre-deployment
- [ ] Environment variables configured
- [ ] Database migrations run
- [ ] SSL certificates configured
- [ ] Security scan completed
- [ ] Performance testing completed

### Deployment
- [ ] Build Docker image
- [ ] Push to registry
- [ ] Update docker-compose.yml
- [ ] Deploy to production
- [ ] Run smoke tests

### Post-deployment
- [ ] Monitor application health
- [ ] Check logs for errors
- [ ] Verify external API connections
- [ ] Test user workflows
- [ ] Backup configuration

### Maintenance
- [ ] Regular updates and patches
- [ ] Security scans
- [ ] Performance monitoring
- [ ] Log rotation and cleanup
- [ ] Backup procedures