package remotewrite

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestEdge_ConcurrentWrite(t *testing.T) {
	r := New()
	h := r.Handler()
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := "metric_name,value\nhttp_requests,42\n"
			req := httptest.NewRequest("POST", "/api/v1/write", strings.NewReader(body))
			req.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
		}()
	}
	wg.Wait()
}
