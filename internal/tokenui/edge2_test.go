package tokenui

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestEdge_ConcurrentPage(t *testing.T) {
	h := New("https://proxy.example.com")
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			h.Page(w, httptest.NewRequest("GET", "/developer/tokens", nil))
		}()
	}
	wg.Wait()
}

func TestEdge_RegisterMultipleTimes(t *testing.T) {
	h := New("https://proxy.example.com")
	mux := http.NewServeMux()
	h.Register(mux)
	// Registering again on different mux should not panic
	mux2 := http.NewServeMux()
	h.Register(mux2)
}
