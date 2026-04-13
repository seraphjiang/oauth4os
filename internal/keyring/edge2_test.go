package keyring

import (
	"sync"
	"testing"
	"time"
)

func TestEdge_ConcurrentCurrentDuringRotate(t *testing.T) {
	r, err := New(2048, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			k := r.Current()
			if k == nil {
				t.Error("Current should never return nil")
			}
		}()
		go func() {
			defer wg.Done()
			r.Rotate()
		}()
	}
	wg.Wait()
}

func TestEdge_StopIdempotent(t *testing.T) {
	r, _ := New(2048, time.Hour)
	r.Stop()
	r.Stop() // must not panic
}
