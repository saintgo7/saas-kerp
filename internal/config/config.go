package config

import (
	"time"
)

// Config holds all application configuration
type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	NATS      NATSConfig      `mapstructure:"nats"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	CORS      CORSConfig      `mapstructure:"cors"`
	RateLimit RateLimitConfig `mapstructure:"ratelimit"`
	Log       LogConfig       `mapstructure:"log"`
}

// AppConfig holds application-level configuration
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Env     string `mapstructure:"env"`
	Debug   bool   `mapstructure:"debug"`
	Port    int    `mapstructure:"port"`
	Version string `mapstructure:"version"`

	// TrustedProxies lists the CIDRs / IPs of reverse proxies that are allowed to
	// set X-Forwarded-For and X-Real-IP. An empty list disables proxy trust
	// entirely, which is the correct setting when the API is reachable directly.
	// Never set this to 0.0.0.0/0: it lets any client forge its own IP and defeat
	// both rate limiting and audit logging.
	TrustedProxies []string `mapstructure:"trusted_proxies"`

	// MaxRequestBodyBytes caps the size of any request body the API will read.
	MaxRequestBodyBytes int64 `mapstructure:"max_request_body_bytes"`
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Name            string        `mapstructure:"name"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`

	// TenantGUC enables the per-request PostgreSQL session variable
	// (app.current_tenant) that the Row Level Security policies read. See
	// internal/database/tenant.go for the mechanism and its trade-offs.
	TenantGUC bool `mapstructure:"tenant_guc"`

	// LogParameters controls whether GORM's SQL traces keep bind parameters.
	// Parameters are interpolated into the traced statement, so leaving this on
	// outside development writes refresh tokens and bcrypt hashes to the log.
	LogParameters bool `mapstructure:"log_parameters"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// NATSConfig holds NATS JetStream configuration
type NATSConfig struct {
	URL       string `mapstructure:"url"`
	ClusterID string `mapstructure:"cluster_id"`
}

// JWTConfig holds JWT authentication configuration
type JWTConfig struct {
	Secret          string        `mapstructure:"secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
	Issuer          string        `mapstructure:"issuer"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
	MaxAge         int      `mapstructure:"max_age"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	RequestsPerSecond int  `mapstructure:"requests_per_second"`
	Burst             int  `mapstructure:"burst"`
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}
