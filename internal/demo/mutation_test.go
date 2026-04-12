package demo

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// Mutation: remove HTML rendering → App must return HTML
func TestMutation_AppHTML(t *testing.T) {
	h := NewHandler("https://proxy.example.com", "demo-client")
	w := httptest.NewRecorder()
	h.App(w, httptest.NewRequest("GET", "/demo/app", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "html") {
		t.Error("must return HTML")
	}
}

// Mutation: remove client_id → page must contain client_id
func TestMutation_ClientIDInPage(t *testing.T) {
	h := NewHandler("https://proxy.example.com", "my-demo-client")
	w := httptest.NewRecorder()
	h.App(w, httptest.NewRequest("GET", "/demo/app", nil))
	if !strings.Contains(w.Body.String(), "my-demo-client") {
		t.Error("demo page must contain the client_id")
	}
}
