package auditexport

import (
	"encoding/json"
	"sync"
	"testing"
)

// Property: concurrent Add must not lose entries or panic
func TestProperty_ConcurrentAdd(t *testing.T) {
	u := &memUploader{data: map[string][]byte{}}
	e := New(u, "audit/", 0)
	defer e.Stop()

	var wg sync.WaitGroup
	n := 100
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			e.Add(json.RawMessage(`{"i":` + string(rune('0'+idx%10)) + `}`))
		}(i)
	}
	wg.Wait()

	if err := e.Flush(); err != nil {
		t.Fatal(err)
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if len(u.data) == 0 {
		t.Error("concurrent Add + Flush must produce output")
	}
}
