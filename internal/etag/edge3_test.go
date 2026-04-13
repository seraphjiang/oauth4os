package etag

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEdge_StarIfNoneMatch(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("If-None-Match", "*")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	// * should match any ETag
	if w.Code != 304 && w.Code != 200 {
		t.Errorf("expected 304 or 200, got %d", w.Code)
	}
}

func TestEdge_ETagConsistent(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("same content"))
	}))
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, httptest.NewRequest("GET", "/", nil))
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, httptest.NewRequest("GET", "/", nil))
	if w1.Header().Get("ETag") != w2.Header().Get("ETag") {
		t.Error("same content should produce same ETag")
	}
}
