package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testMap() *ClientMap {
	return NewClientMap(map[string]*ClientEntry{
		"agent.example.com":      {ClientID: "agent", Scopes: []string{"read:logs-*"}},
		"ci@example.com":         {ClientID: "ci-bot", Scopes: []string{"read:logs-*", "write:logs-*"}},
		"admin.internal":         {ClientID: "admin", Scopes: []string{"admin"}},
	})
}

func TestIdentify_ByCN(t *testing.T) {
	m := testMap()
	cert := &x509.Certificate{Subject: pkix.Name{CommonName: "agent.example.com"}}
	entry, err := m.Identify(cert)
	if err != nil {
		t.Fatal(err)
	}
	if entry.ClientID != "agent" {
		t.Fatalf("client_id = %s", entry.ClientID)
	}
}

func TestIdentify_ByDNSSAN(t *testing.T) {
	m := testMap()
	cert := &x509.Certificate{
		Subject:  pkix.Name{CommonName: "unknown"},
		DNSNames: []string{"admin.internal"},
	}
	entry, err := m.Identify(cert)
	if err != nil {
		t.Fatal(err)
	}
	if entry.ClientID != "admin" {
		t.Fatalf("client_id = %s", entry.ClientID)
	}
}

func TestIdentify_ByEmailSAN(t *testing.T) {
	m := testMap()
	cert := &x509.Certificate{
		Subject:        pkix.Name{CommonName: "unknown"},
		EmailAddresses: []string{"ci@example.com"},
	}
	entry, err := m.Identify(cert)
	if err != nil {
		t.Fatal(err)
	}
	if entry.ClientID != "ci-bot" {
		t.Fatalf("client_id = %s", entry.ClientID)
	}
}

func TestIdentify_CaseInsensitive(t *testing.T) {
	m := testMap()
	cert := &x509.Certificate{Subject: pkix.Name{CommonName: "Agent.Example.COM"}}
	entry, err := m.Identify(cert)
	if err != nil {
		t.Fatal(err)
	}
	if entry.ClientID != "agent" {
		t.Fatalf("client_id = %s", entry.ClientID)
	}
}

func TestIdentify_NoMatch(t *testing.T) {
	m := testMap()
	cert := &x509.Certificate{Subject: pkix.Name{CommonName: "unknown.example.com"}}
	_, err := m.Identify(cert)
	if err == nil {
		t.Fatal("should fail for unknown cert")
	}
}

func TestMiddleware_InjectsHeaders(t *testing.T) {
	m := testMap()
	var gotUser, gotScopes, gotMethod string
	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = r.Header.Get("X-Proxy-User")
		gotScopes = r.Header.Get("X-Proxy-Scopes")
		gotMethod = r.Header.Get("X-Auth-Method")
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.TLS = &tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{
			{Subject: pkix.Name{CommonName: "agent.example.com"}},
		},
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if gotUser != "agent" {
		t.Fatalf("X-Proxy-User = %s", gotUser)
	}
	if gotScopes != "read:logs-*" {
		t.Fatalf("X-Proxy-Scopes = %s", gotScopes)
	}
	if gotMethod != "mtls" {
		t.Fatalf("X-Auth-Method = %s", gotMethod)
	}
}

func TestMiddleware_SkipsBearer(t *testing.T) {
	m := testMap()
	called := false
	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer tok_123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("should pass through when Bearer present")
	}
}

func TestMiddleware_SkipsNoCert(t *testing.T) {
	m := testMap()
	called := false
	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("should pass through when no TLS")
	}
}

func TestMiddleware_DeniesUnknownCert(t *testing.T) {
	m := testMap()
	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.TLS = &tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{
			{Subject: pkix.Name{CommonName: "evil.example.com"}},
		},
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 403 {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}
