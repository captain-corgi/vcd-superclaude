# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Status

This is a documentation-driven project for a hierarchical multi-agent system built with Go. The project currently exists as comprehensive documentation but has not been fully implemented yet. All source files need to be created according to the detailed specifications in the docs/ directory.

## Architecture Overview

The system is designed as a hierarchical multi-agent system with the following components:

### Core Agents
- **Planner Agent**: Analyzes user requests and creates execution plans
- **Executor Agent**: Executes tasks and interacts with external APIs (Jira, Slack, GitHub)
- **Evaluator Agent**: Validates results and generates final reports

### Communication Layer
- **tRPC-A2A Protocol**: Agent-to-agent communication using tRPC
- **Genkit Streaming**: Real-time progress updates
- **WebSocket**: Bidirectional real-time communication

### External Integrations
- **Jira API**: Project management data collection
- **Slack API**: Team communication data collection
- **GitHub API**: Development workflow data collection

### Data Layer
- **Database**: PostgreSQL with connection pooling
- **Cache**: Redis for data caching
- **Storage**: Message persistence for audit trails

## Technology Stack

- **Go 1.21+**: Primary programming language
- **Echo**: High-performance web framework
- **Genkit**: AI framework with streaming capabilities
- **tRPC-A2A-Go**: Agent-to-agent communication protocol
- **Gorilla/WebSocket**: Real-time communication
- **Zap**: Structured JSON logging
- **Docker**: Containerization and deployment

## Project Structure

```
multi-agent-system/
├── cmd/
│   └── server/
│       └── main.go                    # Application entry point
├── internal/
│   ├── agents/
│   │   ├── planner/                   # Planner agent implementation
│   │   ├── executor/                  # Executor agent implementation
│   │   └── evaluator/                 # Evaluator agent implementation
│   ├── communication/
│   │   ├── trpc/                      # tRPC-A2A protocol implementation
│   │   └── streaming/                 # Genkit streaming implementation
│   ├── tools/
│   │   ├── jira/                      # Jira API client
│   │   ├── slack/                     # Slack API client
│   │   └── github/                   # GitHub API client
│   └── web/
│       ├── handlers/                  # HTTP handlers and WebSocket
│       └── static/                    # Static web assets
├── pkg/
│   ├── models/                        # Data models and schemas
│   ├── storage/                       # Database operations
│   ├── config/                        # Configuration management
│   ├── logger/                        # Logging utilities
│   └── metrics/                       # Application metrics
├── configs/
│   ├── app.yaml                       # Application configuration
│   ├── database.yaml                  # Database configuration
│   └── logging.yaml                   # Logging configuration
├── docker/
│   ├── Dockerfile                     # Production Docker image
│   ├── docker-compose.yml             # Production compose
│   └── docker-compose.dev.yml         # Development compose
├── docs/                             # Comprehensive documentation
├── scripts/                          # Utility scripts
├── .env.example                      # Environment variables template
├── Makefile                          # Build and development commands
├── go.mod                            # Go module definition
└── go.sum                            # Go module checksums
```

## Development Commands

### Project Setup
```bash
# Clone and initialize (from scratch)
git clone <repository-url>
cd multi-agent-system
git checkout develop

# Initialize Go modules
go mod init github.com/<username>/multi-agent-system
go mod tidy

# Setup development environment
make dev-setup

# Copy environment configuration
cp .env.example .env
```

### Building and Running
```bash
# Build the application
make build

# Run the application
make run

# Run with Docker Compose (development)
make dev

# Run with Docker Compose (production)
make docker-up
```

### Testing and Quality
```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Lint code
make lint

# Run security scan
make security

# Run all quality checks
make check
```

### Database Operations
```bash
# Run database migrations up
make migrate-up

# Run database migrations down
make migrate-down

# View database logs
make db-logs
```

### Docker Operations
```bash
# Build Docker image
make docker-build

# Start Docker Compose
make docker-up

# Stop Docker Compose
make docker-down

# View Docker logs
make docker-logs
```

## Configuration

### Environment Variables (.env)
Key environment variables include:
```bash
APP_PORT=8080
DATABASE_URL=postgres://postgres:password@localhost:5432/multiagent?sslmode=disable
REDIS_URL=redis://localhost:6379

# API Credentials
JIRA_CLIENT_ID=your-client-id
JIRA_CLIENT_SECRET=your-client-secret
SLACK_CLIENT_ID=your-client-id
SLACK_CLIENT_SECRET=your-client-secret
GITHUB_TOKEN=your-personal-access-token
```

### Configuration Files (configs/)
- **app.yaml**: Application settings, agent configurations, timeouts
- **database.yaml**: Database connection settings, pooling, migrations
- **logging.yaml**: Log levels, output formats, structured logging

## Implementation Status

Based on the documentation, the project is in planning phase with comprehensive specs:

### Completed Documentation
- ✅ System architecture and workflows
- ✅ Agent specifications and responsibilities
- ✅ API integration patterns
- ✅ Deployment strategies
- ✅ Testing methodologies
- ✅ Configuration management

### To Be Implemented
- ⏳ All Go source code (cmd/, internal/, pkg/)
- ⏳ Go modules (go.mod, go.sum)
- ⏳ Docker configurations
- ⏳ Build automation (Makefile)
- ⏳ Configuration files

## Development Guidelines

### Code Structure
- Follow standard Go project layout (cmd/internal/pkg)
- Use clean architecture principles with separation of concerns
- Implement comprehensive error handling and logging
- Follow Go naming conventions and idioms

### Agent Development
- Each agent implements a common Agent interface
- Agents communicate via tRPC-A2A protocol
- Use structured logging for debugging agent interactions
- Implement proper timeout and retry logic

### API Integration
- Use idiomatic Go patterns for external API clients
- Implement authentication and rate limiting
- Add proper error handling and retries
- Cache responses where appropriate

### Testing
- Unit tests for individual components
- Integration tests for agent workflows
- E2E tests for complete user scenarios
- Performance tests for response times

## Key Resources

### Documentation Files
- `docs/overview.md`: System architecture and vision
- `docs/agents.md`: Detailed agent specifications
- `docs/workflow-implementation.md`: Implementation guide
- `docs/api-integration.md`: External API integration patterns
- `docs/deployment.md`: Deployment and DevOps strategies
- `docs/tasks.md`: Task list and implementation roadmap

### Development References
- Use Echo framework for HTTP/WebSocket server
- Implement Genkit streaming for real-time updates
- Configure tRPC-A2A for agent communication
- Use Zap for structured JSON logging
- Setup PostgreSQL with connection pooling
- Configure Redis for caching

## Important Notes

- This is a learning prototype for demonstrating modern Go patterns
- Focus on streaming capabilities, agent coordination, and API integration
- Follow the detailed implementation guides in docs/
- Test thoroughly before deployment
- Monitor performance and error rates closely