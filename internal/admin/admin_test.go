package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/cedar"
	"github.com/seraphjiang/oauth4os/internal/config"
	"github.com/seraphjiang/oauth4os/internal/scope"
)

func setup() (*State, *http.ServeMux) {
	cfg := &config.Config{
		ScopeMapping: map[string]config.Role{
			"admin": {BackendRoles: []string{"all_access"}},
		},
		Tenants: make(map[string]config.Tenant),
	}
	mapper := scope.NewMultiTenantMapper(cfg.ScopeMapping, cfg.Tenants)
	eng := cedar.NewTenantEngine(nil)
	s := NewState(cfg, mapper, eng)
	mux := http.NewServeMux()
	s.Register(mux)
	return s, mux
}

func TestListScopeMappings(t *testing.T) {
	_, mux := setup()
	req := httptest.NewRequest("GET", "/admin/scope-mappings", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result map[string]config.Role
	json.NewDecoder(w.Body).Decode(&result)
	if _, ok := result["admin"]; !ok {
		t.Fatal("expected admin scope mapping")
	}
}

func TestUpdateScopeMappings(t *testing.T) {
	_, mux := setup()
	body, _ := json.Marshal(map[string]config.Role{
		"read:logs": {BackendRoles: []string{"logs_reader"}},
	})
	req := httptest.NewRequest("PUT", "/admin/scope-mappings", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddAndRemoveProvider(t *testing.T) {
	_, mux := setup()
	// Add
	body, _ := json.Marshal(config.Provider{Name: "test", Issuer: "https://test.example.com"})
	req := httptest.NewRequest("POST", "/admin/providers", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	// Duplicate
	req = httptest.NewRequest("POST", "/admin/providers", bytes.NewReader(body))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 409 {
		t.Fatalf("expected 409 conflict, got %d", w.Code)
	}
	// Remove
	req = httptest.NewRequest("DELETE", "/admin/providers/test", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 204 {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	// Remove again
	req = httptest.NewRequest("DELETE", "/admin/providers/test", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUpdateAndRemoveTenant(t *testing.T) {
	_, mux := setup()
	body, _ := json.Marshal(config.Tenant{
		ScopeMapping: map[string]config.Role{"read:*": {BackendRoles: []string{"reader"}}},
	})
	req := httptest.NewRequest("PUT", "/admin/tenants/https://kc.example.com", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// List
	req = httptest.NewRequest("GET", "/admin/tenants", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var tenants map[string]config.Tenant
	json.NewDecoder(w.Body).Decode(&tenants)
	if _, ok := tenants["https://kc.example.com"]; !ok {
		t.Fatal("expected tenant")
	}
	// Remove
	req = httptest.NewRequest("DELETE", "/admin/tenants/https://kc.example.com", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 204 {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestCedarPolicyCRUD(t *testing.T) {
	_, mux := setup()
	// List (should have defaults from setup)
	req := httptest.NewRequest("GET", "/admin/cedar-policies", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("list: expected 200, got %d", w.Code)
	}

	// Add forbid policy
	body, _ := json.Marshal(CedarPolicyInput{ID: "block-secret", Effect: "forbid", Resource: ".secret-index"})
	req = httptest.NewRequest("POST", "/admin/cedar-policies", bytes.NewReader(body))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("add: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Add without ID → 400
	body, _ = json.Marshal(CedarPolicyInput{Effect: "permit"})
	req = httptest.NewRequest("POST", "/admin/cedar-policies", bytes.NewReader(body))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("add no id: expected 400, got %d", w.Code)
	}

	// Remove
	req = httptest.NewRequest("DELETE", "/admin/cedar-policies/block-secret", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 204 {
		t.Fatalf("remove: expected 204, got %d", w.Code)
	}

	// Remove again → 404
	req = httptest.NewRequest("DELETE", "/admin/cedar-policies/block-secret", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Fatalf("remove again: expected 404, got %d", w.Code)
	}
}

func TestRateLimitCRUD(t *testing.T) {
	_, mux := setup()
	// List
	req := httptest.NewRequest("GET", "/admin/rate-limits", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("list: expected 200, got %d", w.Code)
	}

	// Update
	body, _ := json.Marshal(map[string]int{"read:logs-*": 1000, "admin": 30})
	req = httptest.NewRequest("PUT", "/admin/rate-limits", bytes.NewReader(body))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("update: expected 200, got %d", w.Code)
	}

	// Verify
	req = httptest.NewRequest("GET", "/admin/rate-limits", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var limits map[string]int
	json.NewDecoder(w.Body).Decode(&limits)
	if limits["admin"] != 30 {
		t.Fatalf("admin limit = %d, want 30", limits["admin"])
	}
}
