package i18n

import (
	"net/http/httptest"
	"testing"
)

func TestEdge_AcceptLanguageHeader(t *testing.T) {
	r := httptest.NewRequest("GET", "/i18n/consent.json", nil)
	r.Header.Set("Accept-Language", "de")
	w := httptest.NewRecorder()
	Handler(w, r)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestEdge_NoAcceptLanguage(t *testing.T) {
	w := httptest.NewRecorder()
	Handler(w, httptest.NewRequest("GET", "/i18n/consent.json", nil))
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Error("should return translations")
	}
}
