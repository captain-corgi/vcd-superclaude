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

	"github.com/captain-corgi/multi-agent-system/pkg/models"
)

// MockAuthStorage implements AuthStorage for testing
type MockAuthStorage struct {
	mock.Mock
}

func (m *MockAuthStorage) CreateUserSession(ctx context.Context, s *models.UserSession) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}

func (m *MockAuthStorage) GetUserSession(ctx context.Context, sessionID string) (*models.UserSession, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).(*models.UserSession), args.Error(1)
}

func (m *MockAuthStorage) DeleteUserSession(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockAuthStorage) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAuthStorage) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockAuthStorage) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockAuthStorage) Del(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockAuthStorage) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockAuthStorage) CreateOrUpdateUser(ctx context.Context, u *models.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *MockAuthStorage) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthStorage) SeedRoles(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockAuthStorage) GetRoleByName(ctx context.Context, name string) (*models.Role, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockAuthStorage) AssignRoleToUser(ctx context.Context, userID, roleID string) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *MockAuthStorage) GetUserRoles(ctx context.Context, userID string) ([]models.Role, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.Role), args.Error(1)
}

func (m *MockAuthStorage) GetUserRoleNames(ctx context.Context, userID string) ([]string, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockAuthStorage) CreatePersonalAccessToken(ctx context.Context, token *models.PersonalAccessToken) (string, error) {
	args := m.Called(ctx, token)
	return args.String(0), args.Error(1)
}

func (m *MockAuthStorage) GetPersonalAccessTokenByPrefix(ctx context.Context, tokenPrefix string) (*models.PersonalAccessToken, error) {
	args := m.Called(ctx, tokenPrefix)
	return args.Get(0).(*models.PersonalAccessToken), args.Error(1)
}

func (m *MockAuthStorage) UpdatePersonalAccessTokenLastUsed(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *MockAuthStorage) ListPersonalAccessTokens(ctx context.Context, userID string) ([]models.PersonalAccessToken, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.PersonalAccessToken), args.Error(1)
}

func (m *MockAuthStorage) DeletePersonalAccessToken(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

// MockSessionManager implements SessionManagerInterface for testing
type MockSessionManager struct {
	mock.Mock
	storage *MockAuthStorage
}

func (m *MockSessionManager) GetSessionFromCookie(c echo.Context) string {
	args := m.Called(c)
	return args.String(0)
}

func (m *MockSessionManager) ValidateSession(ctx context.Context, sessionID string) (*models.UserSession, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).(*models.UserSession), args.Error(1)
}

func (m *MockSessionManager) DeleteSession(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockSessionManager) ClearSessionCookie(c echo.Context) {
	m.Called(c)
}

func (m *MockSessionManager) GetStorage() AuthStorage {
	return m.storage
}

func TestAuthRequiredMiddleware_SessionAuthentication(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockStorage := &MockAuthStorage{}
	mockSessionMgr := &MockSessionManager{storage: mockStorage}

	// Setup mocks
	sessionID := "test_session_id"
	userID := "test_user_id"
	user := &models.User{
		ID:    userID,
		Email: "test@example.com",
		Name:  "Test User",
	}

	session := &models.UserSession{
		SessionID: sessionID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	mockSessionMgr.On("GetSessionFromCookie", mock.AnythingOfType("*echo.context")).Return(sessionID)
	mockSessionMgr.On("ValidateSession", mock.Anything, sessionID).Return(session, nil)
	mockStorage.On("GetUserByID", mock.Anything, userID).Return(user, nil)

	// Create middleware
	middleware := AuthRequiredMiddleware(mockSessionMgr, logger)

	// Create test request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Add session cookie
	c.Request().AddCookie(&http.Cookie{Name: "mas_session", Value: sessionID})

	// Execute middleware
	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "success"})
	})

	err := handler(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "success")

	// Verify mocks
	mockSessionMgr.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestAuthRequiredMiddleware_PATAuthentication(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockStorage := &MockAuthStorage{}
	mockSessionMgr := &MockSessionManager{storage: mockStorage}

	// Setup mocks
	userID := "test_user_id"
	user := &models.User{
		ID:    userID,
		Email: "test@example.com",
		Name:  "Test User",
	}

	tokenPrefix := "test_token_prefix"
	plainToken := tokenPrefix + "12345678901234567890123456789012" // 64 hex chars
	pat := &models.PersonalAccessToken{
		ID:          "pat_id",
		UserID:      userID,
		Name:        "Test Token",
		TokenPrefix: tokenPrefix,
		TokenHash:   "hashed_token", // Mock won't actually verify bcrypt
		ExpiresAt:   &[]time.Time{time.Now().Add(time.Hour)}[0],
	}

	mockStorage.On("GetPersonalAccessTokenByPrefix", mock.Anything, tokenPrefix).Return(pat, nil)
	mockStorage.On("UpdatePersonalAccessTokenLastUsed", mock.Anything, "pat_id").Return(nil)
	mockStorage.On("GetUserByID", mock.Anything, userID).Return(user, nil)

	// Create middleware
	middleware := AuthRequiredMiddleware(mockSessionMgr, logger)

	// Create test request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+plainToken)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute middleware
	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "success"})
	})

	err := handler(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "success")

	// Verify mocks
	mockStorage.AssertExpectations(t)
}

