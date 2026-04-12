// Package admin provides a REST API for managing proxy configuration at runtime.
// Endpoints: scope mappings, Cedar policies, providers, tenants.
package admin

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/seraphjiang/oauth4os/internal/cedar"
	"github.com/seraphjiang/oauth4os/internal/config"
	"github.com/seraphjiang/oauth4os/internal/scope"
)

// State holds mutable proxy configuration. Thread-safe.
type State struct {
	mu        sync.RWMutex
	cfg       *config.Config
	mapper    *scope.Mapper
	cedarEng  *cedar.TenantEngine
}

// NewState wraps config and live components for admin mutations.
func NewState(cfg *config.Config, mapper *scope.Mapper, cedarEng *cedar.TenantEngine) *State {
	return &State{cfg: cfg, mapper: mapper, cedarEng: cedarEng}
}

// Register mounts admin API routes on the given mux.
func (s *State) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/scope-mappings", s.listScopeMappings)
	mux.HandleFunc("PUT /admin/scope-mappings", s.updateScopeMappings)
	mux.HandleFunc("GET /admin/providers", s.listProviders)
	mux.HandleFunc("POST /admin/providers", s.addProvider)
	mux.HandleFunc("DELETE /admin/providers/{name}", s.removeProvider)
	mux.HandleFunc("GET /admin/tenants", s.listTenants)
	mux.HandleFunc("PUT /admin/tenants/{issuer}", s.updateTenant)
	mux.HandleFunc("DELETE /admin/tenants/{issuer}", s.removeTenant)
	mux.HandleFunc("GET /admin/cedar-policies", s.listCedarPolicies)
	mux.HandleFunc("POST /admin/cedar-policies", s.addCedarPolicy)
	mux.HandleFunc("DELETE /admin/cedar-policies/{id}", s.removeCedarPolicy)
	mux.HandleFunc("GET /admin/rate-limits", s.listRateLimits)
	mux.HandleFunc("PUT /admin/rate-limits", s.updateRateLimits)
	mux.HandleFunc("GET /admin/config", s.getConfig)
}

func (s *State) listScopeMappings(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, s.cfg.ScopeMapping)
}

func (s *State) updateScopeMappings(w http.ResponseWriter, r *http.Request) {
	var mapping map[string]config.Role
	if err := json.NewDecoder(r.Body).Decode(&mapping); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	s.mu.Lock()
	s.cfg.ScopeMapping = mapping
	s.rebuildMapper()
	s.mu.Unlock()
	writeJSON(w, map[string]string{"status": "updated"})
}

func (s *State) listProviders(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, s.cfg.Providers)
}

func (s *State) addProvider(w http.ResponseWriter, r *http.Request) {
	var p config.Provider
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil || p.Name == "" || p.Issuer == "" {
		writeErr(w, http.StatusBadRequest, "name and issuer required")
		return
	}
	s.mu.Lock()
	for _, existing := range s.cfg.Providers {
		if existing.Name == p.Name {
			s.mu.Unlock()
			writeErr(w, http.StatusConflict, "provider already exists")
			return
		}
	}
	s.cfg.Providers = append(s.cfg.Providers, p)
	s.mu.Unlock()
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, p)
}

