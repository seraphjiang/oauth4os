package tokenui

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEdge_PageReturnsHTML(t *testing.T) {
	h := New("https://proxy.example.com")
	w := httptest.NewRecorder()
	h.Page(w, httptest.NewRequest("GET", "/developer/tokens", nil))
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestEdge_RegisterAddsRoute(t *testing.T) {
	h := New("https://proxy.example.com")
	mux := http.NewServeMux()
	h.Register(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/developer/tokens", nil))
	if w.Code == 404 {
		t.Error("Register should add route")
	}
}
