package webhook

import (
	"sync"
	"testing"
)

func TestEdge_ConcurrentSign(t *testing.T) {
	s := NewSender("secret")
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			sig := s.Sign([]byte(`{"n":` + string(rune('0'+n%10)) + `}`))
			if sig == "" {
				t.Error("signature should not be empty")
			}
		}(i)
	}
	wg.Wait()
}

func TestEdge_LargePayloadSign(t *testing.T) {
	s := NewSender("secret")
	big := make([]byte, 1<<20) // 1MB
	sig := s.Sign(big)
	if sig == "" {
		t.Error("large payload should produce signature")
	}
	if !s.Verify(big, sig) {
		t.Error("large payload verify should match")
	}
}
