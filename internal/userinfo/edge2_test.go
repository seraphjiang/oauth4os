package userinfo

import (
	"net/http/httptest"
	"sync"
	"testing"
)

func TestEdge_ConcurrentServeHTTP(t *testing.T) {
	h := New(stubLookup)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := httptest.NewRequest("GET", "/oauth/userinfo", nil)
			r.Header.Set("Authorization", "Bearer valid-tok")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
		}()
	}
	wg.Wait()
}
