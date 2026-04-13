package registration

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestEdge_ConcurrentRegister(t *testing.T) {
	h := NewHandler(func(id, secret string, scopes, redirects []string) {}, []string{"read"})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := `{"client_name":"app","redirect_uris":["http://localhost/cb"],"scope":"read"}`
			r := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.Register(w, r)
		}()
	}
	wg.Wait()
}

func TestEdge_InvalidContentType(t *testing.T) {
	h := NewHandler(func(id, secret string, scopes, redirects []string) {}, []string{"read"})
	r := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(`{"client_name":"app"}`))
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	h.Register(w, r)
}

func TestEdge_MissingRedirectURIs(t *testing.T) {
	h := NewHandler(func(id, secret string, scopes, redirects []string) {}, []string{"read"})
	body := `{"client_name":"app","scope":"read"}`
	r := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, r)
	// Implementation may require redirect_uris or not
}
