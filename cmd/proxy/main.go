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
	"github.com/seraphjiang/oauth4os/internal/mtls"
	"github.com/seraphjiang/oauth4os/internal/webhook"
)

const version = "0.1.0"

//go:embed landing.html
var landingPage string

//go:embed openapi.yaml
var openapiSpec string

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
	_ = backup.NewHandler(
		func() *config.Config { return cfg },
		func() []backup.ClientEntry { return nil },
		func(c *config.Config) { *cfg = *c },
	)

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
	_ = keys // available for token signing in future

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
			tracer.EndSpan(jwtSpan, "error")
			authFailed.Add(1)
			requestsFailed.Add(1)
			http.Error(w, `{"error":"invalid_token"}`, http.StatusUnauthorized)
			return
		}
		tracer.EndSpan(jwtSpan, "ok")
		authSuccess.Add(1)

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

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		logger.Info("shutting down", "signal", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("shutdown error", "error", err)
		}
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

const swaggerPage = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>oauth4os — API Docs</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"/>
<style>
body{margin:0;background:#1a1a2e}
.swagger-ui .topbar{display:none}
.swagger-ui{filter:invert(88%) hue-rotate(180deg)}
.swagger-ui .model-box{background:rgba(0,0,0,.1)}
</style>
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
SwaggerUIBundle({
  url:"/developer/openapi.yaml",
  dom_id:"#swagger-ui",
  deepLinking:true,
  tryItOutEnabled:true,
  defaultModelsExpandDepth:-1,
  docExpansion:"list",
  filter:true,
  requestSnippetsEnabled:true
});
</script>
</body>
</html>`

const developerAnalyticsHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Developer Analytics — oauth4os</title>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4"></script>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#0d1117;color:#e6edf3;padding:24px}
h1{font-size:22px;margin-bottom:4px}
.sub{color:#8b949e;margin-bottom:24px;font-size:14px}
.stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:12px;margin-bottom:24px}
.stat{background:#161b22;border:1px solid #30363d;border-radius:10px;padding:16px;text-align:center}
.stat .val{font-size:28px;font-weight:700;color:#58a6ff}
.stat .lbl{font-size:12px;color:#8b949e;margin-top:4px}
.grid{display:grid;grid-template-columns:1fr 1fr;gap:16px;margin-bottom:24px}
@media(max-width:768px){.grid{grid-template-columns:1fr}}
.card{background:#161b22;border:1px solid #30363d;border-radius:10px;padding:16px}
.card h3{font-size:14px;color:#8b949e;margin-bottom:12px}
canvas{max-height:220px}
table{width:100%;border-collapse:collapse;font-size:13px}
th{text-align:left;color:#8b949e;padding:8px 6px;border-bottom:1px solid #30363d}
td{padding:8px 6px;border-bottom:1px solid #21262d}
.bar{height:6px;background:#238636;border-radius:3px;display:inline-block;vertical-align:middle}
.err{color:#f85149}.ok{color:#3fb950}
.refresh{color:#8b949e;font-size:12px;float:right}
</style>
</head>
<body>
<h1>🔐 Developer Analytics <span class="refresh" id="timer">Refreshing in 5s</span></h1>
<p class="sub">Per-client request metrics, scope usage, and index access patterns.</p>

<div class="stats" id="summary"></div>

<div class="grid">
  <div class="card"><h3>Requests by Client</h3><canvas id="clientChart"></canvas></div>
  <div class="card"><h3>Scope Distribution</h3><canvas id="scopeChart"></canvas></div>
</div>

<div class="grid">
  <div class="card"><h3>Top Indices</h3><canvas id="indexChart"></canvas></div>
  <div class="card">
    <h3>Client Details</h3>
    <table><thead><tr><th>Client</th><th>Requests</th><th>Last Active</th><th></th></tr></thead>
    <tbody id="clientTable"></tbody></table>
  </div>
</div>

<div class="card" style="margin-top:16px">
  <h3>Scope × Client Matrix</h3>
  <table><thead><tr><th>Scope</th><th>Requests</th><th></th></tr></thead>
  <tbody id="scopeTable"></tbody></table>
</div>

<script>
const colors=['#58a6ff','#3fb950','#d29922','#f85149','#bc8cff','#39d2c0','#ff7b72','#79c0ff'];
let clientChart,scopeChart,indexChart;

function initChart(id,type,label){
  return new Chart(document.getElementById(id),{type,data:{labels:[],datasets:[{label,data:[],backgroundColor:colors,borderWidth:0}]},
    options:{responsive:true,plugins:{legend:{display:type==='doughnut',labels:{color:'#8b949e'}}},scales:type==='bar'?{x:{ticks:{color:'#8b949e'}},y:{ticks:{color:'#8b949e'},beginAtZero:true}}:undefined}});
}

function updateChart(chart,labels,data){
  chart.data.labels=labels;chart.data.datasets[0].data=data;chart.update('none');
}

function ago(ts){
  if(!ts)return'—';
  const s=Math.floor((Date.now()-new Date(ts))/1000);
  if(s<60)return s+'s ago';if(s<3600)return Math.floor(s/60)+'m ago';
  return Math.floor(s/3600)+'h ago';
}

async function refresh(){
  try{
    const [aResp,tResp]=await Promise.all([fetch('/admin/analytics'),fetch('/oauth/tokens')]);
    const a=await aResp.json(), tokens=await tResp.json();
    const totalReqs=a.top_clients?.reduce((s,c)=>s+c.requests,0)||0;
    const totalClients=a.top_clients?.length||0;
    const totalScopes=a.scope_distribution?.length||0;
    const totalIndices=a.top_indices?.length||0;
    const activeTokens=Array.isArray(tokens)?tokens.length:0;

    document.getElementById('summary').innerHTML=
      [['Total Requests',totalReqs],['Active Clients',totalClients],['Scopes Used',totalScopes],
       ['Indices Accessed',totalIndices],['Active Tokens',activeTokens]]
      .map(([l,v])=>'<div class="stat"><div class="val">'+v+'</div><div class="lbl">'+l+'</div></div>').join('');

    const cl=a.top_clients||[];
    updateChart(clientChart,cl.map(c=>c.client_id),cl.map(c=>c.requests));

    const sc=a.scope_distribution||[];
    updateChart(scopeChart,sc.map(s=>s.name),sc.map(s=>s.count));

    const ix=a.top_indices||[];
    updateChart(indexChart,ix.map(i=>i.name),ix.map(i=>i.count));

    const maxReq=Math.max(...cl.map(c=>c.requests),1);
    document.getElementById('clientTable').innerHTML=cl.map(c=>
      '<tr><td>'+c.client_id+'</td><td>'+c.requests+'</td><td>'+ago(c.last_seen)+
      '</td><td><span class="bar" style="width:'+Math.round(c.requests/maxReq*80)+'px"></span></td></tr>').join('');

    const maxSc=Math.max(...sc.map(s=>s.count),1);
    document.getElementById('scopeTable').innerHTML=sc.map(s=>
      '<tr><td>'+s.name+'</td><td>'+s.count+'</td><td><span class="bar" style="width:'+
      Math.round(s.count/maxSc*120)+'px"></span></td></tr>').join('');
  }catch(e){document.getElementById('summary').innerHTML='<div class="stat"><div class="val err">Error</div><div class="lbl">'+e.message+'</div></div>';}
}

clientChart=initChart('clientChart','bar','Requests');
scopeChart=initChart('scopeChart','doughnut','Scopes');
indexChart=initChart('indexChart','bar','Requests');
refresh();
setInterval(refresh,5000);
let cd=5;setInterval(()=>{cd--;if(cd<=0)cd=5;document.getElementById('timer').textContent='Refreshing in '+cd+'s';},1000);
</script>
</body>
</html>`

const installScript = `#!/bin/bash
set -e

PROXY_URL="${OAUTH4OS_URL:-https://f5cmk2hxwx.us-west-2.awsapprunner.com}"
INSTALL_DIR="${HOME}/.local/bin"
SCRIPT="${INSTALL_DIR}/oauth4os-demo"

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux|darwin) ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

echo "Installing oauth4os-demo CLI..."
mkdir -p "$INSTALL_DIR"

cat > "$SCRIPT" << 'WRAPPER'
#!/bin/bash
set -e
PROXY="${OAUTH4OS_URL:-https://f5cmk2hxwx.us-west-2.awsapprunner.com}"
TOKEN_FILE="${HOME}/.oauth4os-token"

cmd_login() {
  echo "Registering demo client..."
  REG=$(curl -sf "$PROXY/oauth/register" -d '{"client_name":"cli-demo","scope":"read:logs-* admin"}' -H 'Content-Type: application/json')
  CLIENT_ID=$(echo "$REG" | grep -o '"client_id":"[^"]*"' | cut -d'"' -f4)
  CLIENT_SECRET=$(echo "$REG" | grep -o '"client_secret":"[^"]*"' | cut -d'"' -f4)
  echo "Getting token..."
  TOK=$(curl -sf "$PROXY/oauth/token" -d "grant_type=client_credentials&client_id=${CLIENT_ID}&client_secret=${CLIENT_SECRET}&scope=read:logs-*")
  ACCESS=$(echo "$TOK" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
  echo "$ACCESS" > "$TOKEN_FILE"
  echo "✅ Logged in. Token cached at $TOKEN_FILE"
}

cmd_search() {
  [ ! -f "$TOKEN_FILE" ] && echo "Run: oauth4os-demo login" && exit 1
  TOKEN=$(cat "$TOKEN_FILE")
  QUERY="${1:-*}"
  curl -sf "$PROXY/logs-demo/_search" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"query\":{\"query_string\":{\"query\":\"$QUERY\"}},\"size\":10,\"sort\":[{\"@timestamp\":{\"order\":\"desc\"}}]}" | \
    python3 -m json.tool 2>/dev/null || cat
}

cmd_services() {
  [ ! -f "$TOKEN_FILE" ] && echo "Run: oauth4os-demo login" && exit 1
  TOKEN=$(cat "$TOKEN_FILE")
  curl -sf "$PROXY/logs-demo/_search" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"size":0,"aggs":{"services":{"terms":{"field":"service.keyword","size":20}}}}' | \
    python3 -m json.tool 2>/dev/null || cat
}

cmd_health() {
  curl -sf "$PROXY/health" | python3 -m json.tool 2>/dev/null || curl -sf "$PROXY/health"
}

case "${1:-help}" in
  login)    cmd_login ;;
  search)   cmd_search "$2" ;;
  services) cmd_services ;;
  health)   cmd_health ;;
  *)        echo "Usage: oauth4os-demo <login|search|services|health>"
            echo ""
            echo "  login              Register + get token"
            echo "  search [QUERY]     Search logs (default: *)"
            echo "  services           List services in logs"
            echo "  health             Check proxy health"
            ;;
esac
WRAPPER

chmod +x "$SCRIPT"

# Check PATH
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) echo "Add to PATH: export PATH=\"\$HOME/.local/bin:\$PATH\"" ;;
esac

echo "✅ Installed: $SCRIPT"
echo ""
echo "Quick start:"
echo "  oauth4os-demo login"
echo "  oauth4os-demo search 'level:ERROR'"
echo "  oauth4os-demo services"
`
