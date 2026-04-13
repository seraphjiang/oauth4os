package cache

import (
	"sync"
	"testing"
	"time"
)

func TestEdge_ConcurrentSetGet(t *testing.T) {
	c := New(time.Minute, 1000)
	defer c.Stop()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		k := string(rune('a' + i%26))
		go func() {
			defer wg.Done()
			c.Set(k, 200, nil, []byte("val"))
		}()
		go func() {
			defer wg.Done()
			c.Get(k)
		}()
	}
	wg.Wait()
}

func TestEdge_StopIdempotent(t *testing.T) {
	c := New(time.Minute, 100)
	c.Stop()
	c.Stop() // must not panic (sync.Once)
}
