package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/seraphjiang/oauth4os/internal/admin"
	"github.com/seraphjiang/oauth4os/internal/audit"
	"github.com/seraphjiang/oauth4os/internal/cedar"
	"github.com/seraphjiang/oauth4os/internal/config"
	"github.com/seraphjiang/oauth4os/internal/discovery"
	"github.com/seraphjiang/oauth4os/internal/exchange"
	"github.com/seraphjiang/oauth4os/internal/introspect"
	"github.com/seraphjiang/oauth4os/internal/jwt"
	"github.com/seraphjiang/oauth4os/internal/pkce"
	"github.com/seraphjiang/oauth4os/internal/ratelimit"
	"github.com/seraphjiang/oauth4os/internal/registration"
	"github.com/seraphjiang/oauth4os/internal/scope"
	"github.com/seraphjiang/oauth4os/internal/token"
	"github.com/seraphjiang/oauth4os/internal/tracing"
)

const version = "0.2.0"

// Prometheus-style metrics
var (
	requestsTotal   atomic.Int64
	requestsActive  atomic.Int64
	requestsFailed  atomic.Int64
	authSuccess     atomic.Int64
	authFailed      atomic.Int64
	cedarDenied     atomic.Int64
	rateLimited     atomic.Int64
	upstreamErrors  atomic.Int64
	startTime       = time.Now()
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	validator := jwt.NewValidator(cfg.Providers)
	mapper := scope.NewMultiTenantMapper(cfg.ScopeMapping, cfg.Tenants)
	tokenMgr := token.NewManager()
	auditor := audit.NewJSONAuditor(os.Stdout)
	limiter := ratelimit.New(cfg.RateLimits, 600)

	// Tracing — stdout in dev, noop if OAUTH4OS_TRACING=off
	var tracer tracing.Tracer
	if os.Getenv("OAUTH4OS_TRACING") == "off" {
		tracer = tracing.NoopTracer{}
	} else {
		tracer = tracing.NewStdoutTracer(os.Stderr)
	}

	// Cedar policy engine (multi-tenant)
	defaultPolicies := []cedar.Policy{
		{ID: "default-permit", Effect: cedar.Permit,
			Principal: cedar.Match{Any: true}, Action: cedar.Match{Any: true},
			Resource: cedar.Match{Any: true}},
		{ID: "forbid-security-index", Effect: cedar.Forbid,
			Principal: cedar.Match{Any: true}, Action: cedar.Match{Any: true},
			Resource: cedar.Match{Equals: ".opendistro_security"}},
	}
	policyEngine := cedar.NewTenantEngine(defaultPolicies)
	for issuer, t := range cfg.Tenants {
		var policies []cedar.Policy
		for i, pText := range t.CedarPolicies {
			p, err := cedar.ParsePolicy(fmt.Sprintf("%s-policy-%d", issuer, i), pText)
			if err != nil {
				log.Printf("Warning: invalid Cedar policy for tenant %s: %v", issuer, err)
				continue
			}
			policies = append(policies, p)
		}
		if len(policies) > 0 {
			policyEngine.AddTenant(issuer, policies)
		}
	}

	// Connection-pooled transport for upstream
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	if cfg.TLS.InsecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	engineURL, _ := url.Parse(cfg.Upstream.Engine)
	dashboardsURL, _ := url.Parse(cfg.Upstream.Dashboards)

	engineProxy := httputil.NewSingleHostReverseProxy(engineURL)
	engineProxy.Transport = transport
	engineProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		upstreamErrors.Add(1)
		log.Printf("upstream error: %v", err) // log internally, don't expose
		http.Error(w, `{"error":"upstream_error","message":"upstream unavailable"}`, http.StatusBadGateway)
	}

	dashboardsProxy := httputil.NewSingleHostReverseProxy(dashboardsURL)
	dashboardsProxy.Transport = transport
	dashboardsProxy.ErrorHandler = engineProxy.ErrorHandler

	mux := http.NewServeMux()

	// Issuer URL for discovery + token exchange
	issuerURL := "http://localhost" + cfg.Listen
	if cfg.TLS.Enabled {
		issuerURL = "https://localhost" + cfg.Listen
	}
	if envIssuer := os.Getenv("OAUTH4OS_ISSUER"); envIssuer != "" {
		issuerURL = envIssuer
	}

	// Token endpoints
	exchangeHandler := exchange.NewHandler(
		&exchange.JWTSubjectValidator{Validate: func(token string) (string, string, []string, error) {
			claims, err := validator.Validate(token)
			if err != nil {
				return "", "", nil, err
			}
			return claims.ClientID, claims.Issuer, claims.Scopes, nil
		}},
		&exchange.ManagerAdapter{CreateToken: func(clientID string, scopes []string) (string, string) {
			tok, refresh := tokenMgr.CreateTokenForClient(clientID, scopes)
			return tok.ID, refresh
		}},
		issuerURL,
	)
	mux.HandleFunc("POST /oauth/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.FormValue("grant_type") == exchange.GrantType {
			exchangeHandler.ServeHTTP(w, r)
			return
		}
		tokenMgr.IssueToken(w, r)
	})
	mux.HandleFunc("DELETE /oauth/token/{id}", tokenMgr.RevokeToken)
	mux.HandleFunc("GET /oauth/tokens", tokenMgr.ListTokens)
	mux.HandleFunc("GET /oauth/token/{id}", tokenMgr.GetToken)

	// RFC 7662 Token Introspection
	introAdapter := &introspect.ManagerAdapter{GetToken: tokenMgr.Lookup}
	introHandler := introspect.NewHandler(introAdapter)
	mux.Handle("POST /oauth/introspect", introHandler)

	// PKCE authorization code flow for browser clients
	pkceHandler := pkce.NewHandler(func(clientID string, scopes []string) (string, string) {
		tok, refresh := tokenMgr.CreateTokenForClient(clientID, scopes)
		return tok.ID, refresh
	})
	mux.HandleFunc("GET /oauth/authorize", pkceHandler.Authorize)
	mux.HandleFunc("POST /oauth/authorize/token", pkceHandler.Exchange)

	// Dynamic Client Registration (RFC 7591)
	regHandler := registration.NewHandler(tokenMgr.RegisterClient)
	mux.HandleFunc("POST /oauth/register", regHandler.Register)
	mux.HandleFunc("GET /oauth/register/{client_id}", regHandler.Get)

	// Admin API — runtime config management
	adminState := admin.NewState(cfg, mapper, policyEngine)
	adminState.Register(mux)

	// Health
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","version":"%s","uptime_seconds":%d}`,
			version, int(time.Since(startTime).Seconds()))
	})

	// OIDC Discovery
	var scopeNames []string
	for s := range cfg.ScopeMapping {
		scopeNames = append(scopeNames, s)
	}
	mux.HandleFunc("GET /.well-known/openid-configuration",
		discovery.Handler(discovery.Config{Issuer: issuerURL}, scopeNames))

	// Prometheus metrics
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "# HELP oauth4os_requests_total Total proxy requests\n")
		fmt.Fprintf(w, "# TYPE oauth4os_requests_total counter\n")
		fmt.Fprintf(w, "oauth4os_requests_total %d\n", requestsTotal.Load())
		fmt.Fprintf(w, "# HELP oauth4os_requests_active Currently active requests\n")
		fmt.Fprintf(w, "# TYPE oauth4os_requests_active gauge\n")
		fmt.Fprintf(w, "oauth4os_requests_active %d\n", requestsActive.Load())
		fmt.Fprintf(w, "# HELP oauth4os_requests_failed Failed requests\n")
		fmt.Fprintf(w, "# TYPE oauth4os_requests_failed counter\n")
		fmt.Fprintf(w, "oauth4os_requests_failed %d\n", requestsFailed.Load())
		fmt.Fprintf(w, "# HELP oauth4os_auth_success Successful authentications\n")
		fmt.Fprintf(w, "# TYPE oauth4os_auth_success counter\n")
		fmt.Fprintf(w, "oauth4os_auth_success %d\n", authSuccess.Load())
		fmt.Fprintf(w, "# HELP oauth4os_auth_failed Failed authentications\n")
		fmt.Fprintf(w, "# TYPE oauth4os_auth_failed counter\n")
		fmt.Fprintf(w, "oauth4os_auth_failed %d\n", authFailed.Load())
		fmt.Fprintf(w, "# HELP oauth4os_cedar_denied Cedar policy denials\n")
		fmt.Fprintf(w, "# TYPE oauth4os_cedar_denied counter\n")
		fmt.Fprintf(w, "oauth4os_cedar_denied %d\n", cedarDenied.Load())
		fmt.Fprintf(w, "# HELP oauth4os_rate_limited Rate limited requests\n")
		fmt.Fprintf(w, "# TYPE oauth4os_rate_limited counter\n")
		fmt.Fprintf(w, "oauth4os_rate_limited %d\n", rateLimited.Load())
		fmt.Fprintf(w, "# HELP oauth4os_upstream_errors Upstream connection errors\n")
		fmt.Fprintf(w, "# TYPE oauth4os_upstream_errors counter\n")
		fmt.Fprintf(w, "oauth4os_upstream_errors %d\n", upstreamErrors.Load())
		fmt.Fprintf(w, "# HELP oauth4os_uptime_seconds Proxy uptime\n")
		fmt.Fprintf(w, "# TYPE oauth4os_uptime_seconds gauge\n")
		fmt.Fprintf(w, "oauth4os_uptime_seconds %d\n", int(time.Since(startTime).Seconds()))
	})

	// Proxy all other requests
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requestsTotal.Add(1)
		requestsActive.Add(1)
		defer requestsActive.Add(-1)

		// Inject X-Request-ID for tracing
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			b := make([]byte, 16)
			rand.Read(b)
			reqID = fmt.Sprintf("%x", b)
			r.Header.Set("X-Request-ID", reqID)
		}
		w.Header().Set("X-Request-ID", reqID)

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			// Strip proxy-trust headers on unauthenticated path — prevents impersonation
			r.Header.Del("X-Proxy-User")
			r.Header.Del("X-Proxy-Roles")
			r.Header.Del("X-Proxy-Scopes")
			r.Header.Del("Cookie")
			r.Header.Del("Proxy-Authorization")
			engineProxy.ServeHTTP(w, r)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		// Span: JWT validation
		ctx, jwtSpan := tracer.StartSpan(r.Context(), string(tracing.SpanJWT), nil)
		r = r.WithContext(ctx)
		claims, err := validator.Validate(tokenStr)
		if err != nil {
			tracer.EndSpan(jwtSpan, "error")
			authFailed.Add(1)
			requestsFailed.Add(1)
			http.Error(w, `{"error":"invalid_token"}`, http.StatusUnauthorized)
			return
		}
		tracer.EndSpan(jwtSpan, "ok")
		authSuccess.Add(1)

		// Span: scope mapping
		ctx, scopeSpan := tracer.StartSpan(r.Context(), string(tracing.SpanScope), map[string]string{"issuer": claims.Issuer})
		r = r.WithContext(ctx)
		roles := mapper.MapForIssuer(claims.Issuer, claims.Scopes)
		if len(roles) == 0 {
			tracer.EndSpan(scopeSpan, "error")
			requestsFailed.Add(1)
			http.Error(w, `{"error":"insufficient_scope"}`, http.StatusForbidden)
			return
		}
		tracer.EndSpan(scopeSpan, "ok")

		// Span: Cedar policy evaluation (tenant-scoped)
		index := extractIndex(r.URL.Path)
		ctx, cedarSpan := tracer.StartSpan(r.Context(), string(tracing.SpanCedar), map[string]string{"index": index})
		r = r.WithContext(ctx)
		decision := policyEngine.Evaluate(claims.Issuer, cedar.Request{
			Principal: map[string]string{"sub": claims.ClientID, "scope": strings.Join(claims.Scopes, ",")},
			Action:    r.Method,
			Resource:  map[string]string{"index": index, "path": r.URL.Path},
		})
		if !decision.Allowed {
			tracer.EndSpan(cedarSpan, "error")
			cedarDenied.Add(1)
			requestsFailed.Add(1)
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		tracer.EndSpan(cedarSpan, "ok")

		r.Header.Del("Authorization")
		r.Header.Del("Cookie")
		r.Header.Del("Proxy-Authorization")
		r.Header.Del("X-Forwarded-For")  // proxy sets its own
		r.Header.Del("X-Forwarded-Host")
		r.Header.Set("X-Proxy-User", claims.ClientID)
		r.Header.Set("X-Proxy-Roles", strings.Join(roles, ","))
		r.Header.Set("X-Proxy-Scopes", strings.Join(claims.Scopes, ","))

		auditor.Log(claims.ClientID, claims.Scopes, r.Method, r.URL.Path)

		// Span: upstream forwarding
		ctx, upSpan := tracer.StartSpan(r.Context(), string(tracing.SpanUpstream), map[string]string{"target": r.URL.Path})
		r = r.WithContext(ctx)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			dashboardsProxy.ServeHTTP(w, r)
		} else {
			engineProxy.ServeHTTP(w, r)
		}
		tracer.EndSpan(upSpan, "ok")
	})

	addr := cfg.Listen
	if addr == "" {
		addr = ":8443"
	}

	// Rate limiting middleware wraps the mux
	rateLimited := limiter.Middleware(mux, func(r *http.Request) (string, []string) {
		// Extract client from X-Proxy-User header (set by auth handler)
		// For unauthenticated requests, use remote IP
		if user := r.Header.Get("X-Proxy-User"); user != "" {
			scopes := strings.Split(r.Header.Get("X-Proxy-Scopes"), ",")
			return user, scopes
		}
		// Rate limit by IP for token endpoint abuse prevention
		if strings.HasPrefix(r.URL.Path, "/oauth/token") {
			return r.RemoteAddr, nil
		}
		return "", nil
	})

	// Tracing middleware (outermost) → rate limiting → mux
	traced := tracing.Middleware(rateLimited, tracer)

	srv := &http.Server{
		Addr:         addr,
		Handler:      traced,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Printf("Received %v, shutting down gracefully...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Shutdown error: %v", err)
		}
	}()

	log.Printf("oauth4os v%s listening on %s (tls=%v)", version, addr, cfg.TLS.Enabled)
	log.Printf("  Engine:     %s", cfg.Upstream.Engine)
	log.Printf("  Dashboards: %s", cfg.Upstream.Dashboards)

	if cfg.TLS.Enabled && cfg.TLS.CertFile != "" && cfg.TLS.KeyFile != "" {
		err = srv.ListenAndServeTLS(cfg.TLS.CertFile, cfg.TLS.KeyFile)
	} else {
		err = srv.ListenAndServe()
	}
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
	log.Println("Server stopped")
}

func extractIndex(path string) string {
	path = strings.TrimPrefix(path, "/")
	if idx := strings.IndexByte(path, '/'); idx > 0 {
		return path[:idx]
	}
	return path
}
