package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/seraphjiang/oauth4os/internal/config"
	"github.com/seraphjiang/oauth4os/internal/jwt"
	"github.com/seraphjiang/oauth4os/internal/scope"
	"github.com/seraphjiang/oauth4os/internal/token"
	"github.com/seraphjiang/oauth4os/internal/audit"
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

	engineURL, _ := url.Parse(cfg.Upstream.Engine)
	dashboardsURL, _ := url.Parse(cfg.Upstream.Dashboards)

	engineProxy := httputil.NewSingleHostReverseProxy(engineURL)
	dashboardsProxy := httputil.NewSingleHostReverseProxy(dashboardsURL)

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
			// No OAuth token — pass through to engine (existing auth handles it)
			engineProxy.ServeHTTP(w, r)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := validator.Validate(tokenStr)
		if err != nil {
			http.Error(w, `{"error":"invalid_token","message":"`+err.Error()+`"}`, http.StatusUnauthorized)
			return
		}

		// Map scopes to backend roles
		roles := mapper.Map(claims.Scopes)
		if len(roles) == 0 {
			http.Error(w, `{"error":"insufficient_scope"}`, http.StatusForbidden)
			return
		}

		// Inject backend credentials
		r.Header.Del("Authorization")
		r.Header.Set("X-Proxy-User", claims.ClientID)
		r.Header.Set("X-Proxy-Roles", strings.Join(roles, ","))

		// Audit
		auditor.Log(claims.ClientID, claims.Scopes, r.Method, r.URL.Path)

		// Route: /api/* → Dashboards, everything else → Engine
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
	log.Printf("oauth4os proxy listening on %s", addr)
	log.Printf("  Engine:     %s", cfg.Upstream.Engine)
	log.Printf("  Dashboards: %s", cfg.Upstream.Dashboards)
	log.Fatal(http.ListenAndServe(addr, mux))
}
