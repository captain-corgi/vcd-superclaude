package models

import "time"

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
	Uptime         int64   `json:"uptime"`
	MemoryUsage    int64   `json:"memory_usage"`
	CPUUsage       float64 `json:"cpu_usage"`
	RequestsTotal  int64   `json:"requests_total"`
	ActiveAgents   int     `json:"active_agents"`
	DatabaseStatus string  `json:"database_status"`
	RedisStatus    string  `json:"redis_status"`
}

// HealthCheck represents health check response
type HealthCheck struct {
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Version   string                 `json:"version"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
}

// User represents a user in the system
type User struct {
	ID          string `json:"id" db:"id"`
	GoogleSub   string `json:"google_sub" db:"google_sub"`
	Email       string `json:"email" db:"email"`
	Name        string `json:"name" db:"name"`
	PictureURL  string `json:"picture_url" db:"picture_url"`
	CreatedAt   string `json:"created_at" db:"created_at"`
	UpdatedAt   string `json:"updated_at" db:"updated_at"`
	LastLoginAt string `json:"last_login_at" db:"last_login_at"`
}

// UserSession represents a user session for authentication
type UserSession struct {
	SessionID string    `json:"session_id" db:"session_id"`
	UserID    string    `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
}
