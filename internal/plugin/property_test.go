package plugin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type counterPlugin struct {
	name string
	mu   sync.Mutex
	n    int
}

func (c *counterPlugin) Name() string { return c.name }
func (c *counterPlugin) Authorize(_ *http.Request, _ map[string]interface{}) error {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
	return nil
}

// Property: concurrent Authorize must call all plugins without panic
func TestProperty_ConcurrentAuthorize(t *testing.T) {
	reg := NewRegistry()
	p := &counterPlugin{name: "counter"}
	reg.Register(p)

	var wg sync.WaitGroup
	n := 100
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := httptest.NewRequest("GET", "/", nil)
			reg.Authorize(r, map[string]interface{}{"sub": "user"})
		}()
	}
	wg.Wait()

	p.mu.Lock()
	got := p.n
	p.mu.Unlock()
	if got != n {
		t.Errorf("expected %d calls, got %d", n, got)
	}
}

// Property: concurrent Register + Authorize must not panic
func TestProperty_ConcurrentRegisterAuthorize(t *testing.T) {
	reg := NewRegistry()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			reg.Register(&counterPlugin{name: fmt.Sprintf("p-%d", idx)})
		}(i)
		go func() {
			defer wg.Done()
			r := httptest.NewRequest("GET", "/", nil)
			reg.Authorize(r, nil)
		}()
	}
	wg.Wait()
}
