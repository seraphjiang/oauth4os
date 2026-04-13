// Package mtls provides mutual TLS client authentication.
// Extracts client identity from verified TLS certificates and maps
// CN/SAN to OAuth scopes, as an alternative to Bearer tokens.
package mtls

import (
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"
)

// ClientMap maps certificate CN or SAN to client identity + scopes.
type ClientMap struct {
	entries map[string]*ClientEntry // CN or SAN email/DNS → entry
}

// ClientEntry is the identity and scopes for a certificate subject.
type ClientEntry struct {
	ClientID string   `yaml:"client_id" json:"client_id"`
	Scopes   []string `yaml:"scopes" json:"scopes"`
}

// Config is the YAML-friendly mTLS configuration.
type Config struct {
	Enabled  bool                    `yaml:"enabled"`
	CAFile   string                  `yaml:"ca_file"`   // path to CA cert for client verification
	Clients  map[string]*ClientEntry `yaml:"clients"`   // CN/SAN → identity
}

// NewClientMap builds a lookup from config.
func NewClientMap(clients map[string]*ClientEntry) *ClientMap {
	m := &ClientMap{entries: make(map[string]*ClientEntry)}
	for k, v := range clients {
		m.entries[strings.ToLower(k)] = v
	}
	return m
}

// Identify extracts client identity from a verified TLS peer certificate.
// Checks CN first, then DNS SANs, then email SANs.
func (m *ClientMap) Identify(cert *x509.Certificate) (*ClientEntry, error) {
	if cert == nil {
		return nil, fmt.Errorf("no client certificate provided")
	}
	// Try CN
	if entry, ok := m.entries[strings.ToLower(cert.Subject.CommonName)]; ok {
		return entry, nil
	}
	// Try DNS SANs
	for _, dns := range cert.DNSNames {
		if entry, ok := m.entries[strings.ToLower(dns)]; ok {
			return entry, nil
		}
	}
	// Try email SANs
	for _, email := range cert.EmailAddresses {
		if entry, ok := m.entries[strings.ToLower(email)]; ok {
			return entry, nil
		}
	}
	return nil, fmt.Errorf("no mapping for cert CN=%s", cert.Subject.CommonName)
}

// Middleware extracts mTLS client identity and injects X-Proxy-User/Scopes headers.
// Falls through to next handler if no client cert (allows mixed Bearer + mTLS).
func (m *ClientMap) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip if already authenticated via Bearer
		if r.Header.Get("Authorization") != "" {
			next.ServeHTTP(w, r)
			return
		}
		// Skip if no TLS or no peer certs
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		cert := r.TLS.PeerCertificates[0]
		entry, err := m.Identify(cert)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, `{"error":"mtls_denied","message":"%s"}`, err.Error())
			return
		}
		// Inject identity headers for downstream processing
		r.Header.Set("X-Proxy-User", entry.ClientID)
		r.Header.Set("X-Proxy-Scopes", strings.Join(entry.Scopes, ","))
		r.Header.Set("X-Auth-Method", "mtls")
		next.ServeHTTP(w, r)
	})
}
