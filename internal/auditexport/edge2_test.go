package auditexport

import (
	"encoding/json"
	"sync"
	"testing"
)

func TestEdge_ConcurrentAddFlush(t *testing.T) {
	u := &memUploader{data: map[string][]byte{}}
	e := New(u, "test/", 0)
	defer e.Stop()
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			e.Add(json.RawMessage(`{"action":"test"}`))
		}()
		go func() {
			defer wg.Done()
			e.Flush()
		}()
	}
	wg.Wait()
}
