// Package backup provides export/import of all proxy configuration as a JSON bundle.
package backup

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/seraphjiang/oauth4os/internal/config"
)

// Bundle is the exportable configuration snapshot.
type Bundle struct {
	Version      string                   `json:"version"`
	ExportedAt   string                   `json:"exported_at"`
	Providers    []config.Provider        `json:"providers"`
	ScopeMapping map[string]config.Role   `json:"scope_mapping"`
	Tenants      map[string]config.Tenant `json:"tenants"`
	RateLimits   map[string]int           `json:"rate_limits,omitempty"`
	Clients      []ClientEntry            `json:"clients,omitempty"`
}

// ClientEntry is a registered client (secret excluded from export).
type ClientEntry struct {
	ID     string   `json:"client_id"`
	Scopes []string `json:"scopes"`
}

// ConfigGetter reads current config. Provided by the proxy.
type ConfigGetter func() *config.Config

// ClientLister lists registered clients. Provided by token manager.
type ClientLister func() []ClientEntry

// ConfigApplier applies an imported config. Provided by the proxy.
type ConfigApplier func(*config.Config)

// Handler serves backup/restore endpoints.
type Handler struct {
	mu       sync.RWMutex
	getCfg   ConfigGetter
	listCli  ClientLister
	applyCfg ConfigApplier
}

// NewHandler creates a backup handler.
func NewHandler(getCfg ConfigGetter, listCli ClientLister, applyCfg ConfigApplier) *Handler {
	return &Handler{getCfg: getCfg, listCli: listCli, applyCfg: applyCfg}
}

// Register mounts backup/restore routes.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/backup", h.Export)
	mux.HandleFunc("POST /admin/restore", h.Import)
}

// Export returns the full config as a JSON bundle.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	cfg := h.getCfg()
	clients := h.listCli()
	h.mu.RUnlock()

	bundle := Bundle{
		Version:      "1",
		ExportedAt:   time.Now().UTC().Format(time.RFC3339),
		Providers:    cfg.Providers,
		ScopeMapping: cfg.ScopeMapping,
		Tenants:      cfg.Tenants,
		RateLimits:   cfg.RateLimits,
		Clients:      clients,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=oauth4os-backup.json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(bundle)
}

// Import restores config from a JSON bundle.
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	var bundle Bundle
	if err := json.NewDecoder(r.Body).Decode(&bundle); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}

	h.mu.Lock()
	cfg := h.getCfg()
	cfg.Providers = bundle.Providers
	cfg.ScopeMapping = bundle.ScopeMapping
	cfg.Tenants = bundle.Tenants
	if bundle.RateLimits != nil {
		cfg.RateLimits = bundle.RateLimits
	}
	h.applyCfg(cfg)
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "restored",
		"version":  bundle.Version,
		"imported": bundle.ExportedAt,
	})
}
