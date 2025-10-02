package web

import (
	"context"
	"time"

	"github.com/captain-corgi/multi-agent-system/pkg/models"
)

// AuthStorage is the minimal contract needed by auth/session code.
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
}
