package keyring

import (
	"sync"
	"testing"
	"time"
)

func TestEdge_MultipleRotations(t *testing.T) {
	r, _ := New(2048, time.Hour)
	defer r.Stop()
	ids := map[string]bool{}
	for i := 0; i < 5; i++ {
		r.Rotate()
		ids[r.Current().ID] = true
	}
	if len(ids) < 5 {
		t.Errorf("5 rotations should produce 5 unique IDs, got %d", len(ids))
	}
}

func TestEdge_ConcurrentRotateStop(t *testing.T) {
	r, _ := New(2048, time.Hour)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Rotate()
		}()
	}
	wg.Wait()
	r.Stop()
}
