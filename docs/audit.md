# Audit and Logging System

This document describes the comprehensive audit and logging system implemented in the multi-agent system.

## Overview

The audit and logging system provides:
- Structured logging with configurable levels and outputs
- Comprehensive audit trail for HTTP requests, agent actions, and external API calls
- Automatic data redaction for sensitive information
- Configurable retention policies
- Real-time metrics and monitoring
- Query APIs for audit data

## Configuration

### Logging Configuration

```yaml
logging:
  level: "info"           # debug, info, warn, error
  format: "json"          # json, console
  output: ["stdout"]      # stdout, file
  file: ""                # path to log file (if using file output)
  development: true       # development mode
  datadog:
    enabled: false        # Enable Datadog-compatible logging
    inject_apm: true      # Inject APM correlation fields
    service: ""           # Override service name (defaults to app.name)
    env: ""               # Override environment (defaults to DD_ENV)
    version: ""           # Override version (defaults to DD_VERSION)
```

### Datadog Logging Standards

When `logging.datadog.enabled` is set to `true`, the system uses Datadog-compatible field names and structure:

#### Reserved Fields
- `timestamp` - RFC3339 timestamp
- `status` - Log level (debug, info, warn, error, critical, alert, emergency)
- `message` - Log message
- `service` - Service name
- `env` - Environment
- `version` - Service version
- `host` - Hostname

#### APM Correlation Fields (when `inject_apm: true`)
- `dd.trace_id` - Datadog trace ID
- `dd.span_id` - Datadog span ID

#### Error Fields
- `error.kind` - Error type
- `error.message` - Error message
- `error.stack` - Stack trace

#### HTTP/Access Fields
- `http.method` - HTTP method
- `http.url` - Request URL
- `http.route` - Route pattern
- `http.status_code` - Response status code
- `network.client.ip` - Client IP address
- `user_agent` - User agent string
- `request_id` - Request ID
- `duration_ms` - Request duration in milliseconds

#### User Fields (when available)
- `usr.id` - User ID
- `usr.email` - User email

#### Example Log Output

```json
{
  "timestamp": "2024-01-15T10:30:45Z",
  "status": "info",
  "message": "HTTP request",
  "service": "multi-agent-system",
  "env": "production",
  "version": "1.0.0",
  "host": "server-01",
  "dd.trace_id": 1234567890123456789,
  "dd.span_id": 9876543210987654321,
  "http.method": "GET",
  "http.url": "/api/agents",
  "http.route": "/api/agents",
  "http.status_code": 200,
  "network.client.ip": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "request_id": "evt_abc123def456",
  "duration_ms": 45,
  "usr.id": "user123",
  "usr.email": "user@example.com"
}
```

### Audit Configuration

```yaml
audit:
  enabled: true                    # Enable/disable audit system
  expose_endpoints: true          # Expose audit query endpoints
  categories:
    http: true                    # Audit HTTP requests
    agent: true                   # Audit agent actions
    external: true                 # Audit external API calls
  retention_days: 90              # Days to retain audit events
  redact:
    deny_keys:                    # Keys to redact from payloads
      - "password"
      - "token"
      - "authorization"
      - "cookie"
      - "email"
  file_sink:
    enabled: false                # Enable file sink for audit events
    path: logs/audit.log          # Audit log file path
    max_size_mb: 100              # Max file size before rotation
    max_backups: 5                # Number of backup files
    max_age_days: 30              # Max age of backup files
    compress: true                # Compress backup files
```

## Audit Events

### Event Structure

