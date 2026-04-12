package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
	"github.com/seraphjiang/oauth4os/internal/scope"
)

// Mutation: remove scope mappings endpoint → must return JSON
func TestMutation_ListScopeMappings(t *testing.T) {
	cfg := &config.Config{}
	m := scope.NewMapper(nil)
	s := NewState(cfg, m, nil)
	mux := http.NewServeMux()
	s.Register(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/admin/scope-mappings", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "json") {
		t.Error("must return JSON")
	}
}

// Mutation: remove providers endpoint → must return JSON
func TestMutation_ListProviders(t *testing.T) {
	cfg := &config.Config{}
	m := scope.NewMapper(nil)
	s := NewState(cfg, m, nil)
	mux := http.NewServeMux()
	s.Register(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/admin/providers", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
