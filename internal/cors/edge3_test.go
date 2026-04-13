package cors

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestEdge_ConcurrentRequests(t *testing.T) {
	wrap := Middleware(Config{Origins: []string{"https://example.com"}})
	h := wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := httptest.NewRequest("GET", "/", nil)
			r.Header.Set("Origin", "https://example.com")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
		}()
	}
	wg.Wait()
}

func TestEdge_CORSHeadersOnNormalRequest(t *testing.T) {
	wrap := Middleware(Config{Origins: []string{"https://example.com"}})
	h := wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	r := httptest.NewRequest("GET", "/api/data", nil)
	r.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Error("normal request should include CORS origin header")
	}
}
