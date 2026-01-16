package config

import (
	"errors"
	"fmt"
)

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	var errs []error

	// App validation
	if c.App.Port < 1 || c.App.Port > 65535 {
		errs = append(errs, fmt.Errorf("invalid app.port: %d", c.App.Port))
	}

	if c.App.Env != "development" && c.App.Env != "staging" && c.App.Env != "production" {
		errs = append(errs, fmt.Errorf("invalid app.env: %s (must be development, staging, or production)", c.App.Env))
	}

	// Database validation
	if c.Database.Host == "" {
		errs = append(errs, errors.New("database.host is required"))
	}

	if c.Database.Name == "" {
		errs = append(errs, errors.New("database.name is required"))
	}

	if c.Database.User == "" {
		errs = append(errs, errors.New("database.user is required"))
	}

	if c.Database.MaxOpenConns < 1 {
		errs = append(errs, fmt.Errorf("invalid database.max_open_conns: %d", c.Database.MaxOpenConns))
	}

	// JWT validation
	if c.JWT.Secret == "" {
		errs = append(errs, errors.New("jwt.secret is required"))
	}

	if c.App.Env == "production" && c.JWT.Secret == "change-me-in-production" {
		errs = append(errs, errors.New("jwt.secret must be changed in production"))
	}

	if c.JWT.AccessTokenTTL <= 0 {
		errs = append(errs, errors.New("jwt.access_token_ttl must be positive"))
	}

	if c.JWT.RefreshTokenTTL <= 0 {
		errs = append(errs, errors.New("jwt.refresh_token_ttl must be positive"))
	}

	// CORS validation
	if len(c.CORS.AllowedOrigins) == 0 {
		errs = append(errs, errors.New("cors.allowed_origins must have at least one origin"))
	}

	// Rate limit validation
	if c.RateLimit.Enabled {
		if c.RateLimit.RequestsPerSecond < 1 {
			errs = append(errs, errors.New("ratelimit.requests_per_second must be positive when enabled"))
		}
		if c.RateLimit.Burst < 1 {
			errs = append(errs, errors.New("ratelimit.burst must be positive when enabled"))
		}
	}

	// Log validation
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Log.Level] {
		errs = append(errs, fmt.Errorf("invalid log.level: %s", c.Log.Level))
	}

	validFormats := map[string]bool{"json": true, "console": true}
	if !validFormats[c.Log.Format] {
		errs = append(errs, fmt.Errorf("invalid log.format: %s", c.Log.Format))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
