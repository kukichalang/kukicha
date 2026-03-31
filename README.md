# Kukicha

A programming language designed to be read by humans and written by AI agents. Compiles to Go — single binaries, goroutines, strong typing, no runtime dependencies.

**[kukicha.org](https://kukicha.org)** | [Tutorials](https://kukicha.org/#getting-started) | [Stdlib Reference](stdlib/AGENTS.md) | [Quick Reference](docs/kukicha-quick-reference.md)

---

## See If You Can Follow This

```kukicha
import "stdlib/fetch"
import "stdlib/slice"
import "stdlib/sort"
import "stdlib/table"

type Repo
    name string as "name"
    stars int as "stargazers_count"
    language string as "language"

function main()
    repos := fetch.Get("https://api.github.com/users/golang/repos")
        |> fetch.CheckStatus()
        |> fetch.Json(list of Repo) onerr panic "fetch failed: {error}"

    popular := repos
        |> slice.Filter(r => r.stars > 100)
        |> sort.ByKey(r => r.stars)
        |> slice.Reverse()

    t := table.New("Name", "Stars", "Language")
    for repo in popular
        t |> table.AddRow(repo.name, "{repo.stars}", repo.language)
    t |> table.Print()
```

Fetch repos from GitHub, keep the popular ones, sort by stars, print a table. `|>` pipes results between steps. `onerr panic` means "if this fails, stop." Four stdlib packages, zero boilerplate.

---

## Quickstart

**Requires Go 1.26+** ([download](https://go.dev/dl/)) | Pre-built binaries on [GitHub Releases](https://github.com/kukichalang/kukicha/releases)

```bash
go install github.com/kukichalang/kukicha/cmd/kukicha@v0.0.30
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

### Commands

| Command | What it does |
|---------|-------------|
| `kukicha check file.kuki` | Validate syntax without compiling |
| `kukicha run file.kuki` | Compile and run immediately |
| `kukicha build file.kuki` | Compile to a standalone binary |
| `kukicha pack skill.kuki` | Package a skill for agent pipelines |

---

## Why Kukicha

- **Readable** — English keywords (`and`, `or`, `not`, `equals`), indentation instead of braces, no `&&`/`||`/`__`
- **Safe** — The compiler catches SQL injection, XSS, SSRF, path traversal, command injection, and open redirects at build time
- **42+ stdlib packages** — `fetch`, `slice`, `sort`, `mcp`, `llm`, `html`, `crypto`, `shell`, and [many more](stdlib/AGENTS.md)
- **Pipes + error handling** — `|> step onerr handler` chains replace nested `if err != nil`
- **Compiles to Go** — Single binary, cross-compile, WASM support

---

## Editor Support

- **VS Code:** Search `kukicha-lang` in extensions ([repo](https://github.com/kukichalang/vscode-kukicha))
- **Zed:** [kukichalang/zed-kukicha](https://github.com/kukichalang/zed-kukicha)
- **Other:** `make install-lsp` and point your editor at `kukicha-lsp`

---

## Documentation

- [Agent Workflow Tutorial](docs/tutorials/agent-workflow-tutorial.md) — prompt AI, review, ship
- [Absolute Beginner Tutorial](docs/tutorials/absolute-beginner-tutorial.md) — first program, variables, functions
- [Production Patterns](docs/tutorials/production-patterns-tutorial.md) — databases, auth, retry
- [FAQ](docs/faq.md) | [Contributing](docs/contributing.md)

---

**Version:** 0.0.30 | **License:** [MIT](LICENSE)
