package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/saintgo7/saas-kerp/internal/config"
)

// NewPostgresDB creates a new PostgreSQL connection using GORM.
//
// When cfg.TenantGUC is enabled the returned *gorm.DB routes each statement to
// the connection pinned to that statement's context (see tenant.go), which is
// what makes the Row Level Security policies effective. Prepared-statement
// caching is disabled in that mode because a statement prepared on a pinned
// connection is not valid on any other connection, and GORM's cache is shared
// across requests.
func NewPostgresDB(cfg *config.DatabaseConfig, zapLogger *zap.Logger) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	// Configure GORM logger
	var gormLogger logger.Interface
	if zapLogger != nil {
		gormLogger = newGormLogger(zapLogger, cfg.LogParameters)
	} else {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	gormCfg := &gorm.Config{
		Logger:                                   gormLogger,
		DisableForeignKeyConstraintWhenMigrating: true,
		PrepareStmt:                              !cfg.TenantGUC,
	}

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), gormCfg)
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

	if !cfg.TenantGUC {
		return db, nil
	}

	// Re-open GORM on top of the tenant-aware connection pool. postgres.Config.Conn
	// makes the driver adopt the pool as-is, so this reuses the *sql.DB opened and
	// pinged above rather than establishing a second pool.
	tenantDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 newTenantConnPool(sqlDB),
		PreferSimpleProtocol: true,
	}), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to install tenant connection pool: %w", err)
	}

	return tenantDB, nil
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

// gormZapLogger adapts zap logger to the GORM logger interface.
type gormZapLogger struct {
	logger *zap.Logger
	level  logger.LogLevel
	// logParameters keeps bind parameters in traced SQL. GORM interpolates
	// parameters into the traced statement, so with this on the log receives
	// refresh tokens, bcrypt hashes and every other value the app ever binds.
	logParameters bool
}

var (
	_ logger.Interface  = (*gormZapLogger)(nil)
	_ gorm.ParamsFilter = (*gormZapLogger)(nil)
)

func newGormLogger(zapLogger *zap.Logger, logParameters bool) logger.Interface {
	return &gormZapLogger{
		logger:        zapLogger,
		level:         logger.Info,
		logParameters: logParameters,
	}
}

func (l *gormZapLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.level = level
	return &newLogger
}

// ParamsFilter is called by GORM before it renders a statement for tracing.
// Returning no parameters leaves the placeholders ($1, $2, ...) in place, so
// secrets never reach the log.
func (l *gormZapLogger) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.logParameters {
		return sql, params
	}
	return sql, nil
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
	sqlText, rows := fc()

	switch {
	case err != nil && l.level >= logger.Error:
		l.logger.Error("gorm trace",
			zap.Error(err),
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sqlText),
		)
	case elapsed > 200*time.Millisecond && l.level >= logger.Warn:
		l.logger.Warn("slow query",
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sqlText),
		)
	case l.level >= logger.Info:
		l.logger.Debug("gorm trace",
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sqlText),
		)
	}
}

// PingDB verifies the database connection is usable.
func PingDB(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Stats returns connection pool statistics for observability.
func Stats(db *gorm.DB) (sql.DBStats, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return sql.DBStats{}, err
	}
	return sqlDB.Stats(), nil
}
