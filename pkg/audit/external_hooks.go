package audit

import (
	"go.uber.org/zap"

	"github.com/captain-corgi/multi-agent-system/pkg/config"
)

// ExternalAuditHelper provides audit functionality for external API calls
type ExternalAuditHelper struct {
	recorder AuditRecorder
	config   *config.AuditConfig
	logger   *zap.Logger
}

// NewExternalAuditHelper creates a new external audit helper
func NewExternalAuditHelper(recorder AuditRecorder, config *config.AuditConfig, logger *zap.Logger) *ExternalAuditHelper {
	return &ExternalAuditHelper{
		recorder: recorder,
		config:   config,
		logger:   logger,
	}
}

// RecordJiraCall records a Jira API call
func (h *ExternalAuditHelper) RecordJiraCall(action string, success bool, metadata map[string]interface{}) {
	if !h.config.Enabled || !h.config.Categories.External {
		return
	}

	payload := map[string]interface{}{
		"service":  "jira",
		"action":   action,
		"metadata": metadata,
	}

	event := NewExternalAuditEvent("jira", action, success, payload)
	if err := h.recorder.Record(event); err != nil {
		h.logger.Error("Failed to record Jira API call",
			zap.String("action", action),
			zap.Error(err))
	}
}

// RecordSlackCall records a Slack API call
func (h *ExternalAuditHelper) RecordSlackCall(action string, success bool, metadata map[string]interface{}) {
	if !h.config.Enabled || !h.config.Categories.External {
		return
	}

	payload := map[string]interface{}{
		"service":  "slack",
		"action":   action,
		"metadata": metadata,
	}

	event := NewExternalAuditEvent("slack", action, success, payload)
	if err := h.recorder.Record(event); err != nil {
		h.logger.Error("Failed to record Slack API call",
			zap.String("action", action),
			zap.Error(err))
	}
}

// RecordGitHubCall records a GitHub API call
func (h *ExternalAuditHelper) RecordGitHubCall(action string, success bool, metadata map[string]interface{}) {
	if !h.config.Enabled || !h.config.Categories.External {
		return
	}

	payload := map[string]interface{}{
		"service":  "github",
		"action":   action,
		"metadata": metadata,
	}

	event := NewExternalAuditEvent("github", action, success, payload)
	if err := h.recorder.Record(event); err != nil {
		h.logger.Error("Failed to record GitHub API call",
			zap.String("action", action),
			zap.Error(err))
	}
}

// RecordGenericExternalCall records a generic external API call
func (h *ExternalAuditHelper) RecordGenericExternalCall(service, action string, success bool, metadata map[string]interface{}) {
	if !h.config.Enabled || !h.config.Categories.External {
		return
	}

	payload := map[string]interface{}{
		"service":  service,
		"action":   action,
		"metadata": metadata,
	}

	event := NewExternalAuditEvent(service, action, success, payload)
	if err := h.recorder.Record(event); err != nil {
		h.logger.Error("Failed to record external API call",
			zap.String("service", service),
			zap.String("action", action),
			zap.Error(err))
	}
}

// RecordRateLimitHit records when an external API rate limit is hit
func (h *ExternalAuditHelper) RecordRateLimitHit(service string, metadata map[string]interface{}) {
	if !h.config.Enabled || !h.config.Categories.External {
		return
	}

	payload := map[string]interface{}{
		"service":  service,
		"action":   "rate_limit_hit",
		"metadata": metadata,
	}

	event := NewExternalAuditEvent(service, "rate_limit_hit", false, payload)
	if err := h.recorder.Record(event); err != nil {
		h.logger.Error("Failed to record rate limit hit",
			zap.String("service", service),
			zap.Error(err))
	}
}

// RecordAuthenticationFailure records when external API authentication fails
func (h *ExternalAuditHelper) RecordAuthenticationFailure(service string, metadata map[string]interface{}) {
	if !h.config.Enabled || !h.config.Categories.External {
		return
	}

	payload := map[string]interface{}{
		"service":  service,
		"action":   "auth_failure",
		"metadata": metadata,
	}

	event := NewExternalAuditEvent(service, "auth_failure", false, payload)
	if err := h.recorder.Record(event); err != nil {
		h.logger.Error("Failed to record authentication failure",
			zap.String("service", service),
			zap.Error(err))
	}
}
