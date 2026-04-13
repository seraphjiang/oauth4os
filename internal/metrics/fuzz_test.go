package metrics

import "testing"

// FuzzCounterInc ensures counter never panics on arbitrary label values.
func FuzzCounterInc(f *testing.F) {
	f.Add("GET", "/health", 200)
	f.Add("", "", 0)
	f.Add("POST", "/very/long/path/that/goes/on/and/on", 500)
	f.Add("DELETE", "/\x00null", -1)
	f.Fuzz(func(t *testing.T, method, path string, status int) {
		c := NewCounter()
		c.Inc(Labels{Method: method, Path: path, Status: status})
		c.Inc(Labels{Method: method, Path: path, Status: status})
		// must not panic
	})
}
