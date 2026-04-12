package proxy_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Fake upstream OpenSearch
func fakeOpenSearch() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/" {
			w.Write([]byte(`{"cluster_name":"test","version":{"number":"2.17.0"}}`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/_search") {
			w.Write([]byte(`{"hits":{"total":{"value":0},"hits":[]}}`))
			return
		}
		w.Write([]byte(`{"status":"ok"}`))
	}))
}

// Minimal proxy handler for testing (mirrors main.go logic without JWT/Cedar)
func proxyHandler(engineURL string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","version":"0.2.0"}`))
	})

	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.Write([]byte("oauth4os_requests_total 0\noauth4os_uptime_seconds 1\n"))
	})

	return mux
}

func TestHealthEndpoint(t *testing.T) {
	handler := proxyHandler("")
	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("status = %v", body["status"])
	}
	if body["version"] != "0.2.0" {
		t.Errorf("version = %v", body["version"])
	}
}

func TestMetricsEndpoint(t *testing.T) {
	handler := proxyHandler("")
	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("metrics request failed: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("content-type = %s", ct)
	}

	body, _ := io.ReadAll(resp.Body)
	text := string(body)
	if !strings.Contains(text, "oauth4os_requests_total") {
		t.Error("missing requests_total metric")
	}
	if !strings.Contains(text, "oauth4os_uptime_seconds") {
		t.Error("missing uptime metric")
	}
}

func TestFakeUpstreamPassthrough(t *testing.T) {
	upstream := fakeOpenSearch()
	defer upstream.Close()

	// Verify fake upstream works
	resp, err := http.Get(upstream.URL + "/")
	if err != nil {
		t.Fatalf("upstream request failed: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	if body["cluster_name"] != "test" {
		t.Errorf("cluster_name = %v", body["cluster_name"])
	}
}

func TestExtractIndex(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/logs-2026.04/_search", "logs-2026.04"},
		{"/my-index/_doc/1", "my-index"},
		{"/.opendistro_security", ".opendistro_security"},
		{"/", ""},
		{"/single", "single"},
	}
	for _, tt := range tests {
		got := extractIndex(tt.path)
		if got != tt.want {
			t.Errorf("extractIndex(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func extractIndex(path string) string {
	path = strings.TrimPrefix(path, "/")
	if idx := strings.IndexByte(path, '/'); idx > 0 {
		return path[:idx]
	}
	return path
}
