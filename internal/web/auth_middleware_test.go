package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	"github.com/captain-corgi/multi-agent-system/pkg/config"
	"github.com/captain-corgi/multi-agent-system/pkg/models"
)

func TestAuthRequiredMiddleware_NoCookie(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockStorage := &MockStorage{}
	sessionMgr := NewSessionManager(mockStorage, &config.AuthConfig{}, logger)

	e := echo.New()
	middleware := AuthRequiredMiddleware(sessionMgr, logger)

	nextCalled := false
	e.GET("/api/test", middleware(func(c echo.Context) error {
		nextCalled = true
		return c.String(http.StatusOK, "success")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.False(t, nextCalled, "Next handler should not be called")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Authentication required")
}

func TestAuthRequiredMiddleware_InvalidSession(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockSessionMgr := &MockSessionManager{}

	// Mock GetSessionFromCookie to return a session ID
	mockSessionMgr.GetSessionFromCookieFunc = func(c echo.Context) string {
		return "invalid_session_id"
	}

	// Mock ValidateSession to return error
	mockSessionMgr.ValidateSessionFunc = func(ctx context.Context, sessionID string) (*models.UserSession, error) {
		return nil, assert.AnError
	}

	// Expect ClearSessionCookie to be called
	mockSessionMgr.On("ClearSessionCookie", mock.Anything).Return()

	e := echo.New()
	middleware := AuthRequiredMiddleware(mockSessionMgr, logger)

	nextCalled := false
	e.GET("/api/test", middleware(func(c echo.Context) error {
		nextCalled = true
		return c.String(http.StatusOK, "success")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.AddCookie(&http.Cookie{Name: "mas_session", Value: "invalid_session_id"})
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.False(t, nextCalled, "Next handler should not be called")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid or expired session")
}

func TestAuthRequiredMiddleware_ValidSession(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockStorage := &MockStorage{}
	mockSessionMgr := &MockSessionManager{}

	userID := "user123"
	sessionID := "valid_session_id"

	// Mock GetSessionFromCookie to return a session ID
	mockSessionMgr.GetSessionFromCookieFunc = func(c echo.Context) string {
		return sessionID
	}

	// Mock ValidateSession to return valid session
	mockSessionMgr.ValidateSessionFunc = func(ctx context.Context, sid string) (*models.UserSession, error) {
		return &models.UserSession{
			SessionID: sid,
			UserID:    userID,
			ExpiresAt: time.Now().Add(time.Hour),
		}, nil
	}

	// Mock GetStorage to return mock storage
	mockSessionMgr.On("GetStorage").Return(mockStorage)

	// Mock GetUserByID to return user
	mockStorage.On("GetUserByID", mock.Anything, userID).Return(&models.User{
		ID:    userID,
		Email: "test@example.com",
		Name:  "Test User",
	}, nil)

	e := echo.New()
	middleware := AuthRequiredMiddleware(mockSessionMgr, logger)

	userFromContext := ""
	e.GET("/api/test", middleware(func(c echo.Context) error {
		user, ok := GetUserFromContext(c)
		if ok {
			userFromContext = user.ID
		}
		return c.String(http.StatusOK, "success")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.AddCookie(&http.Cookie{Name: "mas_session", Value: sessionID})
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "success")
	assert.Equal(t, userID, userFromContext, "User should be in context")

	mockStorage.AssertExpectations(t)
}

func TestGetUserFromContext(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// No user in context
	user, ok := GetUserFromContext(c)
	assert.False(t, ok)
	assert.Nil(t, user)

	// Add user to context
	user = &models.User{ID: "user123", Email: "test@example.com"}
	ctx := context.WithValue(c.Request().Context(), UserContextKey, user)
	c.SetRequest(c.Request().WithContext(ctx))

	user, ok = GetUserFromContext(c)
	assert.True(t, ok)
	assert.Equal(t, "user123", user.ID)
}

func TestRequireUser(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// No user in context - should return error
	user, err := RequireUser(c)
	assert.Error(t, err)
	assert.Nil(t, user)

	// Add user to context
	user = &models.User{ID: "user123", Email: "test@example.com"}
	ctx := context.WithValue(c.Request().Context(), UserContextKey, user)
	c.SetRequest(c.Request().WithContext(ctx))

	user, err = RequireUser(c)
	assert.NoError(t, err)
	assert.Equal(t, "user123", user.ID)
}
