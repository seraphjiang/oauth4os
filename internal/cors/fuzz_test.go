package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// FuzzMiddleware ensures CORS middleware never panics on arbitrary Origin.
func FuzzMiddleware(f *testing.F) {
	f.Add("https://example.com")
	f.Add("")
	f.Add("null")
	f.Add("http://evil.com\r\nX-Injected: true")
	f.Add("https://" + string(make([]byte, 1000)))
	f.Fuzz(func(t *testing.T, origin string) {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
		handler := Middleware(Config{Origins: []string{"https://allowed.com"}})(inner)
		r := httptest.NewRequest("OPTIONS", "/", nil)
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r) // must not panic
	})
}
