package par

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestEdge_ConcurrentPush(t *testing.T) {
	h := NewHandler(func(id, secret string) error { return nil })
	mux := http.NewServeMux()
	h.Register(mux)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := "client_id=app&redirect_uri=http://localhost/cb&response_type=code&scope=read"
			r := httptest.NewRequest("POST", "/oauth/par", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
		}()
	}
	wg.Wait()
}

func TestEdge_EmptyBody(t *testing.T) {
	h := NewHandler(nil)
	mux := http.NewServeMux()
	h.Register(mux)
	r := httptest.NewRequest("POST", "/oauth/par", strings.NewReader(""))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code == 201 {
		t.Error("empty body should not succeed")
	}
}
