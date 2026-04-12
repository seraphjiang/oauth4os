package etag

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestETagAdded(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Header().Get("ETag") == "" {
		t.Fatal("expected ETag header")
	}
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestETag304(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))
	// First request to get ETag
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	tag := rec.Header().Get("ETag")

	// Second request with If-None-Match
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("If-None-Match", tag)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req)
	if rec2.Code != 304 {
		t.Fatalf("expected 304, got %d", rec2.Code)
	}
}

func TestETagSkipsPOST(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/", nil))
	if rec.Header().Get("ETag") != "" {
		t.Fatal("POST should not get ETag")
	}
}
