package analytics

import (
	"sync"
	"testing"
)

func TestEdge_ConcurrentRecord(t *testing.T) {
	a := New()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			a.Record("client-"+string(rune('a'+n%5)), []string{"read"}, "logs")
		}(i)
	}
	wg.Wait()
	s := a.Snapshot()
	if len(s.Clients) == 0 {
		t.Error("should have tracked clients")
	}
}

func TestEdge_SnapshotDuringRecord(t *testing.T) {
	a := New()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			a.Record("c1", []string{"read"}, "idx")
		}()
		go func() {
			defer wg.Done()
			a.Snapshot()
		}()
	}
	wg.Wait()
}
