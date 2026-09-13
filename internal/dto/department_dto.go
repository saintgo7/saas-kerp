package dto

import (
	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// maxDepartmentTreeDepth bounds every walk over the department hierarchy in
// this file.
//
// It mirrors the cap the repository puts on its recursive CTEs
// (internal/repository/department_repository_gorm.go), and it exists for the
// same reason. The service already refuses to make a department its own parent
// or a descendant of itself, so a cycle should never reach this code - but the
// response builder is the last place a corrupt parent_id could turn into an
// endless loop, this time inside the API process rather than inside Postgres.
// It therefore refuses to follow a chain it cannot prove terminates instead of
// trusting the rows it was handed.
const maxDepartmentTreeDepth = 64

// DepartmentResponse represents a department in API responses.
type DepartmentResponse struct {
	ID     string `json:"id"`
	Code   string `json:"code"`
	Name   string `json:"name"`
	NameEn string `json:"name_en,omitempty"`

	// ParentID and ManagerID are rendered as an explicit null when unset
	// rather than omitted, which is why they are pointers while the other
	// optional identifiers in this package are omitempty strings. A client
	// assembling an organisation chart has to tell "this department has no
	// parent" from "the server did not mention a parent", and an absent key
	// reads as the second.
	ParentID *string `json:"parent_id"`

	// Level is the stored departments.level column, where a root department is
	// 1. It is not recomputed here; see DepartmentTreeNode.Depth for the
	// position a node actually occupies in the returned tree.
	Level int    `json:"level"`
	Path  string `json:"path,omitempty"`

	// ManagerID references users(id), not employees(id). See the schema in
	// db/migrations/000005_accounting_tables.up.sql.
	ManagerID *string `json:"manager_id"`

	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// FromDepartment converts domain.Department to DepartmentResponse.
func FromDepartment(department *domain.Department) DepartmentResponse {
	resp := DepartmentResponse{
		ID:        department.ID.String(),
		Code:      department.Code,
		Name:      department.Name,
		NameEn:    department.NameEn,
		Level:     department.Level,
		Path:      department.Path,
		IsActive:  department.IsActive,
		CreatedAt: department.CreatedAt.Format(hrTimeFormat),
		UpdatedAt: department.UpdatedAt.Format(hrTimeFormat),
	}

	if department.ParentID != nil {
		parentID := department.ParentID.String()
		resp.ParentID = &parentID
	}
	if department.ManagerID != nil {
		managerID := department.ManagerID.String()
		resp.ManagerID = &managerID
	}

	return resp
}

// FromDepartments converts []domain.Department to []DepartmentResponse.
func FromDepartments(departments []domain.Department) []DepartmentResponse {
	responses := make([]DepartmentResponse, len(departments))
	for i := range departments {
		responses[i] = FromDepartment(&departments[i])
	}
	return responses
}

// DepartmentTreeNode is a department together with its subtree.
type DepartmentTreeNode struct {
	DepartmentResponse

	// Depth is the node's 0-based position in the tree that was actually
	// returned, computed from the links this response contains. It is the
	// value to indent by. Level is the stored column, which the service
	// maintains only for the department being written: moving a department
	// leaves the level of everything underneath it untouched, so a client that
	// indents by Level can draw a subtree at the wrong offset.
	Depth int `json:"depth"`

	// Children is never null, so a client can iterate it without a guard.
	Children []DepartmentTreeNode `json:"children"`
}

// BuildDepartmentTree assembles a flat list of departments into a forest.
//
// The input is what the repository returns for a company, already ordered.
// That order is preserved among siblings and among roots.
//
// A row is treated as a root when it names no parent, when its parent is not
// part of the input, or when following its parent chain does not terminate.
// The last case cannot arise from data the service wrote - it rejects a
// self-parent and a descendant-as-parent - but a row that acquired a cyclic
// parent_id some other way must still produce a finite response rather than
// hanging the request, so the offending link is dropped and the node surfaces
// at the top level where it is visible instead of vanishing.
func BuildDepartmentTree(departments []domain.Department) []DepartmentTreeNode {
	index := make(map[uuid.UUID]int, len(departments))
	for i := range departments {
		// A duplicate id cannot happen (primary key), but if it somehow did,
		// keeping the first occurrence is what makes the map a function.
		if _, seen := index[departments[i].ID]; !seen {
			index[departments[i].ID] = i
		}
	}

	children := make([][]int, len(departments))
	roots := make([]int, 0, len(departments))

	for i := range departments {
		parent := resolveDepartmentParent(departments, index, i)
		if parent < 0 {
			roots = append(roots, i)
			continue
		}
		children[parent] = append(children[parent], i)
	}

	tree := make([]DepartmentTreeNode, 0, len(roots))
	for _, root := range roots {
		tree = append(tree, buildDepartmentSubtree(departments, children, root, 0))
	}
	return tree
}

// resolveDepartmentParent returns the index of the parent to attach a node to,
// or -1 when the node belongs at the top level.
func resolveDepartmentParent(departments []domain.Department, index map[uuid.UUID]int, child int) int {
	parentID := departments[child].ParentID
	if parentID == nil {
		return -1
	}

	parent, ok := index[*parentID]
	if !ok {
		// The parent is outside this result set - an inactive department in an
		// active-only listing, for instance. The node is shown as a root
		// rather than dropped.
		return -1
	}
	if parent == child {
		return -1
	}
	if !departmentChainTerminates(departments, index, parent, child) {
		return -1
	}
	return parent
}

// departmentChainTerminates reports whether walking up from `start` ends at a
// root within the depth cap without passing through `child`.
//
// Both failure modes mean the same thing for the caller: the link from `child`
// to `start` cannot be followed, because doing so would build a structure that
// is not a tree.
func departmentChainTerminates(departments []domain.Department, index map[uuid.UUID]int, start, child int) bool {
	current := start
	for depth := 0; depth < maxDepartmentTreeDepth; depth++ {
		if current == child {
			return false
		}

		parentID := departments[current].ParentID
		if parentID == nil {
			return true
		}
		next, ok := index[*parentID]
		if !ok {
			return true
		}
		if next == current {
			return false
		}
		current = next
	}
	return false
}

// buildDepartmentSubtree renders one node and everything below it.
//
// The recursion is safe by construction: BuildDepartmentTree only records a
// parent link whose chain it has proved terminates, so `children` describes a
// forest. The depth argument is nonetheless checked, because a builder that
// relies on an invariant established elsewhere is one refactor away from
// recursing forever.
func buildDepartmentSubtree(departments []domain.Department, children [][]int, node, depth int) DepartmentTreeNode {
	result := DepartmentTreeNode{
		DepartmentResponse: FromDepartment(&departments[node]),
		Depth:              depth,
		Children:           []DepartmentTreeNode{},
	}
	if depth >= maxDepartmentTreeDepth {
		return result
	}

	for _, child := range children[node] {
		result.Children = append(result.Children, buildDepartmentSubtree(departments, children, child, depth+1))
	}
	return result
}

// DepartmentRequest carries the writable fields of a department. Create and
// update take the same body.
//
// ParentID and ManagerID accept an omitted key, a JSON null and an empty
// string alike, all meaning "none": the department form sends "" for the
// top-level option.
type DepartmentRequest struct {
	Code      string `json:"code" binding:"required,max=20"`
	Name      string `json:"name" binding:"required,max=100"`
	NameEn    string `json:"name_en,omitempty" binding:"max=100"`
	ParentID  string `json:"parent_id,omitempty" binding:"omitempty,uuid"`
	ManagerID string `json:"manager_id,omitempty" binding:"omitempty,uuid"`
	IsActive  *bool  `json:"is_active,omitempty"`
}

// ApplyTo writes the request onto a department.
//
// It never touches CompanyID, ID, Level or Path. The company comes from the
// authenticated session and nothing a client sends may change it; Level is
// derived by the service from the parent; Path is the ltree column, which no
// Go code writes today.
func (r *DepartmentRequest) ApplyTo(department *domain.Department) error {
	parentID, err := parseOptionalUUID(r.ParentID)
	if err != nil {
		return err
	}
	managerID, err := parseOptionalUUID(r.ManagerID)
	if err != nil {
		return err
	}

	department.Code = r.Code
	department.Name = r.Name
	department.NameEn = r.NameEn
	department.ParentID = parentID
	department.ManagerID = managerID
	if r.IsActive != nil {
		department.IsActive = *r.IsActive
	}

	return nil
}

// MoveDepartmentRequest re-parents a department without rewriting its other
// fields. An omitted, null or empty parent_id moves it to the top level.
type MoveDepartmentRequest struct {
	ParentID string `json:"parent_id,omitempty" binding:"omitempty,uuid"`
}

// ParentUUID returns the requested parent, or nil for the top level.
func (r *MoveDepartmentRequest) ParentUUID() (*uuid.UUID, error) {
	return parseOptionalUUID(r.ParentID)
}
