package repository_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// TestPositionRepositoryScopesEveryStatementToTheCompany is the tenant
// isolation test for positions. It uses the dry-run harness declared in
// employee_repository_test.go.
func TestPositionRepositoryScopesEveryStatementToTheCompany(t *testing.T) {
	db, recorder := newHRDryRunDB(t)
	repo := repository.NewPositionRepositoryGorm(db)

	ctx := context.Background()
	companyID := uuid.New()
	positionID := uuid.New()
	active := true

	calls := []struct {
		name string
		run  func()
	}{
		{"Create", func() {
			_ = repo.Create(ctx, &domain.Position{
				TenantModel: domain.TenantModel{CompanyID: companyID},
				Code:        "MGR",
				Name:        "과장",
			})
		}},
		{"GetByID", func() { _, _ = repo.GetByID(ctx, companyID, positionID) }},
		{"GetByCode", func() { _, _ = repo.GetByCode(ctx, companyID, "MGR") }},
		{"List", func() {
			_, _, _ = repo.List(ctx, &repository.PositionFilter{
				CompanyID:  companyID,
				IsActive:   &active,
				SearchTerm: "과",
				Page:       1,
				PageSize:   20,
			})
		}},
		{"Update", func() {
			_ = repo.Update(ctx, &domain.Position{
				TenantModel: domain.TenantModel{
					BaseModel: domain.BaseModel{ID: positionID},
					CompanyID: companyID,
				},
				Code: "MGR",
				Name: "과장",
			})
		}},
		{"Delete", func() { _ = repo.Delete(ctx, companyID, positionID) }},
		{"ExistsByCode", func() { _, _ = repo.ExistsByCode(ctx, companyID, "MGR", &positionID) }},
		{"CountEmployees", func() { _, _ = repo.CountEmployees(ctx, companyID, positionID) }},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			call.run()
			assertEveryStatementIsTenantScoped(t, "PositionRepository."+call.name, recorder)
		})
	}
}

// TestPositionDeleteIsHard records a deliberate asymmetry with employees.
//
// The positions table has no deleted_at column, so its delete is a real
// DELETE. That is why PositionService refuses one while any employee still
// holds the position: the foreign key would otherwise refuse it at the driver
// level, with a message no user can act on.
func TestPositionDeleteIsHard(t *testing.T) {
	db, recorder := newHRDryRunDB(t)
	repo := repository.NewPositionRepositoryGorm(db)

	_ = repo.Delete(context.Background(), uuid.New(), uuid.New())

	if len(recorder.statements) != 1 {
		t.Fatalf("expected exactly one statement, got %d: %v", len(recorder.statements), recorder.statements)
	}
	sql := strings.ToUpper(strings.TrimSpace(recorder.statements[0]))
	if !strings.HasPrefix(sql, "DELETE") {
		t.Errorf("positions has no deleted_at, so the delete must be a real DELETE, got:\n\t%s", recorder.statements[0])
	}
}
