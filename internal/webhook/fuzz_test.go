package webhook

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// FuzzCheck ensures the webhook authorizer never panics on arbitrary webhook responses.
func FuzzCheck(f *testing.F) {
	f.Add(200, `{"allowed":true}`)
	f.Add(200, `{"allowed":false,"reason":"denied"}`)
	f.Add(500, "")
	f.Add(200, "not json")
	f.Add(200, `{}`)
	f.Add(200, `null`)
	f.Fuzz(func(t *testing.T, status int, body string) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			w.Write([]byte(body))
		}))
		defer srv.Close()
		a := NewAuthorizer(Config{URL: srv.URL})
		a.Check(Request{ClientID: "app"}) // must not panic
	})
}

// FuzzVerify ensures HMAC verification never panics on arbitrary signatures.
func FuzzVerify(f *testing.F) {
	f.Add([]byte(`{"event":"test"}`), "sha256=abc123")
	f.Add([]byte{}, "")
	f.Add([]byte(`null`), "sha256=")
	f.Add([]byte(`x`), string(make([]byte, 10000)))
	f.Fuzz(func(t *testing.T, body []byte, sig string) {
		s := NewSender("test-secret")
		s.Verify(body, sig) // must not panic
	})
}
