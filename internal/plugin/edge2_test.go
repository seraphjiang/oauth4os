package plugin

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type passPlugin struct{ name string }

func (p *passPlugin) Name() string                                              { return p.name }
func (p *passPlugin) Authorize(_ *http.Request, _ map[string]interface{}) error { return nil }

func TestEdge_ConcurrentRegisterList(t *testing.T) {
	reg := NewRegistry()
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			reg.Register(&passPlugin{name: string(rune('a' + n%26))})
		}(i)
		go func() {
			defer wg.Done()
			reg.List()
		}()
	}
	wg.Wait()
}

func TestEdge_AuthorizeWithClaims(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&passPlugin{name: "test"})
	r := httptest.NewRequest("GET", "/", nil)
	err := reg.Authorize(r, map[string]interface{}{"sub": "user", "scope": "read"})
	if err != nil {
		t.Errorf("pass plugin should not error: %v", err)
	}
}

func TestEdge_LoadInvalidPath(t *testing.T) {
	reg := NewRegistry()
	err := reg.Load("/nonexistent/plugin.so")
	if err == nil {
		t.Error("invalid path should error")
	}
}
