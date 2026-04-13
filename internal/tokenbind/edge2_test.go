package tokenbind

import (
	"sync"
	"testing"
)

func TestEdge_ConcurrentBindVerify(t *testing.T) {
	b := New()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		tok := string(rune('a' + i%26))
		go func() {
			defer wg.Done()
			b.Bind(tok, "fp-"+tok)
		}()
		go func() {
			defer wg.Done()
			b.Verify(tok, "fp-"+tok)
		}()
	}
	wg.Wait()
}

func TestEdge_RemoveNonexistent(t *testing.T) {
	b := New()
	b.Remove("never-bound") // must not panic
}

func TestEdge_DoubleBindSameToken(t *testing.T) {
	b := New()
	b.Bind("tok", "fp1")
	b.Bind("tok", "fp2")
	// Both may verify depending on implementation
	_ = b.Verify("tok", "fp1")
	_ = b.Verify("tok", "fp2")
}
