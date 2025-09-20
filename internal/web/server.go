package web

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/anh-tuan-l/multi-agent-system/internal/agents"
	"github.com/anh-tuan-l/multi-agent-system/internal/communication/streaming"
	"github.com/anh-tuan-l/multi-agent-system/pkg/config"
	"github.com/anh-tuan-l/multi-agent-system/pkg/storage"
)

type Server struct {
	echo      *echo.Echo
	config    *config.Config
	storage   *storage.Storage
	streamer  *streaming.Streamer
	logger    *zap.Logger
	upgrader  *websocket.Upgrader
	startTime time.Time
	mu        sync.RWMutex
	agents    map[string]agents.Agent
}

type ServerStats struct {
	Uptime        time.Duration          `json:"uptime"`
	RequestsTotal int64                  `json:"requests_total"`
	ErrorsTotal   int64                  `json:"errors_total"`
	ActiveConns   int                    `json:"active_connections"`
	AgentStatus   map[string]string      `json:"agent_status"`
	SystemMetrics map[string]interface{} `json:"system_metrics"`
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
		HandshakeTimeout: 10 * time.Second,
	}
)

func NewServer(cfg *config.Config, storage *storage.Storage, logger *zap.Logger) *Server {
	e := echo.New()

	// Configure Echo
	e.HideBanner = true
	e.Debug = cfg.App.Debug
	e.HTTPErrorHandler = customHTTPErrorHandler

	server := &Server{
		echo:      e,
		config:    cfg,
		storage:   storage,
		streamer:  streaming.NewStreamer(),
		logger:    logger,
		upgrader:  &upgrader,
		startTime: time.Now(),
		agents:    make(map[string]agents.Agent),
	}

	server.setupRoutes()

	return server
}

// Echo returns the Echo instance for testing
func (s *Server) Echo() *echo.Echo {
	return s.echo
}

func (s *Server) setupRoutes() {
	// Health checks
	s.echo.GET("/health", s.healthCheck)
	s.echo.GET("/health/detailed", s.detailedHealthCheck)
	s.echo.GET("/ready", s.readinessCheck)
	s.echo.GET("/live", s.livenessCheck)

	// API routes
	api := s.echo.Group("/api")
	api.POST("/requests", s.createRequest)
	api.GET("/requests/:id", s.getRequest)
	api.GET("/requests", s.listRequests)
	api.GET("/agents/status", s.getAgentStatus)
	api.GET("/agents/:id/metrics", s.getAgentMetrics)
	api.GET("/metrics", s.getMetrics)
	api.GET("/config", s.getConfig)

	// WebSocket route
	s.echo.GET("/ws", s.websocketHandler)

	// Static files
	s.echo.Static("/static", "web/static")
	s.echo.File("/", "web/static/index.html")
}

func (s *Server) Start(address string) error {
	s.logger.Info("Starting server",
		zap.String("address", address),
		zap.String("app_name", s.config.App.Name),
		zap.String("version", s.config.App.Version),
		zap.Int("port", s.config.App.Port),
		zap.Bool("debug", s.config.App.Debug))

	return s.echo.Start(address)
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server")
	return s.echo.Shutdown(ctx)
}

// Health check endpoints
func (s *Server) healthCheck(c echo.Context) error {
	stats := s.getServerStats()

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"uptime":    stats.Uptime.String(),
		"version":   s.config.App.Version,
	})
}

func (s *Server) detailedHealthCheck(c echo.Context) error {
	storageHealth := s.storage.GetStorageStats()
	stats := s.getServerStats()

	health := map[string]interface{}{
		"status":      "healthy",
		"timestamp":   time.Now().UTC(),
		"uptime":      stats.Uptime.String(),
		"version":     s.config.App.Version,
		"database":    storageHealth["database"],
		"redis":       storageHealth["redis"],
		"agents":      s.getAgentStatuses(),
		"system":      stats.SystemMetrics,
		"connections": stats.ActiveConns,
	}

	return c.JSON(http.StatusOK, health)
}

