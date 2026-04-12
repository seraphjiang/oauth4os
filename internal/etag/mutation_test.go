package etag

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// M1: Mutate method guard — PUT/DELETE/PATCH must bypass ETag logic.
func TestMutation_NonGETMethodsSkipped(t *testing.T) {
	for _, m := range []string{"PUT", "DELETE", "PATCH"} {
		rec := httptest.NewRecorder()
		h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		}))
		h.ServeHTTP(rec, httptest.NewRequest(m, "/", nil))
		if rec.Header().Get("ETag") != "" {
			t.Fatalf("%s should not get ETag", m)
		}
	}
}

// M2: HEAD must get ETag (same as GET).
func TestMutation_HEADGetsETag(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("data"))
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("HEAD", "/", nil))
	if rec.Header().Get("ETag") == "" {
		t.Fatal("HEAD should get ETag")
	}
}

// M3: 304 must have empty body.
func TestMutation_304EmptyBody(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	tag := rec.Header().Get("ETag")

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("If-None-Match", tag)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req)
	if rec2.Code != 304 {
		t.Fatalf("expected 304, got %d", rec2.Code)
	}
	if rec2.Body.Len() > 0 {
		t.Fatal("304 response must have empty body")
	}
}

// M4: Different content must produce different ETags.
func TestMutation_DifferentContentDifferentETag(t *testing.T) {
	var body string
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))

	body = "aaa"
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, httptest.NewRequest("GET", "/", nil))

	body = "bbb"
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest("GET", "/", nil))

	if rec1.Header().Get("ETag") == rec2.Header().Get("ETag") {
		t.Fatal("different content must produce different ETags")
	}
}

// M5: Same content must produce same ETag (deterministic).
func TestMutation_SameContentSameETag(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("stable"))
	}))
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, httptest.NewRequest("GET", "/", nil))
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest("GET", "/", nil))
	if rec1.Header().Get("ETag") != rec2.Header().Get("ETag") {
		t.Fatal("same content must produce same ETag")
	}
}

// M6: Wrong If-None-Match must return 200, not 304.
func TestMutation_WrongIfNoneMatchReturns200(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("data"))
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("If-None-Match", `"wrong"`)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("wrong ETag should return 200, got %d", rec.Code)
	}
	if rec.Header().Get("ETag") == "" {
		t.Fatal("response should still have ETag")
	}
}

// M7: Upstream headers must be forwarded.
func TestMutation_UpstreamHeadersForwarded(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "val")
		w.Write([]byte("ok"))
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Header().Get("X-Custom") != "val" {
		t.Fatal("upstream headers must be forwarded")
	}
}
