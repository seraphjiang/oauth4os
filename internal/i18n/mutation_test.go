package i18n

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// Mutation: remove JSON response → must return valid JSON
func TestMutation_JSONResponse(t *testing.T) {
	w := httptest.NewRecorder()
	Handler(w, httptest.NewRequest("GET", "/i18n/consent.json", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "json") {
		t.Error("must return JSON content type")
	}
}

// Mutation: remove consent translations → must include consent strings
func TestMutation_ConsentStrings(t *testing.T) {
	w := httptest.NewRecorder()
	Handler(w, httptest.NewRequest("GET", "/i18n/consent.json", nil))
	body := w.Body.String()
	if body == "" || body == "{}" || body == "null" {
		t.Error("must return non-empty consent translations")
	}
}

// Mutation: remove language fallback → unknown language must fall back to English
func TestMutation_LanguageFallback(t *testing.T) {
	h := Handler()
	r := httptest.NewRequest("GET", "/i18n/consent.json?lang=xx", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// Should return English as fallback
	if !strings.Contains(w.Body.String(), "consent") {
		t.Error("fallback must return consent strings")
	}
}
