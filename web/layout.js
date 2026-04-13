// oauth4os shared layout — inject sidebar + navbar on all pages
// Include via <script src="/web/layout.js"></script>
(function(){
const API=window.location.origin;
const path=window.location.pathname;

const NAV=[
  {group:'OAuth',items:[
    {href:'/register/',icon:'📝',label:'Register App'},
    {href:'/admin/clients',icon:'👥',label:'My Apps'},
    {href:'/tutorial/',icon:'📖',label:'Tutorial'},
    {href:'/explorer/',icon:'🔬',label:'API Explorer'},
    {href:'/playground/',icon:'🧪',label:'Cedar Playground'},
  ]},
  {group:'Admin',items:[
    {href:'/admin/',icon:'⚙️',label:'Console'},
    {href:'/admin/tokens',icon:'🔑',label:'Tokens'},
    {href:'/admin/policies',icon:'📜',label:'Policies'},
    {href:'/admin/keys',icon:'🔐',label:'Keys'},
    {href:'/admin/config/',icon:'🛠',label:'Config'},
    {href:'/admin/tenants',icon:'🏢',label:'Tenants'},
    {href:'/admin/feedback',icon:'💬',label:'Feedback'},
  ]},
  {group:'Monitoring',items:[
    {href:'/dashboard/',icon:'📊',label:'Dashboard'},
    {href:'/logs/',icon:'📋',label:'Live Logs'},
    {href:'/status/',icon:'🟢',label:'Status'},
    {href:'/analytics/',icon:'📈',label:'Analytics'},
  ]},
  {group:'Demo',items:[
    {href:'/demo/',icon:'🖥',label:'Log Viewer'},
    {href:'/demo/services.html',icon:'🗺',label:'Service Map'},
    {href:'/demo/alerts.html',icon:'🔔',label:'Alerts'},
    {href:'/demo/trace.html',icon:'⏱',label:'Traces'},
  ]},
  {group:'Help',items:[
    {href:'/setup/',icon:'🚀',label:'Getting Started'},
    {href:'/changelog/',icon:'📋',label:'Changelog'},
    {href:'/.well-known/openid-configuration',icon:'🔍',label:'Discovery'},
    {href:'/my-feedback',icon:'💬',label:'My Feedback'},
  ]},
];

// CSS
const css=document.createElement('style');
css.textContent=`
.layout-wrap{display:flex;min-height:100vh}
.layout-sidebar{width:240px;background:var(--surface,#161b22);border-right:1px solid var(--border,#30363d);position:fixed;top:0;left:0;bottom:0;overflow-y:auto;z-index:100;transition:transform .2s}
.layout-sidebar.closed{transform:translateX(-240px)}
.layout-main{margin-left:240px;flex:1;min-width:0}
.layout-topbar{height:44px;border-bottom:1px solid var(--border,#30363d);display:flex;align-items:center;padding:0 16px;gap:12px;background:var(--surface,#161b22);position:sticky;top:0;z-index:99}
.layout-topbar .logo{font-size:15px;font-weight:800;color:var(--accent,#58a6ff);text-decoration:none}
.layout-topbar .version{font-size:10px;padding:2px 6px;border-radius:8px;background:var(--accent,#58a6ff);color:#fff;font-weight:600}
.layout-topbar .health{font-size:11px;margin-left:auto}
.layout-topbar .theme-toggle{background:none;border:1px solid var(--border,#30363d);border-radius:4px;padding:2px 6px;cursor:pointer;font-size:13px;color:var(--text,#e6edf3)}
.layout-hamburger{display:none;background:none;border:none;font-size:20px;cursor:pointer;color:var(--text,#e6edf3);padding:4px}
.sidebar-logo{padding:12px 16px;font-size:16px;font-weight:800;border-bottom:1px solid var(--border,#30363d)}
.sidebar-logo a{color:var(--text,#e6edf3);text-decoration:none}
.sidebar-logo span{color:var(--accent,#58a6ff)}
.sidebar-group{padding:8px 0}
.sidebar-group-label{padding:4px 16px;font-size:10px;font-weight:700;text-transform:uppercase;letter-spacing:.5px;color:var(--muted,#8b949e)}
.sidebar-link{display:flex;align-items:center;gap:8px;padding:6px 16px;font-size:12px;color:var(--muted,#8b949e);text-decoration:none;border-radius:0}
.sidebar-link:hover{background:var(--bg,#0d1117);color:var(--text,#e6edf3)}
.sidebar-link.active{background:var(--bg,#0d1117);color:var(--accent,#58a6ff);font-weight:600;border-right:2px solid var(--accent,#58a6ff)}
.sidebar-link .icon{width:18px;text-align:center;font-size:14px}
.layout-breadcrumb{padding:8px 16px;font-size:11px;color:var(--muted,#8b949e);border-bottom:1px solid var(--border,#30363d)}
.layout-breadcrumb a{color:var(--muted,#8b949e);text-decoration:none}
.layout-breadcrumb a:hover{color:var(--accent,#58a6ff)}
@media(max-width:768px){
  .layout-sidebar{transform:translateX(-240px)}
  .layout-sidebar.open{transform:translateX(0)}
  .layout-main{margin-left:0}
  .layout-hamburger{display:block}
  .sidebar-overlay{position:fixed;inset:0;background:rgba(0,0,0,.5);z-index:99;display:none}
  .sidebar-overlay.open{display:block}
}
`;
document.head.appendChild(css);

// Build sidebar HTML
let sidebarHTML=`<div class="sidebar-logo"><a href="/">🔐 oauth<span>4os</span></a></div>`;
NAV.forEach(g=>{
  sidebarHTML+=`<div class="sidebar-group"><div class="sidebar-group-label">${g.group}</div>`;
  g.items.forEach(i=>{
    const active=path===i.href||path===i.href+'index.html'||(i.href!=='/'+' '&&path.startsWith(i.href)&&i.href.length>2);
    sidebarHTML+=`<a href="${i.href}" class="sidebar-link${active?' active':''}"><span class="icon">${i.icon}</span>${i.label}</a>`;
  });
  sidebarHTML+=`</div>`;
});

// Find breadcrumb
let crumb='Home';
NAV.forEach(g=>g.items.forEach(i=>{if(path===i.href||path.startsWith(i.href)&&i.href.length>2)crumb=g.group+' / '+i.label}));

// Inject
const body=document.body;
const existingContent=body.innerHTML;

// Remove existing headers/navs
const wrapper=document.createElement('div');
wrapper.className='layout-wrap';
wrapper.innerHTML=`
<div class="sidebar-overlay" id="sidebarOverlay"></div>
<aside class="layout-sidebar" id="layoutSidebar">${sidebarHTML}</aside>
<div class="layout-main">
  <div class="layout-topbar">
    <button class="layout-hamburger" id="hamburger" aria-label="Toggle menu">☰</button>
    <a href="/" class="logo">oauth4os</a>
    <span class="version" id="layoutVersion">v2</span>
    <span class="health" id="layoutHealth">⏳</span>
    <button class="theme-toggle" id="layoutTheme" aria-label="Toggle theme">🌓</button>
  </div>
  <div class="layout-breadcrumb"><a href="/">Home</a> / ${crumb}</div>
  <div id="layoutContent"></div>
</div>`;

body.innerHTML='';
body.appendChild(wrapper);
document.getElementById('layoutContent').innerHTML=existingContent;

// Remove old headers that got moved into content
document.querySelectorAll('#layoutContent > header, #layoutContent > .layout-topbar').forEach(el=>el.remove());

// Health check
fetch(API+'/health').then(r=>r.json()).then(d=>{
  document.getElementById('layoutHealth').innerHTML=d.status==='ok'?'<span style="color:var(--green,#3fb950)">● Healthy</span>':'<span style="color:var(--red,#f85149)">● Unhealthy</span>';
}).catch(()=>{document.getElementById('layoutHealth').innerHTML='<span style="color:var(--muted)">● Unknown</span>'});

// Version
fetch(API+'/version').then(r=>r.json()).then(d=>{
  document.getElementById('layoutVersion').textContent=d.version||'v2';
}).catch(()=>{});

// Theme
document.getElementById('layoutTheme').addEventListener('click',()=>{
  const t=document.documentElement.dataset.theme==='dark'?'light':'dark';
  document.documentElement.dataset.theme=t;
  localStorage.setItem('oauth4os_theme',t);
});

// Hamburger
document.getElementById('hamburger').addEventListener('click',()=>{
  document.getElementById('layoutSidebar').classList.toggle('open');
  document.getElementById('sidebarOverlay').classList.toggle('open');
});
document.getElementById('sidebarOverlay').addEventListener('click',()=>{
  document.getElementById('layoutSidebar').classList.remove('open');
  document.getElementById('sidebarOverlay').classList.remove('open');
});
})();
