package i18n

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestHandlerReturnsAllTranslations(t *testing.T) {
	w := httptest.NewRecorder()
	Handler(w, httptest.NewRequest("GET", "/i18n", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var data map[string]map[string]string
	if err := json.NewDecoder(w.Body).Decode(&data); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if data["en"]["approve"] != "Approve" {
		t.Errorf("expected en.approve='Approve', got %q", data["en"]["approve"])
	}
	if data["es"]["approve"] != "Aprobar" {
		t.Errorf("expected es.approve='Aprobar', got %q", data["es"]["approve"])
	}
}

func TestHandlerContentType(t *testing.T) {
	w := httptest.NewRecorder()
	Handler(w, httptest.NewRequest("GET", "/i18n", nil))
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}
}

func TestHandlerCacheHeader(t *testing.T) {
	w := httptest.NewRecorder()
	Handler(w, httptest.NewRequest("GET", "/i18n", nil))
	cc := w.Header().Get("Cache-Control")
	if cc == "" {
		t.Error("expected Cache-Control header")
	}
}
