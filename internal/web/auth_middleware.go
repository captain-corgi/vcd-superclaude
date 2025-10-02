package web

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

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

// AuthStorage extends the interface to include PAT methods
type AuthStorage interface {
	// Sessions
	CreateUserSession(ctx context.Context, s *models.UserSession) error
	GetUserSession(ctx context.Context, sessionID string) (*models.UserSession, error)
	DeleteUserSession(ctx context.Context, sessionID string) error
	CleanupExpiredSessions(ctx context.Context) (int64, error)

	// KV for OAuth state
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// Users
	CreateOrUpdateUser(ctx context.Context, u *models.User) error
	GetUserByID(ctx context.Context, userID string) (*models.User, error)

	// Roles
	SeedRoles(ctx context.Context) error
	GetRoleByName(ctx context.Context, name string) (*models.Role, error)
	AssignRoleToUser(ctx context.Context, userID, roleID string) error
	GetUserRoles(ctx context.Context, userID string) ([]models.Role, error)
	GetUserRoleNames(ctx context.Context, userID string) ([]string, error)

	// Personal Access Tokens
	CreatePersonalAccessToken(ctx context.Context, token *models.PersonalAccessToken) (string, error)
	GetPersonalAccessTokenByPrefix(ctx context.Context, tokenPrefix string) (*models.PersonalAccessToken, error)
	UpdatePersonalAccessTokenLastUsed(ctx context.Context, tokenID string) error
	ListPersonalAccessTokens(ctx context.Context, userID string) ([]models.PersonalAccessToken, error)
	DeletePersonalAccessToken(ctx context.Context, tokenID string) error
}

// AuthRequiredMiddleware checks for valid authentication and adds user to context
func AuthRequiredMiddleware(sessionMgr SessionManagerInterface, logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()

			// Try to authenticate via Bearer token first (PAT)
			if user, err := authenticateViaBearerToken(c, sessionMgr, logger); err == nil && user != nil {
				// Add user to request context
				ctx = context.WithValue(ctx, UserContextKey, user)
				c.SetRequest(c.Request().WithContext(ctx))

				// Add user info to response headers for logging middleware
				c.Response().Header().Set("X-User-Id", user.ID)
				c.Response().Header().Set("X-User-Email", user.Email)

				logger.Debug("User authenticated via PAT",
					zap.String("user_id", user.ID),
					zap.String("email", user.Email))

				return next(c)
			}

			// Fall back to session cookie authentication
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

			// Add user to request context
			ctx = context.WithValue(ctx, UserContextKey, user)
			c.SetRequest(c.Request().WithContext(ctx))

			// Add user info to response headers for logging middleware
			c.Response().Header().Set("X-User-Id", user.ID)
			c.Response().Header().Set("X-User-Email", user.Email)

			logger.Debug("User authenticated via session",
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

// authenticateViaBearerToken attempts to authenticate using a Bearer token (PAT)
func authenticateViaBearerToken(c echo.Context, sessionMgr SessionManagerInterface, logger *zap.Logger) (*models.User, error) {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("no authorization header")
	}

	// Check if it's a Bearer token
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("not a bearer token")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return nil, fmt.Errorf("empty bearer token")
	}

	// Use first 32 characters as prefix for lookup
	if len(token) < 32 {
		return nil, fmt.Errorf("token too short")
	}

	tokenPrefix := token[:32]
	ctx := c.Request().Context()
	storage := sessionMgr.GetStorage()

	// Get token from storage
	pat, err := storage.GetPersonalAccessTokenByPrefix(ctx, tokenPrefix)
	if err != nil {
		logger.Debug("Invalid PAT prefix", zap.Error(err))
		return nil, fmt.Errorf("invalid token")
	}

	// Check if token is expired
	if pat.ExpiresAt != nil && time.Now().After(*pat.ExpiresAt) {
		logger.Debug("PAT expired", zap.String("token_id", pat.ID))
		return nil, fmt.Errorf("token expired")
	}

	// Verify token hash using bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(pat.TokenHash), []byte(token)); err != nil {
		logger.Debug("PAT hash mismatch", zap.Error(err))
		return nil, fmt.Errorf("invalid token")
	}

	// Update last used timestamp
	if err := storage.UpdatePersonalAccessTokenLastUsed(ctx, pat.ID); err != nil {
		logger.Warn("Failed to update PAT last used", zap.Error(err))
		// Don't fail authentication for this error
	}

	// Get user information
	user, err := storage.GetUserByID(ctx, pat.UserID)
	if err != nil {
		logger.Error("Failed to get user for PAT",
			zap.String("user_id", pat.UserID),
			zap.Error(err))
		return nil, fmt.Errorf("user not found")
	}

	return user, nil
}

// RequireRolesMiddleware checks if the authenticated user has any of the required roles
func RequireRolesMiddleware(sessionMgr SessionManagerInterface, logger *zap.Logger, requiredRoles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()

			// Get the user from context (set by AuthRequiredMiddleware)
			user, ok := GetUserFromContext(c)
			if !ok {
				logger.Debug("No user in context for role check")
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Authentication required",
				})
			}

			// Get user's roles from storage
			storage := sessionMgr.GetStorage()
			userRoles, err := storage.GetUserRoleNames(ctx, user.ID)
			if err != nil {
				logger.Error("Failed to get user roles",
					zap.String("user_id", user.ID),
					zap.Error(err))
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to verify permissions",
				})
			}

			// Check if user has any of the required roles
			hasRequiredRole := false
			for _, userRole := range userRoles {
				for _, requiredRole := range requiredRoles {
					if userRole == requiredRole {
						hasRequiredRole = true
						break
					}
				}
				if hasRequiredRole {
					break
				}
			}

			if !hasRequiredRole {
				logger.Warn("User lacks required roles",
					zap.String("user_id", user.ID),
					zap.Strings("user_roles", userRoles),
					zap.Strings("required_roles", requiredRoles))

				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "Insufficient permissions",
				})
			}

			logger.Debug("User authorized with required roles",
				zap.String("user_id", user.ID),
				zap.Strings("user_roles", userRoles),
				zap.Strings("required_roles", requiredRoles))

			return next(c)
		}
	}
}

// HasRole is a helper function to check if a user has a specific role
func HasRole(user *models.User, role string, storage AuthStorage, ctx context.Context) (bool, error) {
	userRoles, err := storage.GetUserRoleNames(ctx, user.ID)
	if err != nil {
		return false, err
	}

	for _, userRole := range userRoles {
		if userRole == role {
			return true, nil
		}
	}

	return false, nil
}

// RequireRole is a helper function that can be used in handlers to ensure a specific role
func RequireRole(c echo.Context, role string) error {
	// For this helper, we need to get the storage from the session manager
	// This is a bit hacky, but we'll need to pass it through context or use a different approach
	// For now, return an error asking to use RequireRolesMiddleware instead
	return echo.NewHTTPError(http.StatusBadRequest, "Use RequireRolesMiddleware instead of RequireRole helper")
}
