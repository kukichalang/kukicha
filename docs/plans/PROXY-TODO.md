# Kukicha Proxy — TODO

Supplements the `claude/kukicha-goproxy-integration-S603o` branch.

## Domain

Switch from `proxy.kukicha.org` to `proxy.kukicha.dev` throughout:
- `cmd/kukicha-proxy/main.go` (hosted mode default upstream)
- `cmd/kukicha/proxy.go` (`hostedProxyURL` constant)

## Trusted Prefixes

Default trusted list (bypass cooldown entirely):

| Prefix | Reason |
|--------|--------|
| `github.com/kukichalang/` | First-party packages |
| `golang.org/x/` | Go team maintained, same release process as stdlib |
| `gopkg.in/` | Stable versioned mirrors (yaml.v3, etc.) |

Everything else gets the 7-day cooldown.

Update the `--trusted` flag default in `cmd/kukicha-proxy/main.go`:
```go
trusted := flag.String("trusted",
    "github.com/kukichalang/,golang.org/x/,gopkg.in/",
    "comma-separated trusted module prefixes (bypass cooldown)")
```

## Implementation Issues

### 1. No disk caching

`cacheDir` is declared on the `proxy` struct but never used. Every request hits upstream. Adding a local file cache for `.mod` and `.zip` responses would:
- Enable offline/air-gapped builds
- Speed up repeated builds
- Reduce load on upstream proxies

### 2. JSON-backed seenDB won't scale for hosted mode

The doc comment mentions "SQLite for hosted" but it's not implemented. The JSON file is rewritten in full on every new version seen, and the entire map is held in memory. Fine for local mode, needs SQLite (or similar) for hosted.

### 3. `.info` endpoint bypasses cooldown

`handleInfo` records first-seen but always passes through to upstream. If a client requests `.info` -> `.mod` -> `.zip` directly (skipping `list`), the cooldown is never enforced. The GOPROXY spec allows this request path.

Fix: check cooldown in `handleInfo`, `handleMod`, and `handleZip` — return 404 for versions still in cooldown (unless trusted).

### 4. No sumdb proxying

The proxy doesn't handle `sum.golang.org` verification. Go will still check checksums via its defaults, but a hosted proxy should proxy the sumdb for completeness and to avoid leaking module fetch patterns to Google.

### 5. No graceful shutdown

`http.ListenAndServe` with no context or signal handling. The `stop` command sends SIGINT but the server doesn't catch it for a clean DB flush. Risk of losing recent first-seen records.

Fix: use `http.Server` with `Shutdown()`, catch SIGINT/SIGTERM, flush `seenDB` on exit.

### 6. 410 Gone handling is wrong

`fetchUpstream` treats HTTP 410 the same as 404. Per the GOPROXY spec, 410 means "permanently unavailable, do NOT try other proxies." The current code returns a generic error which makes Go fall through to the next proxy — wrong semantics.

Fix: return 410 to the client when upstream returns 410.
