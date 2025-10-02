package web

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/captain-corgi/multi-agent-system/internal/web/oidc"
	"github.com/captain-corgi/multi-agent-system/pkg/audit"
	"github.com/captain-corgi/multi-agent-system/pkg/models"
)

// AuthHandlers handles authentication routes
type AuthHandlers struct {
	oidcProvider oidc.OIDCProvider
	sessionMgr   *SessionManager
	storage      interface{} // For future use if needed
	logger       *zap.Logger
	auditHelper  *AuditHelper
}

// NewAuthHandlers creates new auth handlers
func NewAuthHandlers(oidcProvider oidc.OIDCProvider, sessionMgr *SessionManager, storage interface{}, logger *zap.Logger, auditHelper *AuditHelper) *AuthHandlers {
	return &AuthHandlers{
		oidcProvider: oidcProvider,
		sessionMgr:   sessionMgr,
		storage:      storage,
		logger:       logger,
		auditHelper:  auditHelper,
	}
}

// GoogleLoginHandler initiates the Google OAuth flow
// GET /auth/google/login
func (h *AuthHandlers) GoogleLoginHandler(c echo.Context) error {
	// Generate a random state parameter for CSRF protection
	state, err := h.generateState()
	if err != nil {
		h.logger.Error("Failed to generate state", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to initiate login",
		})
	}

	// Store state in Redis with expiration (5 minutes)
	ctx := c.Request().Context()
	stateKey := fmt.Sprintf("oauth_state:%s", state)
	storage := h.sessionMgr.GetStorage()
	err = storage.Set(ctx, stateKey, "valid", 5*time.Minute)
	if err != nil {
		h.logger.Error("Failed to store state", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to initiate login",
		})
	}

	// Generate auth URL
	authURL := h.oidcProvider.GetAuthURL(state)

	h.logger.Info("Google login initiated",
		zap.String("state", state),
		zap.String("auth_url", authURL))

	// Redirect to Google
	return c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// GoogleCallbackHandler handles the OAuth callback from Google
// GET /auth/google/callback
func (h *AuthHandlers) GoogleCallbackHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Get parameters
	code := c.QueryParam("code")
	state := c.QueryParam("state")
	errorParam := c.QueryParam("error")

	// Check for errors
	if errorParam != "" {
		errorDescription := c.QueryParam("error_description")
		h.logger.Error("OAuth error from Google",
			zap.String("error", errorParam),
			zap.String("description", errorDescription))

		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "OAuth authentication failed",
		})
	}

	// Validate state parameter
	if state == "" {
		h.logger.Error("Missing state parameter")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid authentication request",
		})
	}

	// Verify state exists and is valid
	stateKey := fmt.Sprintf("oauth_state:%s", state)
	storage := h.sessionMgr.GetStorage()
	stateValid, err := storage.Exists(ctx, stateKey)
	if err != nil || !stateValid {
		h.logger.Error("Invalid or expired state",
			zap.String("state", state),
			zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid authentication request",
		})
	}

	// Delete the state from Redis
	err = storage.Del(ctx, stateKey)
	if err != nil {
		h.logger.Error("Failed to delete state", zap.Error(err))
	}

	// Exchange code for tokens
	tokenInfo, err := h.oidcProvider.ExchangeCode(ctx, code)
	if err != nil {
		h.logger.Error("Failed to exchange code",
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Authentication failed",
		})
	}

	// Verify and parse ID token
	userInfo, err := h.oidcProvider.VerifyToken(ctx, tokenInfo.IDToken)
	if err != nil {
		h.logger.Error("Failed to verify ID token",
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Authentication failed",
		})
	}

	// Create or update user in database
	user := &models.User{
		GoogleSub:  userInfo.Subject,
		Email:      userInfo.Email,
		Name:       userInfo.Name,
		PictureURL: userInfo.Picture,
	}

	err = storage.CreateOrUpdateUser(ctx, user)
	if err != nil {
		h.logger.Error("Failed to create/update user",
			zap.String("google_sub", user.GoogleSub),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to process user",
		})
	}

	// Create session
	sessionID, err := h.sessionMgr.CreateSession(ctx, user.ID)
	if err != nil {
		h.logger.Error("Failed to create session",
			zap.String("user_id", user.ID),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create session",
		})
	}

	// Set session cookie
	h.sessionMgr.SetSessionCookie(c, sessionID)

	// Record audit event
	actor := audit.ActorInfo{
		ID:    user.ID,
		Email: user.Email,
	}

	success := true
	h.auditHelper.RecordCustomEvent(
		"auth",
		"login",
		actor,
		c.RealIP(),
		nil,      // request_id
		&success, // success
		map[string]interface{}{
			"provider": h.oidcProvider.GetProviderName(),
			"method":   "oauth",
		},
	)

	h.logger.Info("User authenticated successfully",
		zap.String("user_id", user.ID),
		zap.String("email", user.Email),
		zap.String("session_id", sessionID))

	// Redirect to frontend or return success
	redirectURL := c.QueryParam("redirect_url")
	if redirectURL == "" {
		redirectURL = "/" // Default redirect
	}

	// Validate redirect URL to prevent open redirects
	if !h.isValidRedirectURL(redirectURL) {
		redirectURL = "/"
	}

	return c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// LogoutHandler clears the user session
