package tlsreload

import (
	"crypto/tls"
	"testing"
	"time"
)

// Mutation: remove GetCertificate → must return loaded cert
func TestMutation_GetCertificate(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := genCert(t, dir, "test")
	r, err := New(certPath, keyPath, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()
	cert, err := r.GetCertificate(&tls.ClientHelloInfo{})
	if err != nil {
		t.Fatal(err)
	}
	if cert == nil {
		t.Error("GetCertificate must return loaded certificate")
	}
}

// Mutation: remove Stop → poll goroutine must terminate
func TestMutation_StopTerminates(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := genCert(t, dir, "test")
	r, err := New(certPath, keyPath, 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		r.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop must terminate poll goroutine")
	}
}
