package demo

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewHandler(t *testing.T) {
	h := NewHandler("https://proxy.example.com/", "client_123")
	if h.proxyURL != "https://proxy.example.com" {
		t.Errorf("expected trailing slash stripped, got %s", h.proxyURL)
	}
	if h.clientID != "client_123" {
		t.Errorf("expected client_123, got %s", h.clientID)
	}
}

func TestAppPage(t *testing.T) {
	h := NewHandler("https://proxy.example.com", "demo-client")
	w := httptest.NewRecorder()
	h.App(w, httptest.NewRequest("GET", "/demo", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "proxy.example.com") {
		t.Error("app page should contain proxy URL")
	}
	if !strings.Contains(body, "demo-client") {
		t.Error("app page should contain client ID")
	}
	if !strings.Contains(body, "oauth4os") {
		t.Error("app page should contain product name")
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html, got %s", ct)
	}
}

func TestCallbackPage(t *testing.T) {
	h := NewHandler("https://proxy.example.com", "demo-client")
	w := httptest.NewRecorder()
	h.Callback(w, httptest.NewRequest("GET", "/demo/callback?code=abc123", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "authorization_code") {
		t.Error("callback should contain grant_type reference")
	}
	if !strings.Contains(body, "proxy.example.com") {
		t.Error("callback should contain proxy URL")
	}
}

func TestRegisterRoutes(t *testing.T) {
	h := NewHandler("https://proxy.example.com", "demo-client")
	mux := http.NewServeMux()
	h.Register(mux)

	// /demo serves app
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/demo", nil))
	if w.Code != 200 {
		t.Errorf("/demo: expected 200, got %d", w.Code)
	}

	// /demo/ also serves app
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/demo/", nil))
	if w.Code != 200 {
		t.Errorf("/demo/: expected 200, got %d", w.Code)
	}

	// /demo/callback serves callback
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/demo/callback", nil))
	if w.Code != 200 {
		t.Errorf("/demo/callback: expected 200, got %d", w.Code)
	}
}
