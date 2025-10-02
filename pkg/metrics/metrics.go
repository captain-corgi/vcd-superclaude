package metrics

import (
	"sync"
	"time"
)

// Metrics represents application metrics
type Metrics struct {
	mu sync.RWMutex

	// Audit metrics
	AuditQueued  int64         `json:"audit_queued"`
	AuditWritten int64         `json:"audit_written"`
	AuditDropped int64         `json:"audit_dropped"`
	AuditLatency time.Duration `json:"audit_latency"`

	// HTTP metrics
	HTTPRequestsTotal int64 `json:"http_requests_total"`
	HTTPErrorsTotal   int64 `json:"http_errors_total"`

	// Agent metrics
	AgentTasksTotal   int64 `json:"agent_tasks_total"`
	AgentTasksSuccess int64 `json:"agent_tasks_success"`
	AgentTasksFailed  int64 `json:"agent_tasks_failed"`

	// External API metrics
	ExternalCallsTotal   int64 `json:"external_calls_total"`
	ExternalCallsSuccess int64 `json:"external_calls_success"`
	ExternalCallsFailed  int64 `json:"external_calls_failed"`
	ExternalRateLimitHit int64 `json:"external_rate_limit_hit"`
}

// Global metrics instance
var globalMetrics = &Metrics{}

// GetMetrics returns the current metrics
func GetMetrics() Metrics {
	globalMetrics.mu.RLock()
	defer globalMetrics.mu.RUnlock()
	return *globalMetrics
}

// IncrementAuditQueued increments the audit queued counter
func IncrementAuditQueued() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.AuditQueued++
}

// IncrementAuditWritten increments the audit written counter
func IncrementAuditWritten() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.AuditWritten++
}

// IncrementAuditDropped increments the audit dropped counter
func IncrementAuditDropped() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.AuditDropped++
}

// UpdateAuditLatency updates the audit latency metric
func UpdateAuditLatency(latency time.Duration) {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.AuditLatency = latency
}

// IncrementHTTPRequests increments the HTTP requests counter
func IncrementHTTPRequests() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.HTTPRequestsTotal++
}

// IncrementHTTPErrors increments the HTTP errors counter
func IncrementHTTPErrors() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.HTTPErrorsTotal++
}

// IncrementAgentTasks increments the agent tasks counter
func IncrementAgentTasks() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.AgentTasksTotal++
}

// IncrementAgentTasksSuccess increments the successful agent tasks counter
func IncrementAgentTasksSuccess() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.AgentTasksSuccess++
}

// IncrementAgentTasksFailed increments the failed agent tasks counter
func IncrementAgentTasksFailed() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.AgentTasksFailed++
}

// IncrementExternalCalls increments the external API calls counter
func IncrementExternalCalls() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.ExternalCallsTotal++
}

// IncrementExternalCallsSuccess increments the successful external API calls counter
func IncrementExternalCallsSuccess() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.ExternalCallsSuccess++
}

// IncrementExternalCallsFailed increments the failed external API calls counter
func IncrementExternalCallsFailed() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.ExternalCallsFailed++
}

// IncrementExternalRateLimitHit increments the external rate limit hit counter
func IncrementExternalRateLimitHit() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	globalMetrics.ExternalRateLimitHit++
}

// Reset resets all metrics to zero
func Reset() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()
	*globalMetrics = Metrics{}
}
