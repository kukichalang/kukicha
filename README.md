# Kukicha

**Brewed from Go.** Use readable forms while you learn, or plain Go when you're fluent; same file, same compiler. The standard library stays steeped in simplicity no matter how you write it. Ships as a single binary.

**[kukicha.org](https://kukicha.org)** | [Quick Reference](docs/kukicha-quick-reference.md) | [Tutorials](https://kukicha.org/#tutorials) | [Stdlib Reference](.claude/skills/stdlib/SKILL.md)

---

## A taste of Kukicha

Triage open GitHub issues with an LLM. Fetch, classify in parallel, keep the urgent ones, sort, print.

```
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
        |> llm.System(`Classify GitHub issues. Reply JSON: {severity:1-5, kind, summary}`)
        |> llm.Ask("{i.title}\n\n{i.body}") onerr return {}

    v := Verdict{number: i.number}
    jsonpkg.UnmarshalString(reply, reference of v) onerr return {}
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

You can read this top-to-bottom without knowing the language: fetch issues, classify four at a time, filter the severe ones, sort, print. The stdlib names carry the weight — `fetch.Json`, `slice.Filter`, `concurrent.MapWithLimit`, `sort.ByKey` — and `onerr` keeps error handling out of the flow. A Go developer can write the same program in plain Go — `[]Issue`, `&v`, `!= nil`, closures — and Kukicha will compile it unchanged.

---

## Two tiers, one stdlib

Kukicha is a strict superset of Go: the language you graduate *into* is Go itself. The Kukicha tier gives you scannable forms while you're learning; the Go tier is waiting whenever you're ready. Pick the one that fits — or mix them in the same file.

| Concept | Kukicha form | Go form |
| --- | --- | --- |
| **Booleans** | `and`, `or`, `not` | `&&`, `\|\|`, `!` |
| **Comparison** | `equals`, `isnt` | `==`, `!=` |
| **Lists** | `list of string` | `[]string` |
| **Maps** | `map of string to int` | `map[string]int` |
| **Pointers** | `reference User`, `reference of user` | `*User`, `&user` |
| **Nil** | `empty` | `nil` |
| **Errors** | `onerr return` | `if err != nil { return err }` |
| **Pipes** | `x \|> h() \|> g() \|> f()` | `f(g(h(x)))` |
| **If-expression** | `x := if cond then a else b` | *(no Go equivalent)* |
| **Enums** | `enum Status` with named variants | `const` + `iota` |
| **Lambdas** | `(x int) => x * 2` | `func(x int) int { return x*2 }` |
| **Interpolation** | `"hi {name}"` | `fmt.Sprintf("hi %s", name)` |

The Kukicha forms are syntactic sugar; the Go forms are the Go they desugar to. Every `.go` file is valid `.kuki` unchanged, and every `.kuki` file transpiles to standard Go before compilation.

What stays constant across both tiers is the [stdlib](.claude/skills/stdlib/SKILL.md) — `fetch`, `slice`, `sort`, `llm`, `mcp`, `concurrent`, `html`, `crypto`, `shell`, and 30+ more. `fetch.Get(...) |> fetch.Json(...) onerr ...` reads the same whether the surrounding code is `list of string` or `[]string`.

---

## Quickstart

**Requires Go 1.26+** ([download](https://go.dev/dl/)) | Pre-built binaries on [GitHub Releases](https://github.com/kukichalang/kukicha/releases)

```
go install github.com/kukichalang/kukicha/cmd/kukicha@v0.1.11
mkdir myapp && cd myapp
kukicha init
```

`kukicha init` initializes a Go module (if `go.mod` is absent), extracts the stdlib, downloads dependencies, and writes an `AGENTS.md` language reference. Add `.kukicha/` to your `.gitignore`.

Create `hello.kuki`:

```
function main()
    name := "World"
    print("Hello, {name}!")
```

```
kukicha run hello.kuki
```

### Commands

| Command | What it does |
| --- | --- |
| `kukicha check file.kuki` | Validate syntax without compiling |
| `kukicha run file.kuki` | Compile and run immediately |
| `kukicha build file.kuki` | Compile to a standalone binary |
| `kukicha fmt -w file.kuki` | Format in place |
| `kukicha brew file.kuki` | Convert back to standalone Go |
| `kukicha-blend file.go` | Suggest Kukicha idioms for Go code |

---

## Why bother

* **Skimmable at every skill level** — beginners read the Kukicha forms, Go developers write plain Go, both call the same readable stdlib
* **Compile-time security checks** — catches SQL injection, XSS, SSRF, path traversal, command injection, and open redirects before you ship
* **42+ batteries-included stdlib packages** — `fetch`, `slice`, `sort`, `mcp`, `llm`, `html`, `crypto`, `shell`, and [more](.claude/skills/stdlib/SKILL.md)
* **Ships as Go** — single binary, cross-compile, WASM, full Go ecosystem
* `kukicha brew file.kuki` converts any file back to standard Go; existing `.go` files compile as `.kuki` unchanged

---

## Starting from Go

Already have a Go codebase? You don't have to rewrite anything, Kukicha can suggest idioms incrementally or convert files on request.

```
# See what your Go code looks like with Kukicha idioms
kukicha-blend main.go

# Convert Go to Kukicha (preview first, then apply)
kukicha-blend --diff main.go
kukicha-blend --apply main.go

# Convert Kukicha back to Go anytime
kukicha brew main.kuki
```

---

## Editor support

* **VS Code:** Search `kukicha-lang` in extensions ([repo](https://github.com/kukichalang/vscode-kukicha))
* **Zed:** [kukichalang/zed-kukicha](https://github.com/kukichalang/zed-kukicha)
* **Other:** `make install-lsp` and point your editor at `kukicha-lsp`

---

## Documentation

* [Quick Reference](docs/kukicha-quick-reference.md) — Go-to-Kukicha translation table
* [Agent Workflow Tutorial](docs/tutorials/agent-workflow-tutorial.md) — prompt AI, review, ship
* [Beginner Tutorial](docs/tutorials/beginner-tutorial.md) — first program, variables, functions
* [Production Patterns](docs/tutorials/production-patterns-tutorial.md) — databases, auth, retry
* [FAQ](docs/faq.md) | [Contributing](docs/contributing.md)

---

**Version:** 0.1.11 | **License:** [MIT](LICENSE)

---

>[!NOTE]
>Portions of this codebase were written with AI assistance. Commits are reviewed by human maintainers before merge.
