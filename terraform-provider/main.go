// terraform-provider-oauth4os — manage clients, scopes, policies, providers.
//
// Uses the oauth4os Admin API + client registration endpoint.
// Build: go build -o terraform-provider-oauth4os

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Client wraps the oauth4os Admin API.
type Client struct {
	URL        string
	AdminToken string
	HTTP       *http.Client
}

func NewClient(proxyURL, adminToken string) *Client {
	return &Client{URL: strings.TrimRight(proxyURL, "/"), AdminToken: adminToken, HTTP: &http.Client{}}
}

func (c *Client) do(method, path string, body interface{}) (map[string]interface{}, int, error) {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.URL+path, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.AdminToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AdminToken)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result, resp.StatusCode, nil
}

func (c *Client) doForm(path string, values url.Values) (map[string]interface{}, int, error) {
	resp, err := c.HTTP.PostForm(c.URL+path, values)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result, resp.StatusCode, nil
}

// --- Client Registration ---

func (c *Client) RegisterClient(name, scope string) (map[string]interface{}, error) {
	result, status, err := c.do("POST", "/oauth/register", map[string]interface{}{
		"client_name": name, "scope": scope,
	})
	if err != nil {
		return nil, err
	}
	if status != 201 {
		return nil, fmt.Errorf("register failed: %d %v", status, result)
	}
	return result, nil
}

func (c *Client) GetClient(clientID string) (map[string]interface{}, error) {
	result, status, err := c.do("GET", "/oauth/register/"+clientID, nil)
	if err != nil {
		return nil, err
	}
	if status == 404 {
		return nil, nil
	}
	return result, nil
}

// --- Scope Mappings ---

func (c *Client) GetScopeMappings() (map[string]interface{}, error) {
	result, _, err := c.do("GET", "/admin/scope-mappings", nil)
	return result, err
}

func (c *Client) UpdateScopeMappings(mappings map[string]interface{}) error {
	_, status, err := c.do("PUT", "/admin/scope-mappings", mappings)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("update scope mappings failed: %d", status)
	}
	return nil
}

// --- Cedar Policies ---

func (c *Client) AddCedarPolicy(id, policy string) error {
	_, status, err := c.do("POST", "/admin/cedar-policies", map[string]interface{}{
		"id": id, "policy": policy,
	})
	if err != nil {
		return err
	}
	if status != 201 && status != 200 {
		return fmt.Errorf("add cedar policy failed: %d", status)
	}
	return nil
}

func (c *Client) GetCedarPolicies() (map[string]interface{}, error) {
	result, _, err := c.do("GET", "/admin/cedar-policies", nil)
	return result, err
}

func (c *Client) DeleteCedarPolicy(id string) error {
	_, status, err := c.do("DELETE", "/admin/cedar-policies/"+id, nil)
	if err != nil {
		return err
	}
	if status != 204 && status != 200 {
		return fmt.Errorf("delete cedar policy failed: %d", status)
	}
	return nil
}

// --- Providers ---

func (c *Client) AddProvider(name, issuer, jwksURI string) error {
	_, status, err := c.do("POST", "/admin/providers", map[string]interface{}{
		"name": name, "issuer": issuer, "jwks_uri": jwksURI,
	})
	if err != nil {
		return err
	}
	if status != 201 && status != 200 {
		return fmt.Errorf("add provider failed: %d", status)
	}
	return nil
}

func (c *Client) GetProviders() (map[string]interface{}, error) {
	result, _, err := c.do("GET", "/admin/providers", nil)
	return result, err
}

func (c *Client) DeleteProvider(name string) error {
	_, status, err := c.do("DELETE", "/admin/providers/"+name, nil)
	if err != nil {
		return err
	}
	if status != 204 && status != 200 {
		return fmt.Errorf("delete provider failed: %d", status)
	}
	return nil
}

// --- Config ---

func (c *Client) GetConfig() (map[string]interface{}, error) {
	result, _, err := c.do("GET", "/admin/config", nil)
	return result, err
}

// --- CLI for Terraform external provider pattern ---
// Usage: terraform-provider-oauth4os <action> <resource> [args...]

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <action> <resource> [args...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Actions: create, read, delete\n")
		fmt.Fprintf(os.Stderr, "  Resources: client, scope-mapping, cedar-policy, provider\n")
		os.Exit(1)
	}

	proxyURL := os.Getenv("OAUTH4OS_URL")
	if proxyURL == "" {
		proxyURL = "http://localhost:8443"
	}
	client := NewClient(proxyURL, os.Getenv("OAUTH4OS_ADMIN_TOKEN"))

	action := os.Args[1]
	resource := os.Args[2]

	var result interface{}
	var err error

	switch resource {
	case "client":
		switch action {
		case "create":
			if len(os.Args) < 5 {
				fatal("Usage: create client <name> <scope>")
			}
			result, err = client.RegisterClient(os.Args[3], os.Args[4])
		case "read":
			if len(os.Args) < 4 {
				fatal("Usage: read client <client_id>")
			}
			result, err = client.GetClient(os.Args[3])
		}

	case "cedar-policy":
		switch action {
		case "create":
			if len(os.Args) < 5 {
				fatal("Usage: create cedar-policy <id> <policy>")
			}
			err = client.AddCedarPolicy(os.Args[3], os.Args[4])
			result = map[string]string{"status": "created"}
		case "read":
			result, err = client.GetCedarPolicies()
		case "delete":
			if len(os.Args) < 4 {
				fatal("Usage: delete cedar-policy <id>")
			}
			err = client.DeleteCedarPolicy(os.Args[3])
			result = map[string]string{"status": "deleted"}
		}

	case "provider":
		switch action {
		case "create":
			if len(os.Args) < 6 {
				fatal("Usage: create provider <name> <issuer> <jwks_uri>")
			}
			err = client.AddProvider(os.Args[3], os.Args[4], os.Args[5])
			result = map[string]string{"status": "created"}
		case "read":
			result, err = client.GetProviders()
		case "delete":
			if len(os.Args) < 4 {
				fatal("Usage: delete provider <name>")
			}
			err = client.DeleteProvider(os.Args[3])
			result = map[string]string{"status": "deleted"}
		}

	case "config":
		result, err = client.GetConfig()

	default:
		fatal("Unknown resource: " + resource)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(result)
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
