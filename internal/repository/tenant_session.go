package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/database"
)

// TenantSession installs the PostgreSQL tenant context for work that happens
// OUTSIDE a request that middleware.TenantSession already covered.
//
// Every authenticated route gets app.current_tenant from the middleware, which
// reads the company id out of the JWT. The authentication routes themselves
// cannot: /auth/login, /auth/refresh and /auth/register have no token yet, so
// they run with no tenant set - and once the application role loses BYPASSRLS
// (000016_least_privilege_app_role) that means every INSERT and UPDATE they
// attempt matches no policy and is rejected.
//
// The calling contract stated in 000021_auth_bootstrap.up.sql is:
//
//  1. call the SECURITY DEFINER bootstrap lookup
//  2. verify the credential
//  3. Begin() the tenant session with the company_id that came back
//  4. do everything else through the ordinary RLS-enforced path
//
// This interface is step 3. It lives in the repository package because it is
// the layer that already owns the *gorm.DB wiring (see UnitOfWork); the service
// layer takes it as a dependency and never sees GORM.
type TenantSession interface {
	// Begin returns a context routed to a connection carrying
	// app.current_tenant = companyID, plus a release function that MUST always
	// be called - normally by defer - to reset the variable and return the
	// connection to the pool.
	//
	// Statements issued with the returned context (every repository call does
	// db.WithContext(ctx)) run inside the tenant; statements issued with the
	// original context do not.
	Begin(ctx context.Context, companyID uuid.UUID) (context.Context, func(), error)
}

// tenantSessionGorm implements TenantSession on top of the pinned-connection
// mechanism in internal/database.
type tenantSessionGorm struct {
	db *gorm.DB
}

// NewTenantSession creates a GORM-backed TenantSession.
func NewTenantSession(db *gorm.DB) TenantSession {
	return &tenantSessionGorm{db: db}
}

func (t *tenantSessionGorm) Begin(ctx context.Context, companyID uuid.UUID) (context.Context, func(), error) {
	return database.AcquireTenantConn(ctx, t.db, companyID)
}
