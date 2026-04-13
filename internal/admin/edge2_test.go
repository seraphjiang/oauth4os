package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
	"github.com/seraphjiang/oauth4os/internal/scope"
)

func TestEdge_AddProviderValid(t *testing.T) {
	s := NewState(&config.Config{}, scope.NewMapper(nil), nil)
	mux := http.NewServeMux()
	s.Register(mux)
	body := `{"name":"github","issuer":"https://github.com","client_id":"abc","client_secret":"xyz"}`
	r := httptest.NewRequest("POST", "/admin/providers", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 200 && w.Code != 201 {
		t.Errorf("expected 200/201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEdge_AddProviderInvalidJSON(t *testing.T) {
	s := NewState(&config.Config{}, scope.NewMapper(nil), nil)
	mux := http.NewServeMux()
	s.Register(mux)
	r := httptest.NewRequest("POST", "/admin/providers", strings.NewReader("not json"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code == 200 {
		t.Error("invalid JSON should fail")
	}
}

func TestEdge_RemoveProviderNotFound(t *testing.T) {
	s := NewState(&config.Config{}, scope.NewMapper(nil), nil)
	mux := http.NewServeMux()
	s.Register(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("DELETE", "/admin/providers/nonexistent", nil))
	if w.Code == 200 {
		t.Error("removing nonexistent provider should fail")
	}
}
