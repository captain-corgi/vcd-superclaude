package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/captain-corgi/multi-agent-system/pkg/audit"
	"github.com/captain-corgi/multi-agent-system/pkg/models"
)

type Storage struct {
	db     *sql.DB
	redis  *redis.Client
	logger *zap.Logger
}

type DatabaseHealth struct {
	Status    string        `json:"status"`
	Latency   time.Duration `json:"latency"`
	LastError error         `json:"last_error,omitempty"`
}

type RedisHealth struct {
	Status     string        `json:"status"`
	Latency    time.Duration `json:"latency"`
	LastError  error         `json:"last_error,omitempty"`
	Connection bool          `json:"connection"`
}

func NewStorage(dbURL, redisURL string, logger *zap.Logger) (*Storage, error) {
	storage := &Storage{
		logger: logger,
	}

	var err error

	// Connect to PostgreSQL
	if dbURL != "" {
		storage.db, err = sql.Open("postgres", dbURL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}

		// Configure connection pool
		storage.db.SetMaxOpenConns(25)
		storage.db.SetMaxIdleConns(5)
		storage.db.SetConnMaxLifetime(5 * time.Minute)

		// Test database connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		start := time.Now()
		err = storage.db.PingContext(ctx)
		storage.logger.Info("Database connection check",
			zap.Duration("latency", time.Since(start)),
			zap.Error(err))

		if err != nil {
			return nil, fmt.Errorf("failed to ping database: %w", err)
		}
	} else {
		storage.logger.Info("Database connection disabled (no URL provided)")
	}

	// Connect to Redis
	if redisURL != "" {
		storage.redis = redis.NewClient(&redis.Options{
			Addr: redisURL,
			// Redis configuration can be extended as needed
		})

		// Test Redis connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		start := time.Now()
		_, err = storage.redis.Ping(ctx).Result()
		storage.logger.Info("Redis connection check",
			zap.Duration("latency", time.Since(start)),
			zap.Error(err))

		if err != nil {
			return nil, fmt.Errorf("failed to connect to Redis: %w", err)
		}
	} else {
		storage.logger.Info("Redis connection disabled (no URL provided)")
	}

	return storage, nil
}

func (s *Storage) Close() error {
	var errs []error

	if s.db != nil {
		if err := s.db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("database close error: %w", err))
		}
	}

	if s.redis != nil {
		if err := s.redis.Close(); err != nil {
			errs = append(errs, fmt.Errorf("redis close error: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("multiple errors during close: %v", errs)
	}

	return nil
}

func (s *Storage) GetDB() *sql.DB {
	return s.db
}

func (s *Storage) GetRedis() *redis.Client {
	return s.redis
}

func (s *Storage) CheckDatabaseHealth() DatabaseHealth {
	health := DatabaseHealth{
		Status: "unknown",
	}

	if s.db == nil {
		health.Status = "disabled"
		return health
	}

	start := time.Now()
	err := s.db.Ping()
	health.Latency = time.Since(start)
	health.LastError = err

	if err != nil {
		health.Status = "unhealthy"
	} else {
		health.Status = "healthy"
	}

	return health
}

func (s *Storage) CheckRedisHealth() RedisHealth {
	health := RedisHealth{
		Status: "unknown",
	}

	if s.redis == nil {
		health.Status = "disabled"
		return health
	}

	start := time.Now()
	_, err := s.redis.Ping(context.Background()).Result()
	health.Latency = time.Since(start)
	health.LastError = err
	health.Connection = err == nil

	if err != nil {
		health.Status = "unhealthy"
	} else {
		health.Status = "healthy"
	}

	return health
}

func (s *Storage) GetStorageStats() map[string]interface{} {
	stats := map[string]interface{}{
		"database":  s.CheckDatabaseHealth(),
		"redis":     s.CheckRedisHealth(),
		"timestamp": time.Now(),
	}

	return stats
}

// Redis Helper Methods
func (s *Storage) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if s.redis == nil {
		return fmt.Errorf("redis not initialized")
	}

	return s.redis.Set(ctx, key, value, ttl).Err()
}

func (s *Storage) Get(ctx context.Context, key string) (string, error) {
	if s.redis == nil {
		return "", fmt.Errorf("redis not initialized")
	}

	return s.redis.Get(ctx, key).Result()
}

func (s *Storage) Del(ctx context.Context, key string) error {
	if s.redis == nil {
		return fmt.Errorf("redis not initialized")
	}

	return s.redis.Del(ctx, key).Err()
}

func (s *Storage) Exists(ctx context.Context, key string) (bool, error) {
	if s.redis == nil {
		return false, fmt.Errorf("redis not initialized")
	}

	result, err := s.redis.Exists(ctx, key).Result()
	return result > 0, err
}

// Database Helper Methods
func (s *Storage) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	return s.db.ExecContext(ctx, query, args...)
}

