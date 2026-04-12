package mtls

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"
)

func BenchmarkIdentify(b *testing.B) {
	m := NewClientMap(map[string]*ClientEntry{
		"service-a": {CN: "service-a.example.com", Scopes: []string{"read"}},
		"service-b": {CN: "service-b.example.com", Scopes: []string{"write"}},
	})
	cert := &x509.Certificate{Subject: pkix.Name{CommonName: "service-a.example.com"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Identify(cert)
	}
}
