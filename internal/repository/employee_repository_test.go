package repository_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// ---------------------------------------------------------------------------
// A database-free harness for checking the SQL a repository builds
// ---------------------------------------------------------------------------
//
// GORM's DryRun mode builds every statement and stops short of sending it, so
// these tests need no PostgreSQL and run in the default `go test ./...` (the
// repository suite that does need a database is behind the `integration` build
// tag). What they check is the one invariant a multi-tenant repository cannot
// get wrong: EVERY statement it issues must mention company_id.
//
// The check is deliberately crude - "does the statement contain company_id" -
// because that is exactly the mistake worth catching: a query written without
// the tenant predicate at all. A wrong predicate is a different class of bug
// and belongs to the integration suite.

// sqlRecorder collects the statements built during one test.
type sqlRecorder struct {
	statements []string
}

// newHRDryRunDB opens a GORM handle that builds PostgreSQL SQL and never
// connects. sql.Open is lazy and DisableAutomaticPing keeps gorm.Open from
// dialling, so no server has to exist.
func newHRDryRunDB(t *testing.T) (*gorm.DB, *sqlRecorder) {
	t.Helper()

	// Port 1 on purpose: nothing listens there, so a bug in this harness fails
	// loudly instead of quietly reaching whatever PostgreSQL the developer
	// happens to be running. SkipDefaultTransaction is what keeps the write
	// finishers from opening a transaction - without it Create, Update and
	// Delete try to connect before they build any SQL.
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "host=127.0.0.1 port=1 user=dryrun password=dryrun dbname=dryrun sslmode=disable",
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		DryRun:                 true,
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("opening a dry-run handle: %v", err)
	}

	recorder := &sqlRecorder{}
	record := func(tx *gorm.DB) {
		if sql := tx.Statement.SQL.String(); sql != "" {
			recorder.statements = append(recorder.statements, sql)
		}
	}

	for name, register := range map[string]func(string, func(*gorm.DB)) error{
		"query":  db.Callback().Query().After("gorm:query").Register,
		"create": db.Callback().Create().After("gorm:create").Register,
		"update": db.Callback().Update().After("gorm:update").Register,
		"delete": db.Callback().Delete().After("gorm:delete").Register,
		"row":    db.Callback().Row().After("gorm:row").Register,
		"raw":    db.Callback().Raw().After("gorm:raw").Register,
	} {
		if err := register("dryrun:record_"+name, record); err != nil {
			t.Fatalf("registering the %s recorder: %v", name, err)
		}
	}

	return db, recorder
}

// assertEveryStatementIsTenantScoped fails when any statement omits
// company_id, and when the call issued no statement at all (which would make
// the assertion vacuous).
func assertEveryStatementIsTenantScoped(t *testing.T, name string, recorder *sqlRecorder) {
	t.Helper()

	if len(recorder.statements) == 0 {
		t.Fatalf("%s built no SQL, so nothing was checked", name)
	}
	for _, sql := range recorder.statements {
		if !strings.Contains(sql, "company_id") {
			t.Errorf("%s built a statement with no company_id predicate:\n\t%s", name, sql)
		}
	}
	recorder.statements = nil
}

