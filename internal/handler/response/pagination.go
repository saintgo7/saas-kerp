package response

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	// DefaultPage is the default page number
	DefaultPage = 1
	// DefaultPerPage is the default items per page
	DefaultPerPage = 20
	// MaxPerPage is the maximum items per page
	MaxPerPage = 100
)

// PaginationParams holds pagination parameters from request
type PaginationParams struct {
	Page    int
	PerPage int
	Offset  int
}

// ParsePagination extracts pagination parameters from the request query and
// clamps them. Both "per_page" and "page_size" are accepted for the page size;
// the handlers in this codebase historically used the latter.
func ParsePagination(c *gin.Context) PaginationParams {
	return ParsePaginationWithDefault(c, DefaultPerPage)
}

// ParsePaginationWithDefault is ParsePagination with a caller-chosen default
// page size (still capped at MaxPerPage).
func ParsePaginationWithDefault(c *gin.Context, defaultPerPage int) PaginationParams {
	if defaultPerPage < 1 || defaultPerPage > MaxPerPage {
		defaultPerPage = DefaultPerPage
	}

	page := parseIntQuery(c, "page", DefaultPage)

	perPage := defaultPerPage
	if raw := c.Query("page_size"); raw != "" {
		perPage = parseIntQuery(c, "page_size", defaultPerPage)
	} else if raw := c.Query("per_page"); raw != "" {
		perPage = parseIntQuery(c, "per_page", defaultPerPage)
	}

	page, perPage = ClampPagination(page, perPage)

	return PaginationParams{
		Page:    page,
		PerPage: perPage,
		Offset:  (page - 1) * perPage,
	}
}

// ClampPagination forces page and perPage into the supported range. Callers must
// route every client-supplied page size through this before it reaches a
// repository LIMIT or a TotalPages division: page_size=0 otherwise panics with
// an integer divide by zero, and an unbounded page size loads a whole table.
func ClampPagination(page, perPage int) (int, int) {
	if page < 1 {
		page = DefaultPage
	}
	if perPage < 1 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}
	return page, perPage
}

// TotalPages computes the page count for a total and a page size. A page size of
// zero or less yields zero pages instead of panicking.
func TotalPages(total int64, perPage int) int {
	if perPage < 1 || total <= 0 {
		return 0
	}
	return int((total + int64(perPage) - 1) / int64(perPage))
}

// parseIntQuery parses an integer from query string with default value
func parseIntQuery(c *gin.Context, key string, defaultVal int) int {
	str := c.Query(key)
	if str == "" {
		return defaultVal
	}

	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultVal
	}

	return val
}

// SortParams holds sorting parameters from request
type SortParams struct {
	Field     string
	Direction string // "asc" or "desc"
}

// ParseSort extracts sorting parameters from request query
func ParseSort(c *gin.Context, allowedFields map[string]bool, defaultField, defaultDirection string) SortParams {
	field := c.Query("sort_by")
	direction := c.Query("sort_dir")

	// Validate field
	if field == "" || !allowedFields[field] {
		field = defaultField
	}

	// Validate direction
	if direction != "asc" && direction != "desc" {
		direction = defaultDirection
	}

	return SortParams{
		Field:     field,
		Direction: direction,
	}
}

// ListQuery combines pagination and sorting
type ListQuery struct {
	PaginationParams
	SortParams
	Search string
}

// ParseListQuery extracts all list query parameters
func ParseListQuery(c *gin.Context, allowedSortFields map[string]bool, defaultSort, defaultDirection string) ListQuery {
	return ListQuery{
		PaginationParams: ParsePagination(c),
		SortParams:       ParseSort(c, allowedSortFields, defaultSort, defaultDirection),
		Search:           c.Query("search"),
	}
}
