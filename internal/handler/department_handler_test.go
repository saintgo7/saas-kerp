package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/repository"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// ---------------------------------------------------------------------------
// In-memory department repository
// ---------------------------------------------------------------------------

// memDepartmentRepo is an in-memory stand-in for the GORM repository.
//
// The tests below run the REAL service on top of it, so what they check is the
// whole path a request takes below the router: handler -> service -> storage.
// That is the only way to see that the hierarchy rules the service enforces
// (no self-parent, no descendant-as-parent) come out of the API as the right
// status codes rather than as a 500.
//
// It deliberately reproduces two behaviours of the GORM repository that the
// handler has to cope with:
//
//   - a missing row is reported as fmt.Errorf("department not found"), an
//     anonymous error that errors.Is cannot match against any sentinel;
//   - every query is filtered by company_id, so another tenant's row is
//     indistinguishable from one that does not exist.
type memDepartmentRepo struct {
	rows  map[uuid.UUID]*domain.Department
	order []uuid.UUID

	// failWith, when set, is returned by GetByID instead of a row. It stands
	// for a driver-level failure whose text must never reach a client.
	failWith error

	voucherEntries map[uuid.UUID]bool
}

func newMemDepartmentRepo() *memDepartmentRepo {
	return &memDepartmentRepo{
		rows:           map[uuid.UUID]*domain.Department{},
		voucherEntries: map[uuid.UUID]bool{},
	}
}

// errMemDepartmentNotFound mirrors internal/repository/department_repository_gorm.go.
func errMemDepartmentNotFound() error {
	return fmt.Errorf("department not found")
}

// seed inserts a row directly, bypassing the service's validation.
func (r *memDepartmentRepo) seed(companyID uuid.UUID, code string, parent *uuid.UUID, level int) *domain.Department {
	department := &domain.Department{
		TenantModel: domain.TenantModel{
			BaseModel: domain.BaseModel{ID: uuid.New()},
			CompanyID: companyID,
		},
		Code:     code,
		Name:     code,
		ParentID: parent,
		Level:    level,
		IsActive: true,
	}
	r.rows[department.ID] = department
	r.order = append(r.order, department.ID)
	return department
}

func (r *memDepartmentRepo) Create(ctx context.Context, department *domain.Department) error {
	if department.ID == uuid.Nil {
		department.ID = uuid.New()
	}
	stored := *department
	r.rows[department.ID] = &stored
	r.order = append(r.order, department.ID)
	return nil
}

func (r *memDepartmentRepo) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Department, error) {
	if r.failWith != nil {
		return nil, r.failWith
	}
	row, ok := r.rows[id]
	if !ok || row.CompanyID != companyID {
		return nil, errMemDepartmentNotFound()
	}
	copied := *row
	return &copied, nil
}

func (r *memDepartmentRepo) GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Department, error) {
	for _, id := range r.order {
		row := r.rows[id]
		if row.CompanyID == companyID && row.Code == code {
			copied := *row
			return &copied, nil
		}
	}
	return nil, errMemDepartmentNotFound()
}

func (r *memDepartmentRepo) List(ctx context.Context, filter *repository.DepartmentFilter) ([]domain.Department, int64, error) {
	var matched []domain.Department
	for _, id := range r.order {
		row := r.rows[id]
		if row.CompanyID != filter.CompanyID {
			continue
		}
		if filter.ParentID != nil && (row.ParentID == nil || *row.ParentID != *filter.ParentID) {
			continue
		}
		if filter.IsActive != nil && row.IsActive != *filter.IsActive {
			continue
		}
		if filter.SearchTerm != "" &&
			!strings.Contains(row.Code, filter.SearchTerm) &&
			!strings.Contains(row.Name, filter.SearchTerm) {
			continue
		}
		matched = append(matched, *row)
	}

	total := int64(len(matched))

	// The same LIMIT/OFFSET arithmetic the GORM repository performs. A page
	// size of zero would make this a no-op there and here; the point of the
	// test is that it never arrives as zero.
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		if offset >= len(matched) {
			return nil, total, nil
		}
		end := offset + filter.PageSize
		if end > len(matched) {
			end = len(matched)
		}
		matched = matched[offset:end]
	}

	return matched, total, nil
}

