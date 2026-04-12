package compress

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Property: compressed output must decompress to original content
func TestProperty_RoundTrip(t *testing.T) {
	payloads := []string{
		"hello world",
		strings.Repeat("log entry\n", 100),
		`{"error":"not_found","status":404}`,
		"",
	}
	for _, payload := range payloads {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(payload))
		})
		handler := Middleware(inner)
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Header().Get("Content-Encoding") != "gzip" {
			// Small payloads may not be compressed
			if w.Body.String() != payload {
				t.Errorf("uncompressed body mismatch for %q", payload)
			}
			continue
		}
		gr, err := gzip.NewReader(w.Body)
		if err != nil {
			t.Fatalf("gzip.NewReader: %v", err)
		}
		got, _ := io.ReadAll(gr)
		gr.Close()
		if string(got) != payload {
			t.Errorf("round-trip mismatch: got %q, want %q", got, payload)
		}
	}
}
