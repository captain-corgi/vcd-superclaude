# AGENTS.md

## Setup commands
- Install deps: `go mod download && go mod tidy`
- Start dev server: `make dev` or `go run cmd/server/main.go`
- Run tests: `make test`
- Run tests with coverage: `make test-coverage`

## Code style
- Go idiomatic code with proper error handling
- Use Go standard formatting: `go fmt ./...`
- Follow Go naming conventions (Pascal for exported, camel for unexported)
- Interface-based design for testability
- Structured logging with Zap

## Dev environment tips
- Use `make dev` to start full development environment with Docker
- Use `make docker-logs` to monitor all service logs
- Run `make check` to run all quality checks (fmt, lint, security)
- Use `make dev-logs` to view development environment logs
- Configure `.env` file from `.env.example` before starting

## Agent development commands
- Build specific agent: `go build ./internal/agents/<agent_name>`
- Test agent: `go test ./internal/agents/<agent_name> -v`
- Run agent integration tests: `make test-integration`
- Debug agent: `go run -tags debug cmd/server/main.go`

## Testing instructions
- Find CI plan in `.github/workflows` (if implemented)
- Run `make test` to run all tests
- Run `make test-coverage` for coverage report
- Run `make lint` to check code quality
- Run `make security` to run security scanning
- Fix errors until all suites are green
- Run `make check` after all changes

## Database operations
- Run migrations: `make migrate-up`
- Rollback migrations: `make migrate-down`
- View database logs: `make db-logs`
- Reset database: `make db-reset`

## API integration testing
- Test Jira integration: `make test-jira`
- Test Slack integration: `make test-slack`
- Test GitHub integration: `make test-github`
- Run all API tests: `make test-apis`

## WebSocket testing
- Test WebSocket connection: `make test-websocket`
- Monitor WebSocket messages: `make test-streaming`
- Test real-time updates: `make test-realtime`

## Performance testing
- Run load tests: `make test-load`
- Monitor performance: `make test-perf`
- Check memory usage: `make test-memory`

## Docker operations
- Build Docker image: `make docker-build`
- Start Docker services: `make docker-up`
- Stop Docker services: `make docker-down`
- View Docker logs: `make docker-logs`
- Restart services: `make docker-restart`

## PR instructions
- Title format: `[multi-agent] <Title>` or `[<agent_name>] <Title>`
- Always run `make fmt && make lint && make test` before committing
- Include test coverage for new features
- Update documentation for agent changes
- Run integration tests for multi-agent workflows
- Ensure all Docker services start successfully

## Agent-specific development

### Planner Agent
- Directory: `internal/agents/planner/`
- Focus: Request parsing, plan creation, task coordination
- Test with: `go test ./internal/agents/planner/`
- Debug: `DEBUG=planner go run cmd/server/main.go`

### Executor Agent
- Directory: `internal/agents/executor/`
- Focus: API calls, rate limiting, error handling
- Test with: `go test ./internal/agents/executor/`
- Debug: `DEBUG=executor go run cmd/server/main.go`

### Evaluator Agent
- Directory: `internal/agents/evaluator/`
- Focus: Data validation, report generation, quality checks
- Test with: `go test ./internal/agents/evaluator/`
- Debug: `DEBUG=evaluator go run cmd/server/main.go`

## Communication layer development
- tRPC-A2A: `internal/communication/trpc/`
- Streaming: `internal/communication/streaming/`
- WebSocket: `internal/web/handlers/`
- Test communication: `make test-communication`

## Tool development
- Jira tools: `internal/tools/jira/`
- Slack tools: `internal/tools/slack/`
- GitHub tools: `internal/tools/github/`
- Test tools: `make test-tools`

## Configuration management
- Edit configs/app.yaml for application settings
- Edit configs/database.yaml for database configuration
- Edit configs/logging.yaml for logging configuration
- Validate configuration: `make validate-config`

## Monitoring and debugging
- View application logs: `make logs`
- Monitor health checks: `make health`
- View metrics: `make metrics`
- Debug mode: `DEBUG=true go run cmd/server/main.go`

## Deployment commands
- Build for production: `make build-prod`
- Deploy with Docker: `make deploy`
- Check deployment status: `make status`
- Rollback deployment: `make rollback`

## Common issues and solutions
- Database connection issues: `make migrate-up` then restart
- Port conflicts: Change APP_PORT in .env
- API rate limits: Check EXECUTOR_RATE_LIMIT in configs
- WebSocket issues: Restart services and check logs
- Memory issues: Adjust Go runtime flags in Makefile