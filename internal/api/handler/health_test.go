package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mugishajc/orderforge/internal/api/handler"
)

func TestLiveness(t *testing.T) {
	h := handler.NewHealthHandler(nil) // liveness does not need db
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	h.Liveness(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
