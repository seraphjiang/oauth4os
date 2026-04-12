// Package client provides a Go SDK for oauth4os.
//
// Usage:
//
//	c := client.New("http://localhost:8443", "my-client", "my-secret",
//	    client.WithScopes("read:logs-*"),
//	)
//	// Token is auto-managed — fetched on first call, refreshed when expired.
//	results, err := c.Search("logs-*", map[string]interface{}{
//	    "query": map[string]interface{}{"match": map[string]string{"level": "error"}},
//	})
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client is an oauth4os SDK client with automatic token management.
type Client struct {
	baseURL      string
	clientID     string
	clientSecret string
	scopes       string
	httpClient   *http.Client

	mu       sync.Mutex
	token    string
	expiry   time.Time
}

// Option configures the client.
type Option func(*Client)

// WithScopes sets the requested scopes.
func WithScopes(scopes ...string) Option {
	return func(c *Client) { c.scopes = strings.Join(scopes, " ") }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// New creates an oauth4os client.
func New(baseURL, clientID, clientSecret string, opts ...Option) *Client {
	c := &Client{
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		clientID:     clientID,
		clientSecret: clientSecret,
		scopes:       "admin",
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Token returns a valid access token, fetching or refreshing as needed.
func (c *Client) Token() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.expiry.Add(-30*time.Second)) {
		return c.token, nil
	}
	return c.fetchToken()
}

func (c *Client) fetchToken() (string, error) {
	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"scope":         {c.scopes},
	}
	resp, err := c.httpClient.PostForm(c.baseURL+"/oauth/token", data)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request returned %d: %s", resp.StatusCode, body)
	}
	var result struct {
		AccessToken string  `json:"access_token"`
		ExpiresIn   float64 `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("token decode failed: %w", err)
	}
	c.token = result.AccessToken
	c.expiry = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return c.token, nil
}

// Do executes an authenticated HTTP request against the proxy.
func (c *Client) Do(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	tok, err := c.Token()
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.httpClient.Do(req)
}

// Search queries an OpenSearch index.
func (c *Client) Search(index string, query map[string]interface{}) ([]map[string]interface{}, error) {
	resp, err := c.Do("POST", "/"+index+"/_search", query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search returned %d: %s", resp.StatusCode, body)
	}
	var result struct {
		Hits struct {
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	docs := make([]map[string]interface{}, len(result.Hits.Hits))
	for i, h := range result.Hits.Hits {
		docs[i] = h.Source
	}
	return docs, nil
}

// Index writes a document to an OpenSearch index.
func (c *Client) Index(index, id string, doc map[string]interface{}) error {
	path := "/" + index + "/_doc"
	if id != "" {
		path += "/" + id
	}
	resp, err := c.Do("POST", path, doc)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("index returned %d: %s", resp.StatusCode, body)
	}
	return nil
}

// Health checks the proxy health endpoint.
func (c *Client) Health() (map[string]interface{}, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}

// CreateToken issues a new scoped token.
func (c *Client) CreateToken(scope string) (string, error) {
	data := url.Values{
		"grant_type": {"client_credentials"},
		"client_id":  {c.clientID},
		"scope":      {scope},
	}
	resp, err := c.httpClient.PostForm(c.baseURL+"/oauth/token", data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		AccessToken string `json:"access_token"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.AccessToken, nil
}

// RevokeToken revokes a token by ID.
func (c *Client) RevokeToken(tokenID string) error {
	req, _ := http.NewRequest("DELETE", c.baseURL+"/oauth/token/"+tokenID, nil)
	tok, err := c.Token()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("revoke returned %d", resp.StatusCode)
	}
	return nil
}

// Register dynamically registers a new client (RFC 7591).
func (c *Client) Register(clientName, scope string) (clientID, clientSecret string, err error) {
	body := map[string]interface{}{
		"client_name": clientName,
		"scope":       scope,
		"grant_types": []string{"client_credentials"},
	}
	b, _ := json.Marshal(body)
	resp, err := c.httpClient.Post(c.baseURL+"/oauth/register", "application/json", bytes.NewReader(b))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		raw, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("register returned %d: %s", resp.StatusCode, raw)
	}
	var result struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.ClientID, result.ClientSecret, nil
}
