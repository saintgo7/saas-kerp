package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Load reads configuration from file and environment variables
func Load() (*Config, error) {
	v := viper.New()

	// Config file settings
	v.SetConfigName("app")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/kerp")

	// Environment variables
	v.SetEnvPrefix("KERP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Set defaults
	setDefaults(v)

	// Read config file (optional - env vars can override)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is OK, we use defaults and env vars
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// App defaults
	v.SetDefault("app.name", "kerp-api")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.debug", true)
	v.SetDefault("app.port", 8080)
	v.SetDefault("app.version", "0.2.0")
	// Trust no proxy by default: gin's own default is 0.0.0.0/0, which lets any
	// client forge X-Forwarded-For. Deployments behind nginx/Traefik must set
	// KERP_APP_TRUSTED_PROXIES to the proxy network CIDR.
	v.SetDefault("app.trusted_proxies", []string{})
	v.SetDefault("app.max_request_body_bytes", 1<<20) // 1 MiB

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.name", "kerp_dev")
	v.SetDefault("database.user", "kerp")
	v.SetDefault("database.password", "")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "5m")
	v.SetDefault("database.tenant_guc", true)
	v.SetDefault("database.log_parameters", false)

	// Redis defaults
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	// NATS defaults
	v.SetDefault("nats.url", "nats://localhost:4222")
	v.SetDefault("nats.cluster_id", "kerp-cluster")

	// JWT defaults
	v.SetDefault("jwt.secret", "change-me-in-production")
	v.SetDefault("jwt.access_token_ttl", "15m")
	v.SetDefault("jwt.refresh_token_ttl", "168h")
	v.SetDefault("jwt.issuer", "kerp-api")

	// CORS defaults
	v.SetDefault("cors.allowed_origins", []string{"http://localhost:3000", "http://localhost:5173"})
	v.SetDefault("cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"})
	v.SetDefault("cors.allowed_headers", []string{"Authorization", "Content-Type", "X-Request-ID"})
	v.SetDefault("cors.max_age", 86400)

	// Rate limit defaults
	// Rate limiting is on by default: an authentication endpoint without it is a
	// free credential-stuffing target.
	v.SetDefault("ratelimit.enabled", true)
	v.SetDefault("ratelimit.requests_per_second", 100)
	v.SetDefault("ratelimit.burst", 200)

	// Log defaults
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
}