func (s *Server) readinessCheck(c echo.Context) error {
	// Check if all required services are ready
	if !s.isReady() {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"status": "not ready",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status": "ready",
	})
}

func (s *Server) livenessCheck(c echo.Context) error {
	stats := s.getServerStats()
	return c.JSON(http.StatusOK, map[string]string{
		"status": "alive",
		"uptime": stats.Uptime.String(),
	})
}

// API endpoints
func (s *Server) createRequest(c echo.Context) error {
	var req agents.Request
	if err := c.Bind(&req); err != nil {
		s.logger.Error("Invalid request format", zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Generate request ID if not provided
	if req.ID == "" {
		req.ID = fmt.Sprintf("req_%d", time.Now().Unix())
	}

	req.Timestamp = time.Now()

	// Process request through planner agent
	response, err := s.processRequest(c.Request().Context(), &req)
	if err != nil {
		s.logger.Error("Request processing failed",
			zap.String("request_id", req.ID),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	s.logger.Info("Request processed successfully",
		zap.String("request_id", req.ID))

	return c.JSON(http.StatusOK, response)
}

func (s *Server) getRequest(c echo.Context) error {
	id := c.Param("id")

	// For now, return a mock response
	// In a real implementation, this would retrieve from database
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":     id,
		"status": "completed",
		"data": map[string]interface{}{
			"result": "Mock data for request " + id,
		},
		"timestamp": time.Now(),
	})
}

func (s *Server) listRequests(c echo.Context) error {
	// Mock data - in real implementation, this would query database
	requests := []map[string]interface{}{
		{
			"id":        "req_1",
			"status":    "completed",
			"timestamp": time.Now().Add(-time.Hour),
			"source":    "web",
		},
		{
			"id":        "req_2",
			"status":    "processing",
			"timestamp": time.Now().Add(-30 * time.Minute),
			"source":    "api",
		},
	}

	return c.JSON(http.StatusOK, requests)
}

func (s *Server) getAgentStatus(c echo.Context) error {
	statuses := s.getAgentStatuses()
	return c.JSON(http.StatusOK, statuses)
}

func (s *Server) getAgentMetrics(c echo.Context) error {
	agentID := c.Param("id")

	s.mu.RLock()
	agent, exists := s.agents[agentID]
	s.mu.RUnlock()

	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Agent not found",
		})
	}

	metrics := agent.GetMetrics()
	return c.JSON(http.StatusOK, metrics)
}

func (s *Server) getMetrics(c echo.Context) error {
	stats := s.getServerMetrics()
	return c.JSON(http.StatusOK, stats)
}

func (s *Server) getConfig(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"app":      s.config.App,
		"agents":   s.config.Agents,
		"database": s.config.Database,
	})
}

// WebSocket handler
func (s *Server) websocketHandler(c echo.Context) error {
	conn, err := s.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		s.logger.Error("WebSocket upgrade failed", zap.Error(err))
		return err
	}
	defer conn.Close()

	subscriberID := fmt.Sprintf("ws_%d", time.Now().Unix())

	// Subscribe to progress updates
	ctx, cancel := context.WithCancel(c.Request().Context())
	defer cancel()

	updates, err := s.streamer.Subscribe(ctx, subscriberID)
	if err != nil {
		s.logger.Error("Failed to subscribe to updates", zap.Error(err))
		return err
	}

	s.logger.Info("WebSocket client connected",
		zap.String("subscriber_id", subscriberID))

	// Send welcome message
	welcome := map[string]interface{}{
		"type":      "welcome",
		"timestamp": time.Now(),
		"message":   "Connected to multi-agent system",
	}

	if err := conn.WriteJSON(welcome); err != nil {
		s.logger.Error("Failed to send welcome message", zap.Error(err))
		return err
	}

	// Handle incoming WebSocket messages
	go s.handleWebSocketMessages(conn, subscriberID)

	// Send updates to WebSocket client
	for update := range updates {
		if err := conn.WriteJSON(update); err != nil {
			s.logger.Error("WebSocket write failed", zap.Error(err))
			break
		}
	}

	s.logger.Info("WebSocket client disconnected",
		zap.String("subscriber_id", subscriberID))

	return nil
}

