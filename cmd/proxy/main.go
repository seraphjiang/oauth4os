package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/seraphjiang/oauth4os/internal/audit"
	"github.com/seraphjiang/oauth4os/internal/cedar"
	"github.com/seraphjiang/oauth4os/internal/config"
	"github.com/seraphjiang/oauth4os/internal/jwt"
	"github.com/seraphjiang/oauth4os/internal/scope"
	"github.com/seraphjiang/oauth4os/internal/token"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	validator := jwt.NewValidator(cfg.Providers)
	mapper := scope.NewMapper(cfg.ScopeMapping)
	tokenMgr := token.NewManager()
	auditor := audit.NewAuditor(os.Stdout)

	// Cedar policy engine — default policies if none configured
	defaultPolicies := []cedar.Policy{
		{ID: "default-permit", Effect: cedar.Permit,
			Principal: cedar.Match{Any: true}, Action: cedar.Match{Any: true},
			Resource: cedar.Match{Any: true}},
		{ID: "forbid-security-index", Effect: cedar.Forbid,
			Principal: cedar.Match{Any: true}, Action: cedar.Match{Any: true},
			Resource: cedar.Match{Equals: ".opendistro_security"}},
	}
	policyEngine := cedar.NewEngine(defaultPolicies)

	// Transport for upstream connections (handles self-signed certs)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.TLS.InsecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	engineURL, _ := url.Parse(cfg.Upstream.Engine)
	dashboardsURL, _ := url.Parse(cfg.Upstream.Dashboards)

	engineProxy := httputil.NewSingleHostReverseProxy(engineURL)
	engineProxy.Transport = transport
	dashboardsProxy := httputil.NewSingleHostReverseProxy(dashboardsURL)
	dashboardsProxy.Transport = transport

	mux := http.NewServeMux()

	// Token endpoints
	mux.HandleFunc("POST /oauth/token", tokenMgr.IssueToken)
	mux.HandleFunc("DELETE /oauth/token/{id}", tokenMgr.RevokeToken)
	mux.HandleFunc("GET /oauth/tokens", tokenMgr.ListTokens)
	mux.HandleFunc("GET /oauth/token/{id}", tokenMgr.GetToken)

	// Health
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","version":"0.1.0"}`)
	})

	// Proxy all other requests
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			engineProxy.ServeHTTP(w, r)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := validator.Validate(tokenStr)
		if err != nil {
			http.Error(w, `{"error":"invalid_token","message":"`+err.Error()+`"}`, http.StatusUnauthorized)
			return
		}

		roles := mapper.Map(claims.Scopes)
		if len(roles) == 0 {
			http.Error(w, `{"error":"insufficient_scope"}`, http.StatusForbidden)
			return
		}

		// Cedar policy evaluation
		index := extractIndex(r.URL.Path)
		decision := policyEngine.Evaluate(cedar.Request{
			Principal: map[string]string{"sub": claims.ClientID, "scope": strings.Join(claims.Scopes, ",")},
			Action:    r.Method,
			Resource:  map[string]string{"index": index, "path": r.URL.Path},
		})
		if !decision.Allowed {
			http.Error(w, `{"error":"forbidden","reason":"`+decision.Reason+`","policy":"`+decision.Policy+`"}`, http.StatusForbidden)
			return
		}

		r.Header.Del("Authorization")
		r.Header.Set("X-Proxy-User", claims.ClientID)
		r.Header.Set("X-Proxy-Roles", strings.Join(roles, ","))

		auditor.Log(claims.ClientID, claims.Scopes, r.Method, r.URL.Path)

		if strings.HasPrefix(r.URL.Path, "/api/") {
			dashboardsProxy.ServeHTTP(w, r)
		} else {
			engineProxy.ServeHTTP(w, r)
		}
	})

	addr := cfg.Listen
	if addr == "" {
		addr = ":8443"
	}

	log.Printf("oauth4os proxy listening on %s (tls=%v)", addr, cfg.TLS.Enabled)
	log.Printf("  Engine:     %s", cfg.Upstream.Engine)
	log.Printf("  Dashboards: %s", cfg.Upstream.Dashboards)

	if cfg.TLS.Enabled && cfg.TLS.CertFile != "" && cfg.TLS.KeyFile != "" {
		log.Fatal(http.ListenAndServeTLS(addr, cfg.TLS.CertFile, cfg.TLS.KeyFile, mux))
	} else {
		log.Fatal(http.ListenAndServe(addr, mux))
	}
}

// extractIndex pulls the index name from an OpenSearch URL path.
// e.g., "/logs-2026.04/_search" → "logs-2026.04"
func extractIndex(path string) string {
	path = strings.TrimPrefix(path, "/")
	if idx := strings.IndexByte(path, '/'); idx > 0 {
		return path[:idx]
	}
	return path
}
