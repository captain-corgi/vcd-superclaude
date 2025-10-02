<!-- d1baf548-7401-4e3a-9858-f0a38faf01fa 8bd1fa9b-355c-401b-9291-f1eb1e8528c5 -->
### Datadog Logging Standards — Adoption Plan

Implement Datadog-compatible structured logging across the app, with APM correlation and consistent field naming. Keep behavior behind config flags and preserve existing Zap usage.

### What “Datadog standard” means (concise)

- JSON logs with reserved fields:
- `timestamp` (RFC3339), `status` (debug|info|warn|error|critical|alert|emergency), `message`, `service`, `env`, `version`, `host`
- APM correlation fields when tracing is active: `dd.trace_id`, `dd.span_id`
- Error fields: `error.kind`, `error.message`, `error.stack`
- HTTP/access fields: `http.method`, `http.url`, `http.route`, `http.status_code`, `network.client.ip`, `user_agent`, `request_id`, latency in ms: `duration` (or `duration_ms`)
- User fields: `usr.id`, `usr.email` (if available)

### Changes to implement

1) Config and flags

- Extend `configs/app.yaml` and `pkg/config/config.go`:
- `logging.datadog.enabled` (bool)
- `logging.datadog.inject_apm` (bool, default true)
- Optional overrides for `service`, `env`, `version` (fallback to `app.name`, `DD_ENV`, `DD_VERSION`)

2) Zap encoder aligned to Datadog

- Add `pkg/logger/datadog.go` to expose a Datadog encoder config:
- Map Zap encoder keys: `timeKey: "timestamp"`, `levelKey: "status"` with lowercase level encoder, `messageKey: "message"`, `nameKey: "logger"`, `callerKey: "logger.source"`, `stacktraceKey: "error.stack"`.
- Add a core wrapper that injects constant fields: `service`, `env`, `version`, `host` (resolve host at startup).
- Provide `logger.NewDatadogLogger(cfg LoggingConfig, app AppConfig)` returning `*logger.Logger`.

3) APM correlation (trace/span IDs)

- Integrate `gopkg.in/DataDog/dd-trace-go.v1` tracing with log injection enabled (configurable):
- Start tracer in `cmd/server/main.go` when `logging.datadog.inject_apm` is true; set env vars or tracer options.
- Use Datadog zap integration to inject `dd.trace_id` and `dd.span_id` (Datadog contrib for Zap) or add a small core that reads IDs from context using `dd-trace-go` APIs and attaches them.
- Ensure graceful shutdown of tracer.

4) Access log middleware (Datadog fields)

- Update `internal/web/middleware.go` `AccessLogMiddleware` to emit Datadog field names:
- `http.method`, `http.url`, `http.route` (when available from Echo), `http.status_code`, `network.client.ip` (use `c.RealIP()`), `user_agent`, `request_id`, `duration_ms`.
- Keep existing audit middleware; add `usr.id`, `usr.email` to logs when present in headers.

5) Error logging mapping

- When logging errors (server startup/shutdown, request errors), add:
- `status: "error"`, `error.kind` (type), `error.message` (err.Error()), and `error.stack` (from Zap stacktraces) for fatal/error paths.

6) Audit logs enrichment

- When recording audit-relevant log lines, add Datadog-friendly keys where applicable (e.g., `usr.id`, `http.method`, `resource`, `request_id`).
- Keep database-stored audit events unchanged; this task focuses on shipped logs format.

7) Docker/Agent integration (optional but recommended)

- Document how to run Datadog Agent sidecar in `docker/docker-compose*.yml` with:
- Container stdout collection, `DD_SERVICE`, `DD_ENV`, `DD_VERSION`, `DD_LOGS_INJECTION=true`
- Set source to `go` or `multi-agent-system` and enable logs

8) Tests

- Unit tests for encoder: ensure keys exist and `status` lowercased.
- Middleware test: simulate request; assert `http.*`, `network.client.ip`, `duration_ms`, `request_id` fields.
- If tracer available in tests, assert `dd.trace_id`, `dd.span_id` presence when a span is active (can be a separate build tag test or mocked core).

9) Docs

- Update `docs/audit.md` and `docs/overview.md` with short “Datadog logging” section: fields used, how to enable, correlation, example log lines.

### File-level edits (targeted)

- `pkg/config/config.go`, `configs/app.yaml`: add `logging.datadog` block (enabled, inject_apm, service/env/version overrides).
- `pkg/logger/datadog.go`: new encoder/core helpers + `NewDatadogLogger`.
- `cmd/server/main.go`: initialize Datadog logger when enabled; start/stop tracer if inject_apm.
- `internal/web/middleware.go`: adjust fields to Datadog keys; add route and duration in ms; include usr fields if present.
- `docs/audit.md`: add Datadog logging conventions and examples.
- `docker` folder: optional Agent compose snippet in docs; not required to change runtime by default.

### Rollout & compatibility

- Feature-flagged; default off. When enabled, JSON keys switch to Datadog schema.
- No changes to log destinations (stdout remains), so existing infra keeps working.

### Todos

- [ ] Add logging.datadog config structs and YAML
- [ ] Implement Datadog zap encoder and field injector
- [ ] Wire Datadog logger in main when enabled
- [ ] Add optional dd-trace-go log injection
- [ ] Update AccessLogMiddleware to Datadog field names
- [ ] Add unit tests for encoder and middleware
- [ ] Update docs with usage and examples
- [ ] (Optional) Provide Datadog Agent compose snippet

### To-dos

- [ ] Add logging.datadog config and YAML keys
- [ ] Implement Datadog Zap encoder and field injector
- [ ] Wire Datadog logger and optional tracer in main.go
- [ ] Emit Datadog fields in AccessLogMiddleware
- [ ] Add unit tests for encoder and middleware
- [ ] Document Datadog logging config and examples