func (r *memDepartmentRepo) Update(ctx context.Context, department *domain.Department) error {
	stored := *department
	r.rows[department.ID] = &stored
	return nil
}

func (r *memDepartmentRepo) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	if row, ok := r.rows[id]; ok && row.CompanyID == companyID {
		delete(r.rows, id)
	}
	return nil
}

func (r *memDepartmentRepo) GetTree(ctx context.Context, companyID uuid.UUID) ([]domain.Department, error) {
	var result []domain.Department
	for _, id := range r.order {
		row := r.rows[id]
		if row.CompanyID == companyID && row.IsActive {
			result = append(result, *row)
		}
	}
	return result, nil
}

func (r *memDepartmentRepo) GetChildren(ctx context.Context, companyID, parentID uuid.UUID) ([]domain.Department, error) {
	var result []domain.Department
	for _, id := range r.order {
		row := r.rows[id]
		if row.CompanyID == companyID && row.ParentID != nil && *row.ParentID == parentID {
			result = append(result, *row)
		}
	}
	return result, nil
}

func (r *memDepartmentRepo) GetAncestors(ctx context.Context, companyID, id uuid.UUID) ([]domain.Department, error) {
	var result []domain.Department
	current, ok := r.rows[id]
	for depth := 0; ok && depth < 64; depth++ {
		if current.ParentID == nil {
			break
		}
		parent, found := r.rows[*current.ParentID]
		if !found || parent.CompanyID != companyID {
			break
		}
		result = append(result, *parent)
		current = parent
	}
	return result, nil
}

// GetDescendants mirrors the recursive CTE: it anchors on parent_id = id and
// therefore never contains the node itself, which is exactly why the service
// has to reject a self-parent separately.
func (r *memDepartmentRepo) GetDescendants(ctx context.Context, companyID, id uuid.UUID) ([]domain.Department, error) {
	var result []domain.Department
	frontier := []uuid.UUID{id}
	visited := map[uuid.UUID]bool{id: true}

	for depth := 0; depth < 64 && len(frontier) > 0; depth++ {
		var next []uuid.UUID
		for _, parentID := range frontier {
			for _, candidateID := range r.order {
				row := r.rows[candidateID]
				if row.CompanyID != companyID || row.ParentID == nil || *row.ParentID != parentID {
					continue
				}
				if visited[row.ID] {
					continue
				}
				visited[row.ID] = true
				result = append(result, *row)
				next = append(next, row.ID)
			}
		}
		frontier = next
	}
	return result, nil
}

func (r *memDepartmentRepo) ExistsByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	for _, id := range r.order {
		row := r.rows[id]
		if row.CompanyID != companyID || row.Code != code {
			continue
		}
		if excludeID != nil && row.ID == *excludeID {
			continue
		}
		return true, nil
	}
	return false, nil
}

func (r *memDepartmentRepo) HasChildren(ctx context.Context, companyID, id uuid.UUID) (bool, error) {
	children, err := r.GetChildren(ctx, companyID, id)
	return len(children) > 0, err
}

func (r *memDepartmentRepo) HasVoucherEntries(ctx context.Context, _, departmentID uuid.UUID) (bool, error) {
	return r.voucherEntries[departmentID], nil
}

func (r *memDepartmentRepo) UpdateActiveStatus(ctx context.Context, companyID uuid.UUID, ids []uuid.UUID, isActive bool) error {
	for _, id := range ids {
		if row, ok := r.rows[id]; ok && row.CompanyID == companyID {
			row.IsActive = isActive
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Harness
// ---------------------------------------------------------------------------

type departmentTestEnv struct {
	engine    *gin.Engine
	repo      *memDepartmentRepo
	companyID uuid.UUID
}

// newDepartmentTestEnv wires the handler to the real service over the
// in-memory repository, on a router carrying the same paths as
// router.registerDepartmentRoutes.
//
// The permission middleware is not installed here; which role may reach which
// route is asserted in internal/router/hr_routes_test.go, and repeating it
// would only pin the same rule twice.
func newDepartmentTestEnv(t *testing.T) *departmentTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := newMemDepartmentRepo()
	handler := NewDepartmentHandler(service.NewDepartmentService(repo))

	env := &departmentTestEnv{
		engine:    gin.New(),
		repo:      repo,
		companyID: uuid.New(),
	}

	api := env.engine.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		appctx.SetUserID(c, uuid.New())
		appctx.SetCompanyID(c, env.companyID)
		c.Next()
	})

	departments := api.Group("/departments")
	departments.GET("", handler.List)
	departments.GET("/tree", handler.Tree)
	departments.GET("/code/:code", handler.GetByCode)
	departments.GET("/:id", handler.GetByID)
	departments.GET("/:id/children", handler.GetChildren)
	departments.GET("/:id/can-delete", handler.CanDelete)
	departments.POST("", handler.Create)
	departments.PUT("/:id", handler.Update)
	departments.POST("/:id/move", handler.Move)
	departments.DELETE("/:id", handler.Delete)

	return env
}

