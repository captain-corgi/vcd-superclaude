# Multi-Agent System

A hierarchical multi-agent system built with Go for automated data collection and report generation from Jira, Slack, and GitHub.

## 🎯 Project Vision

Create a learning prototype that demonstrates:
- **Genkit streaming capabilities** for real-time communication
- **Tool usage patterns** for external API integration
- **tRPC-A2A protocol** for agent-to-agent communication
- **Hierarchical agent coordination** for complex workflows

## 🏗️ System Architecture

```mermaid
graph TB
    subgraph "User Interface"
        UI[Web Interface<br/>Echo + WebSocket]
    end

    subgraph "Agent Layer"
        PLANNER[Planner Agent<br/>Request Analysis & Planning]
        EXECUTOR[Executor Agent<br/>Task Execution & API Calls]
        EVALUATOR[Evaluator Agent<br/>Data Validation & Reporting]
    end

    subgraph "Communication Layer"
        A2A[tRPC-A2A Protocol<br/>Agent-to-Agent Communication]
        STREAM[Genkit Streaming<br/>Real-time Updates]
    end

    subgraph "External APIs"
        JIRA[Jira API]
        SLACK[Slack API]
        GITHUB[GitHub API]
    end

    subgraph "Data Layer"
        STORAGE[Message Persistence]
        CACHE[Data Caching]
    end

    UI --> PLANNER
    PLANNER --> A2A
    A2A --> EXECUTOR
    EXECUTOR --> A2A
    A2A --> EVALUATOR
    EVALUATOR --> UI

    EXECUTOR --> JIRA
    EXECUTOR --> SLACK
    EXECUTOR --> GITHUB

    A2A --> STORAGE
    EXECUTOR --> CACHE
    STREAM --> UI
```

## 🔄 Workflow Process

```mermaid
sequenceDiagram
    participant User as User
    participant UI as Web Interface
    participant Planner as Planner Agent
    participant Executor as Executor Agent
    participant Evaluator as Evaluator Agent
    participant APIs as External APIs

    User->>UI: Submit data collection request
    UI->>Planner: Forward request for analysis
    Planner->>Planner: Parse request and create plan
    Planner->>Executor: Execute plan via tRPC-A2A
    Executor->>APIs: Fetch data from Jira/Slack/GitHub
    APIs-->>Executor: Return data
    Executor->>Evaluator: Submit collected data
    Evaluator->>Evaluator: Validate and generate report
    Evaluator-->>UI: Return final report
    UI-->>User: Display results with real-time updates
```

## 🤖 Agent Hierarchy

### Planner Agent
- **Role**: Analyze user requests and create execution plans
- **Responsibilities**:
  - Parse natural language requests
  - Determine required data sources
  - Create task execution sequence
  - Coordinate with executor agent

### Executor Agent
- **Role**: Execute tasks and interact with external APIs
- **Responsibilities**:
  - Implement Jira, Slack, GitHub API clients
  - Handle authentication and rate limiting
  - Execute data collection tasks
  - Manage error handling and retries

### Evaluator Agent
- **Role**: Validate results and generate final reports
- **Responsibilities**:
  - Validate data completeness and quality
  - Generate structured reports
  - Apply data normalization
  - Ensure output consistency

## 🛠️ Technology Stack

### Core Technologies
- **Go**: Primary programming language
- **Genkit**: AI framework with streaming capabilities
- **Echo**: High-performance web framework
- **tRPC-A2A-Go**: Agent-to-agent communication protocol

### Communication & Real-time
- **Gorilla/WebSocket**: Real-time bidirectional communication
- **Genkit Streaming**: Real-time progress updates
- **Message Persistence**: SQLite/PostgreSQL for message storage

### External Integrations
- **Jira API**: Project management data
- **Slack API**: Team communication data
- **GitHub API**: Development workflow data

### Deployment
- **Docker**: Containerization
- **Docker Compose**: Local development environment
- **Go Modules**: Dependency management

## 📊 Key Features

### Real-time Communication
- WebSocket connections for live updates
- Genkit streaming for agent progress
- tRPC-A2A for structured agent communication

### External API Integration
- Configurable API clients
- Authentication management
- Rate limiting and error handling
- Data caching strategies

### Workflow Management
- Hierarchical task execution
- Message persistence for audit trails
- Error recovery mechanisms
- Progress tracking and reporting

### Web Interface
- Request submission forms
- Real-time progress visualization
- Report display and export
- Configuration management

## 🎯 Learning Objectives

### Genkit Integration
- Streaming capabilities implementation
- Tool system usage patterns
- AI model integration strategies
- Real-time communication patterns

### Multi-Agent Systems
- Hierarchical agent design
- Agent-to-agent communication protocols
- Task coordination and delegation
- Error handling and recovery

### External API Patterns
- RESTful API integration
- Authentication and authorization
- Rate limiting and throttling
- Data normalization and processing

### Web Development
- Real-time web applications
- WebSocket integration
- API design and implementation
- Containerized deployment

## 🚀 Getting Started

