package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/saintgo7/saas-kerp/internal/config"
)

// NewPostgresDB creates a new PostgreSQL connection using GORM
func NewPostgresDB(cfg *config.DatabaseConfig, zapLogger *zap.Logger) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	// Configure GORM logger
	var gormLogger logger.Interface
	if zapLogger != nil {
		gormLogger = newGormLogger(zapLogger)
	} else {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger:                                   gormLogger,
		DisableForeignKeyConstraintWhenMigrating: true,
		PrepareStmt:                              true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// CloseDB closes the database connection
func CloseDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// ScopedDB returns a DB instance scoped to a specific company (multi-tenancy)
func ScopedDB(db *gorm.DB, companyID uuid.UUID) *gorm.DB {
	return db.Where("company_id = ?", companyID)
}

// ScopedDBWithDeleted includes soft-deleted records
func ScopedDBWithDeleted(db *gorm.DB, companyID uuid.UUID) *gorm.DB {
	return db.Unscoped().Where("company_id = ?", companyID)
}

// Transaction wraps a function in a database transaction
func Transaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.Transaction(fn)
}

// gormLogger adapts zap logger to GORM logger interface
type gormZapLogger struct {
	logger *zap.Logger
	level  logger.LogLevel
}

func newGormLogger(zapLogger *zap.Logger) logger.Interface {
	return &gormZapLogger{
		logger: zapLogger,
		level:  logger.Info,
	}
}

func (l *gormZapLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.level = level
	return &newLogger
}

func (l *gormZapLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Info {
		l.logger.Sugar().Infof(msg, data...)
	}
}

func (l *gormZapLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Warn {
		l.logger.Sugar().Warnf(msg, data...)
	}
}

func (l *gormZapLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Error {
		l.logger.Sugar().Errorf(msg, data...)
	}
}

func (l *gormZapLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil && l.level >= logger.Error:
		l.logger.Error("gorm trace",
			zap.Error(err),
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
		)
	case elapsed > 200*time.Millisecond && l.level >= logger.Warn:
		l.logger.Warn("slow query",
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
		)
	case l.level >= logger.Info:
		l.logger.Debug("gorm trace",
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
		)
	}
}
