package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
	"github.com/seraphjiang/oauth4os/internal/scope"
)

func TestEdge_ConcurrentListProviders(t *testing.T) {
	s := NewState(&config.Config{}, scope.NewMapper(nil), nil)
	mux := http.NewServeMux()
	s.Register(mux)
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest("GET", "/admin/providers", nil))
		}()
		go func() {
			defer wg.Done()
			body := `{"name":"p","issuer":"https://x.com","client_id":"a","client_secret":"b"}`
			r := httptest.NewRequest("POST", "/admin/providers", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
		}()
	}
	wg.Wait()
}

func TestEdge_UpdateScopeMappingsInvalidJSON(t *testing.T) {
	s := NewState(&config.Config{}, scope.NewMapper(nil), nil)
	mux := http.NewServeMux()
	s.Register(mux)
	r := httptest.NewRequest("PUT", "/admin/scope-mappings", strings.NewReader("not json"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code == 200 {
		t.Error("invalid JSON should fail")
	}
}
