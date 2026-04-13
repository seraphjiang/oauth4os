package mtls

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"
)

func TestEdge_NewClientMapEmpty(t *testing.T) {
	m := NewClientMap(nil)
	if m == nil {
		t.Error("NewClientMap(nil) should return non-nil")
	}
}

func TestEdge_IdentifyNilCertReturnsError(t *testing.T) {
	m := NewClientMap(nil)
	_, err := m.Identify(nil)
	if err == nil {
		t.Error("nil cert should return error")
	}
}

func TestEdge_IdentifyUnknownCertFails(t *testing.T) {
	m := NewClientMap(map[string]*ClientEntry{
		"known-cn": {ClientID: "app", Scopes: []string{"read"}},
	})
	cert := &x509.Certificate{Subject: pkix.Name{CommonName: "unknown-cn"}}
	_, err := m.Identify(cert)
	if err == nil {
		t.Error("unknown CN should fail")
	}
}

func TestEdge_IdentifyKnownCertPasses(t *testing.T) {
	m := NewClientMap(map[string]*ClientEntry{
		"my-app": {ClientID: "app-1", Scopes: []string{"read"}},
	})
	cert := &x509.Certificate{Subject: pkix.Name{CommonName: "my-app"}}
	entry, err := m.Identify(cert)
	if err != nil {
		t.Fatalf("known CN should pass: %v", err)
	}
	if entry.ClientID != "app-1" {
		t.Errorf("expected app-1, got %q", entry.ClientID)
	}
}
