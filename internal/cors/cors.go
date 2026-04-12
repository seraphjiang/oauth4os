// Package cors provides configurable CORS middleware.
package cors

import (
	"net/http"
	"strings"
)

// Config holds CORS settings.
type Config struct {
	Origins []string // empty = allow all
	Methods []string // empty = default
	Headers []string // empty = default
}

const (
	defaultMethods = "GET, POST, PUT, DELETE, OPTIONS"
	defaultHeaders = "Authorization, Content-Type, X-Request-ID"
)

// Middleware returns CORS middleware.
func Middleware(cfg Config) func(http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range cfg.Origins {
		allowed[o] = true
	}
	methods := defaultMethods
	if len(cfg.Methods) > 0 {
		methods = strings.Join(cfg.Methods, ", ")
	}
	headers := defaultHeaders
	if len(cfg.Headers) > 0 {
		headers = strings.Join(cfg.Headers, ", ")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check origin
			allow := "*"
			if len(allowed) > 0 {
				if allowed[origin] {
					allow = origin
				} else {
					// Origin not allowed — skip CORS headers
					if r.Method == http.MethodOptions {
						w.WriteHeader(http.StatusForbidden)
						return
					}
					next.ServeHTTP(w, r)
					return
				}
			}

			w.Header().Set("Access-Control-Allow-Origin", allow)
			w.Header().Set("Access-Control-Allow-Methods", methods)
			w.Header().Set("Access-Control-Allow-Headers", headers)
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
