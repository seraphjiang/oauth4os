// Package pkce implements Proof Key for Code Exchange (RFC 7636) for browser clients.
package pkce

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"
)

// AuthCode is a pending authorization code.
type AuthCode struct {
	Code                string
	ClientID            string
	Scopes              []string
	CodeChallenge       string
	CodeChallengeMethod string
	CreatedAt           time.Time
	RedirectURI         string
}

// Handler manages PKCE authorization code flow.
type Handler struct {
	codes            map[string]*AuthCode
	pending          map[string]*pendingConsent
	mu               sync.Mutex
	issuer           func(clientID string, scopes []string) (accessToken, refreshToken string)
	validateRedirect func(clientID, uri string) bool
}

// NewHandler creates a PKCE handler. issuer mints tokens; validateRedirect checks redirect_uri allowlist.
func NewHandler(issuer func(clientID string, scopes []string) (string, string), validateRedirect func(clientID, uri string) bool) *Handler {
	return &Handler{
		codes:            make(map[string]*AuthCode),
		issuer:           issuer,
		validateRedirect: validateRedirect,
	}
}

// pendingConsent stores an authorize request awaiting user approval.
type pendingConsent struct {
	ID                  string
	ClientID            string
	Scopes              []string
	CodeChallenge       string
	CodeChallengeMethod string
	RedirectURI         string
	State               string
	CreatedAt           time.Time
}

// Authorize handles GET /oauth/authorize — shows consent screen.
func (h *Handler) Authorize(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	challenge := r.URL.Query().Get("code_challenge")
	method := r.URL.Query().Get("code_challenge_method")
	redirectURI := r.URL.Query().Get("redirect_uri")
	scope := r.URL.Query().Get("scope")
	state := r.URL.Query().Get("state")

	if clientID == "" || redirectURI == "" {
		writeErr(w, http.StatusBadRequest, "invalid_request", "client_id and redirect_uri required")
		return
	}
	if h.validateRedirect != nil && !h.validateRedirect(clientID, redirectURI) {
		writeErr(w, http.StatusBadRequest, "invalid_request", "redirect_uri not registered for client")
		return
	}
	if method == "" {
		method = "S256"
	}
	if method != "S256" {
		writeErr(w, http.StatusBadRequest, "invalid_request", "only S256 code_challenge_method supported")
		return
	}

	consentID := generateCode()
	h.mu.Lock()
	if h.pending == nil {
		h.pending = make(map[string]*pendingConsent)
	}
	h.pending[consentID] = &pendingConsent{
		ID: consentID, ClientID: clientID, Scopes: splitScopes(scope),
		CodeChallenge: challenge, CodeChallengeMethod: method,
		RedirectURI: redirectURI, State: state, CreatedAt: time.Now(),
	}
	h.mu.Unlock()

	renderConsent(w, r, consentID, clientID, splitScopes(scope))
}

// Consent handles POST /oauth/consent — approve or deny.
func (h *Handler) Consent(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	consentID := r.FormValue("consent_id")
	action := r.FormValue("action")

	h.mu.Lock()
	pc, ok := h.pending[consentID]
	if ok {
		delete(h.pending, consentID)
	}
	h.mu.Unlock()

	if !ok || time.Since(pc.CreatedAt) > 10*time.Minute {
		writeErr(w, http.StatusBadRequest, "invalid_request", "consent expired or not found")
		return
	}

	sep := "?"
	if len(pc.RedirectURI) > 0 && pc.RedirectURI[len(pc.RedirectURI)-1] == '?' {
		sep = ""
	}
	stateParam := ""
	if pc.State != "" {
		stateParam = "&state=" + pc.State
	}

	if action != "approve" {
		http.Redirect(w, r, fmt.Sprintf("%s%serror=access_denied%s", pc.RedirectURI, sep, stateParam), http.StatusFound)
		return
	}

	code := generateCode()
	h.mu.Lock()
	h.codes[code] = &AuthCode{
		Code: code, ClientID: pc.ClientID, Scopes: pc.Scopes,
		CodeChallenge: pc.CodeChallenge, CodeChallengeMethod: pc.CodeChallengeMethod,
		CreatedAt: time.Now(), RedirectURI: pc.RedirectURI,
	}
	h.mu.Unlock()

	http.Redirect(w, r, fmt.Sprintf("%s%scode=%s%s", pc.RedirectURI, sep, code, stateParam), http.StatusFound)
}

