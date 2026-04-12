const CACHE = 'logview-v1';
const ASSETS = ['/demo/', '/demo/services.html', '/demo/alerts.html', '/demo/trace.html', '/demo/playground.html'];

self.addEventListener('install', e => {
  e.waitUntil(caches.open(CACHE).then(c => c.addAll(ASSETS)).then(() => self.skipWaiting()));
});

self.addEventListener('activate', e => {
  e.waitUntil(caches.keys().then(ks => Promise.all(ks.filter(k => k !== CACHE).map(k => caches.delete(k)))).then(() => self.clients.claim()));
});

self.addEventListener('fetch', e => {
  const url = new URL(e.request.url);
  // Network-first for API calls, cache-first for static pages
  if (url.pathname.startsWith('/oauth/') || url.pathname.startsWith('/logs') || url.pathname.startsWith('/admin/') || url.pathname.startsWith('/metrics')) {
    e.respondWith(fetch(e.request).catch(() => new Response(JSON.stringify({ error: 'offline' }), { headers: { 'Content-Type': 'application/json' } })));
  } else {
    e.respondWith(caches.match(e.request).then(r => r || fetch(e.request).then(resp => {
      if (resp.ok) { const c = resp.clone(); caches.open(CACHE).then(cache => cache.put(e.request, c)); }
      return resp;
    }).catch(() => caches.match('/demo/'))));
  }
});
