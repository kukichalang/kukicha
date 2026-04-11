# Kukicha

Brewed from what Go leaves on the table. Kukicha is a **strict superset of Go**, rename `.go` to `.kuki` and it compiles unchanged. Then blend in features that didn't fit Go's minimalist philosophy: pipes, `onerr`, enums, if-expressions, readable operators. Not sure? `kukicha brew` gives you standard Go back. The stems dissolve and the tea remains.

**[kukicha.org](https://kukicha.org)** | [Quick Reference](docs/kukicha-quick-reference.md) | [Tutorials](https://kukicha.org/#tutorials) | [Stdlib Reference](docs/SKILL.md)

---

## A taste of Kukicha

Triage open GitHub issues with an LLM. Fetch, classify in parallel, keep the urgent ones, sort, print — end to end in 40 lines, no `if err != nil` ladder.

```kukicha
# triage.kuki — classify open issues with Claude, flag the urgent ones
import "stdlib/concurrent"
import "stdlib/fetch"
import "stdlib/json" as jsonpkg
import "stdlib/llm"
import "stdlib/slice"
import "stdlib/sort"

type Issue
    number int
    title string
    body string

type Verdict
    number int
    severity int # 1 = trivial .. 5 = on fire
    kind string # bug | feature | docs | question
    summary string

func triage(i Issue) Verdict
    reply := llm.New("anthropic:claude-sonnet-4-6")
        |> llm.JSONMode()
        |> llm.System("Classify GitHub issues. Reply JSON: \{severity:1-5, kind, summary\}")
        |> llm.Ask("{i.title}\n\n{i.body}") onerr return Verdict{}

    v := Verdict{number: i.number}
    jsonpkg.UnmarshalString(reply, reference of v) onerr return Verdict{}
    return v

func main()
    issues := fetch.Get("https://api.github.com/repos/golang/go/issues?per_page=20")
        |> fetch.CheckStatus()
        |> fetch.Json(empty list of Issue) onerr panic "github: {error}"

    urgent := issues
        |> concurrent.MapWithLimit(4, triage)
        |> slice.Filter(v => v.severity >= 4)
        |> sort.ByKey(v => -v.severity)

    print("Needs attention:")
    for v in urgent
        print("  [P{v.severity}] {v.kind}  #{v.number}  {v.summary}")
```

Reads like the English description above it. Underneath: typed HTTP→JSON decode with `fetch.Json(list of Issue)`, a pipeline-level `onerr` that catches network, status, and decode in one handler, an LLM builder composed with pipes, structured output decoded straight into a `Verdict` struct, bounded parallelism without goroutine or errgroup bookkeeping, and stdlib `Filter`/`sort.ByKey` chained on the result. Every `err != nil` you'd write in Go is absorbed by `onerr`.

All valid Go is still valid Kukicha — rename `.go` to `.kuki` and it compiles unchanged. 

---

## Quickstart

**Requires Go 1.26+** ([download](https://go.dev/dl/)) | Pre-built binaries on [GitHub Releases](https://github.com/kukichalang/kukicha/releases)

```bash
go install github.com/kukichalang/kukicha/cmd/kukicha@v0.1.5
mkdir myapp && cd myapp
kukicha init
```

`kukicha init` initializes a Go module (if `go.mod` is absent), extracts the stdlib, downloads dependencies, and writes an `AGENTS.md` language reference. Add `.kukicha/` to your `.gitignore`.

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

Go's philosophy is radical simplicity. Some proven patterns from Rust, Elixir, Kotlin, and Python didn't fit that vision. Kukicha picks them up.

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

All Go syntax is also accepted, Kukicha is a strict superset. 

---

## What does Kukicha offer?

- Existing `.go` files compile as `.kuki` unchanged
- `kukicha brew` converts back to standard Go anytime
- Blend in one feature at a time, leave the rest as Go
- **Security at compile time**, catches SQL injection, XSS, SSRF, path traversal, command injection, and open redirects at build time
- **42+ ease-of-use stdlib packages** — `fetch`, `slice`, `sort`, `mcp`, `llm`, `html`, `crypto`, `shell`, and [many more](docs/SKILL.md#stdlib-packages)
- **Ships as Go**, single binary, cross-compile, WASM support and the full Go ecosystem

---

## Editor Support

- **VS Code:** Search `kukicha-lang` in extensions ([repo](https://github.com/kukichalang/vscode-kukicha))
- **Zed:** [kukichalang/zed-kukicha](https://github.com/kukichalang/zed-kukicha)
- **Other:** `make install-lsp` and point your editor at `kukicha-lsp`

---

## Documentation

- [Quick Reference](docs/kukicha-quick-reference.md) — Go-to-Kukicha translation table
- [Agent Workflow Tutorial](docs/tutorials/agent-workflow-tutorial.md) — prompt AI, review, ship
- [Beginner Tutorial](docs/tutorials/beginner-tutorial.md) — first program, variables, functions
- [Production Patterns](docs/tutorials/production-patterns-tutorial.md) — databases, auth, retry
- [FAQ](docs/faq.md) | [Contributing](docs/contributing.md)

---

**Version:** 0.1.5 | **License:** [MIT](LICENSE)