func (s *Storage) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	return s.db.QueryContext(ctx, query, args...)
}

func (s *Storage) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if s.db == nil {
		return nil
	}

	return s.db.QueryRowContext(ctx, query, args...)
}

func (s *Storage) InitializeSchema() error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Create tables schema
	schema := `
	CREATE TABLE IF NOT EXISTS requests (
		id VARCHAR(255) PRIMARY KEY,
		type VARCHAR(50) NOT NULL,
		payload JSONB NOT NULL,
		metadata JSONB,
		timestamp TIMESTAMP NOT NULL,
		source_agent VARCHAR(100) NOT NULL,
		status VARCHAR(50) DEFAULT 'pending'
	);

	CREATE TABLE IF NOT EXISTS responses (
		id VARCHAR(255) PRIMARY KEY,
		request_id VARCHAR(255) NOT NULL,
		success BOOLEAN NOT NULL,
		data JSONB,
		error TEXT,
		metadata JSONB,
		timestamp TIMESTAMP NOT NULL,
		target_agent VARCHAR(100) NOT NULL,
		FOREIGN KEY (request_id) REFERENCES requests(id)
	);

	CREATE TABLE IF NOT EXISTS messages (
		id VARCHAR(255) PRIMARY KEY,
		type VARCHAR(50) NOT NULL,
		content JSONB NOT NULL,
		from_agent VARCHAR(100) NOT NULL,
		to_agent VARCHAR(100) NOT NULL,
		timestamp TIMESTAMP NOT NULL,
		correlation_id VARCHAR(255)
	);

	CREATE TABLE IF NOT EXISTS agent_metrics (
		id SERIAL PRIMARY KEY,
		agent_id VARCHAR(100) NOT NULL,
		requests_processed BIGINT DEFAULT 0,
		requests_success BIGINT DEFAULT 0,
		requests_failed BIGINT DEFAULT 0,
		average_response_time INTERVAL,
		error_rate FLOAT,
		uptime INTERVAL,
		memory_usage BIGINT,
		timestamp TIMESTAMP NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS audit_events (
		id BIGSERIAL PRIMARY KEY,
		event_id VARCHAR(255) UNIQUE,
		category VARCHAR(32) NOT NULL,
		action VARCHAR(128) NOT NULL,
		actor_id VARCHAR(255),
		actor_email VARCHAR(255),
		source_ip INET,
		resource VARCHAR(255),
		request_id VARCHAR(255),
		success BOOLEAN,
		payload JSONB,
		redacted BOOLEAN DEFAULT false,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		google_sub VARCHAR(255) UNIQUE NOT NULL,
		email VARCHAR(320) UNIQUE NOT NULL,
		name VARCHAR(255),
		picture_url TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_login_at TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS user_sessions (
		session_id VARCHAR(255) PRIMARY KEY,
		user_id UUID NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_requests_timestamp ON requests(timestamp);
	CREATE INDEX IF NOT EXISTS idx_requests_status ON requests(status);
	CREATE INDEX IF NOT EXISTS idx_responses_timestamp ON responses(timestamp);
	CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
	CREATE INDEX IF NOT EXISTS idx_agent_metrics_agent_id ON agent_metrics(agent_id);
	CREATE INDEX IF NOT EXISTS idx_agent_metrics_timestamp ON agent_metrics(timestamp);
	CREATE INDEX IF NOT EXISTS idx_audit_events_created_at ON audit_events(created_at);
	CREATE INDEX IF NOT EXISTS idx_audit_events_category ON audit_events(category);
	CREATE INDEX IF NOT EXISTS idx_audit_events_actor_id ON audit_events(actor_id);
	CREATE INDEX IF NOT EXISTS idx_audit_events_request_id ON audit_events(request_id);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_users_google_sub ON users(google_sub);
	CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);
`

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := s.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to create database schema: %w", err)
	}

	s.logger.Info("Database schema initialized successfully")
	return nil
}

