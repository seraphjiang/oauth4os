package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func mockServer() *httptest.Server {
	var tokenCalls atomic.Int64
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/oauth/token" && r.Method == "POST":
			tokenCalls.Add(1)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "tok_test_123",
				"expires_in":   3600,
				"scope":        r.FormValue("scope"),
			})
		case r.URL.Path == "/health":
			json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "version": "test"})
		case r.URL.Path == "/logs-test/_search":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"hits": map[string]interface{}{
					"hits": []map[string]interface{}{
						{"_source": map[string]string{"level": "error", "msg": "test"}},
					},
				},
			})
		case r.URL.Path == "/oauth/register" && r.Method == "POST":
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(map[string]string{
				"client_id": "client_new", "client_secret": "secret_new",
			})
		default:
			w.WriteHeader(200)
		}
	}))
}

func TestTokenAutoFetch(t *testing.T) {
	srv := mockServer()
	defer srv.Close()

	c := New(srv.URL, "test", "secret", WithScopes("read:logs-*"))
	tok, err := c.Token()
	if err != nil {
		t.Fatalf("Token() failed: %v", err)
	}
	if tok != "tok_test_123" {
		t.Fatalf("expected tok_test_123, got %s", tok)
	}
	// Second call should use cache
	tok2, _ := c.Token()
	if tok2 != tok {
		t.Fatal("expected cached token")
	}
}

func TestHealth(t *testing.T) {
	srv := mockServer()
	defer srv.Close()

	c := New(srv.URL, "test", "secret")
	h, err := c.Health()
	if err != nil {
		t.Fatalf("Health() failed: %v", err)
	}
	if h["status"] != "ok" {
		t.Fatalf("expected ok, got %v", h["status"])
	}
}

func TestSearch(t *testing.T) {
	srv := mockServer()
	defer srv.Close()

	c := New(srv.URL, "test", "secret")
	docs, err := c.Search("logs-test", map[string]interface{}{
		"query": map[string]interface{}{"match_all": map[string]interface{}{}},
	})
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}
	if len(docs) != 1 || docs[0]["level"] != "error" {
		t.Fatalf("unexpected results: %v", docs)
	}
}

func TestRegister(t *testing.T) {
	srv := mockServer()
	defer srv.Close()

	c := New(srv.URL, "test", "secret")
	id, secret, err := c.Register("my-agent", "read:*")
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}
	if id != "client_new" || secret != "secret_new" {
		t.Fatalf("unexpected: id=%s secret=%s", id, secret)
	}
}
