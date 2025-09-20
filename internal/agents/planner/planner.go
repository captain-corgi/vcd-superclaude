package planner

import (
	"context"
	"fmt"
	"time"

	"github.com/anh-tuan-l/multi-agent-system/internal/agents"
	"github.com/anh-tuan-l/multi-agent-system/internal/communication/trpc"
	"go.uber.org/zap"
)

type PlannerAgent struct {
	config     *PlannerConfig
	trpcClient *trpc.Client
	logger     *zap.Logger
	status     agents.AgentStatus
	metrics    agents.AgentMetrics
	startTime  time.Time
}

type PlannerConfig struct {
	BaseConfig  agents.AgentConfig `mapstructure:",squash"`
	MaxRequests int                `mapstructure:"max_requests"`
	Timeout     time.Duration      `mapstructure:"timeout"`
}

type ParsedRequest struct {
	Query        string          `json:"query"`
	DataSources  []string        `json:"data_sources"`
	TimeRange    *TimeRange      `json:"time_range,omitempty"`
	OutputFormat string          `json:"output_format"`
	Priority     agents.Priority `json:"priority"`
}

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

func NewPlannerAgent(config *PlannerConfig, trpcClient *trpc.Client, logger *zap.Logger) *PlannerAgent {
	return &PlannerAgent{
		config:     config,
		trpcClient: trpcClient,
		logger:     logger,
		status:     agents.AgentStatusIdle,
		startTime:  time.Now(),
		metrics: agents.AgentMetrics{
			RequestsProcessed: 0,
			RequestsSuccess:   0,
			RequestsFailed:    0,
			Uptime:            0,
		},
	}
}

func (p *PlannerAgent) Initialize(ctx context.Context, config agents.AgentConfig) error {
	p.config.BaseConfig = config
	p.logger.Info("Planner agent initialized",
		zap.String("id", config.ID),
		zap.String("name", config.Name),
		zap.String("type", string(config.Type)))
	return nil
}

func (p *PlannerAgent) ProcessRequest(ctx context.Context, req *agents.Request) (*agents.Response, error) {
	p.status = agents.AgentStatusProcessing
	defer func() { p.status = agents.AgentStatusIdle }()

	start := time.Now()

	p.logger.Info("Processing request",
		zap.String("request_id", req.ID),
		zap.String("request_type", string(req.Type)))

	// Check if request limit exceeded
	if p.config.MaxRequests > 0 && p.metrics.RequestsProcessed >= int64(p.config.MaxRequests) {
		p.updateMetrics(false, time.Since(start))
		return nil, fmt.Errorf("maximum request limit (%d) reached", p.config.MaxRequests)
	}

	// Parse and analyze request
	parsedReq, err := p.parseRequest(req)
	if err != nil {
		p.logger.Error("Request parsing failed",
			zap.String("request_id", req.ID),
			zap.Error(err))
		p.updateMetrics(false, time.Since(start))
		return nil, fmt.Errorf("request parsing failed: %w", err)
	}

	// Create execution plan
	plan, err := p.createExecutionPlan(parsedReq)
	if err != nil {
		p.logger.Error("Plan creation failed",
			zap.String("request_id", req.ID),
			zap.Error(err))
		p.updateMetrics(false, time.Since(start))
		return nil, fmt.Errorf("plan creation failed: %w", err)
	}

	// Delegate to executor
	_, err = p.delegateToExecutor(ctx, plan, req)
	if err != nil {
		p.logger.Error("Delegation to executor failed",
			zap.String("request_id", req.ID),
			zap.Error(err))
		p.updateMetrics(false, time.Since(start))
		return nil, fmt.Errorf("delegation failed: %w", err)
	}

	p.updateMetrics(true, time.Since(start))
	p.logger.Info("Request processed successfully",
		zap.String("request_id", req.ID),
		zap.Duration("duration", time.Since(start)),
		zap.Int("tasks_created", len(plan.Tasks)))

	return &agents.Response{
		ID:          req.ID,
		Success:     true,
		Data:        map[string]interface{}{"plan": plan},
		TargetAgent: req.SourceAgent,
		Timestamp:   time.Now(),
	}, nil
}

