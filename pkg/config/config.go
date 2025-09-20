package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App           AppConfig           `mapstructure:"app"`
	Agents        AgentsConfig        `mapstructure:"agents"`
	Communication CommunicationConfig `mapstructure:"communication"`
	Tools         ToolsConfig         `mapstructure:"tools"`
	Database      DatabaseConfig      `mapstructure:"database"`
}

type AppConfig struct {
	Name    string        `mapstructure:"name"`
	Version string        `mapstructure:"version"`
	Debug   bool          `mapstructure:"debug"`
	Port    int           `mapstructure:"port"`
	Timeout time.Duration `mapstructure:"timeout"`
}

type AgentsConfig struct {
	Planner   PlannerConfig   `mapstructure:"planner"`
	Executor  ExecutorConfig  `mapstructure:"executor"`
	Evaluator EvaluatorConfig `mapstructure:"evaluator"`
}

type PlannerConfig struct {
	MaxRequests int           `mapstructure:"max_requests"`
	Timeout     time.Duration `mapstructure:"timeout"`
}

type ExecutorConfig struct {
	RateLimit  int      `mapstructure:"rate_limit"`
	RetryCount int      `mapstructure:"retry_count"`
	Tools      []string `mapstructure:"tools"`
}

type EvaluatorConfig struct {
	CacheTTL        time.Duration    `mapstructure:"cache_ttl"`
	MaxReportSize   int64            `mapstructure:"max_report_size"`
	ValidationRules []ValidationRule `mapstructure:"validation_rules"`
}

type ValidationRule struct {
	Name     string                 `mapstructure:"name"`
	Type     string                 `mapstructure:"type"`
	Criteria map[string]interface{} `mapstructure:"criteria"`
	Required bool                   `mapstructure:"required"`
}

type CommunicationConfig struct {
	A2A       A2AConfig       `mapstructure:"a2a"`
	Streaming StreamingConfig `mapstructure:"streaming"`
}

type A2AConfig struct {
	Protocol string        `mapstructure:"protocol"`
	Timeout  time.Duration `mapstructure:"timeout"`
	Retries  int           `mapstructure:"retries"`
}

type StreamingConfig struct {
	Enabled    bool `mapstructure:"enabled"`
	BufferSize int  `mapstructure:"buffer_size"`
}

type ToolsConfig struct {
	Jira   JiraConfig   `mapstructure:"jira"`
	Slack  SlackConfig  `mapstructure:"slack"`
	GitHub GitHubConfig `mapstructure:"github"`
}

type JiraConfig struct {
	BaseURL   string          `mapstructure:"base_url"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Cache     CacheConfig     `mapstructure:"cache"`
}

type SlackConfig struct {
	RateLimit  RateLimitConfig  `mapstructure:"rate_limit"`
	Pagination PaginationConfig `mapstructure:"pagination"`
}

type GitHubConfig struct {
	RateLimit          RateLimitConfig `mapstructure:"rate_limit"`
	Cache              CacheConfig     `mapstructure:"cache"`
	RemainingThreshold int             `mapstructure:"remaining_threshold"`
}

type RateLimitConfig struct {
	RequestsPerMinute int `mapstructure:"requests_per_minute"`
	RequestsPerHour   int `mapstructure:"requests_per_hour"`
}

type PaginationConfig struct {
	Limit int `mapstructure:"limit"`
}

type CacheConfig struct {
	TTL time.Duration `mapstructure:"ttl"`
}

type DatabaseConfig struct {
	URL             string        `mapstructure:"url"`
	RedisURL        string        `mapstructure:"redis_url"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()
	viper.SetDefault("app.port", 8080)
	viper.SetDefault("app.timeout", "30s")
	viper.SetDefault("app.debug", false)
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", "5m")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

func LoadFromEnv() (*Config, error) {
	config := &Config{
		App: AppConfig{
			Name:    os.Getenv("APP_NAME"),
			Version: os.Getenv("APP_VERSION"),
			Debug:   os.Getenv("APP_DEBUG") == "true",
			Port:    getEnvInt("APP_PORT", 8080),
			Timeout: getEnvDuration("APP_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			URL:             os.Getenv("DATABASE_URL"),
			RedisURL:        os.Getenv("REDIS_URL"),
			MaxOpenConns:    getEnvInt("DATABASE_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DATABASE_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvDuration("DATABASE_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Agents: AgentsConfig{
			Planner: PlannerConfig{
				MaxRequests: getEnvInt("PLANNER_MAX_REQUESTS", 10),
				Timeout:     getEnvDuration("PLANNER_TIMEOUT", 30*time.Second),
			},
			Executor: ExecutorConfig{
				RateLimit:  getEnvInt("EXECUTOR_RATE_LIMIT", 100),
				RetryCount: getEnvInt("EXECUTOR_RETRY_COUNT", 3),
				Tools:      []string{"jira", "slack", "github"},
			},
			Evaluator: EvaluatorConfig{
				CacheTTL:      getEnvDuration("EVALUATOR_CACHE_TTL", time.Hour),
				MaxReportSize: getEnvInt64("EVALUATOR_MAX_REPORT_SIZE", 10*1024*1024),
			},
		},
	}

	return config, nil
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		_, err := fmt.Sscanf(value, "%d", &intValue)
		if err != nil {
			log.Printf("Invalid integer value for %s: %v", key, err)
			return defaultValue
		}
		return intValue
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		var intValue int64
		_, err := fmt.Sscanf(value, "%d", &intValue)
		if err != nil {
			log.Printf("Invalid integer value for %s: %v", key, err)
			return defaultValue
		}
		return intValue
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		duration, err := time.ParseDuration(value)
		if err != nil {
			log.Printf("Invalid duration value for %s: %v", key, err)
			return defaultValue
		}
		return duration
	}
	return defaultValue
}
