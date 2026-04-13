package configui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
)

// Mutation: remove JSON encoding → JSON endpoint must return valid JSON
func TestMutation_JSONEndpoint(t *testing.T) {
	h := New(func() *config.Config { return &config.Config{} })
	w := httptest.NewRecorder()
	h.JSON(w, httptest.NewRequest("GET", "/admin/config.json", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "json") {
		t.Error("must return JSON content type")
	}
}

// Mutation: remove HTML page → Page must return HTML
func TestMutation_HTMLPage(t *testing.T) {
	h := New(func() *config.Config { return &config.Config{} })
	w := httptest.NewRecorder()
	h.Page(w, httptest.NewRequest("GET", "/admin/config", nil))
	if !strings.Contains(w.Header().Get("Content-Type"), "html") {
		t.Error("Page must return HTML content type")
	}
}

// Mutation: remove Register → routes must be registered on mux
func TestMutation_RegisterRoutes(t *testing.T) {
	h := New(func() *config.Config { return &config.Config{} })
	mux := http.NewServeMux()
	h.Register(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/admin/config", nil))
	if w.Code == 404 {
		t.Error("Register must add config routes to mux")
	}
}
