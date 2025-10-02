package web

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/captain-corgi/multi-agent-system/pkg/models"
)

// ContextKey for storing user in request context
type ContextKey string

const (
	UserContextKey ContextKey = "user"
)

// SessionManagerInterface defines the interface for session management
type SessionManagerInterface interface {
	GetSessionFromCookie(c echo.Context) string
	ValidateSession(ctx context.Context, sessionID string) (*models.UserSession, error)
	DeleteSession(ctx context.Context, sessionID string) error
	ClearSessionCookie(c echo.Context)
	GetStorage() AuthStorage // For accessing storage methods
}

// AuthRequiredMiddleware checks for valid authentication and adds user to context
func AuthRequiredMiddleware(sessionMgr SessionManagerInterface, logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()

			// Get session ID from cookie
			sessionID := sessionMgr.GetSessionFromCookie(c)
			if sessionID == "" {
				logger.Debug("No session cookie found")
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Authentication required",
				})
			}

			// Validate session
			session, err := sessionMgr.ValidateSession(ctx, sessionID)
			if err != nil {
				logger.Debug("Session validation failed",
					zap.String("session_id", sessionID),
					zap.Error(err))

				// Clear invalid cookie
				sessionMgr.ClearSessionCookie(c)

				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid or expired session",
				})
			}

			// Get user information
			storage := sessionMgr.GetStorage()
			user, err := storage.GetUserByID(ctx, session.UserID)
			if err != nil {
				logger.Error("Failed to get user information",
					zap.String("user_id", session.UserID),
					zap.Error(err))

				// Clear invalid session
				sessionMgr.DeleteSession(ctx, sessionID)
				sessionMgr.ClearSessionCookie(c)

				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "User not found",
				})
			}
			if err != nil {
				logger.Error("Failed to get user information",
					zap.String("user_id", session.UserID),
					zap.Error(err))

				// Clear invalid session
				sessionMgr.DeleteSession(ctx, sessionID)
				sessionMgr.ClearSessionCookie(c)

				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "User not found",
				})
			}

			// Add user to request context
			ctx = context.WithValue(ctx, UserContextKey, user)
			c.SetRequest(c.Request().WithContext(ctx))

			// Add user info to response headers for logging middleware
			c.Response().Header().Set("X-User-Id", user.ID)
			c.Response().Header().Set("X-User-Email", user.Email)

			logger.Debug("User authenticated",
				zap.String("user_id", user.ID),
				zap.String("email", user.Email),
				zap.String("session_id", sessionID))

			return next(c)
		}
	}
}

// OptionalAuthMiddleware adds user to context if authenticated, but doesn't require it
func OptionalAuthMiddleware(sessionMgr SessionManagerInterface, logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()

			// Get session ID from cookie
			sessionID := sessionMgr.GetSessionFromCookie(c)
			if sessionID == "" {
				// No session, continue without user context
				return next(c)
			}

			// Validate session
			session, err := sessionMgr.ValidateSession(ctx, sessionID)
			if err != nil {
				// Invalid session, clear cookie and continue
				sessionMgr.ClearSessionCookie(c)
				return next(c)
			}

			// Get user information
			storage := sessionMgr.GetStorage()
			user, err := storage.GetUserByID(ctx, session.UserID)
			if err != nil {
				logger.Error("Failed to get user information",
					zap.String("user_id", session.UserID),
					zap.Error(err))

				// Clear invalid session
				sessionMgr.DeleteSession(ctx, sessionID)
				sessionMgr.ClearSessionCookie(c)

				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "User not found",
				})
			}
			if err != nil {
				// User not found, clear session and continue
				sessionMgr.DeleteSession(ctx, sessionID)
				sessionMgr.ClearSessionCookie(c)
				return next(c)
			}

			// Add user to request context
			ctx = context.WithValue(ctx, UserContextKey, user)
			c.SetRequest(c.Request().WithContext(ctx))

			// Add user info to response headers for logging middleware
			c.Response().Header().Set("X-User-Id", user.ID)
			c.Response().Header().Set("X-User-Email", user.Email)

			logger.Debug("User authenticated (optional)",
				zap.String("user_id", user.ID),
				zap.String("email", user.Email),
				zap.String("session_id", sessionID))

			return next(c)
		}
	}
}

// GetUserFromContext retrieves the user from the request context
func GetUserFromContext(c echo.Context) (*models.User, bool) {
	user, ok := c.Request().Context().Value(UserContextKey).(*models.User)
	return user, ok
}

// RequireUser is a helper function that can be used in handlers to ensure authentication
func RequireUser(c echo.Context) (*models.User, error) {
	user, ok := GetUserFromContext(c)
	if !ok {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Authentication required")
	}
	return user, nil
}
