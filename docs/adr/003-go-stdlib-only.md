# ADR-003: Go Standard Library Only, No Web Framework

## Status
Accepted

## Context
Choosing between Go web frameworks (Gin, Echo, Chi) or the standard library for the HTTP proxy.

## Decision
Use only Go's `net/http`, `net/http/httputil`, and `crypto` packages. No external web framework.

## Rationale
- **Go 1.22 routing** — `mux.HandleFunc("GET /path", handler)` eliminates the main reason for frameworks
- **Fewer dependencies** — smaller attack surface, no supply chain risk
- **Proxy-native** — `httputil.ReverseProxy` is purpose-built for our use case
- **Transparent** — no magic middleware chains, every stage is explicit

## Consequences
- No automatic request binding/validation (we use `json.NewDecoder` directly)
- No built-in middleware chaining (we compose manually: tracing → rate limit → handler)
- Slightly more boilerplate for error responses (solved with `writeErr` helper)
