package jwt

import (
	"sync"
	"testing"
)

func TestEdge_ConcurrentValidate(t *testing.T) {
	v := NewValidator(nil)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v.Validate("invalid-token")
		}()
	}
	wg.Wait()
}

func TestEdge_ValidateNilProviders(t *testing.T) {
	v := NewValidator(nil)
	_, err := v.Validate("eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.sig")
	if err == nil {
		t.Error("no providers should fail validation")
	}
}
