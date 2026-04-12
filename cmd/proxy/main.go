package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/json"
	"encoding/pem"
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
	"github.com/seraphjiang/oauth4os/internal/analytics"
	"github.com/seraphjiang/oauth4os/internal/audit"
	"github.com/seraphjiang/oauth4os/internal/cedar"
	"github.com/seraphjiang/oauth4os/internal/config"
	"github.com/seraphjiang/oauth4os/internal/discovery"
	"github.com/seraphjiang/oauth4os/internal/exchange"
	"github.com/seraphjiang/oauth4os/internal/federation"
	"github.com/seraphjiang/oauth4os/internal/introspect"
	"github.com/seraphjiang/oauth4os/internal/ipfilter"
	"github.com/seraphjiang/oauth4os/internal/jwt"
	"github.com/seraphjiang/oauth4os/internal/keyring"
	"github.com/seraphjiang/oauth4os/internal/logging"
	"github.com/seraphjiang/oauth4os/internal/pkce"
	"github.com/seraphjiang/oauth4os/internal/ratelimit"
	"github.com/seraphjiang/oauth4os/internal/registration"
	"github.com/seraphjiang/oauth4os/internal/scope"
	"github.com/seraphjiang/oauth4os/internal/session"
	"github.com/seraphjiang/oauth4os/internal/sigv4"
	"github.com/seraphjiang/oauth4os/internal/token"
	"github.com/seraphjiang/oauth4os/internal/tracing"
	"github.com/seraphjiang/oauth4os/internal/backup"
	"github.com/seraphjiang/oauth4os/internal/demo"
	"github.com/seraphjiang/oauth4os/internal/tokenui"
	"github.com/seraphjiang/oauth4os/internal/mtls"
	"github.com/seraphjiang/oauth4os/internal/webhook"
)

const version = "0.2.0"

//go:embed landing.html
var landingPage string

//go:embed openapi.yaml
var openapiSpec string

//go:embed install.sh
var installScript string

//go:embed oauth4os-demo.sh
var demoCLIScript string

//go:embed demo-app.html
var demoAppHTML string

//go:embed swagger.html
var swaggerPage string

