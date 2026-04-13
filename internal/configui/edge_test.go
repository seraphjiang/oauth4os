package configui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
)

func TestEdge_PageReturnsHTML(t *testing.T) {
	h := New(func() *config.Config { return &config.Config{Listen: ":8443"} })
	mux := http.NewServeMux()
	h.Register(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/admin/config", nil))
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestEdge_JSONReturnsConfig(t *testing.T) {
	h := New(func() *config.Config { return &config.Config{Listen: ":8443"} })
	mux := http.NewServeMux()
	h.Register(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/admin/config/json", nil))
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "8443") {
		t.Error("JSON should contain config values")
	}
}