// TestEmployeeRepositoryScopesEveryStatementToTheCompany is the tenant
// isolation test for employees. An employee row carries a resident number and
// a salary history; a query that forgets company_id hands one tenant's staff
// records to another.
func TestEmployeeRepositoryScopesEveryStatementToTheCompany(t *testing.T) {
	db, recorder := newHRDryRunDB(t)
	repo := repository.NewEmployeeRepositoryGorm(db)

	ctx := context.Background()
	companyID := uuid.New()
	employeeID := uuid.New()
	otherID := uuid.New()
	hired := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)

	calls := []struct {
		name string
		run  func()
	}{
		{"Create", func() {
			_ = repo.Create(ctx, &domain.Employee{
				TenantModel: domain.TenantModel{CompanyID: companyID},
				EmployeeNo:  "EMP001",
				Name:        "홍길동",
				HireDate:    hired,
				Status:      domain.EmployeeStatusActive,
			})
		}},
		{"GetByID", func() { _, _ = repo.GetByID(ctx, companyID, employeeID) }},
		{"GetByEmployeeNo", func() { _, _ = repo.GetByEmployeeNo(ctx, companyID, "EMP001") }},
		{"List", func() {
			_, _, _ = repo.List(ctx, &repository.EmployeeFilter{
				CompanyID:      companyID,
				DepartmentID:   &otherID,
				PositionID:     &otherID,
				ManagerID:      &otherID,
				Status:         domain.EmployeeStatusActive,
				EmploymentType: domain.EmploymentTypeRegular,
				SearchTerm:     "홍",
				HiredFrom:      &hired,
				HiredTo:        &hired,
				Page:           1,
				PageSize:       20,
			})
		}},
		{"Update", func() {
			_ = repo.Update(ctx, &domain.Employee{
				TenantModel: domain.TenantModel{
					BaseModel: domain.BaseModel{ID: employeeID},
					CompanyID: companyID,
				},
				EmployeeNo: "EMP001",
				Name:       "홍길동",
				HireDate:   hired,
				Status:     domain.EmployeeStatusActive,
			})
		}},
		{"Delete", func() { _ = repo.Delete(ctx, companyID, employeeID) }},
		{"ExistsByEmployeeNo", func() { _, _ = repo.ExistsByEmployeeNo(ctx, companyID, "EMP001", &employeeID) }},
		{"Exists", func() { _, _ = repo.Exists(ctx, companyID, employeeID) }},
		{"DepartmentExists", func() { _, _ = repo.DepartmentExists(ctx, companyID, otherID) }},
		{"GetManagerID", func() { _, _ = repo.GetManagerID(ctx, companyID, employeeID) }},
		{"CountRetainedRecords", func() { _, _ = repo.CountRetainedRecords(ctx, companyID, employeeID) }},
		{"CountDirectReports", func() { _, _ = repo.CountDirectReports(ctx, companyID, employeeID) }},
		{"CountByStatus", func() { _, _ = repo.CountByStatus(ctx, companyID) }},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			call.run()
			assertEveryStatementIsTenantScoped(t, "EmployeeRepository."+call.name, recorder)
		})
	}
}

// TestEmployeeDeleteIsSoft proves the delete is an UPDATE, not a DELETE.
//
// employees is referenced by payrolls, employee_salaries, employee_insurance
// and insurance_monthly_contributions with ON DELETE RESTRICT: the wage ledger
// has to survive for three years. A hard delete would either be refused by the
// database or, worse, destroy the history it is meant to keep.
func TestEmployeeDeleteIsSoft(t *testing.T) {
	db, recorder := newHRDryRunDB(t)
	repo := repository.NewEmployeeRepositoryGorm(db)

	_ = repo.Delete(context.Background(), uuid.New(), uuid.New())

	if len(recorder.statements) != 1 {
		t.Fatalf("expected exactly one statement, got %d: %v", len(recorder.statements), recorder.statements)
	}
	sql := recorder.statements[0]
	if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(sql)), "UPDATE") {
		t.Errorf("delete must be a soft delete (UPDATE ... SET deleted_at), got:\n\t%s", sql)
	}
	if !strings.Contains(sql, "deleted_at") {
		t.Errorf("delete does not write deleted_at:\n\t%s", sql)
	}
}

// TestEmployeeReadsExcludeSoftDeletedRows guards the other half of the soft
// delete: a resigned-and-removed person must not come back in a listing.
func TestEmployeeReadsExcludeSoftDeletedRows(t *testing.T) {
	db, recorder := newHRDryRunDB(t)
	repo := repository.NewEmployeeRepositoryGorm(db)

	ctx := context.Background()
	companyID := uuid.New()

	_, _ = repo.GetByID(ctx, companyID, uuid.New())
	_, _, _ = repo.List(ctx, &repository.EmployeeFilter{CompanyID: companyID, Page: 1, PageSize: 20})
	_, _ = repo.CountByStatus(ctx, companyID)

	if len(recorder.statements) == 0 {
		t.Fatal("no SQL was built")
	}
	for _, sql := range recorder.statements {
		if !strings.Contains(sql, "deleted_at") {
			t.Errorf("a read does not exclude soft-deleted employees:\n\t%s", sql)
		}
	}
}
