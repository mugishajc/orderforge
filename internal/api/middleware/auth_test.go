package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mugishajc/orderforge/internal/api/middleware"
)

func TestAuth_MissingHeader(t *testing.T) {
	handler := middleware.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuth_ValidToken(t *testing.T) {
	var capturedVal string
	handler := middleware.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedVal, _ = r.Context().Value(middleware.UserIDKey).(string)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer test-token-123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if capturedVal != "test-token-123" {
		t.Errorf("expected token test-token-123, got %s", capturedVal)
	}
}

func TestAuth_EmptyBearer(t *testing.T) {
	handler := middleware.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
