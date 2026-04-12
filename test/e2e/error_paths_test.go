package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestErrorPaths verifies error responses don't leak internal details.
// Checks: no stack traces, no internal IPs, no file paths, no Go error strings.

var leakPatterns = []string{
	"dial tcp",       // upstream connection errors
	"connection refused",
	"no such host",
	".go:",           // Go file paths
	"goroutine",      // stack traces
	"runtime.",       // Go runtime
	"panic",          // panics
	"opensearch:9200", // internal hostnames
	"keycloak:8080",
	"127.0.0.1",
	"localhost:9200",
}

func checkNoLeak(t *testing.T, label string, body []byte) {
	t.Helper()
	s := strings.ToLower(string(body))
	for _, p := range leakPatterns {
		if strings.Contains(s, strings.ToLower(p)) {
			t.Errorf("%s: response leaks internal detail %q in: %s", label, p, string(body)[:min(200, len(body))])
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestErrorPath_InvalidToken(t *testing.T) {
	resp, body := proxyRequest("GET", "/_search", "not-a-jwt", nil)
	if resp == nil {
		t.Skip("proxy not running")
	}
	if resp.StatusCode == 200 {
		t.Fatal("invalid token should not return 200")
	}
	checkNoLeak(t, "invalid_token", body)

	// Verify response is valid JSON with "error" field
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err == nil {
		if _, ok := result["error"]; !ok {
			t.Error("error response should have 'error' field")
		}
	}
}

func TestErrorPath_ExpiredToken(t *testing.T) {
	// Use a structurally valid but expired JWT
	expired := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjEwMDAwMDAwMDAsImlzcyI6ImZha2UifQ.fake"
	resp, body := proxyRequest("GET", "/_search", expired, nil)
	if resp == nil {
		t.Skip("proxy not running")
	}
	checkNoLeak(t, "expired_token", body)
}

func TestErrorPath_NoAuth(t *testing.T) {
	req, _ := http.NewRequest("GET", proxyURL+"/_search", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Skip("proxy not running")
	}
	defer resp.Body.Close()
	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	checkNoLeak(t, "no_auth", buf[:n])
}

func TestErrorPath_MalformedJSON(t *testing.T) {
	token := getToken(t, "admin-agent", "admin-agent-secret", "admin")
	resp, body := proxyRequest("POST", "/test-index/_doc", token, nil)
	if resp == nil {
		t.Skip("proxy not running")
	}
	// Even if upstream returns an error, proxy shouldn't add internal details
	checkNoLeak(t, "malformed_json", body)
}
