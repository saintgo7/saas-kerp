package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/errors"
	"github.com/saintgo7/saas-kerp/internal/handler/response"
)

// parsePageParams reads the "page" and "page_size" query parameters and clamps
// them to the supported range.
//
// Every list handler must go through this. Reading page_size straight from the
// query gives two defects at once: page_size=0 reaches the total-pages division
// and panics with "integer divide by zero", and an arbitrarily large page_size
// turns one request into a full-table scan and serialization.
func parsePageParams(c *gin.Context, defaultPageSize int) (page, pageSize int) {
	p := response.ParsePaginationWithDefault(c, defaultPageSize)
	return p.Page, p.PerPage
}

// listMeta builds the pagination metadata for a list response, computing the
// page count without dividing by a client-controlled value.
func listMeta(total int64, page, pageSize int) *dto.MetaInfo {
	page, pageSize = response.ClampPagination(page, pageSize)
	return &dto.MetaInfo{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: response.TotalPages(total, pageSize),
	}
}

// notImplementedRoute answers a reserved collection path that this API does not
// implement yet.
//
// It exists so such a path is matched by its own route rather than by the ":id"
// wildcard next to it. Without it, GET /vouchers/next-number is parsed as a
// voucher ID, fails uuid.Parse and answers 400 "Invalid voucher ID", which tells
// a client that its identifier was malformed when in fact the endpoint does not
// exist.
func notImplementedRoute(c *gin.Context) {
	response.Error(c, http.StatusNotFound, errors.CodeNotFound, "Endpoint not found")
}
