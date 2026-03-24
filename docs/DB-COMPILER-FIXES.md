# Compiler Fixes Discovered During stdlib/db Implementation

Bugs and limitations found while building `stdlib/db`. Each section describes the issue, the workaround taken, and what fixing it would unlock.

---

## 1. No `defer func() { ... }()` (Deferred Anonymous Closure)

**Bug:** Kukicha's `defer` only accepts a direct function call (`defer rows.Close()`). Go's `defer func() { ... }()` pattern — used for panic recovery, cleanup with captured variables, and multi-step teardown — is not supported.

**Where it hit:** Transaction rollback-on-panic. The idiomatic Go pattern is:

```go
defer func() {
    if r := recover(); r != nil {
        tx.Rollback()
        panic(r) // re-panic after cleanup
    }
}()
```

**Workaround:** Extracted the entire transaction body into a named helper function `finishTx(sqlTx, fn)`. This means transactions don't recover from panics — if the callback panics, the `sql.Tx` is left dangling (Go's GC will eventually clean it up, but the connection is held until then).

**What fixing it would unlock:**
- Panic-safe transactions (`defer func() { recover(); rollback; re-panic }()`)
- Multi-step cleanup patterns (`defer func() { flush(); close(); log() }()`)
- Deferred closures that capture loop variables
- Parity with Go's most common `defer` patterns

**Suggested syntax:**
```kukicha
# Option A: defer + indented block (matches go block syntax)
defer
    if r := recover(); r != empty
        sqlTx.Rollback()
        panic(r)

# Option B: defer with arrow lambda
defer () =>
    if r := recover(); r != empty
        sqlTx.Rollback()
        panic(r)
```

---

## 2. No `if init; condition` (Init-Statement If)

**Bug:** Go's `if val, ok := m[key]; ok { ... }` is not supported. The parser doesn't recognize the semicolon-separated init statement inside an `if`.

**Where it hit:** Column-to-field mapping in `structScanners`. The natural pattern for map lookups is:

```kukicha
# This doesn't parse
if idx, ok := tagMap[col]; ok
    return elem.Field(idx).Addr().Interface()
```

**Workaround:** Split into two statements, or extract into a helper function:

```kukicha
# Two statements
idx, ok := tagMap[col]
if ok
    return elem.Field(idx).Addr().Interface()
```

For the column matcher, I extracted a separate `matchColumnToField` function to keep the code readable.

**What fixing it would unlock:**
- Idiomatic map-lookup-and-use in a single expression
- Type assertions with immediate check: `if v, ok := x.(string); ok`
- Cleaner code in any map-heavy logic (config parsing, routing, caching)
- Closer parity with Go patterns that beginners see in tutorials

**Suggested syntax:**
```kukicha
if idx, ok := tagMap[col]; ok
    ptrs[ci] = elem.Field(idx).Addr().Interface()
```

The parser would need to recognize `;` (or a keyword like `then`) as separating an init statement from the condition within an `if`.

---

## 3. Block Lambda `return` Validates Against Enclosing Function

**Bug:** When a block arrow lambda has a different return type from its enclosing function, `return` statements inside the lambda are validated against the *outer* function's signature, not the lambda's.

**Where it hit:** Transaction callbacks in tests. The `db.Transaction` callback returns `error`, but test functions return nothing:

```kukicha
func TestTransaction(t reference testing.T)
    # This fails semantic analysis:
    # "onerr return requires the enclosing function to return an error"
    # "expected 0 return values, got 1"
    err := db.Transaction(pool, (tx db.Tx) =>
        db.TxExec(tx, "INSERT ...", args) onerr return
        return empty
    )
```

The semantic analyzer sees `return empty` and checks it against `TestTransaction`'s return type (void), not the lambda's return type (`error`).

**Workaround:** Extract the lambda body into a named top-level function:

```kukicha
func txInsertDiana(tx db.Tx) error
    _, err := db.TxExec(tx, "INSERT ...", args)
    return err

func TestTransaction(t reference testing.T)
    err := db.Transaction(pool, txInsertDiana)
```

**What fixing it would unlock:**
- Inline transaction callbacks: `db.Transaction(pool, (tx) => ... onerr return ... return empty)`
- `onerr return` inside lambdas (the most ergonomic error propagation)
- Any callback pattern where the callback's return type differs from the caller's
- The full promise of the `db.Transaction` closure-based API as designed

**Impact:** This is the most impactful fix for `stdlib/db` ergonomics. The whole point of the closure-based transaction API is inline usage. Without this fix, users must extract every transaction body into a named function, which defeats the convenience.

---

## 4. Pipe Chain Without `onerr` Wraps in IIFE Returning `any`

**Bug:** When a pipe chain has no `onerr` handler, the compiler wraps multi-return steps in an IIFE: `func() any { val, _ := f(); return val }()`. The return type is `any`, losing the concrete type. This causes `go vet` errors when the next step expects a concrete type.

**Where it hit:** Piping `db.Query` into `db.ScanOne` without `onerr`:

```kukicha
# This generates code that fails go vet:
result, err := db.Query(pool, "SELECT ...") |> db.ScanOne(User{})
```

Generated Go:
```go
result, err := db.ScanOne(func() any { val, _ := db.Query(...); return val }(), User{})
// go vet: cannot use (func() any literal)() as db.Rows
```

With pipeline-level `onerr`, it correctly generates intermediate variables with proper types:
```go
pipe_1, err_2 := db.Query(...)
// err check...
result, err_3 := db.ScanOne(pipe_1, User{})
```

**Workaround:** Always use `onerr` on pipe chains involving error-returning functions (which you should do anyway for database calls).

**What fixing it would unlock:**
- Pipe chains that silently discard errors (rare but valid for fire-and-forget patterns)
- Consistent behavior whether `onerr` is present or not
- No surprising `go vet` failures for seemingly valid Kukicha code

**Suggested fix:** The IIFE should return the concrete type instead of `any`. The compiler already knows the return types from `exprReturnCounts` / `exprTypes` — the IIFE's return type should match the first return value's type.

---

## Priority Order

1. **Lambda return scoping (#3)** — Highest impact. Blocks the primary use case of `db.Transaction` inline callbacks. Affects any stdlib that uses callback patterns with non-void return types.
2. **Deferred anonymous closures (#1)** — Needed for panic-safe transactions and idiomatic Go resource cleanup patterns.
3. **IIFE type erasure (#4)** — Causes confusing `go vet` failures. Low practical impact since `onerr` is almost always used.
4. **Init-statement if (#2)** — Quality-of-life improvement. Workaround is trivial (split into two lines).