func (s *State) removeProvider(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s.mu.Lock()
	found := false
	filtered := s.cfg.Providers[:0]
	for _, p := range s.cfg.Providers {
		if p.Name == name {
			found = true
		} else {
			filtered = append(filtered, p)
		}
	}
	s.cfg.Providers = filtered
	s.mu.Unlock()
	if !found {
		writeErr(w, http.StatusNotFound, "provider not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *State) listTenants(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, s.cfg.Tenants)
}

func (s *State) updateTenant(w http.ResponseWriter, r *http.Request) {
	issuer := r.PathValue("issuer")
	var t config.Tenant
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	s.mu.Lock()
	if s.cfg.Tenants == nil {
		s.cfg.Tenants = make(map[string]config.Tenant)
	}
	s.cfg.Tenants[issuer] = t
	s.rebuildMapper()
	s.rebuildCedar()
	s.mu.Unlock()
	writeJSON(w, map[string]string{"status": "updated", "issuer": issuer})
}

func (s *State) removeTenant(w http.ResponseWriter, r *http.Request) {
	issuer := r.PathValue("issuer")
	s.mu.Lock()
	if _, ok := s.cfg.Tenants[issuer]; !ok {
		s.mu.Unlock()
		writeErr(w, http.StatusNotFound, "tenant not found")
		return
	}
	delete(s.cfg.Tenants, issuer)
	s.rebuildMapper()
	s.rebuildCedar()
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// ── Cedar Policy CRUD ─────────────────────────────────────────────────────────

// CedarPolicyInput is the JSON body for adding a Cedar policy.
type CedarPolicyInput struct {
	ID     string `json:"id"`
	Effect string `json:"effect"` // "permit" or "forbid"
	Resource string `json:"resource,omitempty"` // index pattern to match
}

func (s *State) listCedarPolicies(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, s.cedarEng.ListPolicies())
}

func (s *State) addCedarPolicy(w http.ResponseWriter, r *http.Request) {
	var input CedarPolicyInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.ID == "" {
		writeErr(w, http.StatusBadRequest, "id is required")
		return
	}
	effect := cedar.Permit
	if input.Effect == "forbid" {
		effect = cedar.Forbid
	}
	p := cedar.Policy{
		ID:        input.ID,
		Effect:    effect,
		Principal: cedar.Match{Any: true},
		Action:    cedar.Match{Any: true},
		Resource:  cedar.Match{Any: true},
	}
	if input.Resource != "" {
		p.Resource = cedar.Match{Equals: input.Resource}
	}
	s.mu.Lock()
	s.cedarEng.AddGlobalPolicy(p)
	s.mu.Unlock()
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]string{"status": "created", "id": input.ID})
}

func (s *State) removeCedarPolicy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.Lock()
	removed := s.cedarEng.RemoveGlobalPolicy(id)
	s.mu.Unlock()
	if !removed {
		writeErr(w, http.StatusNotFound, "policy not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Rate Limit CRUD ───────────────────────────────────────────────────────────

func (s *State) listRateLimits(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, s.cfg.RateLimits)
}

func (s *State) updateRateLimits(w http.ResponseWriter, r *http.Request) {
	var limits map[string]int
	if err := json.NewDecoder(r.Body).Decode(&limits); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	s.mu.Lock()
	s.cfg.RateLimits = limits
	s.mu.Unlock()
	writeJSON(w, map[string]string{"status": "updated"})
}

func (s *State) getConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Return sanitized config (no secrets)
	safe := map[string]interface{}{
		"listen":        s.cfg.Listen,
		"upstream":      s.cfg.Upstream,
		"providers":     s.cfg.Providers,
		"scope_mapping": s.cfg.ScopeMapping,
		"tenants":       s.cfg.Tenants,
	}
	writeJSON(w, safe)
}

// rebuildMapper recreates the scope mapper from current config. Caller must hold mu.
func (s *State) rebuildMapper() {
	*s.mapper = *scope.NewMultiTenantMapper(s.cfg.ScopeMapping, s.cfg.Tenants)
}

// rebuildCedar reloads per-tenant Cedar policies. Caller must hold mu.
func (s *State) rebuildCedar() {
	for issuer, t := range s.cfg.Tenants {
		var policies []cedar.Policy
		for i, pText := range t.CedarPolicies {
			p, err := cedar.ParsePolicy(issuer+"-"+string(rune('0'+i)), pText)
			if err == nil {
				policies = append(policies, p)
			}
		}
		if len(policies) > 0 {
			s.cedarEng.AddTenant(issuer, policies)
		}
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
