// Package demo serves the sample log viewer web app and handles OAuth callbacks.
package demo

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
)

// Handler serves the demo web app.
type Handler struct {
	proxyURL string // e.g. "https://f5cmk2hxwx.us-west-2.awsapprunner.com"
	clientID string
}

// NewHandler creates a demo handler.
func NewHandler(proxyURL, clientID string) *Handler {
	return &Handler{proxyURL: strings.TrimRight(proxyURL, "/"), clientID: clientID}
}

// Register mounts demo routes on the mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /demo", h.App)
	mux.HandleFunc("GET /demo/", h.App)
	mux.HandleFunc("GET /demo/callback", h.Callback)
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// App serves the single-page log viewer dashboard.
func (h *Handler) App(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, appHTML, h.proxyURL, h.clientID)
}

// Callback handles the PKCE redirect — exchanges code for token client-side.
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, callbackHTML, h.proxyURL, h.clientID)
}

const callbackHTML = `<!DOCTYPE html>
<html><head><title>oauth4os — Logging in...</title></head>
<body>
<p id="status">Exchanging authorization code...</p>
<script>
const PROXY = %q;
const CLIENT_ID = %q;
(async () => {
  const params = new URLSearchParams(location.search);
  const code = params.get('code');
  const verifier = sessionStorage.getItem('pkce_verifier');
  if (!code || !verifier) {
    document.getElementById('status').textContent = 'Missing code or verifier';
    return;
  }
  try {
    const resp = await fetch(PROXY + '/oauth/token', {
      method: 'POST',
      headers: {'Content-Type': 'application/x-www-form-urlencoded'},
      body: new URLSearchParams({
        grant_type: 'authorization_code',
        code: code,
        client_id: CLIENT_ID,
        code_verifier: verifier,
        redirect_uri: location.origin + '/demo/callback'
      })
    });
    const data = await resp.json();
    if (data.access_token) {
      sessionStorage.setItem('access_token', data.access_token);
      sessionStorage.removeItem('pkce_verifier');
      location.href = '/demo';
    } else {
      document.getElementById('status').textContent = 'Token exchange failed: ' + (data.error || 'unknown');
    }
  } catch (e) {
    document.getElementById('status').textContent = 'Error: ' + e.message;
  }
})();
</script>
</body></html>`