```json
{
  "id": 123,
  "event_id": "abc123...",
  "category": "http|agent|external",
  "action": "POST /api/requests",
  "actor_id": "user123",
  "actor_email": "user@example.com",
  "source_ip": "192.168.1.1",
  "resource": "/api/requests",
  "request_id": "req_123",
  "success": true,
  "payload": {...},
  "redacted": false,
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Event Categories

#### HTTP Events
- **Category**: `http`
- **Actions**: HTTP method + path (e.g., `POST /api/requests`)
- **Triggered**: On mutating HTTP methods (POST, PUT, PATCH, DELETE)
- **Data**: Request body, response status, actor info, source IP

#### Agent Events
- **Category**: `agent`
- **Actions**: `agent_start`, `agent_stop`, `task_execution`, `agent_error`, `agent_communication`
- **Triggered**: On agent lifecycle events and task executions
- **Data**: Agent ID, task type, success status, metadata

#### External API Events
- **Category**: `external`
- **Actions**: Service-specific actions (e.g., `jira_issue_create`, `slack_message_send`)
- **Triggered**: On external API calls
- **Data**: Service name, action, success status, metadata

## Data Redaction

The system automatically redacts sensitive data from audit payloads based on configurable deny keys:

- **Case-insensitive matching**: `password`, `PASSWORD`, `Password` all match
- **Nested object support**: Redacts keys in nested JSON objects
- **Array support**: Redacts keys in array elements
- **Default keys**: `password`, `token`, `authorization`, `cookie`, `email`

### Example Redaction

**Input:**
```json
{
  "username": "john",
  "password": "secret123",
  "user": {
    "email": "john@example.com",
    "token": "abc123"
  }
}
```

**Output:**
```json
{
  "username": "john",
  "password": "[REDACTED]",
  "user": {
    "email": "[REDACTED]",
    "token": "[REDACTED]"
  }
}
```

## API Endpoints

### List Audit Events

```http
GET /api/audit/events
```

**Query Parameters:**
- `category`: Filter by event category (http, agent, external)
- `actor_id`: Filter by actor ID
- `action`: Filter by action
- `resource`: Filter by resource
- `success`: Filter by success status (true/false)
- `start_date`: Start date (RFC3339 format)
- `end_date`: End date (RFC3339 format)
- `page`: Page number (default: 1)
- `page_size`: Page size (default: 10, max: 100)

**Response:**
```json
{
  "events": [...],
  "page": {
    "page": 1,
    "page_size": 10,
    "total": 100,
    "total_pages": 10
  }
}
```

### Get Audit Event

```http
GET /api/audit/events/:id
```

**Parameters:**
- `id`: Event ID (numeric) or event_id (string)

**Response:**
```json
{
  "id": 123,
  "event_id": "abc123...",
  "category": "http",
  "action": "POST /api/requests",
  "actor_id": "user123",
  "actor_email": "user@example.com",
  "source_ip": "192.168.1.1",
  "resource": "/api/requests",
  "request_id": "req_123",
  "success": true,
  "payload": {...},
  "redacted": false,
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Metrics Endpoint

```http
GET /api/metrics
```

**Response:**
```json
{
  "application": {
    "audit_queued": 100,
    "audit_written": 95,
    "audit_dropped": 5,
    "audit_latency": "10ms",
    "http_requests_total": 1000,
    "http_errors_total": 50,
    "agent_tasks_total": 500,
    "agent_tasks_success": 450,
    "agent_tasks_failed": 50,
    "external_calls_total": 200,
    "external_calls_success": 180,
    "external_calls_failed": 20,
    "external_rate_limit_hit": 5
  },
  "server": {
    "uptime": "1h30m",
    "requests_total": 1000,
    "errors_total": 50,
    "active_connections": 10,
    "agent_status": {...},
    "system_metrics": {...}
  },
  "audit": {
    "queued": 10,
    "written": 95,
    "dropped": 5,
    "latency": "10ms"
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

## Usage Examples

### Agent Audit Helper

```go
// Create audit helper
auditHelper := audit.NewAgentAuditHelper(recorder, config, logger)

// Record agent start
auditHelper.RecordAgentStart("planner", map[string]interface{}{
    "version": "1.0.0",
    "config": {...},
})

// Record task execution
auditHelper.RecordAgentTask("executor", "data_collection", true, map[string]interface{}{
    "task_id": "task_123",
    "duration": "5s",
})

// Record agent error
auditHelper.RecordAgentError("evaluator", "validation failed", map[string]interface{}{
    "error_code": "VALIDATION_ERROR",
    "details": {...},
})
```

### External API Audit Helper

```go
// Create external audit helper
externalHelper := audit.NewExternalAuditHelper(recorder, config, logger)

// Record Jira API call
externalHelper.RecordJiraCall("create_issue", true, map[string]interface{}{
    "issue_key": "PROJ-123",
    "project": "PROJ",
})

// Record rate limit hit
externalHelper.RecordRateLimitHit("slack", map[string]interface{}{
    "retry_after": 60,
    "limit": 100,
})
```

## Retention and Cleanup

The system includes an automatic retention job that:
- Runs daily to clean up old audit events
- Removes events older than `retention_days`
- Logs cleanup statistics
- Can be disabled by setting `retention_days` to 0

## Performance Considerations

- **Async Processing**: Audit events are processed asynchronously to avoid blocking main operations
- **Buffering**: Events are buffered in memory before writing to storage
- **Backpressure**: If the buffer is full, events are dropped with metrics tracking
- **Batching**: Multiple events can be processed in batches for efficiency
- **Indexing**: Database indexes optimize query performance

## Security Considerations

- **Data Redaction**: Sensitive data is automatically redacted
- **Access Control**: Audit endpoints should be protected with authentication
- **Data Retention**: Old audit data is automatically purged
- **Log Rotation**: File logs are rotated to prevent disk space issues

## Monitoring and Alerting

Key metrics to monitor:
- **Audit Drop Rate**: High drop rates indicate performance issues
- **Audit Latency**: High latency indicates storage performance issues
- **Error Rates**: Monitor HTTP, agent, and external API error rates
- **Retention Cleanup**: Monitor cleanup job success and duration

## Troubleshooting

### Common Issues

1. **High Audit Drop Rate**
   - Increase buffer size in recorder configuration
   - Check storage performance
   - Monitor database connection pool

2. **High Audit Latency**
   - Check database performance
   - Monitor storage connection health
   - Consider database optimization

3. **Missing Audit Events**
   - Verify audit is enabled in configuration
   - Check category settings (http, agent, external)
   - Verify middleware is properly configured

4. **Storage Issues**
   - Check database connectivity
   - Verify schema initialization
   - Monitor disk space for file sinks

## Docker and Datadog Agent Integration

### Datadog Agent Setup

To collect logs from the multi-agent system with Datadog Agent, add the following to your `docker-compose.yml`:

```yaml
version: '3.8'
services:
  multi-agent-system:
    image: multi-agent-system:latest
    environment:
      - DD_SERVICE=multi-agent-system
      - DD_ENV=production
      - DD_VERSION=1.0.0
      - DD_LOGS_INJECTION=true
    labels:
      com.datadoghq.ad.logs: '[{"source": "go", "service": "multi-agent-system"}]'
    depends_on:
      - datadog-agent

  datadog-agent:
    image: datadog/agent:latest
    environment:
      - DD_API_KEY=${DD_API_KEY}
      - DD_SITE=datadoghq.com
      - DD_LOGS_ENABLED=true
      - DD_LOGS_CONFIG_CONTAINER_COLLECT_ALL=true
      - DD_PROCESS_AGENT_ENABLED=true
      - DD_APM_ENABLED=true
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - /proc/:/host/proc/:ro
      - /sys/fs/cgroup/:/host/sys/fs/cgroup:ro
    ports:
      - "8126:8126"  # APM
      - "8125:8125/udp"  # DogStatsD
```

### Environment Variables

Set these environment variables for proper Datadog integration:

```bash
# Required for APM correlation
DD_SERVICE=multi-agent-system
DD_ENV=production
DD_VERSION=1.0.0
DD_LOGS_INJECTION=true

# Optional: Override in application config
DD_TRACE_AGENT_URL=http://datadog-agent:8126
DD_LOGS_AGENT_URL=http://datadog-agent:8126
```

### Configuration for Production

Enable Datadog logging in your production configuration:

```yaml
logging:
  level: "info"
  format: "json"
  output: ["stdout"]
  development: false
  datadog:
    enabled: true
    inject_apm: true
    service: "multi-agent-system"
    env: "production"
    version: "1.0.0"
```

### Log Collection

The Datadog Agent will automatically collect logs from the container stdout when configured with the labels above. The logs will be parsed and indexed in Datadog with the proper service, environment, and version tags.

### APM Integration

When `inject_apm: true` is set, the system will:
1. Start the Datadog tracer
2. Inject trace and span IDs into log messages
3. Enable correlation between logs and traces in Datadog

This allows you to:
- View logs associated with specific traces
- Navigate from traces to related logs
- Correlate performance issues with log events

