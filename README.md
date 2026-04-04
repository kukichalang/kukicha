# Kukicha

Brewed from what Go leaves on the table. Kukicha is a **strict superset of Go** — rename `.go` to `.kuki` and it compiles unchanged. Then blend in features that didn't fit Go's minimalist philosophy: pipes, `onerr`, enums, if-expressions, readable operators. Not sure? `kukicha brew` gives you standard Go back. The stems dissolve, the tea remains.

**[kukicha.org](https://kukicha.org)** | [Quick Reference](docs/kukicha-quick-reference.md) | [Tutorials](https://kukicha.org/#tutorials) | [Stdlib Reference](docs/SKILL.md)

---

## Go vs Kukicha

```go
// Go — 8 lines of error ceremony
data, err := fetchData()
if err != nil {
    return fmt.Errorf("fetch: %w", err)
}
result, err := parse(data)
if err != nil {
    return fmt.Errorf("parse: %w", err)
}
```

```kukicha
// Kukicha — same thing, 2 lines
result := fetchData()
    |> parse() onerr return explain "pipeline failed"
```

Both are valid Kukicha. The Go version compiles as-is. The Kukicha version is what you graduate to.

---

## Quickstart

**Requires Go 1.26+** ([download](https://go.dev/dl/)) | Pre-built binaries on [GitHub Releases](https://github.com/kukichalang/kukicha/releases)

```bash
go install github.com/kukichalang/kukicha/cmd/kukicha@v0.1.0
mkdir myapp && cd myapp
kukicha init
```

Create `hello.kuki`:

```kukicha
function main()
    name := "World"
    print("Hello, {name}!")
```

```bash
kukicha run hello.kuki
```

### Adopt Gradually

```bash
# See what your Go code looks like with Kukicha idioms
kukicha-blend main.go

# Convert Go to Kukicha (preview first, then apply)
kukicha-blend --diff main.go
kukicha-blend --apply main.go

# Convert Kukicha back to Go anytime
kukicha brew main.kuki
```

### Commands

| Command | What it does |
|---------|-------------|
| `kukicha check file.kuki` | Validate syntax without compiling |
| `kukicha run file.kuki` | Compile and run immediately |
| `kukicha build file.kuki` | Compile to a standalone binary |
| `kukicha brew file.kuki` | Convert back to standalone Go |
| `kukicha fmt -w file.kuki` | Format in place |
| `kukicha-blend file.go` | Suggest Kukicha idioms for Go code |

---

## What Kukicha Adds

Go's philosophy is radical simplicity — and that's genuinely powerful. But some proven patterns from Rust, Elixir, Kotlin, and Python didn't fit that vision. Kukicha picks them up.

| Feature | Go | Kukicha |
|---------|-----|---------|
| **Error handling** | `if err != nil { return err }` | `onerr return` |
| **Pipes** | `f(g(h(x)))` | `x \|> h() \|> g() \|> f()` |
| **If-expressions** | 5-line temp var + if/else | `x := if cond then a else b` |
| **Readable operators** | `&&`, `\|\|`, `!` | `and`, `or`, `not` |
| **Type syntax** | `[]string`, `map[K]V`, `*T` | `list of string`, `map of K to V`, `reference T` |
| **Enums** | `const` + `iota` | `enum Status` with named variants |
| **Lambdas** | `func(x int) int { return x*2 }` | `(x int) => x * 2` |
| **String interpolation** | `fmt.Sprintf("hi %s", name)` | `"hi {name}"` |

All Go syntax is also accepted — Kukicha is a strict superset. Use whichever form you prefer.

---

## Why Not Just Write Go?

- **Zero adoption cost** — your existing `.go` files compile as `.kuki` unchanged
- **Zero lock-in** — `kukicha brew` converts back to standard Go anytime
- **Gradual migration** — blend in one feature at a time, leave the rest as Go
- **Security at compile time** — catches SQL injection, XSS, SSRF, path traversal, command injection, and open redirects at build time
- **42+ stdlib packages** — `fetch`, `slice`, `sort`, `mcp`, `llm`, `html`, `crypto`, `shell`, and [many more](docs/SKILL.md)
- **Ships as Go** — single binary, cross-compile, WASM support, full Go ecosystem

---

## Editor Support

- **VS Code:** Search `kukicha-lang` in extensions ([repo](https://github.com/kukichalang/vscode-kukicha))
- **Zed:** [kukichalang/zed-kukicha](https://github.com/kukichalang/zed-kukicha)
- **Other:** `make install-lsp` and point your editor at `kukicha-lsp`

---

## Documentation

- [Quick Reference](docs/kukicha-quick-reference.md) — Go-to-Kukicha translation table
- [Beginner Tutorial](docs/tutorials/beginner-tutorial.md) — first program, variables, functions
- [Agent Workflow Tutorial](docs/tutorials/agent-workflow-tutorial.md) — prompt AI, review, ship
- [Production Patterns](docs/tutorials/production-patterns-tutorial.md) — databases, auth, retry
- [FAQ](docs/faq.md) | [Contributing](docs/contributing.md)

---

**Version:** 0.1.0 | **License:** [MIT](LICENSE)
