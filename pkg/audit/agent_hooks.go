package audit

import (
	"go.uber.org/zap"

	"github.com/captain-corgi/multi-agent-system/pkg/config"
)

// AgentAuditHelper provides audit functionality for agents
type AgentAuditHelper struct {
	recorder AuditRecorder
	config   *config.AuditConfig
	logger   *zap.Logger
}

// NewAgentAuditHelper creates a new agent audit helper
func NewAgentAuditHelper(recorder AuditRecorder, config *config.AuditConfig, logger *zap.Logger) *AgentAuditHelper {
	return &AgentAuditHelper{
		recorder: recorder,
		config:   config,
		logger:   logger,
	}
}

// RecordAgentStart records when an agent starts
func (h *AgentAuditHelper) RecordAgentStart(agentID string, metadata map[string]interface{}) {
	if !h.config.Enabled || !h.config.Categories.Agent {
		return
	}

	payload := map[string]interface{}{
		"agent_id": agentID,
		"action":   "agent_start",
		"metadata": metadata,
	}

	event := NewAgentAuditEvent(agentID, "agent_start", true, payload)
	if err := h.recorder.Record(event); err != nil {
		h.logger.Error("Failed to record agent start event",
			zap.String("agent_id", agentID),
			zap.Error(err))
	}
}

// RecordAgentStop records when an agent stops
func (h *AgentAuditHelper) RecordAgentStop(agentID string, metadata map[string]interface{}) {
	if !h.config.Enabled || !h.config.Categories.Agent {
		return
	}

	payload := map[string]interface{}{
		"agent_id": agentID,
		"action":   "agent_stop",
		"metadata": metadata,
	}

	event := NewAgentAuditEvent(agentID, "agent_stop", true, payload)
	if err := h.recorder.Record(event); err != nil {
		h.logger.Error("Failed to record agent stop event",
			zap.String("agent_id", agentID),
			zap.Error(err))
	}
}

// RecordAgentTask records when an agent executes a task
func (h *AgentAuditHelper) RecordAgentTask(agentID, taskType string, success bool, metadata map[string]interface{}) {
	if !h.config.Enabled || !h.config.Categories.Agent {
		return
	}

	payload := map[string]interface{}{
		"agent_id":  agentID,
		"task_type": taskType,
		"action":    "task_execution",
		"metadata":  metadata,
	}

	event := NewAgentAuditEvent(agentID, "task_execution", success, payload)
	if err := h.recorder.Record(event); err != nil {
		h.logger.Error("Failed to record agent task event",
			zap.String("agent_id", agentID),
			zap.String("task_type", taskType),
			zap.Error(err))
	}
}

// RecordAgentError records when an agent encounters an error
func (h *AgentAuditHelper) RecordAgentError(agentID string, errorMsg string, metadata map[string]interface{}) {
	if !h.config.Enabled || !h.config.Categories.Agent {
		return
	}

	payload := map[string]interface{}{
		"agent_id": agentID,
		"action":   "agent_error",
		"error":    errorMsg,
		"metadata": metadata,
	}

	event := NewAgentAuditEvent(agentID, "agent_error", false, payload)
	if err := h.recorder.Record(event); err != nil {
		h.logger.Error("Failed to record agent error event",
			zap.String("agent_id", agentID),
			zap.Error(err))
	}
}

// RecordAgentCommunication records agent-to-agent communication
func (h *AgentAuditHelper) RecordAgentCommunication(fromAgent, toAgent, messageType string, success bool, metadata map[string]interface{}) {
	if !h.config.Enabled || !h.config.Categories.Agent {
		return
	}

	payload := map[string]interface{}{
		"from_agent":   fromAgent,
		"to_agent":     toAgent,
		"message_type": messageType,
		"action":       "agent_communication",
		"metadata":     metadata,
	}

	event := NewAgentAuditEvent(fromAgent, "agent_communication", success, payload)
	if err := h.recorder.Record(event); err != nil {
		h.logger.Error("Failed to record agent communication event",
			zap.String("from_agent", fromAgent),
			zap.String("to_agent", toAgent),
			zap.Error(err))
	}
}