// InsertAuditEvent inserts an audit event into the database
func (s *Storage) InsertAuditEvent(ctx context.Context, event audit.AuditEvent) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := `
		INSERT INTO audit_events (
			event_id, category, action, actor_id, actor_email, source_ip,
			resource, request_id, success, payload, redacted, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	var payloadJSON []byte
	if event.Payload != nil {
		payloadJSON, _ = json.Marshal(event.Payload)
	}

	_, err := s.db.ExecContext(ctx, query,
		event.EventID, event.Category, event.Action, event.ActorID, event.ActorEmail,
		event.SourceIP, event.Resource, event.RequestID, event.Success, payloadJSON,
		event.Redacted, event.CreatedAt)

	return err
}

// ListAuditEvents retrieves audit events with filtering and pagination
func (s *Storage) ListAuditEvents(ctx context.Context, filter audit.AuditFilter, pagination audit.Pagination) ([]audit.AuditEvent, audit.Page, error) {
	if s.db == nil {
		return nil, audit.Page{}, fmt.Errorf("database not initialized")
	}

	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if filter.Category != nil {
		whereClause += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, *filter.Category)
		argIndex++
	}

	if filter.ActorID != nil {
		whereClause += fmt.Sprintf(" AND actor_id = $%d", argIndex)
		args = append(args, *filter.ActorID)
		argIndex++
	}

	if filter.Action != nil {
		whereClause += fmt.Sprintf(" AND action = $%d", argIndex)
		args = append(args, *filter.Action)
		argIndex++
	}

	if filter.Resource != nil {
		whereClause += fmt.Sprintf(" AND resource = $%d", argIndex)
		args = append(args, *filter.Resource)
		argIndex++
	}

	if filter.Success != nil {
		whereClause += fmt.Sprintf(" AND success = $%d", argIndex)
		args = append(args, *filter.Success)
		argIndex++
	}

	if filter.StartDate != nil {
		whereClause += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		whereClause += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *filter.EndDate)
		argIndex++
	}

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_events %s", whereClause)
	var total int64
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, audit.Page{}, fmt.Errorf("failed to count audit events: %w", err)
	}

	// Calculate pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	if pagination.Page < 1 {
		pagination.Page = 1
	}
	if pagination.PageSize < 1 {
		pagination.PageSize = 10
	}

	totalPages := int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize))

	// Build SELECT query with pagination
	selectQuery := fmt.Sprintf(`
		SELECT id, event_id, category, action, actor_id, actor_email, source_ip,
		       resource, request_id, success, payload, redacted, created_at
		FROM audit_events %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, pagination.PageSize, offset)

	rows, err := s.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, audit.Page{}, fmt.Errorf("failed to query audit events: %w", err)
	}
	defer rows.Close()

	var events []audit.AuditEvent
	for rows.Next() {
		var event audit.AuditEvent
		var payloadBytes []byte
		err := rows.Scan(
			&event.ID, &event.EventID, &event.Category, &event.Action,
			&event.ActorID, &event.ActorEmail, &event.SourceIP, &event.Resource,
			&event.RequestID, &event.Success, &payloadBytes, &event.Redacted,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, audit.Page{}, fmt.Errorf("failed to scan audit event: %w", err)
		}

		// Unmarshal payload if present
		if len(payloadBytes) > 0 {
			json.Unmarshal(payloadBytes, &event.Payload)
		}

		events = append(events, event)
	}

	page := audit.Page{
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}

	return events, page, nil
}

