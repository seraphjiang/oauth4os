package backup

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Edge cases for backup/restore.

func TestExport_EmptyConfig(t *testing.T) {
	h := NewHandler(
		func() interface{} { return map[string]interface{}{} },
		func() interface{} { return []interface{}{} },
		nil,
	)
	r := httptest.NewRequest("GET", "/admin/config/export", nil)
	w := httptest.NewRecorder()
	h.Export(w, r)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("response should be valid JSON: %v", err)
	}
}

func TestImport_EmptyBody(t *testing.T) {
	applied := false
	h := NewHandler(nil, nil, func(data json.RawMessage) error {
		applied = true
		return nil
	})
	r := httptest.NewRequest("POST", "/admin/config/import", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Import(w, r)
	// Should not crash on empty import
	if w.Code >= 500 {
		t.Errorf("empty import should not cause 5xx, got %d", w.Code)
	}
}

func TestImport_InvalidJSON(t *testing.T) {
	h := NewHandler(nil, nil, func(data json.RawMessage) error { return nil })
	r := httptest.NewRequest("POST", "/admin/config/import", strings.NewReader("{invalid"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Import(w, r)
	if w.Code != http.StatusBadRequest && w.Code < 400 {
		t.Errorf("invalid JSON should return 4xx, got %d", w.Code)
	}
}

func TestExport_ContentType(t *testing.T) {
	h := NewHandler(
		func() interface{} { return nil },
		func() interface{} { return nil },
		nil,
	)
	r := httptest.NewRequest("GET", "/admin/config/export", nil)
	w := httptest.NewRecorder()
	h.Export(w, r)
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "json") {
		t.Errorf("export should return JSON content type, got %s", ct)
	}
}
