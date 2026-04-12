package e2e

import (
	"net/http"
	"testing"
)

// TestEndpointSurvey hits every known proxy endpoint and verifies it responds (not 404).
// This catches wiring regressions — if a handler is removed, this test fails.
func TestEndpointSurvey(t *testing.T) {
	proxy := proxyURL(t)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	endpoints := []struct {
		method string
		path   string
		want   []int // acceptable status codes
	}{
		{"GET", "/health", []int{200}},
		{"GET", "/health/deep", []int{200, 503}},
		{"GET", "/version", []int{200}},
		{"GET", "/metrics", []int{200}},
		{"GET", "/.well-known/openid-configuration", []int{200}},
		{"GET", "/.well-known/jwks.json", []int{200}},
		{"GET", "/oauth/register", []int{200}},
		{"GET", "/oauth/tokens", []int{200, 401}},
		{"GET", "/install.sh", []int{200}},
		{"GET", "/scripts/oauth4os-demo", []int{200}},
		{"GET", "/v1/traces", []int{200}},
		{"GET", "/i18n/consent.json", []int{200}},
		{"GET", "/admin/audit", []int{200, 401, 403}},
		{"GET", "/admin/analytics", []int{200, 401, 403}},
		{"GET", "/admin/health", []int{200, 503}},
		{"GET", "/developer/openapi.yaml", []int{200}},
		{"GET", "/developer/docs", []int{200}},
		{"GET", "/developer/analytics", []int{200}},
		{"GET", "/demo", []int{200}},
		{"GET", "/demo/callback", []int{200}},
		{"GET", "/admin/config", []int{200}},
		{"POST", "/oauth/device/code", []int{200, 400}},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req, _ := http.NewRequest(ep.method, proxy+ep.path, nil)
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			resp.Body.Close()
			for _, want := range ep.want {
				if resp.StatusCode == want {
					return
				}
			}
			t.Errorf("got %d, want one of %v", resp.StatusCode, ep.want)
		})
	}
}
