package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"net/http"
	"net/http/httptest"
	"testing"
)

func certWithCN(cn string) *x509.Certificate {
	return &x509.Certificate{Subject: pkix.Name{CommonName: cn}}
}

// Mutation: remove CN lookup → must identify by CN
func TestMutation_IdentifyCN(t *testing.T) {
	m := NewClientMap(map[string]*ClientEntry{"service-a": {ClientID: "svc-a", Scopes: []string{"read"}}})
	entry, err := m.Identify(certWithCN("service-a"))
	if err != nil || entry.ClientID != "svc-a" {
		t.Errorf("should identify by CN: %v %v", entry, err)
	}
}

// Mutation: remove case normalization → lookup must be case-insensitive
func TestMutation_CaseInsensitive(t *testing.T) {
	m := NewClientMap(map[string]*ClientEntry{"Service-A": {ClientID: "svc-a"}})
	_, err := m.Identify(certWithCN("service-a"))
	if err != nil {
		t.Error("CN lookup must be case-insensitive")
	}
}

// Mutation: remove unknown cert error → unknown cert must return error
func TestMutation_UnknownCert(t *testing.T) {
	m := NewClientMap(map[string]*ClientEntry{"known": {ClientID: "k"}})
	_, err := m.Identify(certWithCN("unknown"))
	if err == nil {
		t.Error("unknown cert must return error")
	}
}

// Mutation: remove header injection → middleware must set X-Proxy-User
func TestMutation_HeaderInjection(t *testing.T) {
	m := NewClientMap(map[string]*ClientEntry{"svc": {ClientID: "my-svc", Scopes: []string{"admin"}}})
	var gotUser, gotScopes string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = r.Header.Get("X-Proxy-User")
		gotScopes = r.Header.Get("X-Proxy-Scopes")
		w.WriteHeader(200)
	})
	handler := m.Middleware(inner)
	r := httptest.NewRequest("GET", "/", nil)
	r.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{certWithCN("svc")}}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if gotUser != "my-svc" {
		t.Errorf("expected X-Proxy-User=my-svc, got %s", gotUser)
	}
	if gotScopes != "admin" {
		t.Errorf("expected X-Proxy-Scopes=admin, got %s", gotScopes)
	}
}

// Mutation: remove Bearer skip → Bearer auth must bypass mTLS
func TestMutation_BearerBypass(t *testing.T) {
	m := NewClientMap(map[string]*ClientEntry{})
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(200) })
	handler := m.Middleware(inner)
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer tok123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if !called {
		t.Error("Bearer auth should bypass mTLS and call next handler")
	}
}

// Mutation: remove X-Proxy-User header → middleware must inject identity
func TestMutation_MiddlewareSetsProxyUser(t *testing.T) {
	m := NewClientMap(map[string]*ClientEntry{
		"service-a.example.com": {ClientID: "service-a", Scopes: []string{"read"}},
	})
	var gotUser string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = r.Header.Get("X-Proxy-User")
		w.WriteHeader(200)
	})
	handler := m.Middleware(inner)
	r := httptest.NewRequest("GET", "/", nil)
	r.TLS = &tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{
			{Subject: pkix.Name{CommonName: "service-a.example.com"}},
		},
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if gotUser != "service-a" {
		t.Errorf("expected X-Proxy-User 'service-a', got %q", gotUser)
	}
}
