package tlsreload

import "testing"

// FuzzNew ensures New never panics on arbitrary cert/key paths.
func FuzzNew(f *testing.F) {
	f.Add("/nonexistent/cert.pem", "/nonexistent/key.pem")
	f.Add("", "")
	f.Add("/dev/null", "/dev/null")
	f.Fuzz(func(t *testing.T, certPath, keyPath string) {
		// Skip device files that can hang
		for _, p := range []string{certPath, keyPath} {
			if len(p) > 4 && p[:5] == "/dev/" {
				t.Skip("skip device files")
			}
		}
		r, err := New(certPath, keyPath, 0)
		if err == nil && r != nil {
			r.Stop()
		}
		// must not panic
	})
}
