package i18n

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestHandlerDefaultEnglish(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/i18n", nil)
	Handler(w, r)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var data map[string]string
	json.NewDecoder(w.Body).Decode(&data)
	if data["approve"] != "Approve" {
		t.Errorf("expected English 'Approve', got %q", data["approve"])
	}
}

func TestHandlerSpanish(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/i18n?lang=es", nil)
	Handler(w, r)
	var data map[string]string
	json.NewDecoder(w.Body).Decode(&data)
	if data["approve"] != "Aprobar" {
		t.Errorf("expected Spanish 'Aprobar', got %q", data["approve"])
	}
}

func TestHandlerUnknownLangFallback(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/i18n?lang=xx", nil)
	Handler(w, r)
	var data map[string]string
	json.NewDecoder(w.Body).Decode(&data)
	if data["approve"] != "Approve" {
		t.Errorf("expected English fallback, got %q", data["approve"])
	}
}