//go:embed analytics.html
var developerAnalyticsHTML string

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
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	validator := jwt.NewValidator(cfg.Providers)
	mapper := scope.NewMultiTenantMapper(cfg.ScopeMapping, cfg.Tenants)
	tokenMgr := token.NewManager()

	// Structured logger — replaces log.Printf
	logger := logging.New(os.Stdout, "info")

	auditor := audit.NewJSONAuditor(os.Stdout)
	auditStore, _ := audit.NewMemoryStore(10000, "")
	auditor.WithStore(auditStore)

	sessionMgr := session.New(map[string]int{"*": 100})

	analyticsTracker := analytics.New()

	// Load persisted clients
	clientStore, err := token.NewClientStore("data/clients.json", tokenMgr)
	if err != nil {
		logger.Info("client store not loaded", "error", err)
		clientStore = nil
	}

	// Pre-register demo clients (idempotent — no redirect URI restriction for demo)
	tokenMgr.RegisterClient("demo-webapp", "", []string{"read:logs"}, nil)
	tokenMgr.RegisterClient("demo-cli", "", []string{"read:logs"}, nil)

	// IP filter — per-client allowlist/denylist
	var ipRules *ipfilter.Rules
	if cfg.IPFilter.Global != nil || len(cfg.IPFilter.Clients) > 0 {
		ipCfg := ipfilter.Config{}
		if cfg.IPFilter.Global != nil {
			ipCfg.Global = &ipfilter.FilterConfig{Allow: cfg.IPFilter.Global.Allow, Deny: cfg.IPFilter.Global.Deny}
		}
		if len(cfg.IPFilter.Clients) > 0 {
			ipCfg.Clients = make(map[string]*ipfilter.FilterConfig)
			for k, v := range cfg.IPFilter.Clients {
				ipCfg.Clients[k] = &ipfilter.FilterConfig{Allow: v.Allow, Deny: v.Deny}
			}
		}
		var err error
		ipRules, err = ipfilter.New(ipCfg)
		if err != nil {
			logger.Fatal("invalid ip_filter config", "error", err)
		}
		logger.Info("IP filter enabled", "clients", len(cfg.IPFilter.Clients))
	}
	limiter := ratelimit.New(cfg.RateLimits, 600)

	// Tracing — stdout in dev, noop if OAUTH4OS_TRACING=off
	var tracer tracing.TracerIface
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
				logger.Warn("Warning: invalid Cedar policy for tenant %s: %v", issuer, err)
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

	// SigV4 signing for AOSS / managed OpenSearch with IAM
	var upstreamTransport http.RoundTripper = transport
	if cfg.Upstream.SigV4 != nil {
		svc := cfg.Upstream.SigV4.Service
		if svc == "" {
			svc = "aoss"
		}
		upstreamTransport = sigv4.New(transport, cfg.Upstream.SigV4.Region, svc)
		logger.Info("SigV4 signing enabled", "region", cfg.Upstream.SigV4.Region, "service", svc)
	}

	engineURL, _ := url.Parse(cfg.Upstream.Engine)
	dashboardsURL, _ := url.Parse(cfg.Upstream.Dashboards)

	engineProxy := httputil.NewSingleHostReverseProxy(engineURL)
	engineProxy.Transport = upstreamTransport
	engineProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		upstreamErrors.Add(1)
		logger.Error("upstream error", "error", err) // log internally, don't expose
		http.Error(w, `{"error":"upstream_error","message":"upstream unavailable"}`, http.StatusBadGateway)
	}

	dashboardsProxy := httputil.NewSingleHostReverseProxy(dashboardsURL)
	dashboardsProxy.Transport = upstreamTransport
	dashboardsProxy.ErrorHandler = engineProxy.ErrorHandler

	// Multi-cluster federation router
	var fedRouter *federation.Router
	if len(cfg.Clusters) > 0 {
		var clusters []federation.Cluster
		for name, c := range cfg.Clusters {
			clusters = append(clusters, federation.Cluster{Name: name, URL: c.Engine, Indices: c.Prefixes})
		}
		fedRouter = federation.New(clusters, transport)
		logger.Info("  Federation: %d clusters", len(cfg.Clusters))
	}

	// Webhook external authorizer (optional)
	var webhookAuth *webhook.Authorizer
	if cfg.Webhook.URL != "" {
		webhookAuth = webhook.NewAuthorizer(webhook.Config{
			URL:      cfg.Webhook.URL,
			Timeout:  cfg.Webhook.Timeout,
			Headers:  cfg.Webhook.Headers,
			FailOpen: cfg.Webhook.FailOpen,
		})
		logger.Info("  Webhook: %s", cfg.Webhook.URL)
	}

	// mTLS client auth (optional)
	var mtlsMap *mtls.ClientMap
	if len(cfg.MTLS.Clients) > 0 {
		entries := make(map[string]*mtls.ClientEntry)
		for cn, c := range cfg.MTLS.Clients {
			entries[cn] = &mtls.ClientEntry{ClientID: c.ClientID, Scopes: c.Scopes}
		}
		mtlsMap = mtls.NewClientMap(entries)
		logger.Info("  mTLS: %d client certs", len(cfg.MTLS.Clients))
	}

	// Backup handler
	backupHandler := backup.NewHandler(
		func() *config.Config { return cfg },
		func() []backup.ClientEntry { return nil },
		func(c *config.Config) { *cfg = *c },
	)

	mux := http.NewServeMux()

	// Register backup routes
	backupHandler.Register(mux)

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
		r.Body = http.MaxBytesReader(w, r.Body, 1<<16) // 64KB max
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
	}, tokenMgr.ValidateRedirectURI)
	mux.HandleFunc("GET /oauth/authorize", pkceHandler.Authorize)
	mux.HandleFunc("POST /oauth/consent", pkceHandler.Consent)
	mux.HandleFunc("POST /oauth/authorize/token", pkceHandler.Exchange)

	// Dynamic Client Registration (RFC 7591)
	// Wrap RegisterClient to persist after mutation
	registerAndSave := func(id, secret string, scopes, redirectURIs []string) {
		tokenMgr.RegisterClient(id, secret, scopes, redirectURIs)
		if clientStore != nil {
			clientStore.Save(tokenMgr)
		}
	}
	regHandler := registration.NewHandler(registerAndSave, nil)
	saveClients := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			next(w, r)
			if clientStore != nil {
				clientStore.Save(tokenMgr)
			}
		}
	}
	mux.HandleFunc("POST /oauth/register", regHandler.Register)
	mux.HandleFunc("GET /oauth/register", regHandler.List)
	mux.HandleFunc("GET /oauth/register/{client_id}", regHandler.Get)
	mux.HandleFunc("PUT /oauth/register/{client_id}", saveClients(regHandler.Update))
	mux.HandleFunc("DELETE /oauth/register/{client_id}", saveClients(regHandler.Delete))
	mux.HandleFunc("POST /oauth/register/{client_id}/rotate", saveClients(regHandler.RotateSecret))

	// Admin API — runtime config management
	adminState := admin.NewState(cfg, mapper, policyEngine)
	adminState.Register(mux)

	// Install script — curl -sL <proxy>/install.sh | bash
	mux.HandleFunc("GET /install.sh", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Cache-Control", "no-cache")
		fmt.Fprint(w, installScript)
	})

	// CLI demo script — downloaded by install.sh
	mux.HandleFunc("GET /scripts/oauth4os-demo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Cache-Control", "no-cache")
		fmt.Fprint(w, demoCLIScript)
	})

	// Demo web app — log viewer with PKCE login
	mux.HandleFunc("GET /demo/app", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, demoAppHTML)
	})

	// Health
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","version":"%s","uptime_seconds":%d}`,
			version, int(time.Since(startTime).Seconds()))
	})

	// Deep health — checks upstream, JWKS, TLS cert
	healthClient := &http.Client{Timeout: 5 * time.Second, Transport: transport}
	mux.HandleFunc("GET /health/deep", func(w http.ResponseWriter, r *http.Request) {
		type check struct {
			Status string `json:"status"`
			Detail string `json:"detail,omitempty"`
		}
		result := map[string]check{}
		overall := "ok"

		// Check upstream OpenSearch
		resp, err := healthClient.Get(cfg.Upstream.Engine)
		if err != nil {
			result["upstream"] = check{"error", err.Error()}
			overall = "degraded"
		} else {
			resp.Body.Close()
			result["upstream"] = check{"ok", fmt.Sprintf("status=%d", resp.StatusCode)}
		}

		// Check JWKS for each provider
		for _, p := range cfg.Providers {
			uri := p.JWKSURI
			if uri == "" || uri == "auto" {
				uri = strings.TrimSuffix(p.Issuer, "/") + "/.well-known/openid-configuration"
			}
			resp, err := healthClient.Get(uri)
			if err != nil {
				result["jwks_"+p.Name] = check{"error", err.Error()}
				overall = "degraded"
			} else {
				resp.Body.Close()
				result["jwks_"+p.Name] = check{"ok", fmt.Sprintf("status=%d", resp.StatusCode)}
			}
		}

		// Check TLS cert expiry if enabled
		if cfg.TLS.Enabled && cfg.TLS.CertFile != "" {
			if certPEM, err := os.ReadFile(cfg.TLS.CertFile); err == nil {
				block, _ := pem.Decode(certPEM)
				if block != nil {
					if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
						days := int(time.Until(cert.NotAfter).Hours() / 24)
						status := "ok"
						if days < 30 {
							status = "warning"
							overall = "degraded"
						}
						result["tls_cert"] = check{status, fmt.Sprintf("expires_in_days=%d", days)}
					}
				}
			}
		}

		result["overall"] = check{overall, ""}
		w.Header().Set("Content-Type", "application/json")
		if overall != "ok" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(result)
	})

	// OIDC Discovery
	var scopeNames []string
	for s := range cfg.ScopeMapping {
		scopeNames = append(scopeNames, s)
	}
	mux.HandleFunc("GET /.well-known/openid-configuration",
		discovery.Handler(discovery.Config{Issuer: issuerURL}, scopeNames))

	// Key rotation + JWKS endpoint
	rotateInterval := 24 * time.Hour
	if v := os.Getenv("OAUTH4OS_KEY_ROTATE_HOURS"); v != "" {
		if h, err := time.ParseDuration(v + "h"); err == nil {
			rotateInterval = h
		}
	}
	keys, err := keyring.New(2048, rotateInterval)
	if err != nil {
		log.Fatalf("Failed to initialize keyring: %v", err)
	}
	defer keys.Stop()
	mux.HandleFunc("GET /.well-known/jwks.json", keys.JWKSHandler())

	// Prometheus metrics
	mux.HandleFunc("GET /admin/audit", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter := audit.QueryFilter{
			ClientID: q.Get("client_id"),
			Event:    q.Get("event"),
			Limit:    100,
		}
		if since := q.Get("since"); since != "" {
			if t, err := time.Parse(time.RFC3339, since); err == nil {
				filter.Since = t
			}
		}
		entries, _ := auditor.Query(filter)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	})

	mux.HandleFunc("GET /admin/analytics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(analyticsTracker.Snapshot())
	})

	mux.HandleFunc("GET /admin/clusters", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if fedRouter != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"clusters": fedRouter.ClusterNames(), "mode": "federation"})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{"clusters": []string{cfg.Upstream.Engine}, "mode": "single"})
		}
	})

	mux.HandleFunc("GET /admin/sessions", func(w http.ResponseWriter, r *http.Request) {
		clientID := r.URL.Query().Get("client_id")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sessionMgr.List(clientID))
	})

	mux.HandleFunc("GET /developer/analytics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, developerAnalyticsHTML)
	})

	mux.HandleFunc("DELETE /admin/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		sessionMgr.Remove(r.PathValue("id"))
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("POST /admin/sessions/logout", func(w http.ResponseWriter, r *http.Request) {
		clientID := r.URL.Query().Get("client_id")
		if clientID == "" {
			http.Error(w, `{"error":"client_id required"}`, http.StatusBadRequest)
			return
		}
		removed := sessionMgr.ForceLogout(clientID)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"removed":%d}`, removed)
	})

	// Backup endpoints
	//backupHandler.Register(mux) // handled by admin API

	// Demo web app (log viewer with PKCE login)
	demoApp := demo.NewHandler(issuerURL, "demo-app")
	demoApp.Register(mux)

	// Token inspector page
	tokenInspector := tokenui.New(issuerURL)
	tokenInspector.Register(mux)

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

	// Developer docs — Swagger UI
	mux.HandleFunc("GET /developer/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Fprint(w, openapiSpec)
	})
	mux.HandleFunc("GET /developer/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, swaggerPage)
	})

	// Serve landing page at root
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, landingPage)
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
			// mTLS client cert auth (alternative to Bearer token)
			if mtlsMap != nil && r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
				entry, err := mtlsMap.Identify(r.TLS.PeerCertificates[0])
				if err == nil {
					authSuccess.Add(1)
					r.Header.Set("X-Proxy-User", entry.ClientID)
					r.Header.Set("X-Proxy-Roles", strings.Join(entry.Scopes, ","))
					engineProxy.ServeHTTP(w, r)
					return
				}
			}
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
			// Fallback: check self-issued tokens
			if clientID, scopes, _, expiresAt, revoked, ok := tokenMgr.Lookup(tokenStr); ok && !revoked && time.Now().Before(expiresAt) {
				tracer.EndSpan(jwtSpan, "ok")
				authSuccess.Add(1)
				claims = &jwt.Claims{Subject: clientID, Scopes: scopes, Issuer: "oauth4os"}
			} else {
				tracer.EndSpan(jwtSpan, "error")
				authFailed.Add(1)
				requestsFailed.Add(1)
				http.Error(w, `{"error":"invalid_token"}`, http.StatusUnauthorized)
				return
			}
		} else {
			tracer.EndSpan(jwtSpan, "ok")
			authSuccess.Add(1)
		}

		// IP filter check
		if ipRules != nil {
			if err := ipRules.Check(claims.ClientID, r.RemoteAddr); err != nil {
				requestsFailed.Add(1)
				http.Error(w, `{"error":"ip_denied"}`, http.StatusForbidden)
				return
			}
		}

		// Session tracking — use token ID as session key
		if !sessionMgr.Create(tokenStr[:16], claims.ClientID, tokenStr[:16], r.RemoteAddr) {
			requestsFailed.Add(1)
			http.Error(w, `{"error":"session_limit_exceeded"}`, http.StatusTooManyRequests)
			return
		}
		sessionMgr.Touch(tokenStr[:16])
		tokenMgr.TouchToken(tokenStr, 1*time.Hour) // sliding window

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

		// Webhook external authorization (optional)
		if webhookAuth != nil {
			if err := webhookAuth.Check(webhook.Request{
				ClientID: claims.ClientID,
				Subject:  claims.Subject,
				Scopes:   claims.Scopes,
				Action:   r.Method,
				Resource: r.URL.Path,
				IP:       r.RemoteAddr,
			}); err != nil {
				requestsFailed.Add(1)
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
		}

		r.Header.Del("Authorization")
		r.Header.Del("Cookie")
		r.Header.Del("Proxy-Authorization")
		r.Header.Del("X-Forwarded-For")  // proxy sets its own
		r.Header.Del("X-Forwarded-Host")
		r.Header.Set("X-Proxy-User", claims.ClientID)
		r.Header.Set("X-Proxy-Roles", strings.Join(roles, ","))
		r.Header.Set("X-Proxy-Scopes", strings.Join(claims.Scopes, ","))

		auditor.Log(claims.ClientID, claims.Scopes, r.Method, r.URL.Path)
		analyticsTracker.Record(claims.ClientID, claims.Scopes, index)

		// Span: upstream forwarding
		ctx, upSpan := tracer.StartSpan(r.Context(), string(tracing.SpanUpstream), map[string]string{"target": r.URL.Path})
		r = r.WithContext(ctx)
		if fedRouter != nil {
			if proxy := fedRouter.Route(r); proxy != nil {
				proxy.ServeHTTP(w, r)
			} else {
				engineProxy.ServeHTTP(w, r)
			}
		} else if strings.HasPrefix(r.URL.Path, "/api/") {
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

	// Graceful shutdown — drain connections, flush state
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		logger.Info("shutting down", "signal", sig)

		// 1. Drain active connections (30s timeout)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("shutdown error", "error", err)
		}

		// 2. Save client state
		if clientStore != nil {
			if err := clientStore.Save(tokenMgr); err != nil {
				logger.Error("failed to save clients on shutdown", "error", err)
			} else {
				logger.Info("client state saved")
			}
		}

		// 3. Flush audit logs
		if auditStore != nil {
			auditStore.Close()
			logger.Info("audit store flushed")
		}

		logger.Info("shutdown complete")
	}()

	logger.Info("listening", "version", version, "addr", addr, "tls", cfg.TLS.Enabled)
	logger.Info("upstream", "engine", cfg.Upstream.Engine)
	logger.Info("upstream", "dashboards", cfg.Upstream.Dashboards)

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

