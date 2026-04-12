// Package tokenui serves the token inspector page at /developer/tokens.
package tokenui

import (
	"fmt"
	"net/http"
)

// Handler serves the token inspector UI.
type Handler struct {
	proxyURL string
}

// New creates a token inspector handler.
func New(proxyURL string) *Handler { return &Handler{proxyURL: proxyURL} }

// Register mounts routes.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /developer/tokens", h.Page)
}

// Page serves the token inspector HTML.
func (h *Handler) Page(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, page, h.proxyURL)
}

const page = `<!DOCTYPE html>
<html><head>
<title>oauth4os — Token Inspector</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#0d1117;color:#c9d1d9}
.hdr{background:#161b22;border-bottom:1px solid #30363d;padding:14px 24px;display:flex;align-items:center;justify-content:space-between}
.hdr h1{font-size:18px;color:#58a6ff}
.hdr a{color:#8b949e;text-decoration:none;font-size:13px}
.ctr{max-width:1100px;margin:24px auto;padding:0 16px}
.stats{display:flex;gap:16px;margin-bottom:20px}
.stat{background:#161b22;border:1px solid #30363d;border-radius:8px;padding:16px 20px;flex:1;text-align:center}
.stat .n{font-size:28px;font-weight:700;color:#58a6ff}
.stat .l{font-size:12px;color:#8b949e;margin-top:4px}
.stat.warn .n{color:#d29922}
.stat.err .n{color:#f85149}
table{width:100%%;background:#161b22;border:1px solid #30363d;border-radius:8px;border-collapse:collapse;overflow:hidden}
th{background:#21262d;text-align:left;padding:10px 14px;font-size:12px;color:#8b949e;text-transform:uppercase;letter-spacing:.5px}
td{padding:10px 14px;border-top:1px solid #21262d;font-size:13px;font-family:'SF Mono',monospace}
tr:hover{background:#1c2128}
.scope{background:#1f6feb22;color:#58a6ff;padding:2px 8px;border-radius:4px;font-size:12px;margin-right:4px}
.exp{font-size:12px}
.exp.ok{color:#3fb950}
.exp.soon{color:#d29922}
.exp.dead{color:#f85149}
.rev-btn{background:#da363322;color:#f85149;border:1px solid #f8514944;padding:4px 12px;border-radius:6px;cursor:pointer;font-size:12px}
.rev-btn:hover{background:#da363355}
.empty{padding:40px;text-align:center;color:#8b949e}
.refresh{background:#21262d;color:#c9d1d9;border:1px solid #30363d;padding:6px 14px;border-radius:6px;cursor:pointer;font-size:13px}
</style>
</head>
<body>
<div class="hdr">
  <h1>🔑 Token Inspector</h1>
  <div><a href="/developer">← Developer Portal</a> &nbsp; <button class="refresh" onclick="load()">↻ Refresh</button></div>
</div>
<div class="ctr">
  <div class="stats">
    <div class="stat"><div class="n" id="s-total">-</div><div class="l">Active Tokens</div></div>
    <div class="stat warn"><div class="n" id="s-expiring">-</div><div class="l">Expiring &lt;10min</div></div>
    <div class="stat err"><div class="n" id="s-expired">-</div><div class="l">Expired</div></div>
  </div>
  <table>
    <thead><tr><th>Token ID</th><th>Client</th><th>Scopes</th><th>Created</th><th>Expires</th><th>Status</th><th></th></tr></thead>
    <tbody id="tbody"><tr><td colspan="7" class="empty">Loading...</td></tr></tbody>
  </table>
</div>
<script>
const PROXY=%q;
const token=sessionStorage.getItem('access_token')||'';

async function load(){
  try{
    const r=await fetch(PROXY+'/oauth/tokens',{headers:{'Authorization':'Bearer '+token}});
    if(!r.ok){document.getElementById('tbody').innerHTML='<tr><td colspan="7" class="empty">Auth required — <a href="/demo" style="color:#58a6ff">login first</a></td></tr>';return}
    const tokens=await r.json()||[];
    const now=Date.now();
    let total=0,expiring=0,expired=0;
    tokens.forEach(t=>{
      const exp=new Date(t.expires_at).getTime();
      if(exp<now)expired++;
      else{total++;if(exp-now<600000)expiring++}
    });
    document.getElementById('s-total').textContent=total;
    document.getElementById('s-expiring').textContent=expiring;
    document.getElementById('s-expired').textContent=expired;
    if(tokens.length===0){document.getElementById('tbody').innerHTML='<tr><td colspan="7" class="empty">No tokens</td></tr>';return}
    tokens.sort((a,b)=>new Date(b.created_at)-new Date(a.created_at));
    document.getElementById('tbody').innerHTML=tokens.map(t=>{
      const exp=new Date(t.expires_at).getTime();
      const rem=exp-now;
      let cls='ok',lbl=fmt(rem);
      if(rem<0){cls='dead';lbl='expired'}
      else if(rem<600000){cls='soon';lbl=fmt(rem)+' left'}
      return '<tr>'+
        '<td title="'+esc(t.id)+'">'+esc(t.id.substring(0,16))+'…</td>'+
        '<td>'+esc(t.client_id)+'</td>'+
        '<td>'+(t.scopes||[]).map(s=>'<span class="scope">'+esc(s)+'</span>').join('')+'</td>'+
        '<td>'+ago(t.created_at)+'</td>'+
        '<td><span class="exp '+cls+'">'+lbl+'</span></td>'+
        '<td>'+(rem<0?'<span class="exp dead">expired</span>':'<span class="exp ok">active</span>')+'</td>'+
        '<td>'+(rem>0?'<button class="rev-btn" onclick="revoke(\''+esc(t.id)+'\')">Revoke</button>':'')+'</td>'+
        '</tr>';
    }).join('');
  }catch(e){document.getElementById('tbody').innerHTML='<tr><td colspan="7" class="empty">Error: '+e.message+'</td></tr>'}
}

async function revoke(id){
  if(!confirm('Revoke token '+id.substring(0,16)+'…?'))return;
  await fetch(PROXY+'/oauth/revoke',{method:'POST',headers:{'Authorization':'Bearer '+token,'Content-Type':'application/x-www-form-urlencoded'},body:'token='+id});
  load();
}

function fmt(ms){if(ms<0)return'expired';const m=Math.floor(ms/60000);if(m<60)return m+'m';const h=Math.floor(m/60);return h+'h '+m%%60+'m'}
function ago(d){const m=Math.floor((Date.now()-new Date(d).getTime())/60000);if(m<1)return'just now';if(m<60)return m+'m ago';const h=Math.floor(m/60);return h+'h ago'}
function esc(s){return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;')}

load();
setInterval(load,15000);
</script>
</body></html>`
