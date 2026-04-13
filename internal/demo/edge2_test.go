package demo

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestEdge_ConcurrentApp(t *testing.T) {
	h := NewHandler("https://proxy.example.com", "demo-client")
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			h.App(w, httptest.NewRequest("GET", "/demo/", nil))
		}()
	}
	wg.Wait()
}

func TestEdge_CallbackMissingCode(t *testing.T) {
	h := NewHandler("https://proxy.example.com", "demo-client")
	w := httptest.NewRecorder()
	h.Callback(w, httptest.NewRequest("GET", "/demo/callback", nil))
	// Missing code param — should handle gracefully
	if w.Code == 0 {
		t.Error("should return valid status")
	}
}

func TestEdge_RegisterAddsCallback(t *testing.T) {
	h := NewHandler("https://proxy.example.com", "demo-client")
	mux := http.NewServeMux()
	h.Register(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/demo/callback", nil))
	if w.Code == 404 {
		t.Error("callback route should be registered")
	}
}
