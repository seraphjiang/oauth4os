package ipfilter

import "testing"

// FuzzCheck ensures Check never panics on arbitrary addresses
func FuzzCheck(f *testing.F) {
	f.Add("app", "10.0.0.1:8080")
	f.Add("", "")
	f.Add("client", "not-an-ip")
	f.Add("x", "[::1]:443")
	f.Add("y", "999.999.999.999:0")
	f.Fuzz(func(t *testing.T, clientID, addr string) {
		r, _ := New(Config{
			Clients: map[string]*FilterConfig{
				"app": {Allow: []string{"10.0.0.0/8"}},
			},
		})
		if r != nil {
			r.Check(clientID, addr) // must not panic
		}
	})
}
