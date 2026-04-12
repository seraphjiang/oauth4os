// Package configui serves an admin UI for viewing and managing proxy configuration.
package configui

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/seraphjiang/oauth4os/internal/config"
)

// Handler serves the config admin UI.
type Handler struct {
	getCfg func() *config.Config
}

// New creates a config UI handler.
func New(getCfg func() *config.Config) *Handler {
	return &Handler{getCfg: getCfg}
}

// Register mounts routes.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/config", h.Page)
	mux.HandleFunc("GET /admin/config/json", h.JSON)
}

// JSON returns the current config as JSON (secrets redacted).
func (h *Handler) JSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.getCfg())
}

// Page serves the config viewer HTML.
func (h *Handler) Page(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, page)
}

const page = `<!DOCTYPE html>
<html><head><title>oauth4os — Config</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,sans-serif;background:#0d1117;color:#c9d1d9}
.hdr{background:#161b22;border-bottom:1px solid #30363d;padding:14px 24px;display:flex;align-items:center;justify-content:space-between}
.hdr h1{font-size:18px;color:#58a6ff}
.hdr a{color:#8b949e;text-decoration:none;font-size:13px}
.ctr{max-width:900px;margin:24px auto;padding:0 16px}
pre{background:#161b22;border:1px solid #30363d;border-radius:8px;padding:20px;font-size:13px;font-family:'SF Mono',monospace;overflow-x:auto;line-height:1.6}
.key{color:#ff7b72}.str{color:#a5d6ff}.num{color:#79c0ff}.bool{color:#d2a8ff}.null{color:#8b949e}
.section{margin-bottom:16px}
.section h2{font-size:14px;color:#8b949e;margin-bottom:8px;text-transform:uppercase;letter-spacing:.5px}
</style></head>
<body>
<div class="hdr"><h1>⚙️ Configuration</h1><a href="/developer">← Developer Portal</a></div>
<div class="ctr">
<div class="section"><h2>Current Configuration</h2><pre id="cfg">Loading...</pre></div>
</div>
<script>
fetch('/admin/config/json').then(r=>r.json()).then(d=>{
  document.getElementById('cfg').innerHTML=syntaxHighlight(JSON.stringify(d,null,2));
}).catch(e=>{document.getElementById('cfg').textContent='Error: '+e.message});
function syntaxHighlight(j){
  return j.replace(/("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g,function(m){
    let c='num';if(/^"/.test(m)){c=/:$/.test(m)?'key':'str'}else if(/true|false/.test(m)){c='bool'}else if(/null/.test(m)){c='null'}
    return'<span class="'+c+'">'+m+'</span>';
  });
}
</script>
</body></html>`
