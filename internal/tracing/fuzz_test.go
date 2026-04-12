package tracing

import "testing"

// FuzzParseTraceparent ensures traceparent parsing never panics.
func FuzzParseTraceparent(f *testing.F) {
	f.Add("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	f.Add("")
	f.Add("00-abc-def-01")
	f.Add("invalid")
	f.Add("00-0000000000000000-0000000000000000-00")
	f.Add("99-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-ff")
	f.Fuzz(func(t *testing.T, header string) {
		ParseTraceparent(header) // must not panic
	})
}
