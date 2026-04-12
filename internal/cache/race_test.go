package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestConcurrentReadWrite(t *testing.T) {
	c := New(1*time.Second, 1000)
	var wg sync.WaitGroup
	// 10 writers + 10 readers concurrently
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.Set(fmt.Sprintf("/path/%d/%d", n, j), 200, nil, []byte("data"))
			}
		}(i)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.Get(fmt.Sprintf("/path/%d/%d", n, j))
			}
		}(i)
	}
	wg.Wait()
}

func TestConcurrentEviction(t *testing.T) {
	c := New(1*time.Second, 5) // tiny cache
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c.Set(fmt.Sprintf("/evict/%d", n), 200, nil, []byte("x"))
		}(i)
	}
	wg.Wait()
	// Should not panic or corrupt
}