func (e *departmentTestEnv) do(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	e.engine.ServeHTTP(w, req)
	return w
}

// decodeEnvelope reads the canonical response envelope.
func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) dto.Response {
	t.Helper()

	var envelope dto.Response
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("response is not the standard envelope: %v (body: %s)", err, w.Body.String())
	}
	return envelope
}

// assertErrorResponse checks the status and error code and returns the body.
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, status int, code string) dto.Response {
	t.Helper()

	if w.Code != status {
		t.Fatalf("status = %d, want %d (body: %s)", w.Code, status, w.Body.String())
	}
	envelope := decodeEnvelope(t, w)
	if envelope.Success {
		t.Fatalf("success = true on an error response: %s", w.Body.String())
	}
	if envelope.Error == nil {
		t.Fatalf("no error object in %s", w.Body.String())
	}
	if envelope.Error.Code != code {
		t.Errorf("error code = %q, want %q (body: %s)", envelope.Error.Code, code, w.Body.String())
	}
	return envelope
}

// ---------------------------------------------------------------------------
// Pagination
// ---------------------------------------------------------------------------

// TestDepartmentListSurvivesAZeroPageSize. page_size=0 reaching the
// total-pages division is an "integer divide by zero" panic, which takes the
// request down with a 500 and no body.
func TestDepartmentListSurvivesAZeroPageSize(t *testing.T) {
	env := newDepartmentTestEnv(t)
	for i := 0; i < 3; i++ {
		env.repo.seed(env.companyID, fmt.Sprintf("D%d", i), nil, 1)
	}

	for _, query := range []string{"page_size=0", "page_size=-5", "page=0&page_size=0", "page_size=abc"} {
		t.Run(query, func(t *testing.T) {
			w := env.do(t, http.MethodGet, "/api/v1/departments?"+query, "")
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
			}

			envelope := decodeEnvelope(t, w)
			if envelope.Meta == nil || envelope.Meta.Pagination == nil {
				t.Fatalf("no pagination metadata in %s", w.Body.String())
			}
			page := envelope.Meta.Pagination
			if page.PerPage != 20 {
				t.Errorf("per_page = %d, want the default 20", page.PerPage)
			}
			if page.Page != 1 {
				t.Errorf("page = %d, want 1", page.Page)
			}
			if page.Total != 3 || page.TotalPages != 1 {
				t.Errorf("total = %d, total_pages = %d; want 3 and 1", page.Total, page.TotalPages)
			}
		})
	}
}

// TestDepartmentListCapsThePageSize keeps one request from serialising a whole
// table.
func TestDepartmentListCapsThePageSize(t *testing.T) {
	env := newDepartmentTestEnv(t)
	env.repo.seed(env.companyID, "CEO", nil, 1)

	w := env.do(t, http.MethodGet, "/api/v1/departments?page_size=100000", "")
	envelope := decodeEnvelope(t, w)
	if envelope.Meta.Pagination.PerPage != 100 {
		t.Errorf("per_page = %d, want the cap of 100", envelope.Meta.Pagination.PerPage)
	}
}

func TestDepartmentListRejectsAMalformedParentFilter(t *testing.T) {
	env := newDepartmentTestEnv(t)

	w := env.do(t, http.MethodGet, "/api/v1/departments?parent_id=not-a-uuid", "")
	assertErrorResponse(t, w, http.StatusBadRequest, "VAL_004")
}

