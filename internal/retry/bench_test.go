package retry

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

type mockRT struct {
	status int
}

func (m *mockRT) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: m.status,
		Body:       http.NoBody,
	}, nil
}

func BenchmarkRoundTrip_NoRetry(b *testing.B) {
	t := &Transport{Base: &mockRT{status: 200}, MaxRetries: 3, BaseDelay: time.Millisecond}
	req, _ := http.NewRequest("GET", "http://localhost/test", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := t.RoundTrip(req)
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}
}

func BenchmarkRoundTrip_WithBody(b *testing.B) {
	t := &Transport{Base: &mockRT{status: 200}, MaxRetries: 3, BaseDelay: time.Millisecond}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "http://localhost/test", strings.NewReader(`{"query":"*"}`))
		resp, _ := t.RoundTrip(req)
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}
}