### Prerequisites
- Go 1.21+
- Docker & Docker Compose
- PostgreSQL (optional, can use SQLite for development)
- Redis (optional, can use in-memory cache for development)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/your-username/multi-agent-system.git
   cd multi-agent-system
   ```

2. **Setup Development Environment**
   ```bash
   # Copy environment variables
   cp .env.example .env

   # Edit environment variables if needed
   nano .env
   ```

3. **Install Dependencies**
   ```bash
   # Download Go modules
   go mod download

   # Tidy dependencies
   go mod tidy
   ```

4. **Run Development Server**
   ```bash
   # Option 1: Run directly with Go
   go run cmd/server/main.go

   # Option 2: Run with Docker Compose
   docker-compose -f docker/docker-compose.dev.yml up -d
   ```

5. **Access Web Interface**
   - Open `http://localhost:8080` in browser
   - Submit data collection requests
   - Monitor real-time progress

## 📝 Project Structure

```
multi-agent-system/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── agents/
│   │   ├── planner/
│   │   ├── executor/
│   │   └── evaluator/
│   ├── communication/
│   │   ├── trpc/
│   │   └── streaming/
│   ├── tools/
│   │   ├── jira/
│   │   ├── slack/
│   │   └── github/
│   └── web/
│       ├── handlers/
│       └── static/
├── pkg/
│   ├── models/
│   ├── storage/
│   └── config/
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
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## 🔄 Development Workflow

### Available Commands

```bash
# Development setup
make dev-setup        # Setup development environment
make dev             # Start development environment with Docker
make dev-logs        # Show development environment logs

# Application commands
make build          # Build the application
make run            # Run the application
make test           # Run tests
make test-coverage  # Run tests with coverage

# Docker commands
make docker-build   # Build Docker image
make docker-up      # Start Docker Compose
make docker-down    # Stop Docker Compose
make docker-logs    # Show Docker Compose logs

# Quality checks
make fmt            # Format code
make lint           # Lint code
make security       # Run security scan
make check          # Run all checks

# Database
make migrate-up     # Run database migrations
make migrate-down   # Rollback database migrations

# Documentation
make docs           # Generate documentation
```

### Environment Configuration

```bash
# .env.example
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
```

## 📈 Success Metrics

### Technical Metrics
- Agent response time < 2 seconds
- API integration success rate > 95%
- WebSocket connection stability
- Memory usage optimization

### Learning Metrics
- Genkit streaming capability demonstration
- tRPC-A2A protocol implementation
- Multi-agent coordination patterns
- External API integration patterns

### User Experience Metrics
- Real-time update responsiveness
- Report generation accuracy
- Error handling clarity
- Interface usability

## 🧪 Testing

The project includes comprehensive testing:

- **Unit Tests**: Individual component testing
- **Integration Tests**: Agent workflow testing
- **Performance Tests**: Response time and resource usage
- **E2E Tests**: Complete user workflow testing

Run tests with:
```bash
make test           # Run all tests
make test-coverage  # Run with coverage report
```

## 🔧 Configuration

### Application Configuration
The application uses YAML configuration files in `configs/`:

- `app.yaml`: Application settings
- `database.yaml`: Database and Redis configuration
- `logging.yaml`: Logging configuration

### API Configuration
External API configuration supports:

- **Authentication**: OAuth2, API keys, bearer tokens
- **Rate Limiting**: Configurable per-API limits
- **Caching**: TTL-based caching strategies
- **Retries**: Exponential backoff retry logic

## 📊 Monitoring & Health Checks

### Health Endpoints
- `GET /health` - Basic health check
- `GET /health/detailed` - Detailed health status
- `GET /ready` - Readiness check
- `GET /live` - Liveness check
- `GET /metrics` - System metrics

### Logging
Structured logging with Zap logger:
- JSON format for machine parsing
- Multiple log levels (debug, info, warn, error)
- Request/response tracing
- Error context and stack traces

## 🚨 Troubleshooting

### Common Issues

#### Database Connection Issues
```bash
# Check PostgreSQL logs
docker-compose logs postgres

# Test database connection
docker-compose exec postgres psql -U postgres -d multiagent -c "SELECT 1"
```

#### Redis Connection Issues
```bash
# Check Redis logs
docker-compose logs redis

# Test Redis connection
docker-compose exec redis redis-cli ping
```

#### Application Issues
```bash
# Check application logs
docker-compose logs app

# Restart application
docker-compose restart app

# Debug mode
docker-compose -f docker/docker-compose.dev.yml up --build
```

### Performance Optimization

#### Database Optimization
- Configure connection pooling
- Optimize query performance
- Implement proper indexing

#### Redis Optimization
- Set memory limits
- Configure eviction policies
- Use appropriate data structures

#### Application Optimization
- Configure Go runtime parameters
- Implement proper caching strategies
- Optimize API call patterns

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite
6. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

This project demonstrates modern Go development practices including:
- Clean architecture and separation of concerns
- Comprehensive error handling
- Real-time communication patterns
- External API integration best practices
- Containerized deployment strategies

## 📚 Additional Resources

- [Go Documentation](https://go.dev/doc/)
- [Echo Framework Documentation](https://echo.labstack.com/docs/)
- [Docker Documentation](https://docs.docker.com/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)