// GetAuditEvent retrieves a single audit event by ID or event_id
func (s *Storage) GetAuditEvent(ctx context.Context, idOrEventID string) (audit.AuditEvent, error) {
	if s.db == nil {
		return audit.AuditEvent{}, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT id, event_id, category, action, actor_id, actor_email, source_ip,
		       resource, request_id, success, payload, redacted, created_at
		FROM audit_events
		WHERE id = $1 OR event_id = $1
	`

	var event audit.AuditEvent
	var payloadBytes []byte
	err := s.db.QueryRowContext(ctx, query, idOrEventID).Scan(
		&event.ID, &event.EventID, &event.Category, &event.Action,
		&event.ActorID, &event.ActorEmail, &event.SourceIP, &event.Resource,
		&event.RequestID, &event.Success, &payloadBytes, &event.Redacted,
		&event.CreatedAt,
	)

	if err != nil {
		return audit.AuditEvent{}, fmt.Errorf("failed to get audit event: %w", err)
	}

	// Unmarshal payload if present
	if len(payloadBytes) > 0 {
		json.Unmarshal(payloadBytes, &event.Payload)
	}

	return event, nil
}

// CleanupAuditEvents removes audit events older than the specified days
func (s *Storage) CleanupAuditEvents(ctx context.Context, retentionDays int) (int64, error) {
	if s.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	if retentionDays <= 0 {
		return 0, nil
	}

	query := `
		DELETE FROM audit_events 
		WHERE created_at < NOW() - INTERVAL '%d days'
	`

	result, err := s.db.ExecContext(ctx, fmt.Sprintf(query, retentionDays))
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup audit events: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// User-related database operations

// CreateOrUpdateUser creates a new user or updates an existing one
func (s *Storage) CreateOrUpdateUser(ctx context.Context, user *models.User) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := `
		INSERT INTO users (google_sub, email, name, picture_url, last_login_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (google_sub)
		DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			picture_url = EXCLUDED.picture_url,
			last_login_at = NOW(),
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRowContext(ctx, query,
		user.GoogleSub, user.Email, user.Name, user.PictureURL).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create or update user: %w", err)
	}

	return nil
}

// GetUserByGoogleSub retrieves a user by their Google subject ID
func (s *Storage) GetUserByGoogleSub(ctx context.Context, googleSub string) (*models.User, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT id, google_sub, email, name, picture_url, created_at, updated_at, last_login_at
		FROM users WHERE google_sub = $1
	`

	user := &models.User{}
	err := s.db.QueryRowContext(ctx, query, googleSub).Scan(
		&user.ID, &user.GoogleSub, &user.Email, &user.Name, &user.PictureURL,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get user by google_sub: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by their ID
func (s *Storage) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT id, google_sub, email, name, picture_url, created_at, updated_at, last_login_at
		FROM users WHERE id = $1
	`

	user := &models.User{}
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID, &user.GoogleSub, &user.Email, &user.Name, &user.PictureURL,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return user, nil
}

// CreateUserSession creates a new user session
func (s *Storage) CreateUserSession(ctx context.Context, session *models.UserSession) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := `
		INSERT INTO user_sessions (session_id, user_id, expires_at)
		VALUES ($1, $2, $3)
	`

	_, err := s.db.ExecContext(ctx, query, session.SessionID, session.UserID, session.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to create user session: %w", err)
	}

	return nil
}

// GetUserSession retrieves a user session by session ID
func (s *Storage) GetUserSession(ctx context.Context, sessionID string) (*models.UserSession, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT session_id, user_id, created_at, expires_at
		FROM user_sessions WHERE session_id = $1 AND expires_at > NOW()
	`

	session := &models.UserSession{}
	err := s.db.QueryRowContext(ctx, query, sessionID).Scan(
		&session.SessionID, &session.UserID, &session.CreatedAt, &session.ExpiresAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get user session: %w", err)
	}

	return session, nil
}

// DeleteUserSession deletes a user session
func (s *Storage) DeleteUserSession(ctx context.Context, sessionID string) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := `DELETE FROM user_sessions WHERE session_id = $1`

	_, err := s.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete user session: %w", err)
	}

	return nil
}

// CleanupExpiredSessions removes expired user sessions
func (s *Storage) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	if s.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	query := `DELETE FROM user_sessions WHERE expires_at < NOW()`

	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}
