# Metaprogramming, Macros, and Stdlib: Reducing Beginner Boilerplate

## The Problem

Beginners face a wall of boilerplate before they can do anything useful. Consider Go's "read a JSON file" idiom:

```go
data, err := os.ReadFile("config.json")
if err != nil {
    log.Fatal(err)
}
var config Config
if err := json.Unmarshal(data, &config); err != nil {
    log.Fatal(err)
}
```

That's 7 lines for something conceptually simple: "load config.json into a Config struct." Three broad strategies exist for reducing this kind of friction: **metaprogramming**, **macros**, and **a rich standard library**. Each makes different trade-offs, and Go has a clear opinion about which ones are acceptable.

---

## Strategy 1: Metaprogramming

**What it is:** Code that writes, inspects, or transforms other code — typically at compile time. Rust's `derive` macros, Lisp's `defmacro`, and Template Haskell are canonical examples.

**How it reduces boilerplate:** You describe *what* you want, and the metaprogramming system generates the repetitive implementation. A single `#[derive(Serialize, Deserialize)]` in Rust replaces dozens of lines of manual serialization code.

**The trade-off:** Metaprogramming adds a layer of indirection. Code that generates code is harder to read, debug, and teach. Error messages from macro-generated code are notoriously opaque. For beginners, metaprogramming can feel like magic — helpful when it works, bewildering when it breaks.

### Go's position

Go explicitly rejects compile-time metaprogramming. There are no macros, no `derive`, no template metaprogramming, no compile-time code generation hooks in the language itself.

The reasoning (from the Go team, repeatedly):

1. **Readability over writability.** Go code is read far more often than it's written. Metaprogramming saves the writer time but costs every future reader time. Go optimizes for the reader.

2. **One obvious way.** If macros exist, every library invents its own DSL. The language fragments into dialects. Go prefers that everyone writes the same straightforward code.

3. **Tooling.** `go vet`, `gopls`, `gofmt`, and the entire tooling ecosystem can reason about Go code because the language is simple and regular. Macros break this property.

Go does offer `go generate` — but this is explicitly a *build step*, not a language feature. It runs external programs that produce `.go` files. The generated code is committed, reviewed, and read like any other Go code. It's metaprogramming in the workflow, not in the language.

---

## Strategy 2: Macros

**What they are:** A specific kind of metaprogramming where you define syntactic transformations. The compiler expands macros before (or during) compilation, replacing a short form with a longer one.

There are two broad families:

| Type | How it works | Examples |
|------|-------------|----------|
| **Textual macros** | String substitution before parsing | C `#define`, C++ templates |
| **Syntactic macros** | Operate on the AST (parsed tree) | Lisp macros, Rust proc macros, Elixir macros |

