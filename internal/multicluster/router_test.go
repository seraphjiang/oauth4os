package multicluster

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testRouter(t *testing.T) *Router {
	t.Helper()
	// Start fake backends
	backendA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"cluster": "a", "path": r.URL.Path})
	}))
	backendB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"cluster": "b", "path": r.URL.Path})
	}))
	t.Cleanup(func() { backendA.Close(); backendB.Close() })

	r, err := NewRouter(Config{Clusters: []Cluster{
		{Name: "prod", Engine: backendA.URL, Prefix: "/prod", Default: true},
		{Name: "staging", Engine: backendB.URL, Prefix: "/staging"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRoute_ByPrefix(t *testing.T) {
	r := testRouter(t)
	req := httptest.NewRequest("GET", "/staging/logs-*/_search", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var resp map[string]string
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["cluster"] != "b" {
		t.Fatalf("expected cluster b, got %s", resp["cluster"])
	}
	if resp["path"] != "/logs-*/_search" {
		t.Fatalf("prefix should be stripped, got path=%s", resp["path"])
	}
}

func TestRoute_ByHeader(t *testing.T) {
	r := testRouter(t)
	req := httptest.NewRequest("GET", "/logs-*/_search", nil)
	req.Header.Set("X-Cluster", "staging")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var resp map[string]string
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["cluster"] != "b" {
		t.Fatalf("expected cluster b via header, got %s", resp["cluster"])
	}
}

func TestRoute_DefaultFallback(t *testing.T) {
	r := testRouter(t)
	req := httptest.NewRequest("GET", "/logs-*/_search", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var resp map[string]string
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["cluster"] != "a" {
		t.Fatalf("expected default cluster a, got %s", resp["cluster"])
	}
}

func TestRoute_NoMatch(t *testing.T) {
	r, _ := NewRouter(Config{Clusters: []Cluster{
		{Name: "prod", Engine: "http://localhost:1", Prefix: "/prod"},
	}})
	req := httptest.NewRequest("GET", "/other/path", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != 502 {
		t.Fatalf("expected 502, got %d", rec.Code)
	}
}

func TestAdd_Remove(t *testing.T) {
	r, _ := NewRouter(Config{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer backend.Close()

	r.Add(Cluster{Name: "new", Engine: backend.URL, Prefix: "/new"})
	if len(r.List()) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(r.List()))
	}

	if !r.Remove("new") {
		t.Fatal("remove should return true")
	}
	if r.Remove("new") {
		t.Fatal("second remove should return false")
	}
	if len(r.List()) != 0 {
		t.Fatal("should be empty")
	}
}

func TestRoute_HeaderOverridesPrefix(t *testing.T) {
	r := testRouter(t)
	// Path says /prod but header says staging
	req := httptest.NewRequest("GET", "/prod/test", nil)
	req.Header.Set("X-Cluster", "staging")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var resp map[string]string
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["cluster"] != "b" {
		t.Fatalf("header should override prefix, got %s", resp["cluster"])
	}
}
