package etag

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestEdge_ConcurrentRequests(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		}()
	}
	wg.Wait()
}

func TestEdge_EmptyBodyETag(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 204 {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestEdge_WeakETagMismatch(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("If-None-Match", `W/"wrong-etag"`)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code == 304 {
		t.Error("wrong ETag should not return 304")
	}
}
