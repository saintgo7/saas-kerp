package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantGUCName is the PostgreSQL session variable holding the current tenant
// UUID. Every Row Level Security policy in db/migrations reads it as
// current_setting('app.current_tenant', true), and an unset value matches no
// rows (fail closed).
//
// There is deliberately no companion "bypass" variable: the previous
// app.is_admin switch was removed from the policies because a GUC that turns RLS
// off is a bypass any SQL injection can reach.
const TenantGUCName = "app.current_tenant"

// --------------------------------------------------------------------------
// Why a pinned connection rather than a plain SET on the pool
// --------------------------------------------------------------------------
//
// `SET app.current_tenant = ...` is a *session* setting. GORM sits on top of
// database/sql, whose pool hands out an arbitrary connection per statement, so a
// SET issued for one request would land on a connection that a different
// tenant's request picks up moments later. That is a cross-tenant read, i.e.
// strictly worse than having no RLS at all.
//
// Two mechanisms are safe:
//
//  1. `SET LOCAL` inside an explicit transaction. Correct, but it only covers
//     statements executed on that transaction, and this codebase runs most reads
//     outside any transaction.
//  2. Pin one connection for the lifetime of the request, set the GUC on it,
//     route every statement of that request to it, and RESET the GUC before
//     returning the connection to the pool.
//
// (2) is implemented here because it needs no change to repository or service
// signatures: the pinned connection travels in the request `context.Context`,
// which every repository already forwards via `db.WithContext(ctx)`.
//
// Cost: a tenant-scoped request holds one pool connection for its whole
// duration, so database.max_open_conns becomes the ceiling on concurrent
// tenant requests. Requests beyond that queue rather than fail.
//
// Why not (1) alone, which is the form the database owner specified: almost
// every read in this codebase runs outside a transaction (repositories call
// db.WithContext(ctx).Find(...) directly). A transaction-local setting applied to
// such a statement is discarded when the implicit single-statement transaction
// ends, so once the application role loses its RLS exemption every one of those
// reads would return zero rows. Wrapping each request in one explicit
// transaction instead is worse: in PostgreSQL any failed statement aborts the
// whole transaction, and several handlers here deliberately continue after a
// query error.
//
// The leak that (1) is meant to prevent is prevented here by the pin: the
// connection carrying the setting is never shared, and it is reset — or
// destroyed, if the reset fails — before it returns to the pool.
//
// SetTenantLocal / WithTenantTx below implement form (1) for callers that do run
// inside a transaction and want the setting scoped to it.

type pinnedConnKey struct{}

// WithPinnedConn returns a context that routes every subsequent GORM statement
// to conn, provided the *gorm.DB was built by NewPostgresDB with tenant GUC
// support enabled.
func WithPinnedConn(ctx context.Context, conn *sql.Conn) context.Context {
	if conn == nil {
		return ctx
	}
	return context.WithValue(ctx, pinnedConnKey{}, conn)
}

// PinnedConn returns the connection pinned to ctx, or nil.
func PinnedConn(ctx context.Context) *sql.Conn {
	if ctx == nil {
		return nil
	}
	conn, _ := ctx.Value(pinnedConnKey{}).(*sql.Conn)
	return conn
}

