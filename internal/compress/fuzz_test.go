package compress

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// FuzzMiddleware ensures compress middleware never panics on arbitrary Accept-Encoding.
func FuzzMiddleware(f *testing.F) {
	f.Add("gzip")
	f.Add("")
	f.Add("deflate, gzip;q=0.5")
	f.Add("br")
	f.Add("identity")
	f.Add("gzip, deflate, br, zstd")
	f.Fuzz(func(t *testing.T, encoding string) {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hello world test data for compression"))
		})
		handler := Middleware(inner)
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Accept-Encoding", encoding)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r) // must not panic
	})
}
