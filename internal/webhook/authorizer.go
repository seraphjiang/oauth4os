// Package webhook implements external webhook-based authorization.
// After JWT validation, the proxy can call an external webhook to make
// a final allow/deny decision — enabling custom auth logic (LDAP, internal
// policy engines, compliance checks) without modifying oauth4os.
package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Request is sent to the external webhook.
type Request struct {
	ClientID string            `json:"client_id"`
	Subject  string            `json:"sub"`
	Scopes   []string          `json:"scopes"`
	Action   string            `json:"action"`  // HTTP method
	Resource string            `json:"resource"` // request path
	IP       string            `json:"ip"`
	Headers  map[string]string `json:"headers,omitempty"`
}

// Response is expected from the webhook.
type Response struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// Authorizer calls an external webhook for auth decisions.
type Authorizer struct {
	url     string
	client  *http.Client
	headers map[string]string // static headers (e.g., API key)
}

// Config holds webhook configuration.
type Config struct {
	URL     string            `yaml:"url"`
	Timeout int               `yaml:"timeout_ms"` // default 2000
	Headers map[string]string `yaml:"headers"`
	// FailOpen: if true, allow request when webhook is unreachable. Default false (fail closed).
	FailOpen bool `yaml:"fail_open"`
}

// NewAuthorizer creates a webhook authorizer.
func NewAuthorizer(cfg Config) *Authorizer {
	timeout := time.Duration(cfg.Timeout) * time.Millisecond
	if timeout == 0 {
		timeout = 2 * time.Second
	}
	return &Authorizer{
		url:     cfg.URL,
		client:  &http.Client{Timeout: timeout},
		headers: cfg.Headers,
	}
}

// Check calls the webhook and returns nil if allowed, error if denied.
func (a *Authorizer) Check(req Request) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("webhook marshal error")
	}

	httpReq, err := http.NewRequest(http.MethodPost, a.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook request error")
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range a.headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("webhook unreachable")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("webhook response malformed")
	}

	if !result.Allowed {
		if result.Reason != "" {
			return fmt.Errorf("denied by webhook")
		}
		return fmt.Errorf("denied by webhook")
	}
	return nil
}
