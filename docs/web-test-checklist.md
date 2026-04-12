# Web UI Test Checklist

Manual test checklist for all oauth4os web pages.

## Pages

| # | Page | Path | Status |
|---|------|------|--------|
| 1 | Landing | `/` | ✅ |
| 2 | Demo Dashboard | `/demo/` | ✅ |
| 3 | Service Map | `/demo/services.html` | ✅ |
| 4 | Alerts | `/demo/alerts.html` | ✅ |
| 5 | Trace Viewer | `/demo/trace.html` | ✅ |
| 6 | PKCE Playground | `/demo/playground.html` | ✅ |
| 7 | Developer Portal | `/developer/` | ✅ |
| 8 | Admin Console | `/admin/` | ✅ |
| 9 | Token Analytics | `/analytics/` | ✅ |
| 10 | Cedar Playground | `/playground/` | ✅ |
| 11 | Benchmark | `/benchmark/` | ✅ |
| 12 | Status | `/status/` | ✅ |
| 13 | Changelog | `/changelog/` | ✅ |
| 14 | 404 | `/404.html` | ✅ |

## Per-Page Checks

### Functional
- [ ] Page loads without JS errors
- [ ] Data fetches complete (or graceful fallback)
- [ ] All buttons/links work
- [ ] Forms submit correctly
- [ ] Error states show friendly message

### Accessibility
- [ ] All interactive elements have aria-labels
- [ ] Keyboard navigation works (Tab, Enter, Escape)
- [ ] Focus visible on all focusable elements
- [ ] `lang="en"` on `<html>`
- [ ] Color contrast passes WCAG AA

### Responsive
- [ ] 1440px (desktop) — no overflow
- [ ] 768px (tablet) — sidebar collapses
- [ ] 375px (mobile) — single column, readable

### Theme
- [ ] Dark mode renders correctly
- [ ] Light mode renders correctly
- [ ] Theme persists across page navigation
- [ ] Theme persists across reload

### Security
- [ ] No XSS: all user/API data escaped via `esc()`
- [ ] No inline event handlers with user data
- [ ] Tokens stored in sessionStorage (not localStorage)
- [ ] 401 response triggers logout

### Performance
- [ ] No unnecessary re-fetches
- [ ] Auto-refresh intervals reasonable (≥15s)
- [ ] Skeletons shown during loading
- [ ] No layout shift on data load
