package dto

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// newTestDepartment builds one row the way the repository would hand it over.
func newTestDepartment(id uuid.UUID, code string, parent *uuid.UUID, level int) domain.Department {
	return domain.Department{
		TenantModel: domain.TenantModel{
			BaseModel: domain.BaseModel{ID: id},
			CompanyID: uuid.New(),
		},
		Code:     code,
		Name:     code,
		ParentID: parent,
		Level:    level,
		IsActive: true,
	}
}

// collectTreeIDs walks the built tree and records every node it can reach,
// along with the depth it was placed at.
func collectTreeIDs(nodes []DepartmentTreeNode, seen map[string]int) {
	for i := range nodes {
		seen[nodes[i].ID] = nodes[i].Depth
		collectTreeIDs(nodes[i].Children, seen)
	}
}

func TestBuildDepartmentTreeNests(t *testing.T) {
	ceo := uuid.New()
	dev := uuid.New()
	frontend := uuid.New()
	sales := uuid.New()

	tree := BuildDepartmentTree([]domain.Department{
		newTestDepartment(ceo, "CEO", nil, 1),
		newTestDepartment(dev, "DEV", &ceo, 2),
		newTestDepartment(sales, "SALES", &ceo, 2),
		newTestDepartment(frontend, "DEV-FE", &dev, 3),
	})

	if len(tree) != 1 {
		t.Fatalf("got %d roots, want 1", len(tree))
	}
	root := tree[0]
	if root.ID != ceo.String() {
		t.Fatalf("root is %s, want the CEO department", root.Code)
	}
	if root.Depth != 0 {
		t.Errorf("root depth = %d, want 0", root.Depth)
	}
	if len(root.Children) != 2 {
		t.Fatalf("root has %d children, want 2", len(root.Children))
	}

	// Sibling order follows the input order, which is the order the
	// repository sorted by (level, code).
	if root.Children[0].Code != "DEV" || root.Children[1].Code != "SALES" {
		t.Errorf("sibling order = %s, %s; want DEV, SALES",
			root.Children[0].Code, root.Children[1].Code)
	}

	grandchildren := root.Children[0].Children
	if len(grandchildren) != 1 || grandchildren[0].ID != frontend.String() {
		t.Fatalf("the frontend team was not nested under DEV: %+v", grandchildren)
	}
	if grandchildren[0].Depth != 2 {
		t.Errorf("grandchild depth = %d, want 2", grandchildren[0].Depth)
	}
}

// TestBuildDepartmentTreeChildrenIsNeverNull: the page iterates children
// without a guard, and `null` is not iterable.
func TestBuildDepartmentTreeChildrenIsNeverNull(t *testing.T) {
	tree := BuildDepartmentTree([]domain.Department{
		newTestDepartment(uuid.New(), "SOLO", nil, 1),
	})

	encoded, err := json.Marshal(tree)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if !strings.Contains(string(encoded), `"children":[]`) {
		t.Errorf("a leaf serialised as %s, want an empty children array", encoded)
	}
	if strings.Contains(string(encoded), `"children":null`) {
		t.Errorf("children serialised as null: %s", encoded)
	}
}

// TestBuildDepartmentTreeSurvivesSelfParent. The service refuses to store this,
// but the response builder must not be the thing that hangs if a row acquires
// it another way.
func TestBuildDepartmentTreeSurvivesSelfParent(t *testing.T) {
	id := uuid.New()
	self := newTestDepartment(id, "LOOP", &id, 1)

	tree := BuildDepartmentTree([]domain.Department{self})

	if len(tree) != 1 {
		t.Fatalf("got %d roots, want the self-parented row surfaced once", len(tree))
	}
	if len(tree[0].Children) != 0 {
		t.Errorf("the self link was followed: %+v", tree[0].Children)
	}
}

// TestBuildDepartmentTreeSurvivesCycle: A is B's parent and B is A's, which
// makes both unreachable from any root. They are shown at the top level rather
// than dropped, and the walk terminates.
func TestBuildDepartmentTreeSurvivesCycle(t *testing.T) {
	a := uuid.New()
	b := uuid.New()

	tree := BuildDepartmentTree([]domain.Department{
		newTestDepartment(a, "A", &b, 1),
		newTestDepartment(b, "B", &a, 1),
	})

	seen := map[string]int{}
	collectTreeIDs(tree, seen)
	if len(seen) != 2 {
		t.Fatalf("the cycle produced %d nodes, want both rows exactly once", len(seen))
	}
	if _, ok := seen[a.String()]; !ok {
		t.Error("A disappeared from the response")
	}
	if _, ok := seen[b.String()]; !ok {
		t.Error("B disappeared from the response")
	}
}

// TestBuildDepartmentTreeParentOutsideResultSet. GetTree selects only active
// departments, so an active child of an inactive parent has a parent id that
// names nothing in the list. It must still be visible.
func TestBuildDepartmentTreeParentOutsideResultSet(t *testing.T) {
	missing := uuid.New()
	child := uuid.New()

	tree := BuildDepartmentTree([]domain.Department{
		newTestDepartment(child, "ORPHAN", &missing, 2),
	})

	if len(tree) != 1 || tree[0].ID != child.String() {
		t.Fatalf("the orphan was not surfaced as a root: %+v", tree)
	}
	if tree[0].ParentID == nil || *tree[0].ParentID != missing.String() {
		t.Error("the stored parent id was rewritten; the response must still report it")
	}
}

