package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/captain-corgi/multi-agent-system/pkg/config"
	"github.com/captain-corgi/multi-agent-system/pkg/models"
)

// SessionManager manages user sessions using Redis and cookies
type SessionManager struct {
	storage    AuthStorage
	config     *config.AuthConfig
	logger     *zap.Logger
	cookieName string
}

// NewSessionManager creates a new session manager
func NewSessionManager(storage AuthStorage, authConfig *config.AuthConfig, logger *zap.Logger) *SessionManager {
	return &SessionManager{
		storage:    storage,
		config:     authConfig,
		logger:     logger,
		cookieName: authConfig.Session.CookieName,
	}
}

// CreateSession creates a new session for the user and returns the session ID
func (sm *SessionManager) CreateSession(ctx context.Context, userID string) (string, error) {
	// Generate a secure random session ID
	sessionID, err := sm.generateSessionID()
	if err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(sm.config.Session.TTL)

	// Create session in database
	session := &models.UserSession{
		SessionID: sessionID,
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}

	err = sm.storage.CreateUserSession(ctx, session)
	if err != nil {
		return "", fmt.Errorf("failed to create session in database: %w", err)
	}

	sm.logger.Info("Session created",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.Time("expires_at", expiresAt))

	return sessionID, nil
}

// GetSession retrieves session information by session ID
func (sm *SessionManager) GetSession(ctx context.Context, sessionID string) (*models.UserSession, error) {
	return sm.storage.GetUserSession(ctx, sessionID)
}

// DeleteSession removes a session
func (sm *SessionManager) DeleteSession(ctx context.Context, sessionID string) error {
	err := sm.storage.DeleteUserSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	sm.logger.Info("Session deleted", zap.String("session_id", sessionID))
	return nil
}

// SetSessionCookie sets the session cookie in the HTTP response
func (sm *SessionManager) SetSessionCookie(c echo.Context, sessionID string) {
	cookie := &http.Cookie{
		Name:     sm.cookieName,
		Value:    sessionID,
		Path:     "/",
		Domain:   sm.config.Session.Domain,
		MaxAge:   int(sm.config.Session.TTL.Seconds()),
		HttpOnly: true,
		Secure:   sm.config.Session.Secure,
		SameSite: sm.parseSameSite(sm.config.Session.SameSite),
	}

	c.SetCookie(cookie)

	sm.logger.Debug("Session cookie set",
		zap.String("session_id", sessionID),
		zap.String("cookie_name", sm.cookieName))
}

// ClearSessionCookie removes the session cookie
func (sm *SessionManager) ClearSessionCookie(c echo.Context) {
	cookie := &http.Cookie{
		Name:     sm.cookieName,
		Value:    "",
		Path:     "/",
		Domain:   sm.config.Session.Domain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   sm.config.Session.Secure,
		SameSite: sm.parseSameSite(sm.config.Session.SameSite),
	}

	c.SetCookie(cookie)

	sm.logger.Debug("Session cookie cleared")
}

// GetSessionFromCookie retrieves the session ID from the request cookie
func (sm *SessionManager) GetSessionFromCookie(c echo.Context) string {
	cookie, err := c.Cookie(sm.cookieName)
	if err != nil {
		return ""
	}

	sessionID := cookie.Value
	if sessionID == "" {
		return ""
	}

	// Basic validation - session ID should be hex string of expected length
	if len(sessionID) != 64 || !sm.isValidHex(sessionID) {
		sm.logger.Warn("Invalid session cookie format", zap.String("session_id", sessionID))
		return ""
	}

	return sessionID
}

// ValidateSession checks if a session is valid and not expired
func (sm *SessionManager) ValidateSession(ctx context.Context, sessionID string) (*models.UserSession, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("empty session ID")
	}

	session, err := sm.GetSession(ctx, sessionID)
	if err != nil {
		sm.logger.Debug("Session validation failed",
			zap.String("session_id", sessionID),
			zap.Error(err))
		return nil, fmt.Errorf("invalid session")
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		// Clean up expired session
		sm.DeleteSession(ctx, sessionID)
		sm.logger.Debug("Session expired",
			zap.String("session_id", sessionID),
			zap.Time("expires_at", session.ExpiresAt))
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// CleanupExpiredSessions removes all expired sessions (should be called periodically)
func (sm *SessionManager) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	return sm.storage.CleanupExpiredSessions(ctx)
}

// Helper functions

func (sm *SessionManager) generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (sm *SessionManager) isValidHex(s string) bool {
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func (sm *SessionManager) parseSameSite(sameSite string) http.SameSite {
	switch strings.ToLower(sameSite) {
	case "strict":
		return http.SameSiteStrictMode
	case "lax":
		return http.SameSiteLaxMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

// GetStorage returns the storage instance (for interface compliance)
func (sm *SessionManager) GetStorage() AuthStorage {
	return sm.storage
}