**How they reduce boilerplate:** A macro like `try!` (Rust's predecessor to `?`) turns three lines of error handling into one expression. Elixir's `defstruct` generates struct definitions, accessors, and protocol implementations from a compact declaration.

**The trade-off:** Textual macros are dangerous — they operate before the compiler understands the code, leading to subtle bugs (C's `#define SQUARE(x) x*x` infamously breaks on `SQUARE(1+2)`). Syntactic macros are safer but create a two-language problem: you need to understand both the base language and the macro language. Debugging macro-expanded code requires mental de-expansion.

### Go's position

Go has no macro system of any kind. The Go team considers macros an anti-feature:

> *"In Go, we want the language to be easy to read and understand. Macros work against that goal."* — Rob Pike

The `if err != nil` pattern is the most frequently cited example. Many proposals have suggested macros or syntactic sugar to shorten it. All have been declined, with the rationale that:

- Explicit error handling makes control flow visible
- A macro that hides `return` statements makes code harder to reason about
- The "cost" of typing `if err != nil` is small compared to the "cost" of not understanding what happens on error

Go 1.13 added `errors.Is` and `errors.As` for better error inspection, but the basic `if err != nil` pattern remains deliberate.

---

## Strategy 3: A Rich Standard Library

**What it is:** Instead of giving users tools to generate code, you give them pre-built functions that handle common tasks in a single call. The complexity is hidden inside library code that users don't need to read or understand.

**How it reduces boilerplate:** Instead of writing 7 lines to read JSON, you call `json.ReadFile("config.json", &config)` — one line, one concept. The library author wrote the boilerplate once; every user benefits.

**The trade-off:** A large stdlib has maintenance costs and can become a bottleneck (the Go team moves slowly on stdlib additions). It also can't handle every case — eventually users need the lower-level primitives. But for the 80% case, a good stdlib function is the most beginner-friendly solution because:

1. **No new concepts.** It's just a function call. No new syntax, no mental model to learn.
2. **Discoverable.** IDE autocomplete and documentation work normally.
3. **Debuggable.** You can step into the library code with a standard debugger.
4. **Composable.** Functions compose with existing language features (pipes, error handling, etc.) without special rules.

### Go's position

Go has a famously comprehensive standard library — `net/http`, `encoding/json`, `os`, `fmt`, `crypto`, and dozens more. The Go proverb is:

> *"A little copying is better than a little dependency."*

But also:

> *"The standard library is there so you don't have to."*

Go solves the boilerplate problem primarily through its stdlib, not through language-level abstraction. This is a deliberate choice: keep the language simple, make the library rich.

---

## What Kukicha Does

Kukicha takes the stdlib approach — augmented by targeted syntactic sugar where Go's syntax is needlessly hostile to beginners.

### The stdlib layer

Kukicha wraps common multi-step Go patterns into single-call stdlib functions:

```kukicha
# Go: 7 lines of ReadFile + Unmarshal + error handling
# Kukicha: one pipeline
config := files.ReadString("config.json")
    |> parse.Json(Config)
    onerr panic "bad config: {error}"
```

```kukicha
# Go: http.NewRequest + client.Do + ioutil.ReadAll + json.Unmarshal
# Kukicha: one pipeline
repos := fetch.Get("https://api.github.com/repos")
    |> fetch.CheckStatus()
    |> fetch.Json(list of Repo)
    onerr panic "{error}"
```

```kukicha
# Go: 15+ lines of flag parsing, subcommand routing, help text
# Kukicha: declarative CLI definition
app := cli.New("todo", "A simple task manager")
app |> cli.Command("add", "Add a task", (args cli.Args) =>
    title := args |> cli.StringArg("title") onerr panic "{error}"
    print("Added: {title}")
)
app |> cli.Run(os.Args)
```

Each stdlib package (`slice`, `fetch`, `files`, `parse`, `cli`, `pg`, `shell`) exists because the raw Go equivalent requires boilerplate that teaches nothing and trips up beginners.

### The syntactic sugar layer

Where Go's *syntax* (not just its library) creates friction, Kukicha adds targeted sugar:

| Friction point | Go | Kukicha | Why it helps beginners |
|---|---|---|---|
| Error handling | `if err != nil { return ..., err }` | `onerr return` | One concept instead of three (if, nil, return) |
| Logical operators | `&&`, `\|\|`, `!` | `and`, `or`, `not` | Reads like English |
| Null | `nil` | `empty` | Descriptive, less jargon |
| Pointers | `*T`, `&x` | `reference T`, `reference of x` | Says what it means |
| Lambdas | `func(x T) T { return expr }` | `(x T) => expr` | Less ceremony |
| Composition | Nested calls: `f(g(h(x)))` | `x \|> h() \|> g() \|> f()` | Left-to-right data flow |

### What Kukicha deliberately does NOT do

- **No user-defined transformations.** The set of expansions is fixed in the compiler. Users cannot create new `onerr`-like constructs. This prevents dialect fragmentation but means the language designer picks winners.
- **No AST-level metaprogramming.** No quasiquoting, no compile-time reflection, no code-as-data.

The bet Kukicha makes: a **closed, curated set** of compiler-integrated transformations avoids the fragmentation and complexity problems of user-extensible macros, while still meaningfully reducing friction. The risk is that if the built-in set doesn't cover a user's case, they have no escape hatch — they're stuck writing verbose Go-style code in Kukicha, or dropping down to raw Go.

---

## Summary: The Three Approaches

| Approach | Reduces boilerplate by | Complexity cost | Go's stance | Kukicha's stance |
|----------|----------------------|-----------------|-------------|------------------|
| **Metaprogramming** | Generating code at compile time | High — new mental model, opaque errors | Rejected (except `go generate` as build tool) | Not exposed to users |
| **Macros** | Syntactic shorthand that expands to longer code | Medium to high — two-language problem | Rejected entirely | Rejected entirely |
| **Rich stdlib** | Pre-built functions for common tasks | Low — just function calls | Primary strategy | Primary strategy, plus pipes to compose them |
| **Syntactic sugar** (renaming) | Simpler spelling of the same construct | Low — 1:1 mapping | Minimal (Go prefers explicitness) | `and`/`or`/`not`, `list of T`, `empty` |
| **Compiler-integrated transforms** | Fixed expansions baked into the compiler | Medium — hidden code generation | N/A (Go doesn't do this) | `onerr`, `\|>`, pipeline error threading |

---

## The Honest Part: Where the Lines Blur

The table above draws clean boundaries. In practice, they're fuzzy — and Kukicha should be upfront about where it sits in the gray areas.

### `onerr` is a macro in everything but name

When you write:

```kukicha
data := fetchData() onerr return
```

The compiler generates:

```go
data, err := fetchData()
if err != nil {
    return "", err  // zero values inferred per return signature
}
```

Multiple statements from a single expression. Zero values the user never mentioned. A return with an arity they didn't specify. By any academic definition, that's a macro expansion.

This is functionally identical to Rust's `?` operator, which is widely acknowledged as a macro-like language feature. The Go team rejected `?` for Go (the [`try` proposal](https://github.com/golang/go/issues/32437), 2019) — so Kukicha is genuinely **disagreeing** with Go here, not just adding a library on top.

### What's the actual distinction?

Not "sugar vs. metaprogramming" — it's between **user-defined** and **compiler-fixed** transformations:

| | User-defined (true macros) | Compiler-fixed (Kukicha's approach) |
|---|---|---|
| **Who writes the rule?** | Library or application author | Language designer |
| **How many rules exist?** | Unbounded — every library adds more | Fixed set, documented once |
| **Can it surprise you?** | Yes — must learn each library's macros | Only if you haven't read the language spec |
| **Tooling impact** | Breaks tools that don't understand the macro | Tools are built to understand the sugar |

And here's how each specific feature honestly classifies:

| Feature | Surface appearance | What actually happens | Honest classification |
|---------|-------------------|----------------------|----------------------|
| `and`/`or`/`not` | Keyword rename | 1:1 token swap | Genuine sugar |
| `list of string` | Type syntax | 1:1 mapping to `[]string` | Genuine sugar |
| `onerr return` | Error shorthand | Generates if-nil-return with inferred zero values | Compiler-integrated transformation |
| `onerr explain "msg"` | Error wrapping | Generates `fmt.Errorf("msg: %w", err)` + return | Compiler-integrated transformation |
| `x \|> f() \|> g()` | Composition | Introduces temporaries, reorders evaluation, threads error checks | Compiler-integrated transformation |
| Pipeline `onerr` | Error catch-all | Wraps entire chain in error-checked sequence | Compiler-integrated transformation |

### The compromises, stated plainly

1. **`onerr` hides control flow.** `onerr return` inserts a `return` the user didn't type. Go's objection to `try` was exactly this — hidden returns make it harder to reason about what a function does. Kukicha's counterargument: the word `return` is visually present on the line where it happens, unlike a macro that might bury a return inside an unrelated call. Go purists would say that's not enough.

2. **The pipe operator rewrites argument order.** `x |> f(a, _)` becomes `f(a, x)`. The reader has to mentally undo a positional transformation. For long chains this is a net win (left-to-right data flow); for one-off calls, `f(a, x)` is arguably clearer.

3. **`list of string` trades brevity for readability.** It's three tokens instead of Go's two (`[]string`). More readable for English speakers, more to type for experienced programmers. This is a deliberate bet on the beginner audience.

4. **The transpiler is itself metaprogramming.** Kukicha-the-language says "no user-facing metaprogramming," but Kukicha-the-tool is a code generator that produces Go. Users don't see the generated Go unless they look, which means they're trusting a transformation they can't easily inspect. Go's `go generate` philosophy is that generated code should be committed and reviewed — Kukicha's generated `.go` is an ephemeral build artifact.

### Where Kukicha draws the line

The compromises above are all **fixed and finite**. There are exactly N syntactic transformations in Kukicha, all documented, all baked into the compiler. No library author can add new ones. No application developer can define a custom `onerr`-like construct.

This is the real commitment: not "no transformations" (that would be Go), but **no user-extensible transformations**. The language can surprise you, but only in ways that fit on one reference card. Whether that's an acceptable trade-off depends on what you value more — Go's principle that even the language shouldn't hide control flow, or Kukicha's principle that fixed, well-documented sugar is a net win for beginners even if it isn't "pure."

Reasonable people disagree. Go chose one side. Kukicha chose the other. Both are internally consistent.
