package cedar

import "testing"

// FuzzParsePolicy tests Cedar policy parser with random inputs.
// Run: go test -fuzz=FuzzParsePolicy -fuzztime=30s ./internal/cedar/
func FuzzParsePolicy(f *testing.F) {
	// Seed corpus — valid and edge-case policies
	f.Add("permit(*, *, *);")
	f.Add("forbid(*, *, *);")
	f.Add("permit(*, *, .opendistro_security);")
	f.Add("forbid(*, GET, logs-*) when { scope == \"admin\" };")
	f.Add("permit(*, *, *) unless { scope == \"readonly\" };")
	f.Add("")
	f.Add("(")
	f.Add("permit")
	f.Add("permit(")
	f.Add("permit(*, *)")
	f.Add("forbid();")
	f.Add("permit(*, *, *) when { };")
	f.Add("permit(*, *, *) when { x == };")
	f.Add(string(make([]byte, 10000))) // large input

	f.Fuzz(func(t *testing.T, input string) {
		// Must not panic — errors are fine
		ParsePolicy("fuzz", input)
	})
}

// FuzzEvaluate tests Cedar engine evaluation with random request fields.
func FuzzEvaluate(f *testing.F) {
	f.Add("admin", "GET", "logs-2026", "/logs-2026/_search")
	f.Add("", "", "", "")
	f.Add("user", "DELETE", ".opendistro_security", "/.opendistro_security")
	f.Add(string(make([]byte, 1000)), "POST", string(make([]byte, 1000)), "/")

	engine := NewEngine([]Policy{
		{ID: "p1", Effect: Permit, Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Any: true}},
		{ID: "p2", Effect: Forbid, Principal: Match{Any: true}, Action: Match{Any: true}, Resource: Match{Value: ".opendistro_security"}},
	})

	f.Fuzz(func(t *testing.T, sub, method, index, path string) {
		// Must not panic
		engine.Evaluate(Request{
			Principal: map[string]string{"sub": sub},
			Action:    method,
			Resource:  map[string]string{"index": index, "path": path},
		})
	})
}

// FuzzGlobMatch tests glob pattern matching with random inputs.
func FuzzGlobMatch(f *testing.F) {
	f.Add("logs-*", "logs-2026")
	f.Add("*", "anything")
	f.Add("", "")
	f.Add("a*b*c", "aXbYc")
	f.Add(string(make([]byte, 500)), string(make([]byte, 500)))

	f.Fuzz(func(t *testing.T, pattern, value string) {
		// Must not panic
		globMatch(pattern, value)
	})
}
