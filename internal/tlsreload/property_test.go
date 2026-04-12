package tlsreload

import (
	"crypto/tls"
	"sync"
	"testing"
)

// Property: concurrent GetCertificate calls must not panic or return nil
func TestProperty_ConcurrentGetCertificate(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := genCert(t, dir, "concurrent")
	r, err := New(certPath, keyPath, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cert, err := r.GetCertificate(&tls.ClientHelloInfo{})
			if err != nil {
				t.Errorf("GetCertificate error: %v", err)
			}
			if cert == nil {
				t.Error("GetCertificate returned nil")
			}
		}()
	}
	wg.Wait()
}
