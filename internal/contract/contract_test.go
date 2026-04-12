package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func fakeProxy() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"version": "1.0.0"})
	})
	mux.HandleFunc("/missing-key", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"other": "val"})
	})
	return httptest.NewServer(mux)
}

func TestMutation_PassingCheck(t *testing.T) {
	srv := fakeProxy()
	defer srv.Close()
	r := New(srv.URL)
	results := r.Run([]Check{
		{Name: "health", Method: "GET", Path: "/health", WantCode: 200, WantType: "json", WantKeys: []string{"status"}},
	})
	if !results[0].Pass {
		t.Errorf("health check should pass: %s", results[0].Error)
	}
}

func TestMutation_WrongStatus(t *testing.T) {
	srv := fakeProxy()
	defer srv.Close()
	r := New(srv.URL)
	results := r.Run([]Check{
		{Name: "404", Method: "GET", Path: "/nonexistent", WantCode: 200},
	})
	if results[0].Pass {
		t.Error("wrong status should fail")
	}
}

func TestMutation_MissingKey(t *testing.T) {
	srv := fakeProxy()
	defer srv.Close()
	r := New(srv.URL)
	results := r.Run([]Check{
		{Name: "missing", Method: "GET", Path: "/missing-key", WantCode: 200, WantType: "json", WantKeys: []string{"expected_key"}},
	})
	if results[0].Pass {
		t.Error("missing key should fail")
	}
}

func TestMutation_DefaultChecksExist(t *testing.T) {
	checks := DefaultChecks()
	if len(checks) < 3 {
		t.Errorf("expected at least 3 default checks, got %d", len(checks))
	}
}
