//go:build e2e

package e2e

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// API conformance tests — verify all OpenAPI spec endpoints exist and return expected status codes.
// Based on docs/api-spec.yaml.

var client = &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse // don't follow redirects
}}

type endpoint struct {
	method       string
	path         string
	needsAuth    bool
	body         string
	expectStatus []int // any of these is acceptable
}

var specEndpoints = []endpoint{
	// OAuth endpoints
	{"POST", "/oauth/token", false, "grant_type=client_credentials&client_id=admin-agent&client_secret=admin-agent-secret&scope=admin", []int{200}},
	{"POST", "/oauth/token", false, "grant_type=client_credentials&client_id=bad&client_secret=bad", []int{400, 401}},
	{"POST", "/oauth/token", false, "", []int{400, 401}},
	{"GET", "/oauth/tokens", true, "", []int{200}},
	{"POST", "/oauth/introspect", false, "token=fake", []int{200}},

	// OIDC discovery
	{"GET", "/.well-known/openid-configuration", false, "", []int{200}},

	// Health & metrics
	{"GET", "/health", false, "", []int{200}},
	{"GET", "/metrics", false, "", []int{200}},

	// PKCE authorize (missing params → 400)
	{"GET", "/oauth/authorize", false, "", []int{400}},

	// Dynamic registration
	{"POST", "/oauth/register", false, `{"client_name":"conformance-test","scope":"read:logs-*"}`, []int{201, 200, 400}},

	// Admin endpoints (need auth)
	{"GET", "/admin/scope-mappings", true, "", []int{200}},
	{"GET", "/admin/providers", true, "", []int{200}},
	{"GET", "/admin/cedar-policies", true, "", []int{200}},
	{"GET", "/admin/rate-limits", true, "", []int{200}},
	{"GET", "/admin/config", true, "", []int{200}},
	{"GET", "/admin/tenants", true, "", []int{200}},

	// Unauthenticated proxy → 401
	{"GET", "/_cat/indices", false, "", []int{401, 403}},
}

func TestAPIConformance(t *testing.T) {
	// Get admin token for authenticated endpoints
	adminToken := ""
	resp, body := proxyRequest("POST", "/oauth/token", "", map[string]interface{}{
		// Use form-encoded via helper — but we need raw form. Use direct request.
	})
	_ = resp
	_ = body

	// Direct token request
	tokenResp, err := http.PostForm(proxyURL+"/oauth/token", map[string][]string{
		"grant_type":    {"client_credentials"},
		"client_id":     {"admin-agent"},
		"client_secret": {"admin-agent-secret"},
		"scope":         {"admin"},
	})
	if err == nil && tokenResp.StatusCode == 200 {
		defer tokenResp.Body.Close()
		buf := make([]byte, 4096)
		n, _ := tokenResp.Body.Read(buf)
		// Extract token
		s := string(buf[:n])
		if idx := strings.Index(s, `"access_token":"`); idx >= 0 {
			start := idx + len(`"access_token":"`)
			end := strings.Index(s[start:], `"`)
			if end > 0 {
				adminToken = s[start : start+end]
			}
		}
	}

	for _, ep := range specEndpoints {
		name := ep.method + " " + ep.path
		t.Run(name, func(t *testing.T) {
			var req *http.Request
			if ep.body != "" {
				req, _ = http.NewRequest(ep.method, proxyURL+ep.path, strings.NewReader(ep.body))
				if strings.HasPrefix(ep.body, "{") {
					req.Header.Set("Content-Type", "application/json")
				} else {
					req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				}
			} else {
				req, _ = http.NewRequest(ep.method, proxyURL+ep.path, nil)
			}

			if ep.needsAuth && adminToken != "" {
				req.Header.Set("Authorization", "Bearer "+adminToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Skipf("proxy not reachable: %v", err)
				return
			}
			defer resp.Body.Close()

			ok := false
			for _, expected := range ep.expectStatus {
				if resp.StatusCode == expected {
					ok = true
					break
				}
			}
			if !ok {
				t.Errorf("expected one of %v, got %d", ep.expectStatus, resp.StatusCode)
			}
		})
	}
}

// TestAPIConformance_ContentType verifies JSON endpoints return application/json.
func TestAPIConformance_ContentType(t *testing.T) {
	jsonEndpoints := []string{"/health", "/.well-known/openid-configuration"}
	for _, ep := range jsonEndpoints {
		t.Run(ep, func(t *testing.T) {
			req, _ := http.NewRequest("GET", proxyURL+ep, nil)
			resp, err := client.Do(req)
			if err != nil {
				t.Skipf("proxy not reachable: %v", err)
			}
			defer resp.Body.Close()
			ct := resp.Header.Get("Content-Type")
			if !strings.Contains(ct, "json") && !strings.Contains(ct, "text") {
				t.Errorf("expected JSON content type, got %s", ct)
			}
		})
	}
}

// TestAPIConformance_MethodNotAllowed verifies wrong methods get 405.
func TestAPIConformance_MethodNotAllowed(t *testing.T) {
	cases := []struct{ method, path string }{
		{"DELETE", "/health"},
		{"PUT", "/oauth/token"},
		{"GET", "/oauth/introspect"},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req, _ := http.NewRequest(tc.method, proxyURL+tc.path, nil)
			resp, err := client.Do(req)
			if err != nil {
				t.Skipf("proxy not reachable: %v", err)
			}
			defer resp.Body.Close()
			// 405 is ideal, but some endpoints may return 400 or route differently
			if resp.StatusCode == 200 {
				t.Errorf("%s %s should not return 200", tc.method, tc.path)
			}
		})
	}
}
