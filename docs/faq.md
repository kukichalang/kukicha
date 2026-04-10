# FAQ

## Coming from Go

**Why not just write Go?**

Your `.go` files compile as `.kuki` unchanged. The question is whether the opt-in features are worth it.

| Go | Kukicha (opt-in) |
|---|---|
| `if err != nil { ... }` | `onerr panic "msg"` |
| `f(g(h(x)))` | `x \|> h() \|> g() \|> f()` |
| 5-line temp var + if/else | `x := if cond then a else b` |
| `&&`, `\|\|`, `!` | `and`, `or`, `not` |
| `*Type`, `&var` | `reference Type`, `reference of var` |
| `[]string{"a", "b"}` | `list of string{"a", "b"}` |
| `func(s string) bool { return ... }` | `(s string) => ...` |
| `const` + `iota` | `enum Status` with named variants |

Both Go and Kukicha syntax compile — use whichever you prefer, or mix them in the same file.

**The pipe operator** is probably the biggest addition. Instead of:

```go
result := strings.ToUpper(strings.TrimSpace(strings.ReplaceAll(text, "\t", " ")))
```

You write:

```kukicha
result := text |> strpkg.Replace("\t", " ") |> strpkg.TrimSpace() |> strpkg.ToUpper()
```

Use `_` to control where the piped value lands when it isn't the first argument:

```kukicha
todo |> json.MarshalWrite(response, _)  # Becomes: json.MarshalWrite(response, todo)
```

**Gradual adoption:**

```bash
kukicha-blend main.go          # see what your Go looks like with Kukicha idioms
kukicha-blend --apply main.go  # convert main.go → main.kuki
kukicha brew main.kuki         # convert back to standalone Go anytime
```

**Where to start:** [Quick Reference](kukicha-quick-reference.md) — a direct Go-to-Kukicha translation table.

---

## Coming from Python

**Why use Kukicha instead of Python?**

The syntax will feel familiar — `and`/`or`/`not`, indentation, `for x in items`, `# comments`. The differences are:

1. **Static types** — function parameters require explicit types; local variables are inferred.
2. **No implicit returns** — use `return` explicitly.
3. **Error handling** — `onerr` instead of exceptions.
4. **Single binary deployment** — no runtime, no virtualenv, no `pip install` on the target.

| Python | Kukicha |
|---|---|
| `f"{name}"` | `"{name}"` (no prefix) |
| `def greet(name: str) -> str:` | `func Greet(name string) string` |
| `**kwargs` / named args | `F(x: 10)` |
| `try: ... except Exception as e:` | `result onerr return` |
| `[x for x in items if pred(x)]` | `items \|> slice.Filter((x T) => pred(x))` |

If your work is ML/data science, stick with Python — Kukicha is aimed at CLI tools, automation, and AI agent tooling.

**Where to start:** [Beginner Tutorial](tutorials/beginner-tutorial.md) or skim and jump to the [Quick Reference](kukicha-quick-reference.md).

---

## Coming from Bash / Shell Scripting

**Why not just keep writing bash scripts?**

| Bash Pain Point | Kukicha Solution |
|---|---|
| Quoting hell (`"${var}"`) | `{var}` in strings |
| `set -e` surprises | `onerr` per operation |
| No real data types | `int`, `string`, `bool`, `list of`, `map of` |
| `$1`, `$2` positional args | Named, typed function parameters |
| `if [ ... ]; then ... fi` | `if condition` with indentation |
| Arrays (`"${arr[@]}"`) | `list of string{"a", "b"}` |

**Where to start:** [Kukicha for Shell Scripters](tutorials/shell-to-kukicha.md)

---

## General Questions

### I don't know how to program. Can I use Kukicha?

Yes:

1. Describe what you want to an AI agent (Claude Code, Cursor, ChatGPT)
2. The agent writes Kukicha code
3. You read and approve it — the English-like syntax makes this possible without a programming background
4. Run `kukicha build` to compile a single binary

See the [Agent Workflow Tutorial](tutorials/agent-workflow-tutorial.md) to get started.

### Does Kukicha have a runtime?

No. The compiler transpiles your code to standard, idiomatic Go. Once compiled by the Go toolchain, there is no trace of Kukicha — just a native Go binary with zero runtime overhead.

### Can I use existing Go libraries?

Yes. Import any Go package (standard library or third-party) and use it directly. The compiler trusts external packages it hasn't seen before, giving you the full Go ecosystem.

### Can I convert back to Go?

Yes. `kukicha brew` converts `.kuki` files to standalone, idiomatic Go — no Kukicha dependency, no generated headers. Zero lock-in.

### Which languages was this inspired by?

Besides Go, so far Python, Elixir, and Nim.

Kukicha also draws inspiration from several Go-adjacent transpiler projects:
[soppo](https://github.com/halcyonnouveau/soppo) (nil-safety, exhaustive pattern matching, `?` error propagation),
[dingo](https://github.com/MadAppGang/dingo) (result types, sum types, watch mode, lambda inference),
[gala](https://github.com/martianoff/gala) (sealed types, Option/Either/Try, LSP architecture), and
[sky](https://github.com/anzellai/sky) (Hindley-Milner inference, ADTs, applicative combinators).

These communities are each scratching their own itch — we respect their work and learn from it.
