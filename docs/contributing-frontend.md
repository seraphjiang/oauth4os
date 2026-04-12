# Contributing to oauth4os Web UI

Guide for frontend contributors working on the oauth4os web pages.

## Architecture

All web UIs are single HTML files with zero dependencies:

```
web/
├── index.html          # Landing page
├── 404.html            # Error page
├── robots.txt          # SEO
├── sitemap.xml         # SEO
├── admin/index.html    # Admin console (8 tabs)
├── analytics/index.html # Token analytics dashboard
├── benchmark/index.html # Performance benchmark tool
├── changelog/index.html # Release history (parses CHANGELOG.md)
├── demo/
│   ├── index.html      # Log viewer dashboard (PWA)
│   ├── services.html   # Service map (canvas)
│   ├── alerts.html     # Alert feed
│   ├── trace.html      # Trace waterfall viewer
│   ├── playground.html # PKCE flow walkthrough
│   ├── manifest.json   # PWA manifest
│   └── sw.js           # Service worker
├── developer/index.html # OAuth app management
├── playground/index.html # Cedar policy playground
├── setup/index.html    # Interactive setup wizard
└── status/index.html   # System status page
```

## Conventions

### CSS Variables (dark/light theme)

```css
:root {
  --bg: #0d1117;      --surface: #161b22;
  --border: #30363d;  --text: #e6edf3;
  --muted: #8b949e;   --accent: #58a6ff;
  --green: #3fb950;   --orange: #f0883e;
  --purple: #bc8cff;  --red: #f85149;
}
[data-theme="light"] { /* override all vars */ }
```

### Theme Toggle

```js
document.getElementById('themeToggle').addEventListener('click', () => {
  const t = document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark';
  document.documentElement.dataset.theme = t;
  localStorage.setItem('oauth4os_theme', t);
});
const saved = localStorage.getItem('oauth4os_theme');
if (saved) document.documentElement.dataset.theme = saved;
```

### XSS Prevention

Always escape user/API data:

```js
function esc(s) {
  const d = document.createElement('div');
  d.textContent = s || '';
  return d.innerHTML;
}
// Use: innerHTML = `<td>${esc(userInput)}</td>`
```

### Token Management

```js
// Get token (shared via sessionStorage)
const token = sessionStorage.getItem('oauth4os_token');

// Auto-logout on 401
if (resp.status === 401) {
  sessionStorage.clear();
  token = null;
  showLogin();
  return;
}
```

## Checklist for New Pages

- [ ] `lang="en"` on `<html>`
- [ ] `<meta name="description">`
- [ ] `<meta name="viewport">`
- [ ] Dark/light theme with CSS variables
- [ ] Theme persisted in localStorage
- [ ] All interactive elements have `aria-label`
- [ ] `:focus-visible` outline style
- [ ] `esc()` function for all dynamic content
- [ ] 375px responsive breakpoint
- [ ] Loading skeletons for async data
- [ ] Error boundary (global error handler)
- [ ] Add to `sitemap.xml` if public
