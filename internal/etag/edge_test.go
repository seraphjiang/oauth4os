package etag

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Edge: matching ETag returns 304
func TestEdge_MatchingETag304(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	}))
	// First request to get ETag
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Fatal("first response should have ETag")
	}

	// Second request with If-None-Match
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("If-None-Match", etag)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 304 {
		t.Errorf("matching ETag should return 304, got %d", w.Code)
	}
}

// Edge: different content gets different ETag
func TestEdge_DifferentContentDifferentETag(t *testing.T) {
	content := "hello"
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(content))
	}))
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, httptest.NewRequest("GET", "/", nil))
	etag1 := w1.Header().Get("ETag")

	content = "world"
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, httptest.NewRequest("GET", "/", nil))
	etag2 := w2.Header().Get("ETag")

	if etag1 == etag2 {
		t.Error("different content should produce different ETags")
	}
}

// Edge: POST requests bypass ETag
func TestEdge_POSTBypassesETag(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("created"))
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/", nil))
	if w.Code != 200 {
		t.Errorf("POST should pass through, got %d", w.Code)
	}
}
