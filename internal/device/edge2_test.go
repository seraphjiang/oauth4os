package device

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestEdge_ConcurrentRequestCode(t *testing.T) {
	h := NewHandler(nil)
	mux := http.NewServeMux()
	h.Register(mux)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := "client_id=app&scope=read"
			r := httptest.NewRequest("POST", "/oauth/device/code", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
		}()
	}
	wg.Wait()
}

func TestEdge_MissingClientID(t *testing.T) {
	h := NewHandler(nil)
	mux := http.NewServeMux()
	h.Register(mux)
	body := "scope=read"
	r := httptest.NewRequest("POST", "/oauth/device/code", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code == 200 && strings.Contains(w.Body.String(), "device_code") {
		t.Error("missing client_id should fail or return error")
	}
}