func TestAuthRequiredMiddleware_NoAuthentication(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockStorage := &MockAuthStorage{}
	mockSessionMgr := &MockSessionManager{storage: mockStorage}

	// Setup mocks - no session or token
	mockSessionMgr.On("GetSessionFromCookie", mock.AnythingOfType("*echo.context")).Return("")

	// Create middleware
	middleware := AuthRequiredMiddleware(mockSessionMgr, logger)

	// Create test request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute middleware
	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "success"})
	})

	err := handler(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// Verify mocks
	mockSessionMgr.AssertExpectations(t)
}

func TestRequireRolesMiddleware_AdminRole(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockStorage := &MockAuthStorage{}
	mockSessionMgr := &MockSessionManager{storage: mockStorage}

	// Setup mocks
	userID := "test_user_id"
	user := &models.User{ID: userID, Email: "admin@example.com"}

	mockStorage.On("GetUserRoleNames", mock.Anything, userID).Return([]string{"admin", "viewer"}, nil)

	// Create middleware
	middleware := RequireRolesMiddleware(mockSessionMgr, logger, "admin")

	// Create test request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set user in context
	ctx := context.WithValue(req.Context(), UserContextKey, user)
	c.SetRequest(req.WithContext(ctx))

	// Execute middleware
	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "success"})
	})

	err := handler(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "success")

	// Verify mocks
	mockStorage.AssertExpectations(t)
}

func TestRequireRolesMiddleware_InsufficientRole(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockStorage := &MockAuthStorage{}
	mockSessionMgr := &MockSessionManager{storage: mockStorage}

	// Setup mocks
	userID := "test_user_id"
	user := &models.User{ID: userID, Email: "user@example.com"}

	mockStorage.On("GetUserRoleNames", mock.Anything, userID).Return([]string{"viewer"}, nil)

	// Create middleware
	middleware := RequireRolesMiddleware(mockSessionMgr, logger, "admin")

	// Create test request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set user in context
	ctx := context.WithValue(req.Context(), UserContextKey, user)
	c.SetRequest(req.WithContext(ctx))

	// Execute middleware
	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "success"})
	})

	err := handler(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// Verify mocks
	mockStorage.AssertExpectations(t)
}

func TestAuthenticateViaBearerToken_ValidToken(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockStorage := &MockAuthStorage{}
	mockSessionMgr := &MockSessionManager{storage: mockStorage}

	// Setup mocks
	userID := "test_user_id"
	user := &models.User{ID: userID, Email: "test@example.com"}

	tokenPrefix := "test_token_prefix"
	pat := &models.PersonalAccessToken{
		ID:          "pat_id",
		UserID:      userID,
		TokenPrefix: tokenPrefix,
		TokenHash:   "$2a$10$test.hash", // bcrypt hash placeholder
		ExpiresAt:   &[]time.Time{time.Now().Add(time.Hour)}[0],
	}

	mockStorage.On("GetPersonalAccessTokenByPrefix", mock.Anything, tokenPrefix).Return(pat, nil)
	mockStorage.On("UpdatePersonalAccessTokenLastUsed", mock.Anything, "pat_id").Return(nil)
	mockStorage.On("GetUserByID", mock.Anything, userID).Return(user, nil)

	// Create test request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPrefix+"12345678901234567890123456789012")
	c := e.NewContext(req, nil)

	// Execute function
	result, err := authenticateViaBearerToken(c, mockSessionMgr, logger)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, userID, result.ID)
	assert.Equal(t, "test@example.com", result.Email)

	// Verify mocks
	mockStorage.AssertExpectations(t)
}

func TestAuthenticateViaBearerToken_InvalidToken(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockStorage := &MockAuthStorage{}
	mockSessionMgr := &MockSessionManager{storage: mockStorage}

	// Setup mocks
	mockStorage.On("GetPersonalAccessTokenByPrefix", mock.Anything, "invalid_prefix").Return((*models.PersonalAccessToken)(nil), assert.AnError)

	// Create test request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid_prefix12345678901234567890123456789012")
	c := e.NewContext(req, nil)

	// Execute function
	result, err := authenticateViaBearerToken(c, mockSessionMgr, logger)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)

	// Verify mocks
	mockStorage.AssertExpectations(t)
}

func TestGetUserFromContext(t *testing.T) {
	user := &models.User{ID: "test_id", Email: "test@example.com"}

	// Create test request with user in context
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), UserContextKey, user)
	req = req.WithContext(ctx)
	c := e.NewContext(req, nil)

	// Test getting user from context
	result, ok := GetUserFromContext(c)

	// Assert
	assert.True(t, ok)
	assert.NotNil(t, result)
	assert.Equal(t, "test_id", result.ID)
	assert.Equal(t, "test@example.com", result.Email)
}

func TestRequireUser(t *testing.T) {
	user := &models.User{ID: "test_id", Email: "test@example.com"}

	// Create test request with user in context
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), UserContextKey, user)
	req = req.WithContext(ctx)
	c := e.NewContext(req, nil)

	// Test requiring user
	result, err := RequireUser(c)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test_id", result.ID)
	assert.Equal(t, "test@example.com", result.Email)
}
