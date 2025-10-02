package web

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/captain-corgi/multi-agent-system/pkg/audit"
	"github.com/captain-corgi/multi-agent-system/pkg/models"
)

// PATHandlers handles personal access token admin endpoints
type PATHandlers struct {
	sessionMgr  *SessionManager
	storage     AuthStorage
	logger      *zap.Logger
	auditHelper *AuditHelper
}

// NewPATHandlers creates new PAT handlers
func NewPATHandlers(sessionMgr *SessionManager, storage AuthStorage, logger *zap.Logger, auditHelper *AuditHelper) *PATHandlers {
	return &PATHandlers{
		sessionMgr:  sessionMgr,
		storage:     storage,
		logger:      logger,
		auditHelper: auditHelper,
	}
}

// CreatePATHandler creates a new personal access token for a user
// POST /api/admin/tokens
func (h *PATHandlers) CreatePATHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Get current user (set by auth middleware)
	user, ok := GetUserFromContext(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	var req struct {
		UserID    string     `json:"user_id" validate:"required"`
		Name      string     `json:"name" validate:"required,min=1,max=255"`
		ExpiresAt *time.Time `json:"expires_at,omitempty"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Validation failed",
		})
	}

	// Create the token
	token := &models.PersonalAccessToken{
		UserID:    req.UserID,
		Name:      req.Name,
		ExpiresAt: req.ExpiresAt,
	}

	plainToken, err := h.storage.CreatePersonalAccessToken(ctx, token)
	if err != nil {
		h.logger.Error("Failed to create PAT",
			zap.String("user_id", req.UserID),
			zap.String("token_name", req.Name),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create token",
		})
	}

	// Record audit event
	actor := audit.ActorInfo{
		ID:    user.ID,
		Email: user.Email,
	}

	success := true
	h.auditHelper.RecordCustomEvent(
		"auth",
		"create_pat",
		actor,
		c.RealIP(),
		nil,
		&success,
		map[string]interface{}{
			"target_user_id": req.UserID,
			"token_name":     req.Name,
			"token_id":       token.ID,
		},
	)

	h.logger.Info("PAT created successfully",
		zap.String("token_id", token.ID),
		zap.String("user_id", req.UserID),
		zap.String("created_by", user.ID))

	// Return the token (only time it will be shown in plain text)
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"token":      plainToken,
		"token_id":   token.ID,
		"name":       token.Name,
		"created_at": token.CreatedAt,
		"expires_at": token.ExpiresAt,
		"warning":    "This token will only be shown once. Store it securely.",
	})
}

// ListPATHandler lists personal access tokens for a user
// GET /api/admin/tokens?user_id=...
func (h *PATHandlers) ListPATHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Verify authentication (user is set by auth middleware)
	if _, ok := GetUserFromContext(c); !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	userID := c.QueryParam("user_id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user_id parameter is required",
		})
	}

	tokens, err := h.storage.ListPersonalAccessTokens(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to list PATs",
			zap.String("user_id", userID),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to list tokens",
		})
	}

	h.logger.Debug("PATs listed successfully",
		zap.String("user_id", userID),
		zap.Int("token_count", len(tokens)))

	return c.JSON(http.StatusOK, map[string]interface{}{
		"tokens": tokens,
		"count":  len(tokens),
	})
}

// DeletePATHandler deletes a personal access token
// DELETE /api/admin/tokens/:id
func (h *PATHandlers) DeletePATHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Get current user (set by auth middleware)
	user, ok := GetUserFromContext(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	tokenID := c.Param("id")
	if tokenID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Token ID is required",
		})
	}

	// Get token details before deletion for audit
	var targetUserID string
	var tokenName string
	if token, err := h.storage.GetPersonalAccessTokenByPrefix(ctx, tokenID); err == nil {
		targetUserID = token.UserID
		tokenName = token.Name
	}

	err := h.storage.DeletePersonalAccessToken(ctx, tokenID)
	if err != nil {
		h.logger.Error("Failed to delete PAT",
			zap.String("token_id", tokenID),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete token",
		})
	}

	// Record audit event
	actor := audit.ActorInfo{
		ID:    user.ID,
		Email: user.Email,
	}

	success := true
	h.auditHelper.RecordCustomEvent(
		"auth",
		"delete_pat",
		actor,
		c.RealIP(),
		nil,
		&success,
		map[string]interface{}{
			"target_user_id": targetUserID,
			"token_name":     tokenName,
			"token_id":       tokenID,
		},
	)

	h.logger.Info("PAT deleted successfully",
		zap.String("token_id", tokenID),
		zap.String("deleted_by", user.ID))

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Token deleted successfully",
	})
}
