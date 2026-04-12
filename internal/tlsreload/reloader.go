// Package tlsreload provides a TLS certificate that auto-reloads
// when the cert/key files change on disk. Uses polling (no fsnotify dep).
package tlsreload

import (
	"crypto/tls"
	"os"
	"sync"
	"time"
)

// Reloader watches cert/key files and reloads on change.
type Reloader struct {
	certPath, keyPath string
	mu                sync.RWMutex
	cert              *tls.Certificate
	modTime           time.Time
	stopCh            chan struct{}
	OnReload          func() // called after successful reload
}

// New loads the initial certificate and starts polling for changes.
func New(certPath, keyPath string, pollInterval time.Duration) (*Reloader, error) {
	r := &Reloader{
		certPath: certPath,
		keyPath:  keyPath,
		stopCh:   make(chan struct{}),
	}
	if err := r.load(); err != nil {
		return nil, err
	}
	if pollInterval > 0 {
		go r.poll(pollInterval)
	}
	return r, nil
}

// GetCertificate implements tls.Config.GetCertificate.
func (r *Reloader) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cert, nil
}

// Stop halts the polling goroutine.
func (r *Reloader) Stop() {
	close(r.stopCh)
}

func (r *Reloader) load() error {
	cert, err := tls.LoadX509KeyPair(r.certPath, r.keyPath)
	if err != nil {
		return err
	}
	info, _ := os.Stat(r.certPath)
	r.mu.Lock()
	r.cert = &cert
	if info != nil {
		r.modTime = info.ModTime()
	}
	r.mu.Unlock()
	return nil
}

func (r *Reloader) poll(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			info, err := os.Stat(r.certPath)
			if err != nil || !info.ModTime().After(r.modTime) {
				continue
			}
			if err := r.load(); err == nil && r.OnReload != nil {
				r.OnReload()
			}
		case <-r.stopCh:
			return
		}
	}
}
