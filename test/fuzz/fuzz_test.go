// Fuzz tests for oauth4os — JWT parser, Cedar parser, scope mapper.
//
// Run: go test ./test/fuzz/ -fuzz=. -fuzztime=30s
//
// Targets crash-inducing inputs: panics, nil derefs, infinite loops.

package fuzz

import (
	"testing"

	"github.com/seraphjiang/oauth4os/internal/cedar"
	"github.com/seraphjiang/oauth4os/internal/config"
	"github.com/seraphjiang/oauth4os/internal/scope"
)

// --- Cedar ParsePolicy fuzzer ---

func FuzzCedarParsePolicy(f *testing.F) {
	// Seed corpus
	f.Add(`permit(*, *, *)`)
	f.Add(`forbid(*, *, .opendistro_security)`)
	f.Add(`permit(*, GET, logs-*) when { principal.scope == "read:logs-*" }`)
	f.Add(`permit(*, *, *) unless { resource.index == ".internal" };`)
	f.Add(`permit(admin, *, *) when { principal.iss == "https://keycloak.example.com" }`)
	f.Add(``)
	f.Add(`permit(`)
	f.Add(`forbid(*, *, *) when { }`)
	f.Add(`permit(*, *, *) when { a == "b" && c != "d" }`)
	f.Add(`permit(*, *, *) when { field contains "value" }`)
	f.Add(`permit(*, *, *) when { field in "a,b,c" }`)

	f.Fuzz(func(t *testing.T, input string) {
		// Must not panic — errors are fine
		cedar.ParsePolicy("fuzz", input)
	})
}

// --- Cedar Evaluate fuzzer ---

func FuzzCedarEvaluate(f *testing.F) {
	f.Add("agent-1", "GET", "logs-2026", "read:logs-*")
	f.Add("", "", "", "")
	f.Add("admin", "DELETE", ".opendistro_security", "admin")
	f.Add("x", "POST", "logs-*", "write:logs-*")

	policies := []cedar.Policy{
		{ID: "p1", Effect: cedar.Permit, Principal: cedar.Match{Any: true}, Action: cedar.Match{Equals: "GET"}, Resource: cedar.Match{Pattern: "logs-*"}},
		{ID: "p2", Effect: cedar.Forbid, Principal: cedar.Match{Any: true}, Action: cedar.Match{Any: true}, Resource: cedar.Match{Equals: ".opendistro_security"}},
		{ID: "p3", Effect: cedar.Permit, Principal: cedar.Match{Any: true}, Action: cedar.Match{Any: true}, Resource: cedar.Match{Any: true},
			When: []cedar.Condition{{Field: "principal.scope", Op: "==", Value: "admin"}}},
	}
	engine := cedar.NewEngine(policies)

	f.Fuzz(func(t *testing.T, sub, action, index, scopeVal string) {
		engine.Evaluate(cedar.Request{
			Principal: map[string]string{"sub": sub, "scope": scopeVal},
			Action:    action,
			Resource:  map[string]string{"index": index},
		})
	})
}

// --- Scope Mapper fuzzer ---

func FuzzScopeMapper(f *testing.F) {
	f.Add("read:logs-*")
	f.Add("admin")
	f.Add("")
	f.Add("read:logs-*,write:metrics")
	f.Add("nonexistent:scope")

	mapping := map[string]config.Role{
		"read:logs-*":    {BackendRoles: []string{"readall"}},
		"write:logs-*":   {BackendRoles: []string{"logstash"}},
		"admin":          {BackendRoles: []string{"all_access"}},
		"read:metrics-*": {BackendRoles: []string{"readall"}},
	}
	m := scope.NewMapper(mapping)

	f.Fuzz(func(t *testing.T, scopeStr string) {
		scopes := []string{scopeStr}
		m.Map(scopes)
	})
}

// --- globMatch fuzzer (via Cedar engine) ---

func FuzzCedarGlobMatch(f *testing.F) {
	f.Add("logs-*", "logs-2026")
	f.Add("*", "anything")
	f.Add("", "")
	f.Add("logs-*-prod", "logs-2026-prod")
	f.Add("*-prod", "logs-prod")

	f.Fuzz(func(t *testing.T, pattern, value string) {
		// Exercise glob matching through a policy evaluation
		p := cedar.Policy{
			ID: "fuzz", Effect: cedar.Permit,
			Principal: cedar.Match{Any: true},
			Action:    cedar.Match{Any: true},
			Resource:  cedar.Match{Pattern: pattern},
		}
		engine := cedar.NewEngine([]cedar.Policy{p})
		engine.Evaluate(cedar.Request{
			Principal: map[string]string{"sub": "fuzz"},
			Action:    "GET",
			Resource:  map[string]string{"index": value},
		})
	})
}
