package loadshed

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// FuzzMiddleware ensures loadshed never panics on concurrent requests.
func FuzzMiddleware(f *testing.F) {
	f.Add("GET", "/search")
	f.Add("POST", "/")
	f.Add("", "")
	f.Fuzz(func(t *testing.T, method, path string) {
		s := New(2)
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
		handler := s.Middleware(inner)
		r := httptest.NewRequest(method, "/"+path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r) // must not panic
	})
}
