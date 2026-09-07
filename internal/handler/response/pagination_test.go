package response

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestClampPagination(t *testing.T) {
	cases := []struct {
		name              string
		page, perPage     int
		wantPage, wantPer int
	}{
		{"zero page size is replaced, not passed through", 1, 0, 1, DefaultPerPage},
		{"negative page size is replaced", 1, -5, 1, DefaultPerPage},
		{"page size above the cap is capped", 1, 100000000, 1, MaxPerPage},
		{"zero page becomes the first page", 0, 20, 1, 20},
		{"negative page becomes the first page", -3, 20, 1, 20},
		{"values inside the range are untouched", 4, 50, 4, 50},
		{"the cap itself is allowed", 1, MaxPerPage, 1, MaxPerPage},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			page, perPage := ClampPagination(tc.page, tc.perPage)
			if page != tc.wantPage || perPage != tc.wantPer {
				t.Errorf("ClampPagination(%d, %d) = (%d, %d), want (%d, %d)",
					tc.page, tc.perPage, page, perPage, tc.wantPage, tc.wantPer)
			}
		})
	}
}

// TotalPages is the division that used to panic with page_size=0.
func TestTotalPages(t *testing.T) {
	cases := []struct {
		total   int64
		perPage int
		want    int
	}{
		{0, 20, 0},
		{1, 20, 1},
		{20, 20, 1},
		{21, 20, 2},
		{100, 7, 15},
		{42, 0, 0},  // must not panic
		{42, -1, 0}, // must not panic
		{-5, 20, 0},
	}

	for _, tc := range cases {
		if got := TotalPages(tc.total, tc.perPage); got != tc.want {
			t.Errorf("TotalPages(%d, %d) = %d, want %d", tc.total, tc.perPage, got, tc.want)
		}
	}
}

func TestParsePaginationWithDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)

	parse := func(query string, def int) PaginationParams {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/?"+query, nil)
		return ParsePaginationWithDefault(c, def)
	}

	if got := parse("page_size=0", 20); got.PerPage != DefaultPerPage {
		t.Errorf("page_size=0 gave PerPage=%d, want %d", got.PerPage, DefaultPerPage)
	}
	if got := parse("page_size=100000", 20); got.PerPage != MaxPerPage {
		t.Errorf("an unbounded page size was not capped: %d", got.PerPage)
	}
	// "12abc" used to parse as 12 through fmt.Sscanf; strconv.Atoi rejects it and
	// the default applies.
	if got := parse("page_size=12abc", 20); got.PerPage != 20 {
		t.Errorf("a malformed page size gave PerPage=%d, want the default 20", got.PerPage)
	}
	if got := parse("", 50); got.PerPage != 50 || got.Page != 1 {
		t.Errorf("empty query gave (%d, %d), want (1, 50)", got.Page, got.PerPage)
	}
	if got := parse("page=3&per_page=25", 20); got.PerPage != 25 || got.Page != 3 || got.Offset != 50 {
		t.Errorf("per_page alias not honoured: %+v", got)
	}
	if got := parse("page=2&page_size=10", 20); got.Offset != 10 {
		t.Errorf("offset = %d, want 10", got.Offset)
	}
}
