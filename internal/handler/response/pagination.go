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

// ParsePagination extracts pagination parameters from request query
func ParsePagination(c *gin.Context) PaginationParams {
	page := parseIntQuery(c, "page", DefaultPage)
	perPage := parseIntQuery(c, "per_page", DefaultPerPage)

	// Enforce limits
	if page < 1 {
		page = DefaultPage
	}
	if perPage < 1 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}

	offset := (page - 1) * perPage

	return PaginationParams{
		Page:    page,
		PerPage: perPage,
		Offset:  offset,
	}
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
