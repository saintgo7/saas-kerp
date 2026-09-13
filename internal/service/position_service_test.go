package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/service"
)

func newPosition(companyID uuid.UUID, code string) *domain.Position {
	return &domain.Position{
		TenantModel: domain.TenantModel{CompanyID: companyID},
		Code:        code,
		Name:        "과장",
		RankLevel:   4,
		IsActive:    true,
	}
}

// TestPositionCodeIsUniquePerCompany, and only per company.
func TestPositionCodeIsUniquePerCompany(t *testing.T) {
	repo := newFakePositionRepo()
	svc := service.NewPositionService(repo)
	ctx := context.Background()
	companyID := uuid.New()

	if err := svc.Create(ctx, newPosition(companyID, "MGR")); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Create(ctx, newPosition(companyID, "MGR")); !errors.Is(err, domain.ErrPositionCodeExists) {
		t.Fatalf("Create() = %v, want ErrPositionCodeExists", err)
	}
	if err := svc.Create(ctx, newPosition(uuid.New(), "MGR")); err != nil {
		t.Fatalf("another company must be free to use MGR, got %v", err)
	}
}

// TestPositionValidationRunsBeforeTheWrite.
func TestPositionValidationRunsBeforeTheWrite(t *testing.T) {
	repo := newFakePositionRepo()
	svc := service.NewPositionService(repo)
	ctx := context.Background()

	position := newPosition(uuid.New(), "MGR")
	min, max := 5000000.0, 3000000.0
	position.MinSalary = &min
	position.MaxSalary = &max

	if err := svc.Create(ctx, position); !errors.Is(err, domain.ErrPositionSalaryRange) {
		t.Fatalf("Create() = %v, want ErrPositionSalaryRange", err)
	}
	if len(repo.positions) != 0 {
		t.Error("an invalid position was stored")
	}
}

// TestAPositionHeldByEmployeesCannotBeDeleted.
//
// positions has no deleted_at, so the delete is a real DELETE and
// employees.position_id would refuse it at the driver level. The service turns
// that into an answer a user can act on.
func TestAPositionHeldByEmployeesCannotBeDeleted(t *testing.T) {
	repo := newFakePositionRepo()
	svc := service.NewPositionService(repo)
	ctx := context.Background()
	companyID := uuid.New()

	position := newPosition(companyID, "MGR")
	if err := svc.Create(ctx, position); err != nil {
		t.Fatalf("Create: %v", err)
	}
	repo.employees[position.ID] = 7

	canDelete, reason, err := svc.CanDelete(ctx, companyID, position.ID)
	if err != nil {
		t.Fatalf("CanDelete: %v", err)
	}
	if canDelete {
		t.Error("a position held by employees must not be deletable")
	}
	if reason == "" {
		t.Error("CanDelete must explain why not")
	}

	if err := svc.Delete(ctx, companyID, position.ID); !errors.Is(err, domain.ErrPositionInUse) {
		t.Fatalf("Delete() = %v, want ErrPositionInUse", err)
	}
	if _, ok := repo.positions[position.ID]; !ok {
		t.Error("the position was deleted anyway")
	}

	repo.employees[position.ID] = 0
	if err := svc.Delete(ctx, companyID, position.ID); err != nil {
		t.Fatalf("Delete() = %v, want nil once unused", err)
	}
}

// TestPositionsAreInvisibleToOtherCompanies.
func TestPositionsAreInvisibleToOtherCompanies(t *testing.T) {
	repo := newFakePositionRepo()
	svc := service.NewPositionService(repo)
	ctx := context.Background()

	owner := uuid.New()
	intruder := uuid.New()

	position := newPosition(owner, "MGR")
	if err := svc.Create(ctx, position); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := svc.GetByID(ctx, intruder, position.ID); !errors.Is(err, domain.ErrPositionNotFound) {
		t.Errorf("GetByID across tenants = %v, want ErrPositionNotFound", err)
	}
	if _, err := svc.GetByCode(ctx, intruder, "MGR"); !errors.Is(err, domain.ErrPositionNotFound) {
		t.Errorf("GetByCode across tenants = %v, want ErrPositionNotFound", err)
	}
	if err := svc.Delete(ctx, intruder, position.ID); !errors.Is(err, domain.ErrPositionNotFound) {
		t.Errorf("Delete across tenants = %v, want ErrPositionNotFound", err)
	}

	stolen := *position
	stolen.CompanyID = intruder
	stolen.Name = "탈취"
	if err := svc.Update(ctx, &stolen); !errors.Is(err, domain.ErrPositionNotFound) {
		t.Errorf("Update across tenants = %v, want ErrPositionNotFound", err)
	}
	if repo.positions[position.ID].Name == "탈취" {
		t.Error("another tenant rewrote the position")
	}

	listed, _, err := svc.List(ctx, &service.PositionFilter{CompanyID: intruder})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listed) != 0 {
		t.Errorf("another tenant's list returned %d positions", len(listed))
	}
}

// TestPositionListClampsThePageSize.
func TestPositionListClampsThePageSize(t *testing.T) {
	svc := service.NewPositionService(newFakePositionRepo())

	filter := &service.PositionFilter{CompanyID: uuid.New(), Page: 0, PageSize: 0}
	if _, _, err := svc.List(context.Background(), filter); err != nil {
		t.Fatalf("List: %v", err)
	}
	if filter.Page != 1 || filter.PageSize < 1 {
		t.Errorf("List left page=%d page_size=%d unclamped", filter.Page, filter.PageSize)
	}
}
