package cache

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

// Property: concurrent Get/Set/expiry must never corrupt or panic
func TestProperty_ConcurrentGetSetExpiry(t *testing.T) {
	c := New(50*time.Millisecond, 100)
	var wg sync.WaitGroup
	// Writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				c.Set("/path/"+strconv.Itoa(n)+"/"+strconv.Itoa(j), 200, nil, []byte("data"))
			}
		}(i)
	}
	// Readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				c.Get("/path/" + strconv.Itoa(n) + "/" + strconv.Itoa(j))
			}
		}(i)
	}
	wg.Wait()
	// Let entries expire
	time.Sleep(100 * time.Millisecond)
	// Reads after expiry must return nil
	if e := c.Get("/path/0/0"); e != nil {
		t.Error("expired entry should return nil")
	}
}
