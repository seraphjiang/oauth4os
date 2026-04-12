package keyring

import (
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// Property: concurrent Current + JWKSHandler reads during rotation must not panic
func TestProperty_ConcurrentJWKSRead(t *testing.T) {
	r, err := New(2048, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()
	var wg sync.WaitGroup
	// Readers
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				r.Current()
				w := httptest.NewRecorder()
				r.JWKSHandler().ServeHTTP(w, httptest.NewRequest("GET", "/.well-known/jwks.json", nil))
			}
		}()
	}
	// Rotator
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 3; j++ {
			r.rotate()
			time.Sleep(time.Millisecond)
		}
	}()
	wg.Wait()
}
