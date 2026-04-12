package backup

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
)

func emptyCfg() *config.Config {
	return &config.Config{}
}

func noCli() []ClientEntry { return nil }

func noopApply(c *config.Config) {}

func TestExport_EmptyConfig(t *testing.T) {
	h := NewHandler(emptyCfg, noCli, noopApply)
	w := httptest.NewRecorder()
	h.Export(w, httptest.NewRequest("GET", "/admin/backup", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var b Bundle
	if err := json.Unmarshal(w.Body.Bytes(), &b); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if b.Version != "1" {
		t.Errorf("expected version 1, got %s", b.Version)
	}
}

func TestExport_ContentType(t *testing.T) {
	h := NewHandler(emptyCfg, noCli, noopApply)
	w := httptest.NewRecorder()
	h.Export(w, httptest.NewRequest("GET", "/admin/backup", nil))
	if !strings.Contains(w.Header().Get("Content-Type"), "json") {
		t.Error("export should return JSON")
	}
	if !strings.Contains(w.Header().Get("Content-Disposition"), "attachment") {
		t.Error("export should be downloadable")
	}
}

func TestImport_InvalidJSON(t *testing.T) {
	h := NewHandler(emptyCfg, noCli, noopApply)
	r := httptest.NewRequest("POST", "/admin/restore", strings.NewReader("{bad"))
	w := httptest.NewRecorder()
	h.Import(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestImport_AppliesConfig(t *testing.T) {
	var applied *config.Config
	h := NewHandler(emptyCfg, noCli, func(c *config.Config) { applied = c })
	body := `{"version":"1","providers":[{"name":"test","issuer":"https://test.example.com"}]}`
	r := httptest.NewRequest("POST", "/admin/restore", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.Import(w, r)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if applied == nil || len(applied.Providers) != 1 {
		t.Error("config should have been applied with 1 provider")
	}
}
