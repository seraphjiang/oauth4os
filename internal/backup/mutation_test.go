package backup

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
)

func TestMutation_ExportJSON(t *testing.T) {
	h := NewHandler(
		func() *config.Config { return &config.Config{} },
		func() []ClientEntry { return nil },
		nil,
	)
	w := httptest.NewRecorder()
	h.Export(w, httptest.NewRequest("GET", "/admin/backup/export", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		t.Fatalf("export must return valid JSON: %v", err)
	}
}

func TestMutation_ContentType(t *testing.T) {
	h := NewHandler(
		func() *config.Config { return &config.Config{} },
		func() []ClientEntry { return nil },
		nil,
	)
	w := httptest.NewRecorder()
	h.Export(w, httptest.NewRequest("GET", "/admin/backup/export", nil))
	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("export must set Content-Type: application/json")
	}
}

func TestMutation_ImportInvalid(t *testing.T) {
	h := NewHandler(nil, nil, func(c *config.Config) {})
	r := httptest.NewRequest("POST", "/admin/backup/import", nil)
	w := httptest.NewRecorder()
	h.Import(w, r)
	if w.Code == 200 {
		t.Error("empty body import should not return 200")
	}
}

// Mutation: remove Import success → valid import must return 200
func TestMutation_ImportValid(t *testing.T) {
	applied := false
	h := NewHandler(
		func() *config.Config { return &config.Config{} },
		func() []ClientEntry { return nil },
		func(c *config.Config) { applied = true },
	)
	body := `{"config":{},"clients":[]}`
	r := httptest.NewRequest("POST", "/admin/backup/import", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Import(w, r)
	if w.Code != 200 {
		t.Errorf("valid import should return 200, got %d: %s", w.Code, w.Body.String())
	}
	if !applied {
		t.Error("Import must call applyCfg")
	}
}
