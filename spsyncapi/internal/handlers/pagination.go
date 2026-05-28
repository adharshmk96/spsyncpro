package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 100
)

// Pagination describes a paginated result set.
type Pagination struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

// parsePagination reads page and limit query parameters.
// Returns page, limit, offset. Writes 400 and returns ok=false on invalid input.
func parsePagination(c *gin.Context) (page, limit, offset int, ok bool) {
	page = defaultPage
	limit = defaultLimit

	if raw := c.Query("page"); raw != "" {
		p, err := strconv.Atoi(raw)
		if err != nil || p < 1 {
			respondError(c, http.StatusBadRequest, "page must be a positive integer")
			return 0, 0, 0, false
		}
		page = p
	}

	if raw := c.Query("limit"); raw != "" {
		l, err := strconv.Atoi(raw)
		if err != nil || l < 1 {
			respondError(c, http.StatusBadRequest, "limit must be a positive integer")
			return 0, 0, 0, false
		}
		if l > maxLimit {
			respondError(c, http.StatusBadRequest, fmt.Sprintf("limit must not exceed %d", maxLimit))
			return 0, 0, 0, false
		}
		limit = l
	}

	offset = (page - 1) * limit
	return page, limit, offset, true
}