// ---------------------------------------------------------------------------
// Tenant isolation
// ---------------------------------------------------------------------------

// TestCreateDepartmentIgnoresACompanyIdInTheBody. The company comes from the
// session; a body that names another one must not place the record there.
func TestCreateDepartmentIgnoresACompanyIdInTheBody(t *testing.T) {
	env := newDepartmentTestEnv(t)
	otherCompany := uuid.New()

	body := fmt.Sprintf(`{"code":"DEV","name":"개발본부","company_id":%q}`, otherCompany)
	w := env.do(t, http.MethodPost, "/api/v1/departments", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", w.Code, w.Body.String())
	}

	if len(env.repo.rows) != 1 {
		t.Fatalf("stored %d rows, want 1", len(env.repo.rows))
	}
	for _, row := range env.repo.rows {
		if row.CompanyID != env.companyID {
			t.Errorf("the department was filed under %s, want the session company %s",
				row.CompanyID, env.companyID)
		}
	}
}

// TestAnotherTenantsDepartmentIsNotReadable.
func TestAnotherTenantsDepartmentIsNotReadable(t *testing.T) {
	env := newDepartmentTestEnv(t)
	foreign := env.repo.seed(uuid.New(), "THEIRS", nil, 1)

	w := env.do(t, http.MethodGet, "/api/v1/departments/"+foreign.ID.String(), "")
	assertErrorResponse(t, w, http.StatusNotFound, "RES_001")

	w = env.do(t, http.MethodDelete, "/api/v1/departments/"+foreign.ID.String(), "")
	assertErrorResponse(t, w, http.StatusNotFound, "RES_001")

	if _, still := env.repo.rows[foreign.ID]; !still {
		t.Error("another tenant's department was deleted")
	}
}

// ---------------------------------------------------------------------------
// Error translation
// ---------------------------------------------------------------------------

// TestGetDepartmentAnswers404ForAMissingRow.
//
// The repository reports a missing department as an anonymous
// fmt.Errorf("department not found"), which errors.Is cannot match. Without
// the normalisation in respondDepartmentError, every request for a department
// that does not exist answers 500 and logs a fault.
func TestGetDepartmentAnswers404ForAMissingRow(t *testing.T) {
	env := newDepartmentTestEnv(t)

	paths := []string{
		"/api/v1/departments/" + uuid.New().String(),
		"/api/v1/departments/code/NOPE",
		"/api/v1/departments/" + uuid.New().String() + "/can-delete",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			w := env.do(t, http.MethodGet, path, "")
			assertErrorResponse(t, w, http.StatusNotFound, "RES_001")
		})
	}
}

func TestDepartmentRoutesRejectAMalformedID(t *testing.T) {
	env := newDepartmentTestEnv(t)

	w := env.do(t, http.MethodGet, "/api/v1/departments/not-a-uuid", "")
	assertErrorResponse(t, w, http.StatusBadRequest, "VAL_004")

	w = env.do(t, http.MethodDelete, "/api/v1/departments/not-a-uuid", "")
	assertErrorResponse(t, w, http.StatusBadRequest, "VAL_004")
}

// TestDepartmentErrorsDoNotLeakDriverText. A storage failure is a 500 with a
// fixed message; the driver's text - table names, columns, bind parameters -
// stays in the log.
func TestDepartmentErrorsDoNotLeakDriverText(t *testing.T) {
	env := newDepartmentTestEnv(t)
	const leak = `pq: relation "departments" does not exist (company_id=00000000)`
	env.repo.failWith = fmt.Errorf("%s", leak)

	w := env.do(t, http.MethodGet, "/api/v1/departments/"+uuid.New().String(), "")
	assertErrorResponse(t, w, http.StatusInternalServerError, "SRV_001")

	if strings.Contains(w.Body.String(), "relation") || strings.Contains(w.Body.String(), "pq:") {
		t.Errorf("the driver error reached the client: %s", w.Body.String())
	}
}