// POST /auth/logout
func (h *AuthHandlers) LogoutHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Get session ID from cookie
	sessionID := h.sessionMgr.GetSessionFromCookie(c)
	if sessionID != "" {
		// Delete session from database
		err := h.sessionMgr.DeleteSession(ctx, sessionID)
		if err != nil {
			h.logger.Error("Failed to delete session", zap.Error(err))
		}

		// Get user info for audit
		session, err := h.sessionMgr.ValidateSession(ctx, sessionID)
		if err == nil && session != nil {
			var user *models.User
			storage := h.sessionMgr.GetStorage()
			user, err = storage.GetUserByID(ctx, session.UserID)
			if err == nil && user != nil {
				actor := audit.ActorInfo{
					ID:    user.ID,
					Email: user.Email,
				}

				success := true
				h.auditHelper.RecordCustomEvent(
					"auth",
					"logout",
					actor,
					c.RealIP(),
					nil,
					&success,
					map[string]interface{}{
						"session_id": sessionID,
					},
				)
			}
		}
	}

	// Clear session cookie
	h.sessionMgr.ClearSessionCookie(c)

	h.logger.Info("User logged out", zap.String("session_id", sessionID))

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Logged out successfully",
	})
}

// MeHandler returns the current user's information
// GET /api/me
func (h *AuthHandlers) MeHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Get session ID from cookie
	sessionID := h.sessionMgr.GetSessionFromCookie(c)
	if sessionID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Not authenticated",
		})
	}

	// Validate session
	session, err := h.sessionMgr.ValidateSession(ctx, sessionID)
	if err != nil {
		h.logger.Debug("Invalid session in /me", zap.Error(err))
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid session",
		})
	}

	// Get user information
	var user *models.User
	storage := h.sessionMgr.GetStorage()
	user, err = storage.GetUserByID(ctx, session.UserID)
	if err != nil {
		h.logger.Error("Failed to get user for /me",
			zap.String("user_id", session.UserID),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get user information",
		})
	}

	return c.JSON(http.StatusOK, user)
}

// Helper functions

func (h *AuthHandlers) generateState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (h *AuthHandlers) isValidRedirectURL(redirectURL string) bool {
	// Parse the URL
	u, err := url.Parse(redirectURL)
	if err != nil {
		return false
	}

	// Only allow relative URLs or same origin
	if u.IsAbs() {
		// For now, only allow relative URLs to prevent open redirects
		// In production, you might want to allow specific domains
		return false
	}

	// Allow relative URLs that start with /
	return strings.HasPrefix(redirectURL, "/")
}
