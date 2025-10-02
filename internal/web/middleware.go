package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"

	"github.com/captain-corgi/multi-agent-system/pkg/audit"
	"github.com/captain-corgi/multi-agent-system/pkg/config"
)

// RequestIDMiddleware adds a request ID to each request
func RequestIDMiddleware() echo.MiddlewareFunc {
	return middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		Generator: func() string {
			return audit.GenerateEventID()
		},
	})
}

// AccessLogMiddleware logs HTTP requests with structured logging
func AccessLogMiddleware(logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			req := c.Request()
			res := c.Response()
			err := next(c)
			latency := time.Since(start)

			// Extract user info from headers if available
			userID := req.Header.Get("X-User-Id")
			userEmail := req.Header.Get("X-User-Email")

			// Build log fields with Datadog standard field names
			fields := []zap.Field{
				zap.String("http.method", req.Method),
				zap.String("http.url", req.RequestURI),
				zap.String("http.route", c.Path()), // Echo route pattern
				zap.Int("http.status_code", res.Status),
				zap.String("network.client.ip", c.RealIP()),
				zap.String("user_agent", req.UserAgent()),
				zap.String("request_id", res.Header().Get(echo.HeaderXRequestID)),
				zap.Int64("duration_ms", latency.Milliseconds()),
				zap.Int64("bytes_in", req.ContentLength),
				zap.Int64("bytes_out", res.Size),
			}

			// Add user fields if available
			if userID != "" {
				fields = append(fields, zap.String("usr.id", userID))
			}
			if userEmail != "" {
				fields = append(fields, zap.String("usr.email", userEmail))
			}

			logger.Info("HTTP request", fields...)
			return err
		}
	}
}

// AuditMiddleware records HTTP audit events for mutating operations
func AuditMiddleware(recorder audit.AuditRecorder, auditConfig *config.AuditConfig, logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip if audit is disabled or HTTP category is disabled
			if !auditConfig.Enabled || !auditConfig.Categories.HTTP {
				return next(c)
			}

			// Only audit mutating methods
			method := c.Request().Method
			if !isMutatingMethod(method) {
				return next(c)
			}

			// Extract actor info and source IP
			actor := audit.ExtractActorInfo(c.Request())
			sourceIP := audit.ExtractSourceIP(c.Request())
			requestID := c.Response().Header().Get(echo.HeaderXRequestID)

			// Capture request body for audit
			var requestBody map[string]interface{}
			if req := c.Request(); req != nil && req.Body != nil {
				const maxBody = 1 << 20 // 1MB
				bodyBytes, _ := io.ReadAll(io.LimitReader(req.Body, maxBody))
				req.Body.Close()
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				if len(bodyBytes) > 0 {
					_ = json.Unmarshal(bodyBytes, &requestBody)
				}
			}

			// Execute the handler
			err := next(c)

			// Record audit event
			statusCode := c.Response().Status
			path := c.Request().URL.Path

			auditEvent := audit.NewHTTPAuditEvent(
				method,
				path,
				statusCode,
				actor,
				sourceIP,
				requestID,
				requestBody,
			)

			if recordErr := recorder.Record(auditEvent); recordErr != nil {
				logger.Error("Failed to record audit event",
					zap.String("request_id", requestID),
					zap.Error(recordErr))
			}

			return err
		}
	}
}

// isMutatingMethod checks if the HTTP method is a mutating operation
func isMutatingMethod(method string) bool {
	mutatingMethods := []string{"POST", "PUT", "PATCH", "DELETE"}
	for _, m := range mutatingMethods {
		if strings.EqualFold(method, m) {
			return true
		}
	}
	return false
}

// AuditContextMiddleware adds audit context to the request context
func AuditContextMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Add audit context to request context
			ctx := context.WithValue(c.Request().Context(), "audit_actor", audit.ExtractActorInfo(c.Request()))
			ctx = context.WithValue(ctx, "audit_source_ip", audit.ExtractSourceIP(c.Request()))
			ctx = context.WithValue(ctx, "audit_request_id", c.Response().Header().Get(echo.HeaderXRequestID))

			// Update request with new context
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

// AuditHelper provides helper functions for recording audit events
type AuditHelper struct {
	recorder audit.AuditRecorder
	config   *config.AuditConfig
	logger   *zap.Logger
}

// NewAuditHelper creates a new audit helper
func NewAuditHelper(recorder audit.AuditRecorder, config *config.AuditConfig, logger *zap.Logger) *AuditHelper {
	return &AuditHelper{
		recorder: recorder,
		config:   config,
		logger:   logger,
	}
}

// RecordAgentEvent records an agent audit event
func (h *AuditHelper) RecordAgentEvent(agentID, action string, success bool, payload map[string]interface{}) {
	if !h.config.Enabled || !h.config.Categories.Agent {
		return
	}

	event := audit.NewAgentAuditEvent(agentID, action, success, payload)
	if err := h.recorder.Record(event); err != nil {
		h.logger.Error("Failed to record agent audit event",
			zap.String("agent_id", agentID),
			zap.String("action", action),
			zap.Error(err))
	}
}

// RecordExternalEvent records an external API audit event
func (h *AuditHelper) RecordExternalEvent(service, action string, success bool, payload map[string]interface{}) {
	if !h.config.Enabled || !h.config.Categories.External {
		return
	}

	event := audit.NewExternalAuditEvent(service, action, success, payload)
	if err := h.recorder.Record(event); err != nil {
		h.logger.Error("Failed to record external audit event",
			zap.String("service", service),
			zap.String("action", action),
			zap.Error(err))
	}
}

// RecordCustomEvent records a custom audit event
func (h *AuditHelper) RecordCustomEvent(category, action string, actor audit.ActorInfo, sourceIP string, requestID *string, success *bool, payload map[string]interface{}) {
	if !h.config.Enabled {
		return
	}

	event := audit.NewAuditEvent(category, action, actor, sourceIP, requestID)
	event.Success = success
	event.Payload = payload

	if err := h.recorder.Record(event); err != nil {
		h.logger.Error("Failed to record custom audit event",
			zap.String("category", category),
			zap.String("action", action),
			zap.Error(err))
	}
}
