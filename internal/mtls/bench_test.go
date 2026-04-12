package mtls

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"
)

func BenchmarkIdentify(b *testing.B) {
	m := NewClientMap(map[string]*ClientEntry{
		"service-a.example.com": {ClientID: "service-a", Scopes: []string{"read"}},
		"service-b.example.com": {ClientID: "service-b", Scopes: []string{"write"}},
	})
	cert := &x509.Certificate{Subject: pkix.Name{CommonName: "service-a.example.com"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Identify(cert)
	}
}
