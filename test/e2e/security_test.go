//go:build e2e

package e2e

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// Security scanning tests — path traversal, SSRF, header injection.

// TestSecurity_PathTraversal verifies proxy blocks directory traversal attempts.
func TestSecurity_PathTraversal(t *testing.T) {
	admin := getToken(t, "admin-agent", "admin-agent-secret", "admin")
	paths := []string{
		"/../etc/passwd",
		"/..%2f..%2fetc/passwd",
		"/%2e%2e/%2e%2e/etc/passwd",
		"/logs-2026/../../_cluster/settings",
		"/.opendistro_security",
		"/_plugins/_security/api/internalusers",
	}
	client := &http.Client{Timeout: 5 * time.Second}
	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			req, err := http.NewRequest("GET", proxyURL+p, nil)
			if err != nil {
				return
			}
			req.Header.Set("Authorization", "Bearer "+admin)
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			// Should not return 200 for traversal paths
			if resp.StatusCode == 200 && strings.Contains(p, "..") {
				t.Errorf("path traversal %s returned 200", p)
			}
		})
	}
}

// TestSecurity_HostHeaderInjection verifies proxy doesn't use client Host header for upstream.
func TestSecurity_HostHeaderInjection(t *testing.T) {
	admin := getToken(t, "admin-agent", "admin-agent-secret", "admin")
	req, _ := http.NewRequest("GET", proxyURL+"/_cat/health", nil)
	req.Header.Set("Authorization", "Bearer "+admin)
	req.Host = "evil.example.com"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Skipf("proxy not reachable: %v", err)
	}
	defer resp.Body.Close()
	// Proxy should still work — it uses configured upstream, not Host header
	if resp.StatusCode >= 500 {
		t.Errorf("Host header injection caused %d", resp.StatusCode)
	}
}

// TestSecurity_OversizedHeaders verifies proxy handles huge headers gracefully.
func TestSecurity_OversizedHeaders(t *testing.T) {
	req, _ := http.NewRequest("GET", proxyURL+"/health", nil)
	req.Header.Set("X-Huge", strings.Repeat("A", 64*1024))
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return // connection reset is acceptable
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		t.Errorf("oversized header caused %d", resp.StatusCode)
	}
}

// TestSecurity_MethodOverride verifies X-HTTP-Method-Override doesn't bypass auth.
func TestSecurity_MethodOverride(t *testing.T) {
	req, _ := http.NewRequest("GET", proxyURL+"/_search", nil)
	req.Header.Set("X-HTTP-Method-Override", "DELETE")
	// No auth token — should be rejected regardless of override
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Skipf("proxy not reachable: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		t.Error("method override without auth should not return 200")
	}
}

// TestSecurity_CRLFInjection verifies proxy handles CRLF in headers.
func TestSecurity_CRLFInjection(t *testing.T) {
	req, _ := http.NewRequest("GET", proxyURL+"/health", nil)
	req.Header.Set("X-Injected", "value\r\nX-Evil: injected")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return // Go's HTTP client may reject this
	}
	defer resp.Body.Close()
	// Check response doesn't contain injected header
	if resp.Header.Get("X-Evil") == "injected" {
		t.Error("CRLF injection succeeded")
	}
}

// TestSecurity_InternalEndpointsRequireAuth verifies management endpoints aren't open.
func TestSecurity_InternalEndpointsRequireAuth(t *testing.T) {
	endpoints := []string{
		"/oauth/tokens",
		"/oauth/introspect",
		"/_cluster/settings",
		"/_cat/indices",
	}
	client := &http.Client{Timeout: 5 * time.Second}
	for _, ep := range endpoints {
		t.Run(ep, func(t *testing.T) {
			req, _ := http.NewRequest("GET", proxyURL+ep, nil)
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				t.Errorf("%s accessible without auth", ep)
			}
		})
	}
}
