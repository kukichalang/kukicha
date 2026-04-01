# Proxy Branch — Issues to Address

Tracked from review of `claude/kukicha-goproxy-integration-S603o`.

## Must Fix

### 0. Wrong domain
Branch uses `proxy.kukicha.dev` throughout — we own `kukicha.org`, not `kukicha.dev`. Change to `proxy.kukicha.org` in `cmd/kukicha-proxy/main.go` and `cmd/kukicha/proxy.go`.

### 1. Missing SQLite go.mod dependency
`github.com/glebarez/go-sqlite` is imported in `cmd/kukicha-proxy/main.go` but not added to `go.mod`. Build will fail.

### 2. GOPROXY override is implicit
`buildProxyChain()` is called unconditionally in `buildCommand` and `runCommand`, overriding GOPROXY for every user. This should either respect an existing GOPROXY setting or be opt-in.

### 3. Non-atomic JSON persistence
`seenDB.persist()` writes directly to `seen.json`. A crash mid-write corrupts the file. Write to a temp file and `os.Rename` instead.

## Should Fix

### 4. Daemon doesn't survive terminal close
`proxyStartCommand` daemon mode uses `cmd.Start()` without `SysProcAttr{Setsid: true}`. The proxy dies when the parent terminal closes.

### 5. No proxy subcommand tests
`proxyCommand` routing in `cmd/kukicha/main.go` has no test coverage.

### 6. `kukicha audit` doesn't use proxy chain
`GOPROXY` is set for `build` and `run` but not for `auditCommand`.

## Minor

- Double blank line in `cmd/kukicha/init.go` (lines 66-67)
- `parseDuration` only handles integer days (`7d`), not fractional (`1.5d`)
- No rate limiting on the proxy server
