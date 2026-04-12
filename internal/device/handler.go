// Package device implements OAuth 2.0 Device Authorization Grant (RFC 8628).
// For CLI/IoT devices without a browser — user authorizes on a separate device.
package device

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type code struct {
	DeviceCode string
	UserCode   string
	ClientID   string
	Scopes     []string
	ExpiresAt  time.Time
	Approved   bool
	Denied     bool
	Interval   int // poll interval seconds
}

// Handler manages device authorization flow.
type Handler struct {
	mu     sync.Mutex
	codes  map[string]*code // device_code → code
	byUser map[string]*code // user_code → code
	issuer func(clientID string, scopes []string) (accessToken, refreshToken string)
}

// NewHandler creates a device flow handler.
func NewHandler(issuer func(string, []string) (string, string)) *Handler {
	return &Handler{
		codes:  make(map[string]*code),
		byUser: make(map[string]*code),
		issuer: issuer,
	}
}

// Register mounts device flow endpoints.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /oauth/device/code", h.RequestCode)
	mux.HandleFunc("POST /oauth/device/token", h.PollToken)
	mux.HandleFunc("GET /oauth/device", h.UserPage)
	mux.HandleFunc("POST /oauth/device/approve", h.Approve)
}

func randomCode(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func userCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	s := strings.ToUpper(hex.EncodeToString(b))
	return s[:4] + "-" + s[4:]
}

// RequestCode handles POST /oauth/device/code — device requests authorization.
func (h *Handler) RequestCode(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	clientID := r.FormValue("client_id")
	if clientID == "" {
		writeJSON(w, 400, map[string]string{"error": "invalid_request", "error_description": "client_id required"})
		return
	}
	scopes := strings.Fields(r.FormValue("scope"))

	dc := randomCode(20)
	uc := userCode()
	c := &code{
		DeviceCode: dc,
		UserCode:   uc,
		ClientID:   clientID,
		Scopes:     scopes,
		ExpiresAt:  time.Now().Add(10 * time.Minute),
		Interval:   5,
	}

	h.mu.Lock()
	h.codes[dc] = c
	h.byUser[uc] = c
	h.mu.Unlock()

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	verifyURI := fmt.Sprintf("%s://%s/oauth/device", scheme, r.Host)

	writeJSON(w, 200, map[string]interface{}{
		"device_code":               dc,
		"user_code":                 uc,
		"verification_uri":          verifyURI,
		"verification_uri_complete": verifyURI + "?user_code=" + uc,
		"expires_in":                600,
		"interval":                  5,
	})
}

// PollToken handles POST /oauth/device/token — device polls for token.
func (h *Handler) PollToken(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	deviceCode := r.FormValue("device_code")
	if r.FormValue("grant_type") != "urn:ietf:params:oauth:grant-type:device_code" {
		writeJSON(w, 400, map[string]string{"error": "unsupported_grant_type"})
		return
	}

	h.mu.Lock()
	c, ok := h.codes[deviceCode]
	h.mu.Unlock()

	if !ok || time.Now().After(c.ExpiresAt) {
		writeJSON(w, 400, map[string]string{"error": "expired_token"})
		return
	}
	if c.Denied {
		writeJSON(w, 400, map[string]string{"error": "access_denied"})
		return
	}
	if !c.Approved {
		writeJSON(w, 400, map[string]string{"error": "authorization_pending"})
		return
	}

	// Approved — issue token and clean up
	access, refresh := h.issuer(c.ClientID, c.Scopes)
	h.mu.Lock()
	delete(h.codes, deviceCode)
	delete(h.byUser, c.UserCode)
	h.mu.Unlock()

	writeJSON(w, 200, map[string]interface{}{
		"access_token":  access,
		"refresh_token": refresh,
		"token_type":    "Bearer",
		"expires_in":    3600,
	})
}

// UserPage serves the approval page where the user enters the code.
func (h *Handler) UserPage(w http.ResponseWriter, r *http.Request) {
	uc := r.URL.Query().Get("user_code")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, userPageHTML, uc)
}

// Approve handles POST /oauth/device/approve — user approves or denies.
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	uc := r.FormValue("user_code")
	action := r.FormValue("action") // "approve" or "deny"

	h.mu.Lock()
	c, ok := h.byUser[uc]
	if ok {
		if action == "deny" {
			c.Denied = true
		} else {
			c.Approved = true
		}
	}
	h.mu.Unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if !ok {
		fmt.Fprint(w, `<html><body style="background:#0d1117;color:#f85149;font-family:sans-serif;display:flex;justify-content:center;align-items:center;height:80vh"><h2>Invalid or expired code</h2></body></html>`)
		return
	}
	if action == "deny" {
		fmt.Fprint(w, `<html><body style="background:#0d1117;color:#d29922;font-family:sans-serif;display:flex;justify-content:center;align-items:center;height:80vh"><h2>Access denied. You can close this tab.</h2></body></html>`)
	} else {
		fmt.Fprint(w, `<html><body style="background:#0d1117;color:#3fb950;font-family:sans-serif;display:flex;justify-content:center;align-items:center;height:80vh"><h2>✅ Device authorized! You can close this tab.</h2></body></html>`)
	}
}

// Cleanup removes expired codes.
func (h *Handler) Cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for k, c := range h.codes {
		if time.Now().After(c.ExpiresAt) {
			delete(h.byUser, c.UserCode)
			delete(h.codes, k)
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

const userPageHTML = `<!DOCTYPE html>
<html><head><title>oauth4os — Device Authorization</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,sans-serif;background:#0d1117;color:#c9d1d9;display:flex;justify-content:center;align-items:center;height:100vh}
.card{background:#161b22;border:1px solid #30363d;border-radius:12px;padding:40px;text-align:center;max-width:400px}
h2{color:#58a6ff;margin-bottom:16px}
p{color:#8b949e;margin-bottom:20px;font-size:14px}
input{background:#0d1117;border:1px solid #30363d;color:#c9d1d9;padding:12px;border-radius:6px;font-size:20px;text-align:center;letter-spacing:4px;width:200px;margin-bottom:16px}
.btns{display:flex;gap:12px;justify-content:center}
button{padding:10px 24px;border-radius:6px;border:none;font-size:14px;font-weight:600;cursor:pointer}
.approve{background:#238636;color:#fff}
.deny{background:#21262d;color:#c9d1d9;border:1px solid #30363d}
</style></head>
<body>
<div class="card">
<h2>🔐 Device Authorization</h2>
<p>Enter the code shown on your device</p>
<form method="POST" action="/oauth/device/approve">
<input name="user_code" value="%s" placeholder="XXXX-XXXX" required><br>
<div class="btns">
<button type="submit" name="action" value="approve" class="approve">Authorize</button>
<button type="submit" name="action" value="deny" class="deny">Deny</button>
</div>
</form>
</div>
</body></html>`
