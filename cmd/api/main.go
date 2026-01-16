package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/saintgo7/saas-kerp/internal/auth"
	"github.com/saintgo7/saas-kerp/internal/config"
	"github.com/saintgo7/saas-kerp/internal/database"
	"github.com/saintgo7/saas-kerp/internal/handler"
	"github.com/saintgo7/saas-kerp/internal/router"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := initLogger(cfg)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting K-ERP API Server",
		zap.String("name", cfg.App.Name),
		zap.String("version", cfg.App.Version),
		zap.String("env", cfg.App.Env),
	)

	// Initialize database
	db, err := database.NewPostgresDB(&cfg.Database, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer func() {
		if err := database.CloseDB(db); err != nil {
			logger.Error("Error closing database connection", zap.Error(err))
		}
	}()
	logger.Info("Database connection established")

	// Initialize Redis
	rdb := database.NewRedisClient(&cfg.Redis)
	defer func() {
		if err := database.CloseRedis(rdb); err != nil {
			logger.Error("Error closing Redis connection", zap.Error(err))
		}
	}()

	// Ping Redis
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := database.PingRedis(ctx, rdb); err != nil {
		logger.Warn("Redis connection failed (non-fatal)", zap.Error(err))
	} else {
		logger.Info("Redis connection established")
	}
	cancel()

	// Initialize NATS (optional, non-fatal if fails)
	nc, err := database.NewNATSConnection(&cfg.NATS)
	if err != nil {
		logger.Warn("NATS connection failed (non-fatal)", zap.Error(err))
	} else {
		defer database.CloseNATS(nc)
		logger.Info("NATS connection established")
	}

	// Initialize JWT service
	jwtService := auth.NewJWTService(&cfg.JWT)

	// Initialize handlers
	handlers := handler.NewHandlers(db, rdb, logger, jwtService, cfg.App.Version)

	// Initialize router
	r := router.New(cfg, logger, jwtService, handlers)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      r.Engine(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Server listening",
			zap.Int("port", cfg.App.Port),
			zap.String("address", srv.Addr),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited gracefully")
}

// initLogger initializes the zap logger based on configuration
func initLogger(cfg *config.Config) (*zap.Logger, error) {
	var zapCfg zap.Config

	if cfg.IsDevelopment() {
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		zapCfg = zap.NewProductionConfig()
	}

	// Set log level
	switch cfg.Log.Level {
	case "debug":
		zapCfg.Level.SetLevel(zap.DebugLevel)
	case "info":
		zapCfg.Level.SetLevel(zap.InfoLevel)
	case "warn":
		zapCfg.Level.SetLevel(zap.WarnLevel)
	case "error":
		zapCfg.Level.SetLevel(zap.ErrorLevel)
	}

	// Set encoding format
	if cfg.Log.Format == "console" {
		zapCfg.Encoding = "console"
	}

	return zapCfg.Build()
}
