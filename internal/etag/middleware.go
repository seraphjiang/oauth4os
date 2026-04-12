// Package etag provides ETag-based conditional response middleware.
package etag

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
)

// Middleware adds ETag headers and handles If-None-Match for 304 responses.
// Note: buffers the full response body to compute the hash. Best suited for
// small, static responses (health, discovery, config). Not for large proxy responses.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			next.ServeHTTP(w, r)
			return
		}
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, r)

		body := rec.Body.Bytes()
		tag := fmt.Sprintf(`"%x"`, sha256.Sum256(body))

		if r.Header.Get("If-None-Match") == tag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		w.Header().Set("ETag", tag)
		w.WriteHeader(rec.Code)
		w.Write(body)
	})
}
