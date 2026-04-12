package tlsreload

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func genCert(t *testing.T, dir, name string) (certPath, keyPath string) {
	t.Helper()
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: name},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")
	cf, _ := os.Create(certPath)
	pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: der})
	cf.Close()
	kb, _ := x509.MarshalECPrivateKey(priv)
	kf, _ := os.Create(keyPath)
	pem.Encode(kf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: kb})
	kf.Close()
	return
}

func TestLoadAndGetCertificate(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := genCert(t, dir, "test1")
	r, err := New(certPath, keyPath, 0)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := r.GetCertificate(nil)
	if err != nil || cert == nil {
		t.Fatal("expected certificate")
	}
}

func TestReloadOnChange(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := genCert(t, dir, "original")
	r, _ := New(certPath, keyPath, 50*time.Millisecond)
	defer r.Stop()

	reloaded := make(chan struct{}, 1)
	r.OnReload = func() { reloaded <- struct{}{} }

	// Overwrite with new cert
	time.Sleep(100 * time.Millisecond) // ensure modtime differs
	genCert(t, dir, "updated")

	select {
	case <-reloaded:
		// success
	case <-time.After(2 * time.Second):
		t.Error("expected reload callback")
	}
}

func TestInvalidCert(t *testing.T) {
	dir := t.TempDir()
	_, err := New(filepath.Join(dir, "nope.pem"), filepath.Join(dir, "nope.key"), 0)
	if err == nil {
		t.Error("expected error for missing cert")
	}
}
