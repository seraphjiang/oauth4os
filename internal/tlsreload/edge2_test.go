package tlsreload

import (
	"testing"
	"time"
)

func TestEdge_NewWithValidCert(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := genCert(t, dir, "test")
	r, err := New(certPath, keyPath, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()
	cert, err := r.GetCertificate(nil)
	if err != nil || cert == nil {
		t.Error("should return valid certificate")
	}
}

func TestEdge_StopIdempotent(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := genCert(t, dir, "test")
	r, _ := New(certPath, keyPath, 0)
	r.Stop()
	r.Stop() // must not panic
}

func TestEdge_PollInterval(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := genCert(t, dir, "test")
	r, err := New(certPath, keyPath, 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(250 * time.Millisecond)
	r.Stop()
}
