package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
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

	CREATE INDEX IF NOT EXISTS idx_requests_timestamp ON requests(timestamp);
	CREATE INDEX IF NOT EXISTS idx_requests_status ON requests(status);
	CREATE INDEX IF NOT EXISTS idx_responses_timestamp ON responses(timestamp);
	CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
	CREATE INDEX IF NOT EXISTS idx_agent_metrics_agent_id ON agent_metrics(agent_id);
	CREATE INDEX IF NOT EXISTS idx_agent_metrics_timestamp ON agent_metrics(timestamp);
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
