package audit

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// ExtractActorInfo extracts actor information from HTTP request headers
func ExtractActorInfo(r *http.Request) ActorInfo {
	actorID := r.Header.Get("X-User-Id")
	actorEmail := r.Header.Get("X-User-Email")

	// Fallback to system if no actor info
	if actorID == "" {
		actorID = "system"
	}

	return ActorInfo{
		ID:    actorID,
		Email: actorEmail,
	}
}

// ExtractSourceIP extracts the source IP from the request
func ExtractSourceIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fallback to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// GenerateEventID generates a unique event ID
func GenerateEventID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// NewAuditEvent creates a new audit event with common fields
func NewAuditEvent(category, action string, actor ActorInfo, sourceIP string, requestID *string) AuditEvent {
	eventID := GenerateEventID()
	now := time.Now()

	return AuditEvent{
		EventID:    eventID,
		Category:   category,
		Action:     action,
		ActorID:    &actor.ID,
		ActorEmail: &actor.Email,
		SourceIP:   &sourceIP,
		RequestID:  requestID,
		CreatedAt:  now,
	}
}

// NewHTTPAuditEvent creates a new HTTP audit event
func NewHTTPAuditEvent(method, path string, statusCode int, actor ActorInfo, sourceIP string, requestID string, payload map[string]interface{}) AuditEvent {
	event := NewAuditEvent("http", fmt.Sprintf("%s %s", method, path), actor, sourceIP, &requestID)
	event.Resource = &path
	success := statusCode >= 200 && statusCode < 400
	event.Success = &success
	event.Payload = payload
	return event
}

// NewAgentAuditEvent creates a new agent audit event
func NewAgentAuditEvent(agentID, action string, success bool, payload map[string]interface{}) AuditEvent {
	event := NewAuditEvent("agent", action, ActorInfo{ID: "system"}, "", nil)
	event.Resource = &agentID
	event.Success = &success
	event.Payload = payload
	return event
}

// NewExternalAuditEvent creates a new external API audit event
func NewExternalAuditEvent(service, action string, success bool, payload map[string]interface{}) AuditEvent {
	event := NewAuditEvent("external", action, ActorInfo{ID: "system"}, "", nil)
	event.Resource = &service
	event.Success = &success
	event.Payload = payload
	return event
}
