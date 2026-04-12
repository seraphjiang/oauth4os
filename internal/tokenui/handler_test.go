package tokenui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	h := New("https://proxy.example.com")
	if h.proxyURL != "https://proxy.example.com" {
		t.Errorf("expected proxy URL, got %s", h.proxyURL)
	}
}

func TestPage(t *testing.T) {
	h := New("https://proxy.example.com")
	w := httptest.NewRecorder()
	h.Page(w, httptest.NewRequest("GET", "/developer/tokens", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "proxy.example.com") {
		t.Error("page should contain proxy URL")
	}
	if !strings.Contains(body, "Token Inspector") {
		t.Error("page should contain title")
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html, got %s", ct)
	}
}

func TestRegister(t *testing.T) {
	h := New("https://proxy.example.com")
	mux := http.NewServeMux()
	h.Register(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/developer/tokens", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