// AcquireTenantConn pins a pooled connection, sets the tenant session variables
// on it and returns a context routed to that connection together with a release
// function. The release function resets the session variables before handing the
// connection back to the pool and must always be called, normally by `defer`.
//
// A nil companyID (uuid.Nil) is rejected: an empty tenant GUC makes
// current_tenant_id() return NULL, which every RLS policy treats as "match
// nothing", and silently returning zero rows is worse than failing loudly.
func AcquireTenantConn(ctx context.Context, db *gorm.DB, companyID uuid.UUID) (context.Context, func(), error) {
	if db == nil {
		return ctx, func() {}, fmt.Errorf("database: nil gorm handle")
	}
	if companyID == uuid.Nil {
		return ctx, func() {}, fmt.Errorf("database: refusing to pin a connection without a tenant")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return ctx, func() {}, err
	}

	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return ctx, func() {}, err
	}

	// set_config(..., false) sets the value for the session, which on a pinned
	// connection means "for this request". The reset in the release function is
	// what keeps it from outliving the request; see the note above on why the
	// transaction-local form cannot be used on its own here.
	if _, err := conn.ExecContext(ctx, "SELECT set_config($1, $2, false)", TenantGUCName, companyID.String()); err != nil {
		_ = conn.Close()
		return ctx, func() {}, fmt.Errorf("database: failed to set %s: %w", TenantGUCName, err)
	}

	release := func() {
		// Reset on a context without the request's cancellation so a cancelled or
		// timed-out request still cleans up its connection.
		resetCtx := context.WithoutCancel(ctx)
		if _, err := conn.ExecContext(resetCtx, "SELECT set_config($1, '', false)", TenantGUCName); err != nil {
			// The connection is in an unknown state. Mark it bad so the pool
			// discards it rather than handing this tenant's value to the next
			// request.
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
		}
		_ = conn.Close()
	}

	return WithPinnedConn(ctx, conn), release, nil
}

// SetTenantLocal sets the tenant variable for the duration of the current
// transaction (set_config's third argument is is_local=true, the form the
// database owner specified). tx MUST be an open transaction: outside one the
// setting is discarded at the end of the statement and provides no isolation.
func SetTenantLocal(tx *gorm.DB, companyID uuid.UUID) error {
	if companyID == uuid.Nil {
		return fmt.Errorf("database: refusing to set an empty tenant")
	}
	return tx.Exec("SELECT set_config(?, ?, true)", TenantGUCName, companyID.String()).Error
}

// WithTenantTx runs fn inside a transaction whose tenant session variable is set
// with SET LOCAL, so PostgreSQL's RLS policies apply to every statement fn
// issues through tx.
func WithTenantTx(ctx context.Context, db *gorm.DB, companyID uuid.UUID, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := SetTenantLocal(tx, companyID); err != nil {
			return err
		}
		return fn(tx)
	})
}

// --------------------------------------------------------------------------
// ConnPool wrapper
// --------------------------------------------------------------------------

// tenantConnPool routes statements to the connection pinned to the statement's
// context when there is one, and to the shared pool otherwise. GORM passes the
// context of every statement into these methods, which is what makes the
// per-request pin reach repositories without changing their signatures.
type tenantConnPool struct {
	db *sql.DB
}

var (
	_ gorm.ConnPool         = (*tenantConnPool)(nil)
	_ gorm.ConnPoolBeginner = (*tenantConnPool)(nil)
	_ gorm.GetDBConnector   = (*tenantConnPool)(nil)
)

func newTenantConnPool(db *sql.DB) *tenantConnPool {
	return &tenantConnPool{db: db}
}

func (p *tenantConnPool) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	if conn := PinnedConn(ctx); conn != nil {
		return conn.PrepareContext(ctx, query)
	}
	return p.db.PrepareContext(ctx, query)
}

func (p *tenantConnPool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if conn := PinnedConn(ctx); conn != nil {
		return conn.ExecContext(ctx, query, args...)
	}
	return p.db.ExecContext(ctx, query, args...)
}

func (p *tenantConnPool) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if conn := PinnedConn(ctx); conn != nil {
		return conn.QueryContext(ctx, query, args...)
	}
	return p.db.QueryContext(ctx, query, args...)
}

func (p *tenantConnPool) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if conn := PinnedConn(ctx); conn != nil {
		return conn.QueryRowContext(ctx, query, args...)
	}
	return p.db.QueryRowContext(ctx, query, args...)
}

// BeginTx starts a transaction on the pinned connection when the request has
// one, so the transaction inherits the tenant session variable.
func (p *tenantConnPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	if conn := PinnedConn(ctx); conn != nil {
		return conn.BeginTx(ctx, opts)
	}
	return p.db.BeginTx(ctx, opts)
}

// GetDBConn lets gorm.DB.DB() reach the underlying pool (health checks, stats).
func (p *tenantConnPool) GetDBConn() (*sql.DB, error) {
	return p.db, nil
}

// Ping satisfies gorm.Open's automatic ping check.
func (p *tenantConnPool) Ping() error {
	return p.db.Ping()
}
