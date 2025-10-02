package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App           AppConfig           `mapstructure:"app"`
	Agents        AgentsConfig        `mapstructure:"agents"`
	Communication CommunicationConfig `mapstructure:"communication"`
	Tools         ToolsConfig         `mapstructure:"tools"`
	Database      DatabaseConfig      `mapstructure:"database"`
	Logging       LoggingConfig       `mapstructure:"logging"`
	Audit         AuditConfig         `mapstructure:"audit"`
	Auth          AuthConfig          `mapstructure:"auth"`
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

type LoggingConfig struct {
	Level       string           `mapstructure:"level"`
	Format      string           `mapstructure:"format"`
	Output      []string         `mapstructure:"output"`
	File        string           `mapstructure:"file"`
	Development bool             `mapstructure:"development"`
	Datadog     DatadogLogConfig `mapstructure:"datadog"`
}

type DatadogLogConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	InjectAPM bool   `mapstructure:"inject_apm"`
	Service   string `mapstructure:"service"`
	Env       string `mapstructure:"env"`
	Version   string `mapstructure:"version"`
}

type AuditConfig struct {
	Enabled         bool            `mapstructure:"enabled"`
	ExposeEndpoints bool            `mapstructure:"expose_endpoints"`
	Categories      AuditCategories `mapstructure:"categories"`
	RetentionDays   int             `mapstructure:"retention_days"`
	Redact          AuditRedact     `mapstructure:"redact"`
	FileSink        AuditFileSink   `mapstructure:"file_sink"`
}

type AuditCategories struct {
	HTTP     bool `mapstructure:"http"`
	Agent    bool `mapstructure:"agent"`
	External bool `mapstructure:"external"`
}

type AuditRedact struct {
	DenyKeys []string `mapstructure:"deny_keys"`
}

type AuditFileSink struct {
	Enabled    bool   `mapstructure:"enabled"`
	Path       string `mapstructure:"path"`
	MaxSizeMB  int    `mapstructure:"max_size_mb"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAgeDays int    `mapstructure:"max_age_days"`
	Compress   bool   `mapstructure:"compress"`
}

type AuthConfig struct {
	Providers ProvidersConfig `mapstructure:"providers"`
	Session   SessionConfig   `mapstructure:"session"`
}

type ProvidersConfig struct {
	Google GoogleConfig `mapstructure:"google"`
}

type GoogleConfig struct {
	ClientID        string   `mapstructure:"client_id"`
	ClientSecret    string   `mapstructure:"client_secret"`
	RedirectBaseURL string   `mapstructure:"redirect_base_url"`
	AllowedDomains  []string `mapstructure:"allowed_domains"`
}

type SessionConfig struct {
	CookieName string        `mapstructure:"cookie_name"`
	Domain     string        `mapstructure:"domain"`
	Secure     bool          `mapstructure:"secure"`
	TTL        time.Duration `mapstructure:"ttl"`
	SameSite   string        `mapstructure:"samesite"` // lax/strict/none
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
	viper.SetDefault("auth.providers.google.redirect_base_url", "http://localhost:8080")
	viper.SetDefault("auth.session.cookie_name", "mas_session")
	viper.SetDefault("auth.session.secure", false)
	viper.SetDefault("auth.session.ttl", "24h")
	viper.SetDefault("auth.session.samesite", "lax")

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
		Auth: AuthConfig{
			Providers: ProvidersConfig{
				Google: GoogleConfig{
					ClientID:        os.Getenv("GOOGLE_CLIENT_ID"),
					ClientSecret:    os.Getenv("GOOGLE_CLIENT_SECRET"),
					RedirectBaseURL: getEnvString("AUTH_REDIRECT_BASE_URL", "http://localhost:8080"),
					AllowedDomains:  getEnvStringSlice("AUTH_ALLOWED_DOMAINS", []string{}),
				},
			},
			Session: SessionConfig{
				CookieName: getEnvString("AUTH_COOKIE_NAME", "mas_session"),
				Domain:     getEnvString("AUTH_COOKIE_DOMAIN", ""),
				Secure:     getEnvBool("AUTH_COOKIE_SECURE", false),
				TTL:        getEnvDuration("AUTH_SESSION_TTL", 24*time.Hour),
				SameSite:   getEnvString("AUTH_COOKIE_SAMESITE", "lax"),
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

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true"
	}
	return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing
		if value == "" {
			return defaultValue
		}
		return strings.Split(value, ",")
	}
	return defaultValue
}