// Helper methods
func (s *Server) processRequest(ctx context.Context, req *agents.Request) (*agents.Response, error) {
	// This would interact with the planner agent
	// For now, return a mock response

	plan := &agents.ExecutionPlan{
		ID: fmt.Sprintf("plan_%d", time.Now().Unix()),
		Tasks: []agents.Task{
			{
				ID:         "task_1",
				Type:       agents.TaskTypeDataCollection,
				DataSource: "jira",
				Parameters: map[string]interface{}{
					"query": req.Payload["query"],
				},
				Status: agents.TaskStatusPending,
			},
		},
		CreatedAt:         time.Now(),
		EstimatedDuration: 30 * time.Second,
		Status:            agents.PlanStatusCreated,
	}

	return &agents.Response{
		ID:          req.ID,
		Success:     true,
		Data:        map[string]interface{}{"plan": plan},
		TargetAgent: req.SourceAgent,
		Timestamp:   time.Now(),
	}, nil
}

func (s *Server) handleWebSocketMessages(conn *websocket.Conn, subscriberID string) {
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("WebSocket read error", zap.Error(err))
			}
			break
		}

		msgType := msg["type"].(string)

		switch msgType {
		case "ping":
			response := map[string]interface{}{
				"type":      "pong",
				"timestamp": time.Now(),
			}
			conn.WriteJSON(response)

		case "subscribe":
			// Handle subscription requests
			if topics, ok := msg["topics"].([]interface{}); ok {
				s.logger.Info("WebSocket subscription",
					zap.String("subscriber_id", subscriberID),
					zap.Any("topics", topics))
			}

		case "get_status":
			response := map[string]interface{}{
				"type":      "status",
				"timestamp": time.Now(),
				"status":    s.getAgentStatuses(),
			}
			conn.WriteJSON(response)

		default:
			s.logger.Warn("Unknown WebSocket message type",
				zap.String("type", msgType),
				zap.String("subscriber_id", subscriberID))
		}
	}
}

func (s *Server) getServerStats() ServerStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := ServerStats{
		Uptime:        time.Since(s.startTime),
		ActiveConns:   0, // Would track active connections
		AgentStatus:   make(map[string]string),
		SystemMetrics: s.getSystemMetrics(),
	}

	for id, agent := range s.agents {
		stats.AgentStatus[id] = string(agent.Status())
	}

	return stats
}

func (s *Server) getServerMetrics() map[string]interface{} {
	return map[string]interface{}{
		"uptime":         time.Since(s.startTime).String(),
		"requests_total": 0,      // Would track total requests
		"memory_usage":   "50MB", // Would get actual memory usage
		"cpu_usage":      "10%",  // Would get actual CPU usage
		"active_agents":  len(s.agents),
	}
}

func (s *Server) getSystemMetrics() map[string]interface{} {
	return map[string]interface{}{
		"uptime":  time.Since(s.startTime).String(),
		"memory":  "50MB",
		"cpu":     "10%",
		"version": s.config.App.Version,
		"port":    s.config.App.Port,
	}
}

func (s *Server) getAgentStatuses() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	statuses := make(map[string]string)
	for id, agent := range s.agents {
		statuses[id] = string(agent.Status())
	}
	return statuses
}

func (s *Server) isReady() bool {
	// Check if all required services are ready
	return true // For now, always return ready
}

// Register an agent with the server
func (s *Server) RegisterAgent(agent agents.Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	agentID := agent.Status() // This should be agent.ID in real implementation
	s.agents[string(agentID)] = agent

	s.logger.Info("Agent registered",
		zap.String("agent_id", string(agentID)))

	return nil
}

// Custom HTTP error handler
func customHTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	message := "Internal Server Error"

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		message = he.Message.(string)
	}

	c.Logger().Error(err)

	c.JSON(code, map[string]interface{}{
		"error":     message,
		"code":      code,
		"path":      c.Request().URL.Path,
		"method":    c.Request().Method,
		"timestamp": time.Now(),
	})
}
