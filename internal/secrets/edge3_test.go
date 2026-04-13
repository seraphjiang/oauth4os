package secrets

import (
	"os"
	"sync"
	"testing"
)

func TestEdge_ConcurrentResolve(t *testing.T) {
	os.Setenv("TEST_CONC_SECRET", "value")
	defer os.Unsetenv("TEST_CONC_SECRET")
	r := New()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Resolve("env:TEST_CONC_SECRET")
		}()
	}
	wg.Wait()
}
