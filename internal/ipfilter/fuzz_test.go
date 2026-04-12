package ipfilter

import (
	"testing"
)

// FuzzExtractIP ensures IP extraction never panics on arbitrary input.
func FuzzExtractIP(f *testing.F) {
	f.Add("192.168.1.1:8080")
	f.Add("10.0.0.1")
	f.Add("[::1]:443")
	f.Add("")
	f.Add("not-an-ip")
	f.Add("999.999.999.999:0")
	f.Fuzz(func(t *testing.T, addr string) {
		extractIP(addr) // must not panic
	})
}

// FuzzCheck ensures Check never panics on arbitrary client/addr.
func FuzzCheck(f *testing.F) {
	f.Add("client", "1.2.3.4:80")
	f.Add("", "")
	f.Add("x", "[::1]:443")
	f.Fuzz(func(t *testing.T, client, addr string) {
		r, err := New(Config{Filters: []FilterConfig{{ClientID: "app", Allow: []string{"10.0.0.0/8"}}}})
		if err != nil {
			return
		}
		r.Check(client, addr) // must not panic
	})
}
