package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParsePaginationDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	page, limit, offset, ok := parsePagination(c)
	if !ok {
		t.Fatal("expected ok")
	}
	if page != 1 || limit != 20 || offset != 0 {
		t.Fatalf("defaults: page=%d limit=%d offset=%d", page, limit, offset)
	}
}

func TestParsePaginationInvalidPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?page=0", nil)

	_, _, _, ok := parsePagination(c)
	if ok {
		t.Fatal("expected invalid page")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d", w.Code)
	}
}

func TestParsePaginationLimitTooHigh(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?limit=200", nil)

	_, _, _, ok := parsePagination(c)
	if ok {
		t.Fatal("expected limit error")
	}
}
