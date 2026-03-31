# Kukicha Proxy — TODO

Supplements the `claude/kukicha-goproxy-integration-S603o` branch.

All items resolved as of commit `42b8d14`.

## Domain ✓

Switched from `proxy.kukicha.org` to `proxy.kukicha.dev` in `cmd/kukicha-proxy/main.go` and `cmd/kukicha/proxy.go`.

## Trusted Prefixes ✓

Default `--trusted` flag now includes `github.com/kukichalang/,golang.org/x/,gopkg.in/`.

## Implementation Issues

### 1. No disk caching ✓

`cachedProxyPassthrough` checks a local file cache before hitting upstream and writes responses to disk. `.mod` and `.zip` are served from cache on subsequent requests, enabling offline/air-gapped builds.

### 2. JSON-backed seenDB won't scale for hosted mode ✓

`seenStore` interface added. `--mode=hosted` selects `sqliteSeenStore` (backed by `glebarez/go-sqlite`); local mode retains the JSON-backed `seenDB`.

### 3. `.info` endpoint bypasses cooldown ✓

`handleInfo`, `handleMod`, and `handleZip` now call `firstSeen` (to start the clock) and return 404 for versions still within cooldown. Trusted prefixes bypass enforcement as before.

### 4. No sumdb proxying ✓

`handleSumDB` forwards `sumdb/*` paths to `sum.golang.org`, keeping module fetch patterns off Google's infrastructure.

### 5. No graceful shutdown ✓

Replaced `http.ListenAndServe` with `http.Server` + `Shutdown()`. SIGINT/SIGTERM handler calls `store.close()` before shutdown to flush pending writes.

### 6. 410 Gone handling is wrong ✓

`fetchUpstream` returns `errGone` for HTTP 410. All callers propagate it as 410 to the client instead of falling through to the next proxy.
