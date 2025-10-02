<!-- 5b27d6dc-a0f6-4ef0-8589-eff0385e2e87 50686ac3-a93f-44c7-8c39-adb499b5bd0c -->
### Goal

Make `internal/web` tests pass by removing concrete `*storage.Storage` and `interface{}` casts in auth/session code and using narrow interfaces that both real storage and test mocks implement.

### Changes

- Define storage interfaces in `internal/web` used by auth/session:
```12:60:internal/web/auth_storage.go
package web

import (
  "context"
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
```

- Update `internal/web/session.go`:
  - Change `SessionManager.storage` to `AuthStorage` (not `interface{}`)
  - Remove all type assertions; call methods directly
  - Keep `GetStorage() AuthStorage` for middleware/handlers

- Update `internal/web/auth_middleware.go`:
  - Change `SessionManagerInterface.GetStorage() AuthStorage`
  - Replace storage type assertions with direct calls to `GetStorage().GetUserByID(...)`

- Update `internal/web/auth_handlers.go`:
  - Use `s := h.sessionMgr.GetStorage()` and call `Set/Exists/Del` for state and `CreateOrUpdateUser/GetUserByID` for users

- Update `internal/web/server.go` wiring:
  - No functional change; `storage.Storage` already implements all methods used by `AuthStorage`

- Update tests:
  - In `internal/web/session_test.go` and `internal/web/auth_middleware_test.go`, change `MockStorage` to implement `AuthStorage` interface
  - Remove `NewSessionManagerWithStorage`; update `NewSessionManager` to accept `AuthStorage` instead of `*storage.Storage` so tests can pass mocks in directly
  - Adjust mocks to expect `GetStorage()` where needed

- Optional vet warnings (if in scope):
  - Avoid copying structs with `sync.RWMutex` in `pkg/metrics` and `pkg/audit` by returning pointers or value wrappers; separate PR if desired

### Acceptance Criteria

- `go test ./internal/web -v` passes locally
- No type assertions to concrete storage types remain in `internal/web` auth/session code
- No go vet errors from `internal/web` related to auth/session changes

### To-dos

- [ ] Add `AuthStorage` interface in `internal/web/auth_storage.go`
- [ ] Change `SessionManager` to use `AuthStorage`; remove type assertions
- [ ] Update `AuthRequiredMiddleware` to use `AuthStorage` via `GetStorage()`
- [ ] Use `AuthStorage` for state and users in handlers
- [ ] Update `NewSessionManager` to accept `AuthStorage`
- [ ] Modify mocks to implement `AuthStorage`; update tests to new constructor
- [ ] Run `go test ./internal/web -v` and ensure green