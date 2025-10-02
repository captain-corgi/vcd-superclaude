package web

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	"github.com/captain-corgi/multi-agent-system/pkg/config"
	"github.com/captain-corgi/multi-agent-system/pkg/models"
)

// MockStorage is a mock implementation of AuthStorage for testing
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) CreateUserSession(ctx context.Context, session *models.UserSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockStorage) GetUserSession(ctx context.Context, sessionID string) (*models.UserSession, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserSession), args.Error(1)
}

func (m *MockStorage) DeleteUserSession(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockStorage) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockStorage) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockStorage) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) Del(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockStorage) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockStorage) CreateOrUpdateUser(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockStorage) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockStorage) GetUserByGoogleSub(ctx context.Context, googleSub string) (*models.User, error) {
	args := m.Called(ctx, googleSub)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// MockSessionManager is a mock session manager for testing middleware
type MockSessionManager struct {
	mock.Mock
	GetSessionFromCookieFunc func(c echo.Context) string
	ValidateSessionFunc      func(ctx context.Context, sessionID string) (*models.UserSession, error)
}

func (m *MockSessionManager) CreateSession(ctx context.Context, userID string) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *MockSessionManager) GetSession(ctx context.Context, sessionID string) (*models.UserSession, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserSession), args.Error(1)
}

func (m *MockSessionManager) DeleteSession(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockSessionManager) SetSessionCookie(c echo.Context, sessionID string) {
	m.Called(c, sessionID)
}

func (m *MockSessionManager) ClearSessionCookie(c echo.Context) {
	m.Called(c)
}

func (m *MockSessionManager) GetSessionFromCookie(c echo.Context) string {
	if m.GetSessionFromCookieFunc != nil {
		return m.GetSessionFromCookieFunc(c)
	}
	args := m.Called(c)
	return args.String(0)
}

func (m *MockSessionManager) ValidateSession(ctx context.Context, sessionID string) (*models.UserSession, error) {
	if m.ValidateSessionFunc != nil {
		return m.ValidateSessionFunc(ctx, sessionID)
	}
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserSession), args.Error(1)
}

func (m *MockSessionManager) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSessionManager) generateSessionID() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockSessionManager) isValidHex(s string) bool {
	args := m.Called(s)
	return args.Bool(0)
}

func (m *MockSessionManager) parseSameSite(sameSite string) http.SameSite {
	args := m.Called(sameSite)
	return args.Get(0).(http.SameSite)
}

func (m *MockSessionManager) GetStorage() AuthStorage {
	args := m.Called()
	return args.Get(0).(AuthStorage)
}

func TestNewSessionManager(t *testing.T) {
	logger := zaptest.NewLogger(t)
	authConfig := &config.AuthConfig{
		Session: config.SessionConfig{
			CookieName: "test_session",
			TTL:        time.Hour,
		},
	}

	mockStorage := &MockStorage{}
	sessionMgr := NewSessionManager(mockStorage, authConfig, logger)

	assert.NotNil(t, sessionMgr)
	assert.Equal(t, "test_session", sessionMgr.cookieName)
}

func TestSessionManager_CreateSession(t *testing.T) {
	logger := zaptest.NewLogger(t)
	authConfig := &config.AuthConfig{
		Session: config.SessionConfig{
			TTL: time.Hour,
		},
	}

	mockStorage := &MockStorage{}
	sessionMgr := NewSessionManager(mockStorage, authConfig, logger)

	ctx := context.Background()
	userID := "user123"

	// Mock the CreateUserSession call
	mockStorage.On("CreateUserSession", ctx, mock.AnythingOfType("*models.UserSession")).Return(nil)

	sessionID, err := sessionMgr.CreateSession(ctx, userID)

	assert.NoError(t, err)
	assert.NotEmpty(t, sessionID)
	assert.Len(t, sessionID, 64) // 32 bytes hex encoded

	mockStorage.AssertExpectations(t)
}

