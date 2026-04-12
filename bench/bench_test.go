// Benchmarks for oauth4os core components.
//
// Run: go test ./bench/ -bench=. -benchmem -count=3
//
// Measures:
//   - Scope mapping (Map scopes → backend roles)
//   - Cedar policy evaluation (permit/deny decisions)
//   - Cedar glob matching (index pattern matching)
//   - Full proxy round-trip (HTTP request through auth middleware)

package bench

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/seraphjiang/oauth4os/internal/cedar"
	"github.com/seraphjiang/oauth4os/internal/config"
	"github.com/seraphjiang/oauth4os/internal/scope"
)

// --- Scope Mapper Benchmarks ---

func newMapper(n int) *scope.Mapper {
	mapping := make(map[string]config.Role, n)
	for i := 0; i < n; i++ {
		mapping[fmt.Sprintf("read:index-%d", i)] = config.Role{
			BackendRoles: []string{fmt.Sprintf("reader_%d", i)},
		}
	}
	return scope.NewMapper(mapping)
}

func BenchmarkScopeMapper_1Scope(b *testing.B) {
	m := newMapper(10)
	scopes := []string{"read:index-0"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Map(scopes)
	}
}

func BenchmarkScopeMapper_5Scopes(b *testing.B) {
	m := newMapper(100)
	scopes := []string{"read:index-0", "read:index-10", "read:index-20", "read:index-30", "read:index-40"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Map(scopes)
	}
}

func BenchmarkScopeMapper_Miss(b *testing.B) {
	m := newMapper(100)
	scopes := []string{"nonexistent:scope"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Map(scopes)
	}
}

// --- Cedar Engine Benchmarks ---

func newCedarEngine(nPolicies int) *cedar.Engine {
	policies := make([]cedar.Policy, nPolicies)
	for i := 0; i < nPolicies; i++ {
		policies[i] = cedar.Policy{
			ID: fmt.Sprintf("policy-%d", i), Effect: cedar.Permit,
			Principal: cedar.Match{Any: true},
			Action:    cedar.Match{Equals: "GET"},
			Resource:  cedar.Match{Pattern: fmt.Sprintf("logs-%d-*", i)},
			When: []cedar.Condition{
				{Field: "principal.scope", Op: "==", Value: fmt.Sprintf("read:logs-%d", i)},
			},
		}
	}
	return cedar.NewEngine(policies)
}

func BenchmarkCedar_1Policy(b *testing.B) {
	engine := newCedarEngine(1)
	req := cedar.Request{
		Principal: map[string]string{"scope": "read:logs-0"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs-0-2026"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Evaluate(req)
	}
}

func BenchmarkCedar_10Policies(b *testing.B) {
	engine := newCedarEngine(10)
	req := cedar.Request{
		Principal: map[string]string{"scope": "read:logs-9"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs-9-2026"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Evaluate(req)
	}
}

func BenchmarkCedar_100Policies(b *testing.B) {
	engine := newCedarEngine(100)
	req := cedar.Request{
		Principal: map[string]string{"scope": "read:logs-99"},
		Action:    "GET",
		Resource:  map[string]string{"index": "logs-99-2026"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Evaluate(req)
	}
}

func BenchmarkCedar_DenyAll(b *testing.B) {
	engine := newCedarEngine(10)
	req := cedar.Request{
		Principal: map[string]string{"scope": "write:metrics"},
		Action:    "DELETE",
		Resource:  map[string]string{"index": "metrics-2026"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Evaluate(req)
	}
}

func BenchmarkCedar_ForbidOverride(b *testing.B) {
	policies := []cedar.Policy{
		{ID: "allow-all", Effect: cedar.Permit, Principal: cedar.Match{Any: true}, Action: cedar.Match{Any: true}, Resource: cedar.Match{Any: true}},
		{ID: "deny-security", Effect: cedar.Forbid, Principal: cedar.Match{Any: true}, Action: cedar.Match{Any: true}, Resource: cedar.Match{Equals: ".opendistro_security"}},
	}
	engine := cedar.NewEngine(policies)
	req := cedar.Request{
		Principal: map[string]string{"sub": "agent"},
		Action:    "GET",
		Resource:  map[string]string{"index": ".opendistro_security"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Evaluate(req)
	}
}

// --- Proxy Round-Trip Benchmark ---

func BenchmarkProxyRoundTrip(b *testing.B) {
	// Simulate the proxy auth middleware path (no real OpenSearch)
	mapper := newMapper(10)
	engine := newCedarEngine(10)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Extract token
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.WriteHeader(200) // passthrough
			return
		}
		_ = strings.TrimPrefix(auth, "Bearer ")

		// 2. Map scopes (skip JWT validation — no real JWKS in bench)
		roles := mapper.Map([]string{"read:index-0"})

		// 3. Cedar evaluation
		decision := engine.Evaluate(cedar.Request{
			Principal: map[string]string{"scope": "read:index-0"},
			Action:    r.Method,
			Resource:  map[string]string{"index": "logs-0-2026"},
		})

		if !decision.Allowed || len(roles) == 0 {
			w.WriteHeader(403)
			return
		}

		r.Header.Set("X-Proxy-Roles", strings.Join(roles, ","))
		w.WriteHeader(200)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := server.Client()
	req, _ := http.NewRequest("GET", server.URL+"/logs-0-2026/_search", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Do(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkProxyRoundTrip_Passthrough(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := server.Client()
	req, _ := http.NewRequest("GET", server.URL+"/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Do(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
