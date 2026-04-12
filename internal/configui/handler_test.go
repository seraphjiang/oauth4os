package configui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
)

func testCfg() *config.Config {
	return &config.Config{
		Upstream: config.Upstream{Engine: "http://localhost:9200"},
	}
}

func TestJSON(t *testing.T) {
	h := New(testCfg)
	w := httptest.NewRecorder()
	h.JSON(w, httptest.NewRequest("GET", "/admin/config.json", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var data map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&data); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestPage(t *testing.T) {
	h := New(testCfg)
	w := httptest.NewRecorder()
	h.Page(w, httptest.NewRequest("GET", "/admin/config", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "text/html") {
		t.Error("expected text/html content type")
	}
}

func TestRegister(t *testing.T) {
	h := New(testCfg)
	mux := http.NewServeMux()
	h.Register(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/admin/config", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
