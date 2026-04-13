package introspect

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestEdge_ConcurrentServeHTTP(t *testing.T) {
	h := NewHandler(&stubLookup{})
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := "token=valid-tok"
			r := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
		}()
	}
	wg.Wait()
}

func TestEdge_ResponseContentType(t *testing.T) {
	h := NewHandler(&stubLookup{})
	body := "token=valid-tok"
	r := httptest.NewRequest("POST", "/oauth/introspect", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "json") {
		t.Errorf("introspect should return JSON, got %q", ct)
	}
}