func (p *PlannerAgent) parseRequest(req *agents.Request) (*ParsedRequest, error) {
	// Extract query from payload
	query, ok := req.Payload["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required and must be a string")
	}

	// Extract data sources
	var dataSources []string
	if ds, ok := req.Payload["data_sources"].([]interface{}); ok {
		for _, dsItem := range ds {
			if dsStr, ok := dsItem.(string); ok {
				dataSources = append(dataSources, dsStr)
			}
		}
	}

	// If no data sources specified, use all available
	if len(dataSources) == 0 {
		dataSources = []string{"jira", "slack", "github"}
	}

	// Extract time range if provided
	var timeRange *TimeRange
	if tr, ok := req.Payload["time_range"].(map[string]interface{}); ok {
		if startStr, ok := tr["start"].(string); ok {
			startTime, err := time.Parse(time.RFC3339, startStr)
			if err == nil {
				endTime := time.Now()
				if endStr, ok := tr["end"].(string); ok {
					if et, err := time.Parse(time.RFC3339, endStr); err == nil {
						endTime = et
					}
				}
				timeRange = &TimeRange{Start: startTime, End: endTime}
			}
		}
	}

	// Extract output format
	outputFormat := "json"
	if of, ok := req.Payload["output_format"].(string); ok {
		outputFormat = of
	}

	// Extract priority
	priority := agents.PriorityNormal
	if p, ok := req.Payload["priority"].(string); ok {
		priority = agents.Priority(p)
	}

	return &ParsedRequest{
		Query:        query,
		DataSources:  dataSources,
		TimeRange:    timeRange,
		OutputFormat: outputFormat,
		Priority:     priority,
	}, nil
}

func (p *PlannerAgent) createExecutionPlan(req *ParsedRequest) (*agents.ExecutionPlan, error) {
	tasks := make([]agents.Task, 0, len(req.DataSources))

	for i, source := range req.DataSources {
		taskID := fmt.Sprintf("task_%s_%d", source, i+1)

		// Prepare parameters for the tool
		parameters := map[string]interface{}{
			"query": req.Query,
		}

		if req.TimeRange != nil {
			parameters["time_range"] = map[string]interface{}{
				"start": req.TimeRange.Start.Format(time.RFC3339),
				"end":   req.TimeRange.End.Format(time.RFC3339),
			}
		}

		if source == "jira" && req.TimeRange != nil {
			parameters["jql"] = fmt.Sprintf(`created > "%s" AND created < "%s"`,
				req.TimeRange.Start.Format("2006-01-02"),
				req.TimeRange.End.Format("2006-01-02"))
		}

		task := agents.Task{
			ID:         taskID,
			Type:       agents.TaskTypeDataCollection,
			DataSource: source,
			Parameters: parameters,
			Priority:   req.Priority,
			Status:     agents.TaskStatusPending,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Retries:    0,
		}
		tasks = append(tasks, task)
	}

	plan := &agents.ExecutionPlan{
		ID:                fmt.Sprintf("plan_%d", time.Now().Unix()),
		Tasks:             tasks,
		CreatedAt:         time.Now(),
		EstimatedDuration: time.Duration(len(tasks)) * 30 * time.Second, // 30 seconds per task
		ResourceRequirements: map[string]int{
			"concurrent_tasks": 3, // Allow 3 concurrent tasks
		},
		Status: agents.PlanStatusCreated,
	}

	p.logger.Info("Execution plan created",
		zap.String("plan_id", plan.ID),
		zap.Int("tasks_count", len(tasks)))

	return plan, nil
}

func (p *PlannerAgent) delegateToExecutor(ctx context.Context, plan *agents.ExecutionPlan, req *agents.Request) (*agents.Response, error) {
	message := &agents.Message{
		ID:          fmt.Sprintf("msg_%d", time.Now().Unix()),
		Type:        agents.MessageTypeRequest,
		Content:     plan,
		FromAgent:   p.config.BaseConfig.ID,
		ToAgent:     "executor",
		Timestamp:   time.Now(),
		Correlation: plan.ID,
	}

	ctx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	defer cancel()

	return p.trpcClient.SendMessage(ctx, message)
}