// TestBuildDepartmentTreeIsBounded: a chain deeper than the cap terminates and
// still accounts for every row.
func TestBuildDepartmentTreeIsBounded(t *testing.T) {
	const chain = maxDepartmentTreeDepth * 2

	departments := make([]domain.Department, 0, chain)
	ids := make([]uuid.UUID, chain)
	for i := 0; i < chain; i++ {
		ids[i] = uuid.New()
		var parent *uuid.UUID
		if i > 0 {
			parent = &ids[i-1]
		}
		departments = append(departments, newTestDepartment(ids[i], "D", parent, i+1))
	}

	seen := map[string]int{}
	collectTreeIDs(BuildDepartmentTree(departments), seen)

	// Nothing is lost. A link is only recorded when the chain above it can be
	// walked to a root inside the cap, so a parent sits at depth 63 at the
	// deepest and its children at depth 64 - exactly where the subtree builder
	// stops descending. Everything below that surfaces as a new root instead
	// of being silently truncated away.
	if len(seen) != chain {
		t.Fatalf("a %d-deep chain produced %d nodes, want all of them", chain, len(seen))
	}
	for id, depth := range seen {
		if depth > maxDepartmentTreeDepth {
			t.Fatalf("%s was placed at depth %d, past the cap of %d",
				id, depth, maxDepartmentTreeDepth)
		}
	}
}

// TestFromDepartmentRendersAnAbsentParentAsNull. A client that builds a tree
// has to tell "no parent" from "not mentioned".
func TestFromDepartmentRendersAnAbsentParentAsNull(t *testing.T) {
	root := newTestDepartment(uuid.New(), "CEO", nil, 1)

	encoded, err := json.Marshal(FromDepartment(&root))
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if !strings.Contains(string(encoded), `"parent_id":null`) {
		t.Errorf("a root department serialised as %s, want an explicit null parent_id", encoded)
	}
	if !strings.Contains(string(encoded), `"manager_id":null`) {
		t.Errorf("an unmanaged department serialised as %s, want an explicit null manager_id", encoded)
	}
}

// TestDepartmentRequestApplyToLeavesTheTenantAlone. The company, the
// identifier and the derived level are not client input.
func TestDepartmentRequestApplyToLeavesTheTenantAlone(t *testing.T) {
	companyID := uuid.New()
	id := uuid.New()
	parent := uuid.New()

	department := domain.Department{
		TenantModel: domain.TenantModel{
			BaseModel: domain.BaseModel{ID: id},
			CompanyID: companyID,
		},
		Level:    3,
		Path:     "root.dev",
		IsActive: true,
	}

	req := DepartmentRequest{Code: "DEV", Name: "개발본부", ParentID: parent.String()}
	if err := req.ApplyTo(&department); err != nil {
		t.Fatalf("ApplyTo() error = %v", err)
	}

	if department.CompanyID != companyID {
		t.Errorf("CompanyID changed to %s", department.CompanyID)
	}
	if department.ID != id {
		t.Errorf("ID changed to %s", department.ID)
	}
	if department.Level != 3 {
		t.Errorf("Level = %d; the service derives it from the parent, the request must not", department.Level)
	}
	if department.Path != "root.dev" {
		t.Errorf("Path = %q; no request field writes the ltree column", department.Path)
	}
	if department.ParentID == nil || *department.ParentID != parent {
		t.Errorf("ParentID = %v, want %s", department.ParentID, parent)
	}
	if !department.IsActive {
		t.Error("an omitted is_active flipped the department to inactive")
	}
}

// TestDepartmentRequestTreatsBlankReferencesAsNone. The department form sends
// "" for the top-level option and for "no manager".
func TestDepartmentRequestTreatsBlankReferencesAsNone(t *testing.T) {
	parent := uuid.New()
	department := domain.Department{ParentID: &parent, ManagerID: &parent}

	req := DepartmentRequest{Code: "CEO", Name: "대표이사", ParentID: "", ManagerID: ""}
	if err := req.ApplyTo(&department); err != nil {
		t.Fatalf("ApplyTo() error = %v", err)
	}

	if department.ParentID != nil {
		t.Errorf("ParentID = %v, want nil", department.ParentID)
	}
	if department.ManagerID != nil {
		t.Errorf("ManagerID = %v, want nil", department.ManagerID)
	}
}

// TestMoveDepartmentRequestParentUUID covers the three ways a client says
// "move it to the top level".
func TestMoveDepartmentRequestParentUUID(t *testing.T) {
	for _, body := range []string{`{}`, `{"parent_id":""}`, `{"parent_id":null}`} {
		var req MoveDepartmentRequest
		if err := json.Unmarshal([]byte(body), &req); err != nil {
			t.Fatalf("Unmarshal(%s) error = %v", body, err)
		}
		parent, err := req.ParentUUID()
		if err != nil {
			t.Fatalf("ParentUUID() for %s error = %v", body, err)
		}
		if parent != nil {
			t.Errorf("%s gave parent %s, want nil", body, parent)
		}
	}

	target := uuid.New()
	req := MoveDepartmentRequest{ParentID: target.String()}
	parent, err := req.ParentUUID()
	if err != nil {
		t.Fatalf("ParentUUID() error = %v", err)
	}
	if parent == nil || *parent != target {
		t.Errorf("ParentUUID() = %v, want %s", parent, target)
	}
}
