# Multi-Agent System Makefile
# Build automation and development scripts

.PHONY: help build run test fmt lint clean dev-setup dev docker-build docker-up docker-down migrate-up migrate-down check

# Default target
help: ## Show this help message
	@echo "Multi-Agent System Development Commands:"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "%-20s %-50s\n", "Target", "Description"} /^[a-zA-Z_-]+:.*?##/ { printf "%-20s %-50s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# Project Setup
dev-setup: ## Initialize development environment
	@echo "🚀 Setting up development environment..."
	go mod tidy
	cp .env.example .env
	@echo "✅ Development environment setup complete!"

# Build Commands
build: ## Build the application
	@echo "🏗️  Building application..."
	go build -o bin/server cmd/server/main.go
	@echo "✅ Build complete: bin/server"

run: ## Run the application locally
	@echo "🚀 Starting application..."
	go run cmd/server/main.go

run-debug: ## Run the application with debugging
	@echo "🔍 Starting application with debugging..."
	go run -tags debug cmd/server/main.go

# Testing Commands
test: ## Run all tests
	@echo "🧪 Running tests..."
	go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "🧪 Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

test-unit: ## Run unit tests only
	@echo "🧪 Running unit tests..."
	go test -short -v ./...

test-integration: ## Run integration tests
	@echo "🧪 Running integration tests..."
	go test -run TestIntegration -v ./...

# Quality Commands
fmt: ## Format code
	@echo "🎨 Formatting code..."
	go fmt ./...
	@echo "✅ Code formatted"

lint: ## Run linter
	@echo "🔍 Running linter..."
	golangci-lint run
	@echo "✅ Linting complete"

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	go vet ./...
	@echo "✅ Vet complete"

security: ## Run security scan
	@echo "🛡️  Running security scan..."
	gosec ./...
	@echo "✅ Security scan complete"

# Quality Check (Combined)
check: fmt vet lint test ## Run all quality checks
	@echo "✅ All quality checks passed!"

# Database Commands
migrate-up: ## Run database migrations up
	@echo "📊 Running database migrations up..."
	go run cmd/server/main.go migrate-up
	@echo "✅ Migrations up complete"

migrate-down: ## Run database migrations down
	@echo "📊 Running database migrations down..."
	go run cmd/server/main.go migrate-down
	@echo "✅ Migrations down complete"

db-logs: ## View database logs
	@echo "📊 Viewing database logs..."
	docker-compose -f docker/docker-compose.yml logs postgres

# Docker Commands
docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -f docker/Dockerfile -t multi-agent-system:latest .

docker-up: ## Start Docker Compose (production)
	@echo "🐳 Starting Docker Compose (production)..."
	docker-compose -f docker/docker-compose.yml up -d

docker-down: ## Stop Docker Compose
	@echo "🛑 Stopping Docker Compose..."
	docker-compose -f docker/docker-compose.yml down

docker-logs: ## View Docker logs
	@echo "📊 Viewing Docker logs..."
	docker-compose -f docker/docker-compose.yml logs -f

docker-dev: ## Start development environment with Docker
	@echo "🐳 Starting development environment with Docker..."
	docker-compose -f docker/docker-compose.dev.yml --profile dev up

docker-dev-down: ## Stop development Docker environment
	@echo "🛑 Stopping development Docker environment..."
	docker-compose -f docker/docker-compose.dev.yml down

# Development Commands
dev: ## Start development environment
	@echo "🚀 Starting development environment..."
	docker-compose -f docker/docker-compose.dev.yml --profile dev up -d --build

dev-logs: ## View development environment logs
	@echo "📊 Viewing development logs..."
	docker-compose -f docker/docker-compose.dev.yml logs -f

dev-shell: ## Open shell in development container
	@echo "🚀 Opening shell in development container..."
	docker-compose -f docker/docker-compose.dev.yml exec app sh

# Cleanup Commands
clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean -cache
	@echo "✅ Cleanup complete"

# Documentation
docs: ## Generate documentation
	@echo "📚 Generating documentation..."
	go doc -all > docs/api-reference.txt
	@echo "✅ Documentation generated: docs/api-reference.txt"

# Profiling Commands
profile-cpu: ## CPU profiling
	@echo "📊 Running CPU profiling..."
	go run -cpuprofile=cpu.prof cmd/server/main.go
	go tool pprof cpu.prof

profile-mem: ## Memory profiling
	@echo "💾 Running memory profiling..."
	go run -memprofile=mem.prof cmd/server/main.go
	go tool pprof mem.prof

# Release Commands
release-test: ## Run release tests
	@echo "🧪 Running release tests..."
	test -z "$$(git status --porcelain)" || (echo "Working directory not clean" && exit 1)
	go test -v -race -short ./...
	@echo "✅ Release tests passed"

release-build: ## Build for release
	@echo "🏗️  Building for release..."
	CGO_ENABLED=0 GOOS=linux go build -o bin/server-linux cmd/server/main.go
	CGO_ENABLED=0 GOOS=darwin go build -o bin/server-darwin cmd/server/main.go
	CGO_ENABLED=0 GOOS=windows go build -o bin/server-windows.exe cmd/server/main.go
	@echo "✅ Release builds complete"

# Quick Development
quick-dev: dev-setup dev ## Quick development setup and start