func (p *PlannerAgent) updateMetrics(success bool, duration time.Duration) {
	p.metrics.RequestsProcessed++
	p.metrics.Uptime = time.Since(p.startTime)

	if success {
		p.metrics.RequestsSuccess++
	} else {
		p.metrics.RequestsFailed++
	}

	// Update average response time
	total := p.metrics.RequestsProcessed
	p.metrics.AverageResponseTime = time.Duration(
		(int64(p.metrics.AverageResponseTime)*(total-1) + int64(duration)) / total,
	)

	// Calculate error rate
	if p.metrics.RequestsProcessed > 0 {
		p.metrics.ErrorRate = float64(p.metrics.RequestsFailed) / float64(p.metrics.RequestsProcessed) * 100
	}
}

// Interface implementation stubs
func (p *PlannerAgent) StreamProgress(ctx context.Context, req *agents.Request) (<-chan agents.ProgressUpdate, error) {
	progressChan := make(chan agents.ProgressUpdate)

	go func() {
		defer close(progressChan)

		// Simulate progress updates
		updates := []struct {
			status   string
			progress float64
			message  string
		}{
			{"parsing", 10, "Parsing request..."},
			{"validating", 30, "Validating request..."},
			{"planning", 60, "Creating execution plan..."},
			{"delegating", 90, "Delegating to executor..."},
			{"completed", 100, "Plan created successfully"},
		}

		for _, update := range updates {
			select {
			case <-ctx.Done():
				return
			case progressChan <- agents.ProgressUpdate{
				ID:        fmt.Sprintf("progress_%d", time.Now().Unix()),
				Agent:     p.config.BaseConfig.ID,
				Status:    update.status,
				Progress:  update.progress,
				Message:   update.message,
				Timestamp: time.Now(),
				Metadata:  map[string]string{"plan_id": req.ID},
			}:
				time.Sleep(100 * time.Millisecond) // Simulate work
			}
		}
	}()

	return progressChan, nil
}

func (p *PlannerAgent) SendMessage(ctx context.Context, targetAgent string, message *agents.Message) error {
	// For now, use the tRPC client
	_, err := p.trpcClient.SendMessage(ctx, message)
	return err
}

func (p *PlannerAgent) ReceiveMessage(ctx context.Context, message *agents.Message) error {
	// Handle incoming messages from other agents
	p.logger.Info("Received message",
		zap.String("from", message.FromAgent),
		zap.String("type", string(message.Type)),
		zap.String("correlation", message.Correlation))

	// Handle response messages
	if message.Type == agents.MessageTypeResponse {
		if respData, ok := message.Content.(map[string]interface{}); ok {
			if planID, ok := respData["plan_id"].(string); ok {
				p.logger.Info("Received executor response for plan",
					zap.String("plan_id", planID))
			}
		}
	}

	return nil
}

func (p *PlannerAgent) Start(ctx context.Context) error {
	p.status = agents.AgentStatusStarting
	defer func() { p.status = agents.AgentStatusIdle }()

	p.logger.Info("Starting planner agent",
		zap.String("id", p.config.BaseConfig.ID))

	// Initialize health check
	go p.healthCheck(ctx)

	p.status = agents.AgentStatusIdle
	return nil
}

func (p *PlannerAgent) Stop(ctx context.Context) error {
	p.status = agents.AgentStatusStopping
	defer func() { p.status = agents.AgentStatusStopped }()

	p.logger.Info("Stopping planner agent",
		zap.String("id", p.config.BaseConfig.ID))

	return nil
}

func (p *PlannerAgent) Status() agents.AgentStatus {
	return p.status
}

func (p *PlannerAgent) GetMetrics() agents.AgentMetrics {
	p.metrics.Uptime = time.Since(p.startTime)
	return p.metrics
}

func (p *PlannerAgent) healthCheck(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Health check logic here
			p.logger.Debug("Health check",
				zap.String("status", string(p.status)),
				zap.Int64("requests_processed", p.metrics.RequestsProcessed),
				zap.Float64("error_rate", p.metrics.ErrorRate))
		}
	}
}