// Exchange handles POST /oauth/token with grant_type=authorization_code.
func (h *Handler) Exchange(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	code := r.FormValue("code")
	verifier := r.FormValue("code_verifier")
	redirectURI := r.FormValue("redirect_uri")

	if code == "" {
		writeErr(w, http.StatusBadRequest, "invalid_request", "code required")
		return
	}

	h.mu.Lock()
	ac, ok := h.codes[code]
	if ok {
		delete(h.codes, code) // one-time use
	}
	h.mu.Unlock()

	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid_grant", "authorization code not found or already used")
		return
	}

	// Expire after 10 minutes
	if time.Since(ac.CreatedAt) > 10*time.Minute {
		writeErr(w, http.StatusBadRequest, "invalid_grant", "authorization code expired")
		return
	}

	if ac.RedirectURI != redirectURI {
		writeErr(w, http.StatusBadRequest, "invalid_grant", "redirect_uri mismatch")
		return
	}

	// Verify PKCE if code_challenge was provided at authorization time
	if ac.CodeChallenge != "" {
		if verifier == "" {
			writeErr(w, http.StatusBadRequest, "invalid_request", "code_verifier required for PKCE")
			return
		}
		hash := sha256.Sum256([]byte(verifier))
		computed := base64.RawURLEncoding.EncodeToString(hash[:])
		if subtle.ConstantTimeCompare([]byte(computed), []byte(ac.CodeChallenge)) != 1 {
			writeErr(w, http.StatusBadRequest, "invalid_grant", "code_verifier failed verification")
			return
		}
	}

	accessToken, refreshToken := h.issuer(ac.ClientID, ac.Scopes)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": refreshToken,
	})
}

// Cleanup removes expired codes. Call periodically.
func (h *Handler) Cleanup() {
	h.mu.Lock()
	for k, v := range h.codes {
		if time.Since(v.CreatedAt) > 10*time.Minute {
			delete(h.codes, k)
		}
	}
	h.mu.Unlock()
}

func generateCode() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// scopeDescriptions maps scope prefixes to human-readable descriptions.
var scopeDescriptions = map[string]string{
	"read":     "Read data from indices",
	"write":    "Write and modify data in indices",
	"admin":    "Manage cluster settings and configuration",
	"delete":   "Delete documents and indices",
	"monitor":  "View cluster health and metrics",
	"search":   "Search across indices",
	"create":   "Create new indices and mappings",
	"manage":   "Manage index settings and aliases",
	"openid":   "Access your user profile",
	"profile":  "View your profile information",
	"email":    "View your email address",
	"offline_access": "Maintain access when you're not present",
}

func describeScope(s string) string {
	if d, ok := scopeDescriptions[s]; ok {
		return d
	}
	// Try prefix match: "read:logs-*" → "read"
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			if d, ok := scopeDescriptions[s[:i]]; ok {
				return d + " (" + s[i+1:] + ")"
			}
			break
		}
	}
	return "Access: " + s
}

func scopeIcon(s string) string {
	switch {
	case len(s) >= 4 && s[:4] == "read":
		return "👁"
	case len(s) >= 5 && s[:5] == "write":
		return "✏️"
	case len(s) >= 5 && s[:5] == "admin":
		return "⚙️"
	case len(s) >= 6 && s[:6] == "delete":
		return "🗑"
	case len(s) >= 7 && s[:7] == "monitor":
		return "📊"
	default:
		return "🔑"
	}
}

type consentStrings struct {
	Title, Subtitle, AppLabel, Permissions, WriteWarn, Deny, Approve, Footer string
}

var consentI18n = map[string]consentStrings{
	"en": {"Authorize — oauth4os", "An application is requesting access", "Application", "Requested permissions", "This app is requesting write access to your data", "Deny", "Approve", "Authorizing will redirect you back to the application"},
}

func detectLang(r *http.Request) string {
	if a := r.Header.Get("Accept-Language"); len(a) >= 2 {
		return a[:2]
	}
	return "en"
}

