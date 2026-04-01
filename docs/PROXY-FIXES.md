# Proxy Branch — Issues to Address

Tracked from review of `claude/kukicha-goproxy-integration-S603o`.

Rewritten from Go to Kukicha (`cmd/kukicha-proxy/main.kuki`).

## Fixed

### 0. Wrong domain
Changed `proxy.kukicha.dev` to `proxy.kukicha.org` in `main.kuki` and `cmd/kukicha/proxy.go`.

### 1. Missing SQLite go.mod dependency
`github.com/glebarez/go-sqlite` is now imported via `import "..." as _` in the `.kuki` source. Dependency resolved by `kukicha build` via `go mod tidy`.

### 3. Non-atomic JSON persistence
`persist()` now writes to a `.tmp` file and renames atomically via `os.Rename`.

## Still To Do

### 2. GOPROXY override is implicit
`buildProxyChain()` is called unconditionally in `buildCommand` and `runCommand`, overriding GOPROXY for every user. This should either respect an existing GOPROXY setting or be opt-in.

### 4. Daemon doesn't survive terminal close
`proxyStartCommand` daemon mode uses `cmd.Start()` without `SysProcAttr{Setsid: true}`. The proxy dies when the parent terminal closes.

### 5. No proxy subcommand tests
`proxyCommand` routing in `cmd/kukicha/main.go` has no test coverage.

### 6. `kukicha audit` doesn't use proxy chain
`GOPROXY` is set for `build` and `run` but not for `auditCommand`.

### 7. Test file needs updating
`cmd/kukicha-proxy/main_test.go` references old method names (`firstSeen`, `close`) that were renamed to `FirstSeen`, `Shutdown` for interface satisfaction. Update the test file.

## Minor

- Double blank line in `cmd/kukicha/init.go` (lines 66-67)
- `parseDuration` only handles integer days (`7d`), not fractional (`1.5d`)
- No rate limiting on the proxy server

## Compiler Limitations Found

- ~~**Interface return type**: Kukicha semantic checker rejects `return concretePtr` when function returns an interface type. Workaround: append to `list of Interface` and return `list[0]`.~~ **Fixed** — `typeAnnotationToTypeInfo` now resolves named types as interfaces when they match a user-defined or Go stdlib interface.
- ~~**`onerr discard` on multi-return**: `w.Write(data) onerr discard` generates `_ = w.Write(data)` but `Write` returns `(int, error)`. Workaround: use `_, _ = w.Write(data)`.~~ **Fixed** — fallback now emits a bare call, valid for any return count.
- ~~**`close` is a keyword**: Can't use `close` as a method name. Renamed to `Shutdown`.~~ **Fixed** — `close` is now accepted as an identifier in name contexts (method names, field names).
- ~~**`0o755` octal literals**: Not recognized by the lexer. Use `0755` instead.~~ **Fixed** — lexer now supports `0o`, `0x`, and `0b` prefixes.
- **`onerr as e` standalone**: `{e}` works in string interpolation but `e` is not accessible as a standalone identifier (e.g., `log.Fatal(e)` fails). Use `onerr panic "{error}"` instead.
- ~~**Switch scope sharing**: Variables declared in different `when` branches share scope, causing redeclaration errors. Use `if/else if` instead.~~ **Fixed** — each `when` branch now has its own scope.
- ~~**Type inside function**: `type` declarations inside function bodies are not supported. Must be at top level.~~ **Fixed** — parser now emits a clear error message instead of silently failing.
