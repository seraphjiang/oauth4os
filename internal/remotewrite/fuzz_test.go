package remotewrite

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// FuzzHandler ensures the remote write HTTP handler never panics on arbitrary input.
func FuzzHandler(f *testing.F) {
	f.Add(`{"timeseries":[{"labels":{"__name__":"test"},"samples":[{"value":1}]}]}`)
	f.Add(`{}`)
	f.Add(``)
	f.Add(`not json`)
	f.Add(`{"timeseries":null}`)
	f.Add(`{"timeseries":[{"labels":{},"samples":[]}]}`)
	f.Add(strings.Repeat(`{"timeseries":[`, 100))
	f.Fuzz(func(t *testing.T, body string) {
		r := New()
		h := r.Handler()
		req := httptest.NewRequest("POST", "/api/v1/write", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req) // must not panic
	})
}
