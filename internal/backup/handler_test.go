package backup

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
)

func testSetup() (*Handler, *http.ServeMux) {
	cfg := &config.Config{
		Providers:    []config.Provider{{Name: "kc", Issuer: "https://kc.example.com"}},
		ScopeMapping: map[string]config.Role{"admin": {BackendRoles: []string{"all_access"}}},
		Tenants:      map[string]config.Tenant{},
	}
	h := NewHandler(
		func() *config.Config { return cfg },
		func() []ClientEntry { return []ClientEntry{{ID: "test-client", Scopes: []string{"admin"}}} },
		func(c *config.Config) { *cfg = *c },
	)
	mux := http.NewServeMux()
	h.Register(mux)
	return h, mux
}

func TestExport(t *testing.T) {
	_, mux := testSetup()
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/admin/backup", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var bundle Bundle
	json.NewDecoder(w.Body).Decode(&bundle)
	if bundle.Version != "1" || len(bundle.Providers) != 1 || len(bundle.Clients) != 1 {
		t.Fatalf("unexpected bundle: %+v", bundle)
	}
}

func TestRoundTrip(t *testing.T) {
	_, mux := testSetup()

	// Export
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/admin/backup", nil))
	exported := w.Body.Bytes()

	// Modify the bundle
	var bundle Bundle
	json.Unmarshal(exported, &bundle)
	bundle.Providers = append(bundle.Providers, config.Provider{Name: "new", Issuer: "https://new.example.com"})
	modified, _ := json.Marshal(bundle)

	// Import
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/admin/restore", bytes.NewReader(modified)))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Re-export and verify
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/admin/backup", nil))
	var result Bundle
	json.NewDecoder(w.Body).Decode(&result)
	if len(result.Providers) != 2 {
		t.Fatalf("expected 2 providers after restore, got %d", len(result.Providers))
	}
}
