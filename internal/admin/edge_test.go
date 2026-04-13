package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
	"github.com/seraphjiang/oauth4os/internal/scope"
)

func testAdmin() (*State, *http.ServeMux) {
	s := NewState(&config.Config{}, scope.NewMapper(nil), nil)
	mux := http.NewServeMux()
	s.Register(mux)
	return s, mux
}

func TestEdge_ListScopeMappingsReturns200(t *testing.T) {
	_, mux := testAdmin()
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/admin/scope-mappings", nil))
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestEdge_ListProvidersReturns200(t *testing.T) {
	_, mux := testAdmin()
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/admin/providers", nil))
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestEdge_ListTenantsReturns200(t *testing.T) {
	_, mux := testAdmin()
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/admin/tenants", nil))
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
