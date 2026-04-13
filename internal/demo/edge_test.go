package demo

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEdge_AppReturnsHTML(t *testing.T) {
	h := NewHandler("https://proxy.example.com", "demo-client")
	w := httptest.NewRecorder()
	h.App(w, httptest.NewRequest("GET", "/demo/", nil))
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestEdge_RegisterAddsRoutes(t *testing.T) {
	h := NewHandler("https://proxy.example.com", "demo-client")
	mux := http.NewServeMux()
	h.Register(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/demo/", nil))
	if w.Code == 404 {
		t.Error("Register should add /demo/ route")
	}
}
