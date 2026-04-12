// Package ciba implements Client Initiated Backchannel Authentication (CIBA).
// Backend services request user auth via backchannel — user approves on a separate
// device/app, service polls for token. No browser redirect needed.
package ciba

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type authRequest struct {
	RequestID string
	ClientID  string
	Subject   string // login_hint
	Scopes    []string
	ExpiresAt time.Time
	Approved  bool
	Denied    bool
	Interval  int
}

// Handler manages CIBA flow.
type Handler struct {
	mu       sync.Mutex
	requests map[string]*authRequest
	issuer   func(clientID string, scopes []string) (string, string)
}

// NewHandler creates a CIBA handler.
func NewHandler(issuer func(string, []string) (string, string)) *Handler {
	return &Handler{
		requests: make(map[string]*authRequest),
		issuer:   issuer,
	}
}

// Register mounts CIBA endpoints.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /oauth/bc-authorize", h.Initiate)
	mux.HandleFunc("POST /oauth/bc-token", h.Poll)
	mux.HandleFunc("POST /oauth/bc-approve", h.Approve)
}

// Initiate handles POST /oauth/bc-authorize — start backchannel auth.
func (h *Handler) Initiate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	clientID := r.FormValue("client_id")
	loginHint := r.FormValue("login_hint")
	if clientID == "" || loginHint == "" {
		writeJSON(w, 400, map[string]string{"error": "invalid_request", "error_description": "client_id and login_hint required"})
		return
	}

	b := make([]byte, 16)
	rand.Read(b)
	reqID := hex.EncodeToString(b)

	h.mu.Lock()
	h.requests[reqID] = &authRequest{
		RequestID: reqID,
		ClientID:  clientID,
		Subject:   loginHint,
		Scopes:    splitScope(r.FormValue("scope")),
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Interval:  5,
	}
	h.mu.Unlock()

	writeJSON(w, 200, map[string]interface{}{
		"auth_req_id": reqID,
		"expires_in":  300,
		"interval":    5,
	})
}

// Poll handles POST /oauth/bc-token — client polls for token.
func (h *Handler) Poll(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	reqID := r.FormValue("auth_req_id")

	h.mu.Lock()
	req, ok := h.requests[reqID]
	if !ok || time.Now().After(req.ExpiresAt) {
		h.mu.Unlock()
		writeJSON(w, 400, map[string]string{"error": "expired_token"})
		return
	}
	if req.Denied {
		delete(h.requests, reqID)
		h.mu.Unlock()
		writeJSON(w, 400, map[string]string{"error": "access_denied"})
		return
	}
	if !req.Approved {
		h.mu.Unlock()
		writeJSON(w, 400, map[string]string{"error": "authorization_pending"})
		return
	}
	clientID, scopes := req.ClientID, req.Scopes
	delete(h.requests, reqID)
	h.mu.Unlock()

	access, refresh := h.issuer(clientID, scopes)
	writeJSON(w, 200, map[string]interface{}{
		"access_token":  access,
		"refresh_token": refresh,
		"token_type":    "Bearer",
		"expires_in":    3600,
	})
}

// Approve handles POST /oauth/bc-approve — user approves or denies.
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	reqID := r.FormValue("auth_req_id")
	action := r.FormValue("action")

	h.mu.Lock()
	req, ok := h.requests[reqID]
	if ok {
		if action == "deny" {
			req.Denied = true
		} else {
			req.Approved = true
		}
	}
	h.mu.Unlock()

	if !ok {
		writeJSON(w, 404, map[string]string{"error": "not_found"})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func splitScope(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ' ' {
			if i > start {
				out = append(out, s[start:i])
			}
			start = i + 1
		}
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