func renderConsent(w http.ResponseWriter, r *http.Request, consentID, clientID string, scopes []string) {
	lang := detectLang(r)
	s, ok := consentI18n[lang]
	if !ok {
		s = consentI18n["en"]
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html lang="%s"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s</title><style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,system-ui,sans-serif;background:#0d1117;color:#e6edf3;min-height:100vh;display:flex;align-items:center;justify-content:center}
.card{background:#161b22;border:1px solid #30363d;border-radius:16px;padding:40px;max-width:440px;width:100%%;margin:20px}
.logo{text-align:center;font-size:24px;font-weight:700;margin-bottom:8px}
.logo span{color:#58a6ff}
.subtitle{text-align:center;color:#8b949e;font-size:14px;margin-bottom:28px}
.app-name{background:#1c2128;border:1px solid #30363d;border-radius:10px;padding:14px 18px;display:flex;align-items:center;gap:12px;margin-bottom:24px}
.app-icon{width:40px;height:40px;background:linear-gradient(135deg,#58a6ff,#bc8cff);border-radius:10px;display:flex;align-items:center;justify-content:center;font-size:20px;flex-shrink:0}
.app-label{font-size:12px;color:#8b949e}
.app-id{font-size:15px;font-weight:600}
h3{font-size:13px;color:#8b949e;text-transform:uppercase;letter-spacing:.5px;margin-bottom:12px}
.scopes{list-style:none;margin-bottom:28px}
.scopes li{display:flex;align-items:flex-start;gap:10px;padding:10px 14px;background:#1c2128;border:1px solid #30363d;border-radius:8px;margin-bottom:6px}
.scope-icon{font-size:16px;flex-shrink:0;margin-top:1px}
.scope-name{font-size:13px;font-weight:600;color:#e6edf3}
.scope-desc{font-size:12px;color:#8b949e;margin-top:2px}
.warn{background:#1c1507;border-color:#533d08;border-radius:8px;padding:10px 14px;font-size:12px;color:#f0883e;margin-bottom:24px;display:flex;align-items:center;gap:8px}
.buttons{display:flex;gap:10px}
.btn{flex:1;padding:12px;border:none;border-radius:10px;font-size:14px;font-weight:600;cursor:pointer;transition:all .15s}
.btn:hover{transform:translateY(-1px)}
.btn-approve{background:#238636;color:#fff}
.btn-approve:hover{background:#2ea043}
.btn-deny{background:#21262d;color:#c9d1d9;border:1px solid #30363d}
.btn-deny:hover{background:#30363d}
.footer{text-align:center;margin-top:20px;font-size:11px;color:#484f58}
</style></head><body><div class="card">
<div class="logo">🔐 oauth<span>4os</span></div>
<div class="subtitle">%s</div>
<div class="app-name"><div class="app-icon">🔗</div><div><div class="app-label">%s</div><div class="app-id">`, lang, s.Title, s.Subtitle, s.AppLabel)
	// Escape clientID
	for _, c := range clientID {
		switch c {
		case '<':
			fmt.Fprint(w, "&lt;")
		case '>':
			fmt.Fprint(w, "&gt;")
		case '&':
			fmt.Fprint(w, "&amp;")
		case '"':
			fmt.Fprint(w, "&quot;")
		default:
			fmt.Fprintf(w, "%c", c)
		}
	}
	fmt.Fprintf(w, `</div></div></div><h3>%s</h3><ul class="scopes">`, s.Permissions)
	hasWrite := false
	for _, s := range scopes {
		if len(s) >= 5 && (s[:5] == "write" || s[:5] == "admin") {
			hasWrite = true
		}
		fmt.Fprintf(w, `<li><span class="scope-icon">%s</span><div><div class="scope-name">%s</div><div class="scope-desc">%s</div></div></li>`,
			scopeIcon(s), template.HTMLEscapeString(s), template.HTMLEscapeString(describeScope(s)))
	}
	fmt.Fprint(w, `</ul>`)
	if hasWrite {
		fmt.Fprintf(w, `<div class="warn">⚠️ %s</div>`, s.WriteWarn)
	}
	fmt.Fprintf(w, `<form method="POST" action="/oauth/consent"><input type="hidden" name="consent_id" value="%s">`, consentID)
	fmt.Fprint(w, `<div class="buttons"><button type="submit" name="action" value="deny" class="btn btn-deny">Deny</button><button type="submit" name="action" value="approve" class="btn btn-approve">Approve</button></div></form>`)
	fmt.Fprint(w, `<div class="footer">Authorizing will redirect you back to the application</div></div>
<script>
(async()=>{try{const r=await fetch('/i18n/consent.json');if(!r.ok)return;const t=await r.json();const l=navigator.language?.slice(0,2)||'en';const s=t[l]||t.en;if(!s||l==='en')return;
document.querySelector('.subtitle').textContent=s.subtitle;
document.querySelector('h3').textContent=s.permissions;
document.querySelectorAll('.btn-approve').forEach(b=>b.textContent=s.approve);
document.querySelectorAll('.btn-deny').forEach(b=>b.textContent=s.deny);
document.querySelector('.footer').textContent=s.footer;
document.querySelector('.app-label').textContent=s.app_label;
document.title=s.title;
}catch{}})();
</script>
</body></html>`)

}

func splitScopes(s string) []string {
	if s == "" {
		return nil
	}
	var scopes []string
	for _, p := range split(s) {
		if p != "" {
			scopes = append(scopes, p)
		}
	}
	return scopes
}

func split(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			if i > start {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func writeErr(w http.ResponseWriter, status int, code, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": code, "error_description": desc})
}
