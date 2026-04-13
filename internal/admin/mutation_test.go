package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/cedar"
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

// Mutation: remove updateScopeMappings → POST must update mappings
func TestMutation_UpdateScopeMappings(t *testing.T) {
	cfg := &config.Config{}
	m := scope.NewMapper(nil)
	s := NewState(cfg, m, nil)
	mux := http.NewServeMux()
	s.Register(mux)
	body := `{"admin":{"backend_user":"admin","backend_roles":["all_access"]}}`
	r := httptest.NewRequest("PUT", "/admin/scope-mappings", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Errorf("update should return 200, got %d: %s", w.Code, w.Body.String())
	}
}

// Mutation: remove listTenants → must return tenant list
func TestMutation_ListTenants(t *testing.T) {
	cfg := &config.Config{}
	m := scope.NewMapper(nil)
	s := NewState(cfg, m, nil)
	mux := http.NewServeMux()
	s.Register(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/admin/tenants", nil))
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// Mutation: remove listCedarPolicies → must return policies
func TestMutation_ListCedarPolicies(t *testing.T) {
	cfg := &config.Config{}
	m := scope.NewMapper(nil)
	te := cedar.NewTenantEngine(nil)
	s := NewState(cfg, m, te)
	mux := http.NewServeMux()
	s.Register(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/admin/cedar-policies", nil))
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
