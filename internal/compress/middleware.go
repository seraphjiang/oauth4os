// Package compress provides gzip response compression middleware.
package compress

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var gzPool = sync.Pool{
	New: func() any {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		return w
	},
}

type gzipWriter struct {
	http.ResponseWriter
	gz *gzip.Writer
}

func (g *gzipWriter) Write(b []byte) (int, error) { return g.gz.Write(b) }

// Middleware compresses responses when the client accepts gzip.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gz := gzPool.Get().(*gzip.Writer)
		defer gzPool.Put(gz)
		gz.Reset(w)
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Del("Content-Length")
		next.ServeHTTP(&gzipWriter{ResponseWriter: w, gz: gz}, r)
	})
}
