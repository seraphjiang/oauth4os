package introspect

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEdge_EmptyTokenParam(t *testing.T) {
	h := NewHandler(&stubLookup{})
	body := "token="
	r := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Errorf("empty token should return 200 with active:false, got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), `"active":true`) {
		t.Error("empty token should not be active")
	}
}

func TestEdge_LargeToken(t *testing.T) {
	h := NewHandler(&stubLookup{})
	body := "token=" + strings.Repeat("x", 10000)
	r := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	// Should not panic on large token
	if w.Code == 0 {
		t.Error("should return valid status")
	}
}
