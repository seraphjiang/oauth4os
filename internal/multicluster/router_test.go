package multicluster

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouteByPrefix(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Got-Path", r.URL.Path)
		w.WriteHeader(200)
	}))
	defer backend.Close()

	r, err := NewRouter(Config{Clusters: []Cluster{
		{Name: "logs", Engine: backend.URL, Prefix: "/logs"},
		{Name: "metrics", Engine: backend.URL, Prefix: "/metrics"},
		{Name: "default", Engine: backend.URL, Default: true},
	}})
	if err != nil {
		t.Fatal(err)
	}

	// Prefix match
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/logs/_search", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Default fallback
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/unknown/_search", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200 from default, got %d", w.Code)
	}
}

func TestRouteByHeader(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Routed", r.Header.Get("X-Routed-Cluster"))
		w.WriteHeader(200)
	}))
	defer backend.Close()

	r, _ := NewRouter(Config{Clusters: []Cluster{
		{Name: "prod", Engine: backend.URL},
		{Name: "staging", Engine: backend.URL},
	}})

	req := httptest.NewRequest("GET", "/_search", nil)
	req.Header.Set("X-Cluster", "staging")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Header().Get("X-Routed") != "staging" {
		t.Fatalf("expected staging, got %s", w.Header().Get("X-Routed"))
	}
}

func TestNoMatchReturns502(t *testing.T) {
	r, _ := NewRouter(Config{Clusters: []Cluster{
		{Name: "only", Engine: "http://localhost:1", Prefix: "/only"},
	}})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/other/_search", nil))
	if w.Code != 502 {
		t.Fatalf("expected 502, got %d", w.Code)
	}
}

func TestAddRemove(t *testing.T) {
	r, _ := NewRouter(Config{})
	if len(r.List()) != 0 {
		t.Fatal("expected empty")
	}
	r.Add(Cluster{Name: "a", Engine: "http://localhost:1", Prefix: "/a"})
	if len(r.List()) != 1 {
		t.Fatal("expected 1")
	}
	r.Remove("a")
	if len(r.List()) != 0 {
		t.Fatal("expected 0 after remove")
	}
}
