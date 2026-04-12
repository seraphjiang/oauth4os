// Package contract verifies API responses match expected contracts.
package contract

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Check represents a single contract check.
type Check struct {
	Name     string
	Method   string
	Path     string
	WantCode int
	WantKeys []string // JSON keys that must be present
	WantType string   // expected Content-Type substring
}

// Result captures a contract check outcome.
type Result struct {
	Check  Check
	Status int
	Pass   bool
	Error  string
}

// Runner executes contract checks against a base URL.
type Runner struct {
	BaseURL string
	Client  *http.Client
}

// New creates a contract test runner.
func New(baseURL string) *Runner {
	return &Runner{BaseURL: baseURL, Client: &http.Client{Timeout: 10 * time.Second}}
}

// Run executes all checks and returns results.
func (r *Runner) Run(checks []Check) []Result {
	results := make([]Result, len(checks))
	for i, c := range checks {
		results[i] = r.run(c)
	}
	return results
}

func (r *Runner) run(c Check) Result {
	req, err := http.NewRequest(c.Method, r.BaseURL+c.Path, nil)
	if err != nil {
		return Result{Check: c, Error: err.Error()}
	}
	resp, err := r.Client.Do(req)
	if err != nil {
		return Result{Check: c, Error: err.Error()}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	res := Result{Check: c, Status: resp.StatusCode, Pass: true}

	if c.WantCode != 0 && resp.StatusCode != c.WantCode {
		res.Pass = false
		res.Error = fmt.Sprintf("status: want %d, got %d", c.WantCode, resp.StatusCode)
		return res
	}

	if c.WantType != "" {
		ct := resp.Header.Get("Content-Type")
		if ct == "" || !contains(ct, c.WantType) {
			res.Pass = false
			res.Error = fmt.Sprintf("content-type: want %q in %q", c.WantType, ct)
			return res
		}
	}

	if len(c.WantKeys) > 0 {
		var m map[string]interface{}
		if err := json.Unmarshal(body, &m); err != nil {
			res.Pass = false
			res.Error = "response is not valid JSON"
			return res
		}
		for _, k := range c.WantKeys {
			if _, ok := m[k]; !ok {
				res.Pass = false
				res.Error = fmt.Sprintf("missing key %q", k)
				return res
			}
		}
	}

	return res
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// DefaultChecks returns the standard OAuth proxy contract checks.
func DefaultChecks() []Check {
	return []Check{
		{Name: "health", Method: "GET", Path: "/health", WantCode: 200, WantType: "json", WantKeys: []string{"status"}},
		{Name: "version", Method: "GET", Path: "/version", WantCode: 200, WantType: "json", WantKeys: []string{"version"}},
		{Name: "oidc-config", Method: "GET", Path: "/.well-known/openid-configuration", WantCode: 200, WantType: "json", WantKeys: []string{"issuer", "jwks_uri"}},
		{Name: "jwks", Method: "GET", Path: "/.well-known/jwks.json", WantCode: 200, WantType: "json", WantKeys: []string{"keys"}},
		{Name: "metrics", Method: "GET", Path: "/metrics", WantCode: 200},
	}
}
