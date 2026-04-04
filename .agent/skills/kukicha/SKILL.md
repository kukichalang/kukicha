---
name: kukicha
description: Help write, debug, and understand Kukicha code - a strict superset of Go with pipes, onerr, enums, and readable operators. Use when working with .kuki files, discussing Kukicha syntax, error handling with onerr, pipe operators, or the Kukicha compiler/transpiler.
---

# Kukicha Language Skill

Kukicha (茎) is a strict superset of Go — all valid Go compiles as-is. Kukicha adds pipes, `onerr`, enums, if-expressions, and readable operators on top. Full language reference is in `docs/SKILL.md`.

**For compiler errors and diagnostics**, read `.agent/skills/kukicha/troubleshooting.md`.

## Gotchas

### `{error}` vs `{err}` in onerr blocks

Inside any `onerr` handler, the caught error is always named `error`, never `err`. Using `{err}` is a **compile-time error**. To use a custom name, write `onerr as <ident>` — then both `{error}` and `{<ident>}` are valid inside that block.

```kukicha
# CORRECT — canonical name
result := fetch.Get(url) onerr
    print("failed: {error}")
    return

# CORRECT — named alias (onerr as e)
result := fetch.Get(url) onerr as e
    print("failed: {e}")    # {e} and {error} both work here
    return

# WRONG — compiler rejects {err} inside onerr
result := fetch.Get(url) onerr
    print("failed: {err}")    # error: use {error} not {err} inside onerr
    return
```

### `kukicha init` required before stdlib imports

```bash
kukicha init    # run once per project: go mod init + extract stdlib
```

### Auto-imports for interpolated strings

The compiler auto-imports `fmt` when any string interpolation is used, including `error ""` literals with `{expr}`. No manual `import "fmt"` is needed.

```kukicha
# fmt is auto-imported — no manual import needed
func doThing(name string) error
    return error "failed for {name}"
```

### `in` is not a membership operator

```kukicha
# WRONG
if item in items
    ...

# CORRECT
if slices.Contains(items, item)
    ...

# 'in' only works in for loops
for item in items
    process(item)
```

### `fetch.Json` — compile-time type hint, not a runtime value

| Argument | Decodes |
|----------|---------|
| `fetch.Json(list of Repo)` | JSON array → `[]Repo` |
| `fetch.Json(empty Repo)` | JSON object → `Repo` |
| `fetch.Json(map of string to string)` | JSON object → `map[string]string` |

Wrong shape = runtime decode error with no compile-time warning.

### Struct literals must be inline — no multiline form

```kukicha
# CORRECT
todo := Todo{id: 1, title: "Learn Kukicha", completed: false}

# WRONG — multiline struct literals do not parse
todo := Todo{
    id: 1,
    title: "Learn Kukicha",
}
```

### Piped switch — pipe a value into a switch

```kukicha
user.Role |> switch
    when "admin"
        grantAccess()
    when "guest"
        denyAccess()
    otherwise
        checkPermissions()
```

The compiler wraps the switch in an IIFE: `func() { switch role { ... } }()`.

### Pipeline-level onerr — onerr at end of pipe chains

```kukicha
processed := data
    |> parse.Json(list of User)
    |> fetch.EnrichWithDB()
    |> validate.Safe()
    onerr panic "pipeline failed: {error}"
```

If *any* function in the pipe returns a Go `error`, the pipeline short-circuits to the `onerr` block. The compiler generates `if err != nil` checks between each stage.

### `stdlib/iterator` — lazy iteration via Go 1.23 iter.Seq

```kukicha
import "stdlib/iterator"
names := repos
    |> iterator.Values()
    |> iterator.Filter((r Repo) => r.Stars > 100)
    |> iterator.Map((r Repo) => r.Name)
    |> iterator.Take(5)
    |> iterator.Collect()
```

Functions: `Values`, `Filter`, `Map`, `FlatMap`, `Take`, `Skip`, `Enumerate`, `Chunk`, `Zip`, `Reduce`, `Collect`, `Any`, `All`, `Find`.

### Enums — declaration, dot access, and exhaustiveness

```kukicha
enum Status
    OK = 200
    NotFound = 404
    Error = 500

status := Status.OK    # Dot access → transpiles to StatusOK in Go

switch status
    when Status.OK
        print("ok")
    when Status.NotFound
        print("not found")
    when Status.Error
        print("error")
    # Omitting a case without 'otherwise' → compiler warning
```

- Underlying type (int or string) inferred from values — all cases must match
- Integer enums warn if no case has value 0 (zero-value safety)
- Auto-generated `String()` method (skipped if user defines one)
- Cross-package enums from stdlib are resolved automatically

### Shorthand `.Field` / `.Method()` — pipe context only

```kukicha
# CORRECT — shorthand in a pipe
name := user |> .Name
result := data |> .Process()

# WRONG — shorthand outside a pipe (compile error)
name := .Name
result := .Process()
```

Shorthand dot syntax is syntactic sugar for pipe receivers. Using it outside a pipe expression is a compile error: `shorthand .X can only be used in a pipe expression`.

### `any2` in stdlib source is a compiler placeholder — not user syntax

When reading stdlib `.kuki` files you will see `any2` in function signatures. Do not use it in application code — it is a compiler-reserved name for a second generic type parameter.