const appHTML = `<!DOCTYPE html>
<html><head>
<title>oauth4os — Log Viewer Demo</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #0d1117; color: #c9d1d9; }
  .header { background: #161b22; border-bottom: 1px solid #30363d; padding: 12px 24px; display: flex; align-items: center; justify-content: space-between; }
  .header h1 { font-size: 18px; color: #58a6ff; }
  .header .user { font-size: 13px; color: #8b949e; }
  .header button { background: #21262d; color: #c9d1d9; border: 1px solid #30363d; padding: 6px 12px; border-radius: 6px; cursor: pointer; }
  .container { max-width: 1000px; margin: 24px auto; padding: 0 16px; }
  .search-bar { display: flex; gap: 8px; margin-bottom: 16px; }
  .search-bar input { flex: 1; background: #0d1117; border: 1px solid #30363d; color: #c9d1d9; padding: 10px 14px; border-radius: 6px; font-size: 14px; }
  .search-bar select { background: #0d1117; border: 1px solid #30363d; color: #c9d1d9; padding: 10px; border-radius: 6px; }
  .search-bar button { background: #238636; color: #fff; border: none; padding: 10px 20px; border-radius: 6px; cursor: pointer; font-weight: 600; }
  .results { background: #161b22; border: 1px solid #30363d; border-radius: 6px; overflow: hidden; }
  .log-entry { padding: 8px 14px; border-bottom: 1px solid #21262d; font-family: 'SF Mono', monospace; font-size: 13px; display: flex; gap: 12px; }
  .log-entry:last-child { border-bottom: none; }
  .log-entry .ts { color: #8b949e; min-width: 180px; }
  .log-entry .svc { color: #d2a8ff; min-width: 100px; }
  .log-entry .lvl { min-width: 60px; font-weight: 600; }
  .lvl-ERROR { color: #f85149; } .lvl-WARN { color: #d29922; } .lvl-INFO { color: #3fb950; } .lvl-DEBUG { color: #8b949e; }
  .log-entry .msg { color: #c9d1d9; }
  .scope-demo { margin-top: 16px; padding: 16px; background: #161b22; border: 1px solid #30363d; border-radius: 6px; }
  .scope-demo h3 { margin-bottom: 8px; font-size: 14px; }
  .scope-result { font-family: monospace; font-size: 13px; padding: 4px 0; }
  .scope-ok { color: #3fb950; } .scope-fail { color: #f85149; }
  .login-page { display: flex; flex-direction: column; align-items: center; justify-content: center; height: 80vh; }
  .login-page h1 { font-size: 32px; color: #58a6ff; margin-bottom: 8px; }
  .login-page p { color: #8b949e; margin-bottom: 24px; }
  .login-page button { background: #238636; color: #fff; border: none; padding: 14px 32px; border-radius: 8px; font-size: 16px; cursor: pointer; font-weight: 600; }
  .empty { padding: 40px; text-align: center; color: #8b949e; }
  #count { font-size: 13px; color: #8b949e; margin-bottom: 8px; }
</style>
</head>
<body>
<div id="app"></div>
<script>
const PROXY = %q;
const CLIENT_ID = %q;
const token = sessionStorage.getItem('access_token');

function renderLogin() {
  document.getElementById('app').innerHTML =
    '<div class="login-page">' +
    '<h1>🔐 oauth4os Log Viewer</h1>' +
    '<p>Search OpenSearch logs through the oauth4os proxy with PKCE auth</p>' +
    '<button onclick="startLogin()">Login with oauth4os</button>' +
    '</div>';
}

async function startLogin() {
  const verifier = Array.from(crypto.getRandomValues(new Uint8Array(32)), b => b.toString(16).padStart(2,'0')).join('');
  sessionStorage.setItem('pkce_verifier', verifier);
  const encoder = new TextEncoder();
  const digest = await crypto.subtle.digest('SHA-256', encoder.encode(verifier));
  const challenge = btoa(String.fromCharCode(...new Uint8Array(digest))).replace(/\+/g,'-').replace(/\//g,'_').replace(/=+$/,'');
  const redirect = location.origin + '/demo/callback';
  location.href = PROXY + '/oauth/authorize?response_type=code&client_id=' + CLIENT_ID +
    '&redirect_uri=' + encodeURIComponent(redirect) +
    '&code_challenge=' + challenge + '&code_challenge_method=S256&scope=read:logs-*';
}

function renderDashboard() {
  document.getElementById('app').innerHTML =
    '<div class="header"><h1>🔐 oauth4os Log Viewer</h1><div><span class="user">Token: ' + token.substring(0,12) + '...</span> <button onclick="logout()">Logout</button></div></div>' +
    '<div class="container">' +
    '<div class="search-bar">' +
    '<input id="q" placeholder="Search logs... (e.g. level:ERROR, payment timeout)" onkeydown="if(event.key===\'Enter\')search()">' +
    '<select id="svc"><option value="">All services</option><option>payment</option><option>auth</option><option>cart</option><option>shipping</option><option>inventory</option></select>' +
    '<select id="lvl"><option value="">All levels</option><option>ERROR</option><option>WARN</option><option>INFO</option><option>DEBUG</option></select>' +
    '<button onclick="search()">Search</button></div>' +
    '<div id="count"></div>' +
    '<div class="results" id="results"><div class="empty">Enter a query to search logs</div></div>' +
    '<div class="scope-demo"><h3>Scope enforcement demo</h3>' +
    '<button onclick="testScope(\'GET\',\'read\')">Try read ✅</button> ' +
    '<button onclick="testScope(\'PUT\',\'write\')">Try write ❌</button>' +
    '<div id="scope-results"></div></div>' +
    '</div>';
}

async function search() {
  const q = document.getElementById('q').value;
  const svc = document.getElementById('svc').value;
  const lvl = document.getElementById('lvl').value;
  let query = { query: { bool: { must: [] } } };
  if (q) query.query.bool.must.push({ query_string: { query: q } });
  if (svc) query.query.bool.must.push({ match: { service: svc } });
  if (lvl) query.query.bool.must.push({ match: { level: lvl } });
  if (query.query.bool.must.length === 0) query = { query: { match_all: {} } };
  query.size = 50;
  query.sort = [{ timestamp: { order: 'desc' } }];
  try {
    const resp = await fetch(PROXY + '/logs-*/_search', {
      method: 'POST',
      headers: { 'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json' },
      body: JSON.stringify(query)
    });
    if (resp.status === 401 || resp.status === 403) { logout(); return; }
    const data = await resp.json();
    const hits = (data.hits && data.hits.hits) || [];
    document.getElementById('count').textContent = (data.hits.total.value || hits.length) + ' results';
    if (hits.length === 0) {
      document.getElementById('results').innerHTML = '<div class="empty">No results</div>';
      return;
    }
    document.getElementById('results').innerHTML = hits.map(h => {
      const s = h._source || {};
      return '<div class="log-entry"><span class="ts">' + (s.timestamp||'') + '</span><span class="svc">' + (s.service||'') + '</span><span class="lvl lvl-' + (s.level||'') + '">' + (s.level||'') + '</span><span class="msg">' + escHtml(s.message||'') + '</span></div>';
    }).join('');
  } catch(e) {
    document.getElementById('results').innerHTML = '<div class="empty">Error: ' + e.message + '</div>';
  }
}

async function testScope(method, label) {
  try {
    const resp = await fetch(PROXY + '/logs-demo/_doc/scope-test', {
      method: method,
      headers: { 'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json' },
      body: method === 'PUT' ? '{"test":true}' : undefined
    });
    const el = document.getElementById('scope-results');
    if (resp.ok || resp.status === 200) {
      el.innerHTML += '<div class="scope-result scope-ok">' + label + ' → ' + resp.status + ' ✅ allowed</div>';
    } else {
      el.innerHTML += '<div class="scope-result scope-fail">' + label + ' → ' + resp.status + ' ❌ denied</div>';
    }
  } catch(e) {
    document.getElementById('scope-results').innerHTML += '<div class="scope-result scope-fail">' + label + ' → error</div>';
  }
}

function escHtml(s) { return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }
function logout() { sessionStorage.clear(); location.href = '/demo'; }

if (token) { renderDashboard(); } else { renderLogin(); }
</script>
</body></html>`
