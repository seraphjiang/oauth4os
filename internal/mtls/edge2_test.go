package mtls

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"
)

func TestEdge_IdentifyByDNSSAN(t *testing.T) {
	m := NewClientMap(map[string]*ClientEntry{
		"app.example.com": {ClientID: "app", Scopes: []string{"read"}},
	})
	cert := &x509.Certificate{
		Subject:  pkix.Name{CommonName: "other"},
		DNSNames: []string{"app.example.com"},
	}
	entry, err := m.Identify(cert)
	if err != nil {
		t.Fatalf("DNS SAN should match: %v", err)
	}
	if entry.ClientID != "app" {
		t.Errorf("expected app, got %q", entry.ClientID)
	}
}

func TestEdge_IdentifyCaseInsensitive(t *testing.T) {
	m := NewClientMap(map[string]*ClientEntry{
		"my-app": {ClientID: "app", Scopes: []string{"read"}},
	})
	cert := &x509.Certificate{Subject: pkix.Name{CommonName: "MY-APP"}}
	entry, err := m.Identify(cert)
	if err != nil {
		t.Fatalf("case insensitive match should work: %v", err)
	}
	if entry.ClientID != "app" {
		t.Errorf("expected app, got %q", entry.ClientID)
	}
}
