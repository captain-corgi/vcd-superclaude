package agents

import (
	"context"
	"time"
)

type Agent interface {
	// Core functionality
	Initialize(ctx context.Context, config AgentConfig) error
	ProcessRequest(ctx context.Context, req *Request) (*Response, error)
	StreamProgress(ctx context.Context, req *Request) (<-chan ProgressUpdate, error)

	// Communication methods
	SendMessage(ctx context.Context, targetAgent string, message *Message) error
	ReceiveMessage(ctx context.Context, message *Message) error

	// Lifecycle management
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Status() AgentStatus
	GetMetrics() AgentMetrics
}

type AgentConfig struct {
	ID           string                 `mapstructure:"id"`
	Name         string                 `mapstructure:"name"`
	Type         AgentType              `mapstructure:"type"`
	Enabled      bool                   `mapstructure:"enabled"`
	Dependencies []string               `mapstructure:"dependencies"`
	Settings     map[string]interface{} `mapstructure:"settings"`
}

type Request struct {
	ID          string                 `json:"id"`
	Type        RequestType            `json:"type"`
	Payload     map[string]interface{} `json:"payload"`
	Metadata    map[string]string      `json:"metadata"`
	Timestamp   time.Time              `json:"timestamp"`
	SourceAgent string                 `json:"source_agent"`
}

type Response struct {
	ID          string                 `json:"id"`
	Success     bool                   `json:"success"`
	Data        map[string]interface{} `json:"data"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]string      `json:"metadata"`
	Timestamp   time.Time              `json:"timestamp"`
	TargetAgent string                 `json:"target_agent"`
}

type Message struct {
	ID          string        `json:"id"`
	Type        MessageType   `json:"type"`
	Content     interface{}   `json:"content"`
	FromAgent   string        `json:"from_agent"`
	ToAgent     string        `json:"to_agent"`
	Timestamp   time.Time     `json:"timestamp"`
	Correlation string        `json:"correlation_id"`
}

type ProgressUpdate struct {
	ID          string            `json:"id"`
	Agent       string            `json:"agent"`
	Status      string            `json:"status"`
	Progress    float64           `json:"progress"`
	Message     string            `json:"message"`
	Timestamp   time.Time         `json:"timestamp"`
	Metadata    map[string]string `json:"metadata"`
}

type AgentStatus string

type RequestType string

type MessageType string

type AgentMetrics struct {
	RequestsProcessed   int64         `json:"requests_processed"`
	RequestsSuccess     int64         `json:"requests_success"`
	RequestsFailed      int64         `json:"requests_failed"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	ErrorRate           float64       `json:"error_rate"`
	Uptime              time.Duration `json:"uptime"`
	MemoryUsage         uint64        `json:"memory_usage"`
}

// Type definitions
type AgentType string

const (
	AgentTypePlanner   AgentType = "planner"
	AgentTypeExecutor  AgentType = "executor"
	AgentTypeEvaluator AgentType = "evaluator"
)

const (
	RequestTypeDataCollection RequestType = "data_collection"
	RequestTypeExecution     RequestType = "execution"
	RequestTypeEvaluation    RequestType = "evaluation"
	RequestTypeStatusQuery   RequestType = "status_query"
)

const (
	MessageTypeRequest  MessageType = "request"
	MessageTypeResponse MessageType = "response"
	MessageTypeProgress MessageType = "progress"
	MessageTypeError    MessageType = "error"
	MessageTypeNotification MessageType = "notification"
)

const (
	AgentStatusIdle       AgentStatus = "idle"
	AgentStatusProcessing AgentStatus = "processing"
	AgentStatusError      AgentStatus = "error"
	AgentStatusStopped    AgentStatus = "stopped"
	AgentStatusStarting   AgentStatus = "starting"
	AgentStatusStopping   AgentStatus = "stopping"
)

type ExecutionPlan struct {
	ID              string                 `json:"id"`
	Tasks           []Task                 `json:"tasks"`
	Dependencies    []Dependency           `json:"dependencies"`
	CreatedAt       time.Time              `json:"created_at"`
	EstimatedDuration time.Duration        `json:"estimated_duration"`
	ResourceRequirements map[string]int     `json:"resource_requirements"`
	Status          PlanStatus             `json:"status"`
}

type Task struct {
	ID           string                 `json:"id"`
	Type         TaskType               `json:"type"`
	DataSource   string                 `json:"data_source"`
	Parameters   map[string]interface{} `json:"parameters"`
	Dependencies []string               `json:"dependencies"`
	Priority     Priority                `json:"priority"`
	Status       TaskStatus             `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Retries      int                    `json:"retries"`
}

type Dependency struct {
	TaskID     string `json:"task_id"`
	DependsOn  string `json:"depends_on"`
	Type       string `json:"type"` // "sequential", "parallel", "conditional"
}

type TaskType string

type TaskStatus string

type PlanStatus string

type Priority string

const (
	TaskTypeDataCollection TaskType = "data_collection"
	TaskTypeDataProcessing TaskType = "data_processing"
	TaskTypeReportGeneration TaskType = "report_generation"
	TaskTypeValidation     TaskType = "validation"
)

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

const (
	PlanStatusCreated    PlanStatus = "created"
	PlanStatusRunning    PlanStatus = "running"
	PlanStatusCompleted  PlanStatus = "completed"
	PlanStatusFailed     PlanStatus = "failed"
	PlanStatusCancelled  PlanStatus = "cancelled"
)

const (
	PriorityLow      Priority = "low"
	PriorityNormal   Priority = "normal"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

type ToolResult struct {
	Success   bool                   `json:"success"`
	Data      interface{}            `json:"data"`
	Metadata  map[string]string      `json:"metadata"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Cached    bool                   `json:"cached,omitempty"`
}

type ToolSchema struct {
	Name        string              `json:"name"`
	Version     string              `json:"version"`
	Description string              `json:"description"`
	Parameters  map[string]*Parameter `json:"parameters"`
	Required    []string            `json:"required"`
	Returns     *Parameter          `json:"returns"`
}

type Parameter struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Required    bool                   `json:"required"`
	Default     interface{}            `json:"default,omitempty"`
	Enum        []string               `json:"enum,omitempty"`
	Format      string                 `json:"format,omitempty"`
	Example     interface{}            `json:"example,omitempty"`
	Constraints map[string]interface{} `json:"constraints,omitempty"`
}