func TestCreateDepartmentRejectsADuplicateCode(t *testing.T) {
	env := newDepartmentTestEnv(t)
	env.repo.seed(env.companyID, "DEV", nil, 1)

	w := env.do(t, http.MethodPost, "/api/v1/departments", `{"code":"DEV","name":"개발본부"}`)
	assertErrorResponse(t, w, http.StatusConflict, "RES_002")
}

func TestCreateDepartmentRequiresCodeAndName(t *testing.T) {
	env := newDepartmentTestEnv(t)

	w := env.do(t, http.MethodPost, "/api/v1/departments", `{"name":"개발본부"}`)
	assertErrorResponse(t, w, http.StatusBadRequest, "VAL_001")

	w = env.do(t, http.MethodPost, "/api/v1/departments", `{"code":"DEV"}`)
	assertErrorResponse(t, w, http.StatusBadRequest, "VAL_001")

	w = env.do(t, http.MethodPost, "/api/v1/departments", `{"code":"DEV","name":"개발","parent_id":"nope"}`)
	assertErrorResponse(t, w, http.StatusBadRequest, "VAL_001")
}

// ---------------------------------------------------------------------------
// Hierarchy rules
// ---------------------------------------------------------------------------

// TestUpdateDepartmentRejectsSelfParent.
//
// A department that is its own parent makes every recursive walk over the
// hierarchy loop forever - the reason the repository caps its CTEs and the
// service checks the case explicitly. This asserts the refusal survives the
// trip through HTTP as a 400 that names the problem, not a 500.
func TestUpdateDepartmentRejectsSelfParent(t *testing.T) {
	env := newDepartmentTestEnv(t)
	department := env.repo.seed(env.companyID, "DEV", nil, 1)

	body := fmt.Sprintf(`{"code":"DEV","name":"개발본부","parent_id":%q}`, department.ID)
	w := env.do(t, http.MethodPut, "/api/v1/departments/"+department.ID.String(), body)

	envelope := assertErrorResponse(t, w, http.StatusBadRequest, "VAL_002")
	if !strings.Contains(envelope.Error.Message, "own parent") {
		t.Errorf("message = %q, want it to explain the self-reference", envelope.Error.Message)
	}

	if stored := env.repo.rows[department.ID]; stored.ParentID != nil {
		t.Error("the self-reference was written to storage anyway")
	}
}

// TestUpdateDepartmentRejectsADescendantAsParent: moving a parent underneath
// its own child closes the same loop one level further out.
func TestUpdateDepartmentRejectsADescendantAsParent(t *testing.T) {
	env := newDepartmentTestEnv(t)
	root := env.repo.seed(env.companyID, "CEO", nil, 1)
	middle := env.repo.seed(env.companyID, "DEV", &root.ID, 2)
	leaf := env.repo.seed(env.companyID, "DEV-FE", &middle.ID, 3)

	body := fmt.Sprintf(`{"code":"CEO","name":"대표이사","parent_id":%q}`, leaf.ID)
	w := env.do(t, http.MethodPut, "/api/v1/departments/"+root.ID.String(), body)

	assertErrorResponse(t, w, http.StatusBadRequest, "VAL_002")
	if stored := env.repo.rows[root.ID]; stored.ParentID != nil {
		t.Error("the cycle was written to storage anyway")
	}
}

func TestMoveDepartmentRejectsSelfParent(t *testing.T) {
	env := newDepartmentTestEnv(t)
	department := env.repo.seed(env.companyID, "DEV", nil, 1)

	body := fmt.Sprintf(`{"parent_id":%q}`, department.ID)
	w := env.do(t, http.MethodPost, "/api/v1/departments/"+department.ID.String()+"/move", body)

	assertErrorResponse(t, w, http.StatusBadRequest, "VAL_002")
	if stored := env.repo.rows[department.ID]; stored.ParentID != nil {
		t.Error("the self-reference was written to storage anyway")
	}
}

func TestMoveDepartmentRejectsADescendantAsParent(t *testing.T) {
	env := newDepartmentTestEnv(t)
	root := env.repo.seed(env.companyID, "CEO", nil, 1)
	child := env.repo.seed(env.companyID, "DEV", &root.ID, 2)

	body := fmt.Sprintf(`{"parent_id":%q}`, child.ID)
	w := env.do(t, http.MethodPost, "/api/v1/departments/"+root.ID.String()+"/move", body)

	assertErrorResponse(t, w, http.StatusBadRequest, "VAL_002")
}

