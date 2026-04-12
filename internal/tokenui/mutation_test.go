package tokenui

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// Mutation: remove HTML rendering → Page must return HTML
func TestMutation_PageHTML(t *testing.T) {
	h := New("https://proxy.example.com")
	w := httptest.NewRecorder()
	h.Page(w, httptest.NewRequest("GET", "/oauth/tokens", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "html") {
		t.Error("must return HTML")
	}
}

// Mutation: remove proxy URL → page must contain proxy URL
func TestMutation_ProxyURL(t *testing.T) {
	h := New("https://my-proxy.example.com")
	w := httptest.NewRecorder()
	h.Page(w, httptest.NewRequest("GET", "/oauth/tokens", nil))
	if !strings.Contains(w.Body.String(), "my-proxy.example.com") {
		t.Error("page must contain the proxy URL")
	}
}
