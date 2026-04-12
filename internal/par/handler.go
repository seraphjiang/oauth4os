// Package par implements Pushed Authorization Requests (RFC 9126).
// Clients POST auth params to /oauth/par, get a request_uri, then redirect
// the user to /oauth/authorize?request_uri=... — prevents param tampering.
package par

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

type request struct {
	ClientID     string
	Scopes       []string
	RedirectURI  string
	State        string
	CodeChallenge string
	CodeMethod   string
	ExpiresAt    time.Time
}

// Handler manages PAR requests.
type Handler struct {
	mu       sync.Mutex
	requests map[string]*request // request_uri → request
	authClient func(id, secret string) error
}

// NewHandler creates a PAR handler.
func NewHandler(authClient func(id, secret string) error) *Handler {
	return &Handler{
		requests:   make(map[string]*request),
		authClient: authClient,
	}
}

// Register mounts PAR endpoints.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /oauth/par", h.Push)
}

// Push handles POST /oauth/par — client pushes authorization parameters.
func (h *Handler) Push(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")
	if basicID, basicSecret, ok := r.BasicAuth(); ok && clientID == "" {
		clientID = basicID
		clientSecret = basicSecret
	}

	if clientID == "" {
		writeErr(w, 400, "invalid_request", "client_id required")
		return
	}
	if h.authClient != nil && clientSecret != "" {
		if err := h.authClient(clientID, clientSecret); err != nil {
			writeErr(w, 401, "invalid_client", "authentication failed")
			return
		}
	}

	b := make([]byte, 16)
	rand.Read(b)
	uri := "urn:ietf:params:oauth:request_uri:" + hex.EncodeToString(b)

	req := &request{
		ClientID:      clientID,
		Scopes:        strings.Fields(r.FormValue("scope")),
		RedirectURI:   r.FormValue("redirect_uri"),
		State:         r.FormValue("state"),
		CodeChallenge: r.FormValue("code_challenge"),
		CodeMethod:    r.FormValue("code_challenge_method"),
		ExpiresAt:     time.Now().Add(60 * time.Second),
	}

	h.mu.Lock()
	h.requests[uri] = req
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"request_uri": uri,
		"expires_in":  60,
	})
}

// Resolve looks up a pushed request_uri. Returns nil if expired or not found.
func (h *Handler) Resolve(requestURI string) (clientID string, scopes []string, redirectURI, state, codeChallenge, codeMethod string, ok bool) {
	h.mu.Lock()
	req, found := h.requests[requestURI]
	if found {
		delete(h.requests, requestURI) // one-time use
	}
	h.mu.Unlock()
	if !found || time.Now().After(req.ExpiresAt) {
		return "", nil, "", "", "", "", false
	}
	return req.ClientID, req.Scopes, req.RedirectURI, req.State, req.CodeChallenge, req.CodeMethod, true
}

// Cleanup removes expired requests.
func (h *Handler) Cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for k, r := range h.requests {
		if time.Now().After(r.ExpiresAt) {
			delete(h.requests, k)
		}
	}
}

func writeErr(w http.ResponseWriter, status int, code, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": code, "error_description": desc})
}