// TestMoveDepartmentToTheTopLevel is the other half of the move: an empty
// parent detaches the subtree and the level is recomputed.
func TestMoveDepartmentToTheTopLevel(t *testing.T) {
	env := newDepartmentTestEnv(t)
	root := env.repo.seed(env.companyID, "CEO", nil, 1)
	child := env.repo.seed(env.companyID, "DEV", &root.ID, 2)

	w := env.do(t, http.MethodPost, "/api/v1/departments/"+child.ID.String()+"/move", `{"parent_id":""}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}

	stored := env.repo.rows[child.ID]
	if stored.ParentID != nil {
		t.Errorf("ParentID = %v, want nil", stored.ParentID)
	}
	if stored.Level != 1 {
		t.Errorf("Level = %d, want 1", stored.Level)
	}
}

func TestMoveDepartmentAnswers404ForAMissingRow(t *testing.T) {
	env := newDepartmentTestEnv(t)

	w := env.do(t, http.MethodPost, "/api/v1/departments/"+uuid.New().String()+"/move", `{}`)
	assertErrorResponse(t, w, http.StatusNotFound, "RES_001")
}

// TestCreateDepartmentDerivesTheLevelFromTheParent: the level is the service's
// to compute, never the client's to send.
func TestCreateDepartmentDerivesTheLevelFromTheParent(t *testing.T) {
	env := newDepartmentTestEnv(t)
	root := env.repo.seed(env.companyID, "CEO", nil, 1)

	body := fmt.Sprintf(`{"code":"DEV","name":"개발본부","parent_id":%q,"level":99}`, root.ID)
	w := env.do(t, http.MethodPost, "/api/v1/departments", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", w.Code, w.Body.String())
	}

	var created dto.DepartmentResponse
	payload, _ := json.Marshal(decodeEnvelope(t, w).Data)
	if err := json.Unmarshal(payload, &created); err != nil {
		t.Fatalf("cannot read the created department: %v", err)
	}
	if created.Level != 2 {
		t.Errorf("level = %d, want 2 (one below the root)", created.Level)
	}
}

// ---------------------------------------------------------------------------
// Deletion
// ---------------------------------------------------------------------------

func TestDeleteDepartmentRefusesWhileItHasChildren(t *testing.T) {
	env := newDepartmentTestEnv(t)
	root := env.repo.seed(env.companyID, "CEO", nil, 1)
	env.repo.seed(env.companyID, "DEV", &root.ID, 2)

	w := env.do(t, http.MethodDelete, "/api/v1/departments/"+root.ID.String(), "")
	envelope := assertErrorResponse(t, w, http.StatusConflict, "RES_003")
	if !strings.Contains(envelope.Error.Message, "sub-department") {
		t.Errorf("message = %q, want it to name the sub-departments", envelope.Error.Message)
	}

	if _, still := env.repo.rows[root.ID]; !still {
		t.Error("the department was deleted despite having children")
	}
}

func TestDeleteDepartmentRefusesWhileVoucherEntriesReferenceIt(t *testing.T) {
	env := newDepartmentTestEnv(t)
	department := env.repo.seed(env.companyID, "DEV", nil, 1)
	env.repo.voucherEntries[department.ID] = true

	w := env.do(t, http.MethodDelete, "/api/v1/departments/"+department.ID.String(), "")
	envelope := assertErrorResponse(t, w, http.StatusConflict, "RES_003")
	if !strings.Contains(envelope.Error.Message, "voucher") {
		t.Errorf("message = %q, want it to name the voucher entries", envelope.Error.Message)
	}
}

func TestDeleteDepartmentSucceedsForALeaf(t *testing.T) {
	env := newDepartmentTestEnv(t)
	department := env.repo.seed(env.companyID, "DEV", nil, 1)

	w := env.do(t, http.MethodDelete, "/api/v1/departments/"+department.ID.String(), "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %s)", w.Code, w.Body.String())
	}
	if _, still := env.repo.rows[department.ID]; still {
		t.Error("the department is still stored")
	}
}

func TestCanDeleteReportsTheReason(t *testing.T) {
	env := newDepartmentTestEnv(t)
	root := env.repo.seed(env.companyID, "CEO", nil, 1)
	leaf := env.repo.seed(env.companyID, "DEV", &root.ID, 2)

	w := env.do(t, http.MethodGet, "/api/v1/departments/"+root.ID.String()+"/can-delete", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"can_delete":false`) {
		t.Errorf("can_delete for a parent = %s, want false", body)
	}

	w = env.do(t, http.MethodGet, "/api/v1/departments/"+leaf.ID.String()+"/can-delete", "")
	if !strings.Contains(w.Body.String(), `"can_delete":true`) {
		t.Errorf("can_delete for a leaf = %s, want true", w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Tree
// ---------------------------------------------------------------------------

// TestDepartmentTreeIsNested is what the organisation chart is drawn from.
func TestDepartmentTreeIsNested(t *testing.T) {
	env := newDepartmentTestEnv(t)
	root := env.repo.seed(env.companyID, "CEO", nil, 1)
	dev := env.repo.seed(env.companyID, "DEV", &root.ID, 2)
	env.repo.seed(env.companyID, "DEV-FE", &dev.ID, 3)
	env.repo.seed(uuid.New(), "THEIRS", nil, 1)

	w := env.do(t, http.MethodGet, "/api/v1/departments/tree", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}

	var tree []dto.DepartmentTreeNode
	payload, _ := json.Marshal(decodeEnvelope(t, w).Data)
	if err := json.Unmarshal(payload, &tree); err != nil {
		t.Fatalf("cannot read the tree: %v (body: %s)", err, w.Body.String())
	}

	if len(tree) != 1 {
		t.Fatalf("got %d roots, want 1 - another tenant's chart may have leaked", len(tree))
	}
	if tree[0].Code != "CEO" {
		t.Fatalf("root = %s, want CEO", tree[0].Code)
	}
	if len(tree[0].Children) != 1 || tree[0].Children[0].Code != "DEV" {
		t.Fatalf("DEV is not nested under CEO: %+v", tree[0].Children)
	}
	if len(tree[0].Children[0].Children) != 1 {
		t.Fatalf("DEV-FE is not nested under DEV: %+v", tree[0].Children[0])
	}
}

// TestDepartmentTreeSurvivesAStoredCycle. The service refuses to create one,
// but a row that acquired a cyclic parent_id another way - a manual UPDATE, a
// restored dump - must not hang the request that reads the chart.
func TestDepartmentTreeSurvivesAStoredCycle(t *testing.T) {
	env := newDepartmentTestEnv(t)
	a := env.repo.seed(env.companyID, "A", nil, 1)
	b := env.repo.seed(env.companyID, "B", &a.ID, 2)
	env.repo.rows[a.ID].ParentID = &b.ID

	w := env.do(t, http.MethodGet, "/api/v1/departments/tree", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}

	var tree []dto.DepartmentTreeNode
	payload, _ := json.Marshal(decodeEnvelope(t, w).Data)
	if err := json.Unmarshal(payload, &tree); err != nil {
		t.Fatalf("cannot read the tree: %v", err)
	}
	if len(tree) != 2 {
		t.Errorf("got %d roots, want both rows of the cycle surfaced once each", len(tree))
	}
}

func TestDepartmentChildrenListsOneLevel(t *testing.T) {
	env := newDepartmentTestEnv(t)
	root := env.repo.seed(env.companyID, "CEO", nil, 1)
	dev := env.repo.seed(env.companyID, "DEV", &root.ID, 2)
	env.repo.seed(env.companyID, "DEV-FE", &dev.ID, 3)

	w := env.do(t, http.MethodGet, "/api/v1/departments/"+root.ID.String()+"/children", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}

	var children []dto.DepartmentResponse
	payload, _ := json.Marshal(decodeEnvelope(t, w).Data)
	if err := json.Unmarshal(payload, &children); err != nil {
		t.Fatalf("cannot read the children: %v", err)
	}
	if len(children) != 1 || children[0].Code != "DEV" {
		t.Errorf("children = %+v, want only the direct child DEV", children)
	}
}
