package i18n

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEdge_HandlerReturnsJSON(t *testing.T) {
	w := httptest.NewRecorder()
	Handler(w, httptest.NewRequest("GET", "/i18n/consent.json", nil))
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "json") {
		t.Error("should return JSON content type")
	}
}

func TestEdge_HandlerContainsEnglish(t *testing.T) {
	w := httptest.NewRecorder()
	Handler(w, httptest.NewRequest("GET", "/i18n/consent.json", nil))
	if !strings.Contains(w.Body.String(), "en") {
		t.Error("response should contain English translations")
	}
}
