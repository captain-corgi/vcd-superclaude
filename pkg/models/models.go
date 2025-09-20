package models

// APIRequest represents the external API request structure
type APIRequest struct {
	Type     string                 `json:"type" validate:"required,oneof=create,read,update,delete"`
	Query    string                 `json:"query,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// APIResponse represents the external API response structure
type APIResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Message string                 `json:"message,omitempty"`
}

// AgentStatus represents the status of an agent
type AgentStatus struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	LastActive string `json:"last_active"`
}

// SystemMetrics represents system performance metrics
type SystemMetrics struct {
	Uptime          int64   `json:"uptime"`
	MemoryUsage     int64   `json:"memory_usage"`
	CPUUsage        float64 `json:"cpu_usage"`
	RequestsTotal   int64   `json:"requests_total"`
	ActiveAgents    int     `json:"active_agents"`
	DatabaseStatus  string  `json:"database_status"`
	RedisStatus     string  `json:"redis_status"`
}

// HealthCheck represents health check response
type HealthCheck struct {
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Version   string                 `json:"version"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
}