package audit

import (
	"time"
)

// AuditEvent represents an audit event in the database
type AuditEvent struct {
	ID         int64                  `json:"id" db:"id"`
	EventID    string                 `json:"event_id" db:"event_id"`
	Category   string                 `json:"category" db:"category"`
	Action     string                 `json:"action" db:"action"`
	ActorID    *string                `json:"actor_id" db:"actor_id"`
	ActorEmail *string                `json:"actor_email" db:"actor_email"`
	SourceIP   *string                `json:"source_ip" db:"source_ip"`
	Resource   *string                `json:"resource" db:"resource"`
	RequestID  *string                `json:"request_id" db:"request_id"`
	Success    *bool                  `json:"success" db:"success"`
	Payload    map[string]interface{} `json:"payload" db:"payload"`
	Redacted   bool                   `json:"redacted" db:"redacted"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
}

// AuditFilter represents filters for querying audit events
type AuditFilter struct {
	Category  *string    `json:"category,omitempty"`
	ActorID   *string    `json:"actor_id,omitempty"`
	Action    *string    `json:"action,omitempty"`
	Resource  *string    `json:"resource,omitempty"`
	Success   *bool      `json:"success,omitempty"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}

// Pagination represents pagination parameters
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// Page represents pagination metadata
type Page struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// AuditRecorder interface for recording audit events
type AuditRecorder interface {
	Record(event AuditEvent) error
	Close() error
}

// ActorInfo represents actor information extracted from request headers
type ActorInfo struct {
	ID    string
	Email string
}
