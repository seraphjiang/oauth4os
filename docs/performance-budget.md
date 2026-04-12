# Web Performance Budget

Target: all pages load in <1s on 3G, <200ms on broadband.

## Page Size Budget

| Page | Size | Budget | Status |
|------|------|--------|--------|
| Landing | 31KB | <50KB | ✅ |
| Demo Dashboard | 35KB | <50KB | ✅ |
| Admin Console | 23KB | <30KB | ✅ |
| PKCE Playground | 22KB | <30KB | ✅ |
| Developer Portal | 16KB | <20KB | ✅ |
| Trace Viewer | 14KB | <20KB | ✅ |
| Alert Feed | 14KB | <20KB | ✅ |
| Cedar Playground | 12KB | <15KB | ✅ |
| Analytics | 11KB | <15KB | ✅ |
| Benchmark | 11KB | <15KB | ✅ |
| Service Map | 10KB | <15KB | ✅ |
| Setup Wizard | 7KB | <10KB | ✅ |
| Status | 5KB | <10KB | ✅ |
| Changelog | 4KB | <10KB | ✅ |
| 404 | 1KB | <5KB | ✅ |

Total: 254KB across 15 pages (avg 17KB/page).

## Rules

1. Zero external dependencies (no CDN, no npm)
2. All CSS inline in `<style>` (no FOUC)
3. All JS inline in `<script>` (no extra requests)
4. No images — use emoji/SVG/canvas only
5. Gzip: ~60-70% compression ratio expected

## Network Budget

| Metric | Budget | Actual |
|--------|--------|--------|
| First Contentful Paint | <500ms | ~100ms (inline CSS) |
| Time to Interactive | <1s | ~200ms (no external deps) |
| Total Transfer Size | <50KB/page | avg 17KB |
| HTTP Requests | 1 (HTML only) | 1 ✅ |
| External Dependencies | 0 | 0 ✅ |

## Monitoring

Check page sizes after changes:
```bash
wc -c web/**/*.html web/*.html | sort -rn | head -20
```

Flag any page exceeding its budget in code review.
