package dpop

import (
	"encoding/json"
	"sync"
	"testing"
)

func TestEdge_ConcurrentThumbprint(t *testing.T) {
	jwk := json.RawMessage(`{"kty":"EC","crv":"P-256","x":"test","y":"test"}`)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tp := JWKThumbprint(jwk)
			if tp == "" {
				t.Error("thumbprint should not be empty")
			}
		}()
	}
	wg.Wait()
}

func TestEdge_EmptyJWK(t *testing.T) {
	tp := JWKThumbprint(json.RawMessage(`{}`))
	if tp == "" {
		t.Error("empty JWK should still produce thumbprint")
	}
}

func TestEdge_NilJWK(t *testing.T) {
	tp := JWKThumbprint(nil)
	// Should not panic
	_ = tp
}