func TestSessionManager_GetSession(t *testing.T) {
	logger := zaptest.NewLogger(t)
	authConfig := &config.AuthConfig{}
	mockStorage := &MockStorage{}
	sessionMgr := NewSessionManager(mockStorage, authConfig, logger)

	ctx := context.Background()
	sessionID := "test_session_id"
	expectedSession := &models.UserSession{
		SessionID: sessionID,
		UserID:    "user123",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	mockStorage.On("GetUserSession", ctx, sessionID).Return(expectedSession, nil)

	session, err := sessionMgr.GetSession(ctx, sessionID)

	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, expectedSession.SessionID, session.SessionID)
	assert.Equal(t, expectedSession.UserID, session.UserID)

	mockStorage.AssertExpectations(t)
}

func TestSessionManager_ValidateSession_Valid(t *testing.T) {
	logger := zaptest.NewLogger(t)
	authConfig := &config.AuthConfig{}
	mockStorage := &MockStorage{}
	sessionMgr := NewSessionManager(mockStorage, authConfig, logger)

	ctx := context.Background()
	sessionID := "test_session_id"
	futureTime := time.Now().Add(time.Hour)
	expectedSession := &models.UserSession{
		SessionID: sessionID,
		UserID:    "user123",
		ExpiresAt: futureTime,
	}

	mockStorage.On("GetUserSession", ctx, sessionID).Return(expectedSession, nil)

	session, err := sessionMgr.ValidateSession(ctx, sessionID)

	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, expectedSession, session)

	mockStorage.AssertExpectations(t)
}

func TestSessionManager_ValidateSession_Expired(t *testing.T) {
	logger := zaptest.NewLogger(t)
	authConfig := &config.AuthConfig{}
	mockStorage := &MockStorage{}
	sessionMgr := NewSessionManager(mockStorage, authConfig, logger)

	ctx := context.Background()
	sessionID := "test_session_id"
	pastTime := time.Now().Add(-time.Hour)
	expiredSession := &models.UserSession{
		SessionID: sessionID,
		UserID:    "user123",
		ExpiresAt: pastTime,
	}

	mockStorage.On("GetUserSession", ctx, sessionID).Return(expiredSession, nil)
	mockStorage.On("DeleteUserSession", ctx, sessionID).Return(nil)

	session, err := sessionMgr.ValidateSession(ctx, sessionID)

	assert.Error(t, err)
	assert.Nil(t, session)
	assert.Contains(t, err.Error(), "session expired")

	mockStorage.AssertExpectations(t)
}

func TestSessionManager_DeleteSession(t *testing.T) {
	logger := zaptest.NewLogger(t)
	authConfig := &config.AuthConfig{}
	mockStorage := &MockStorage{}
	sessionMgr := NewSessionManager(mockStorage, authConfig, logger)

	ctx := context.Background()
	sessionID := "test_session_id"

	mockStorage.On("DeleteUserSession", ctx, sessionID).Return(nil)

	err := sessionMgr.DeleteSession(ctx, sessionID)

	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

func TestSessionManager_CleanupExpiredSessions(t *testing.T) {
	logger := zaptest.NewLogger(t)
	authConfig := &config.AuthConfig{}
	mockStorage := &MockStorage{}
	sessionMgr := NewSessionManager(mockStorage, authConfig, logger)

	ctx := context.Background()

	mockStorage.On("CleanupExpiredSessions", ctx).Return(int64(5), nil)

	count, err := sessionMgr.CleanupExpiredSessions(ctx)

	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)
	mockStorage.AssertExpectations(t)
}

func TestSessionManager_GenerateSessionID(t *testing.T) {
	logger := zaptest.NewLogger(t)
	authConfig := &config.AuthConfig{}
	mockStorage := &MockStorage{}
	sessionMgr := NewSessionManager(mockStorage, authConfig, logger)

	sessionID, err := sessionMgr.generateSessionID()

	assert.NoError(t, err)
	assert.Len(t, sessionID, 64) // 32 bytes hex encoded
}

func TestSessionManager_IsValidHex(t *testing.T) {
	logger := zaptest.NewLogger(t)
	authConfig := &config.AuthConfig{}
	mockStorage := &MockStorage{}
	sessionMgr := NewSessionManager(mockStorage, authConfig, logger)

	assert.True(t, sessionMgr.isValidHex("0123456789abcdef"))
	assert.True(t, sessionMgr.isValidHex("0123456789ABCDEF"))
	assert.False(t, sessionMgr.isValidHex("gggggggggggggggg"))
	assert.False(t, sessionMgr.isValidHex("0123456789abcdefg")) // too long
}
