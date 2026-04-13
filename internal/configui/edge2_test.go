package configui

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/config"
)

func TestEdge_ConcurrentJSON(t *testing.T) {
	h := New(func() *config.Config { return &config.Config{Listen: ":8443"} })
	mux := http.NewServeMux()
	h.Register(mux)
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest("GET", "/admin/config/json", nil))
		}()
	}
	wg.Wait()
}
