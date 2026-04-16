# CLAUDE.md

Kukicha is a strict superset of Go that adds pipes, `onerr`, enums,
if-expressions, and readable operators on top of standard Go syntax.
Current version: **0.1.9**

**When writing `.kuki` files in this repo, always use Kukicha syntax** —
`and`/`or`/`not`, `equals`/`isnt`, `list of T`, `map of K to V`,
`reference T`, `empty`, 4-space indentation, `onerr`, pipes, enums, and the
`stdlib/*` packages. All valid Go also compiles, but falling back to raw Go
forms in `.kuki` files is a bug in the writing style — we dogfood Kukicha
everywhere. Only reach for a Go-style construct when Kukicha has no
equivalent (e.g. `dereference ptr`, lambdas `(x T) => expr`, `go` blocks,
enum variants — these have no Go alternative and must be written the
Kukicha way).

## Kukicha vs Go Syntax

Kukicha forms on the left are the ones to write. The Go column shows what
they transpile to — **do not** use it as a menu of "also valid" choices.

| Kukicha (write this) | Go equivalent (avoid in `.kuki` files) |
|----------------------|----------------------------------------|
| `and`, `or`, `not` | `&&`, `\|\|`, `!` |
| `equals`, `isnt` | `==`, `!=` |
| `empty` | `nil` |
| `list of string` | `[]string` |
| `map of string to int` | `map[string]int` |
| `reference User` | `*User` |
| `reference of user` | `&user` |
| `dereference ptr` | `*ptr` (no Go-style form accepted) |
| 4-space indentation | `{ }` braces |
| `func Method on t T` | `func (t T) Method()` |
| `(x T) => expr` | `func(x T) T { return expr }` (no Go-style form accepted) |
| `go` + indented block | `go func() { ... }()` (no Go-style form accepted) |
| `enum Status` with `OK = 200` | `const (StatusOK = 200; ...)` |
| `enum Shape` with `Circle` / `Point` (variant enum) | (Go has no sum types — use the Kukicha form) |
| `if v is Circle as c` | `if _, ok := v.(Circle); ok` |
| `if COND then EXPR else EXPR` (if-expression) | (Go has no ternary — use the Kukicha form) |
| `{field: val}` / `{e1, e2}` when type is inferrable | `T{field: val}` / `[]T{e1, e2}` (always) |
| `type X = pkg.T` (transparent alias) | `type X = pkg.T` |

## Keyword Aliases (English-Friendly Forms)

| Short form | English alias | When to use |
|-----------|--------------|-------------|
| `func`    | `function`   | Beginner-facing code and tutorials |
| `var`     | `variable`   | Top-level variable declarations in beginner-facing code |
| `const`   | `constant`   | Beginner-facing const declarations |

All forms compile identically. Use `func`/`var`/`const` in idiomatic/production code, and `function`/`variable`/`constant` when writing beginner tutorials or agent-generated code aimed at non-programmers.

## Generic Type Placeholders (stdlib authoring only)

| Placeholder | Go equivalent | Constraint | Used for |
|-------------|---------------|------------|----------|
| `any` | `T` | `any` (unconstrained) | First type parameter |
| `any2` | `K` | `comparable` | Second type parameter (e.g., map key) |
| `ordered` | `K` | `cmp.Ordered` | Second type parameter requiring ordering |
| `result` | `R` | `any` (unconstrained) | Second unconstrained type parameter (e.g., transform output) |

## Compiler Directives

```kukicha
# kuki:deprecated "Use NewFunc instead"    # warns at every call site
# kuki:security "category"                 # categories: sql, html, fetch, files, redirect, shell
```

Run `make genstdlibregistry` after adding or changing directives on stdlib functions.

## Build & Test Commands

```bash
make build                # Build the kukicha compiler
make blend                # Build the kukicha-blend Go→Kukicha converter
make test                 # Run all tests
make lint                 # Run golangci-lint (errcheck, unused, staticcheck, etc.)
make vet                  # Run go vet on everything including stdlib
make modernize            # Check for outdated Go patterns (go fix -diff)
make fmt                  # Format all .kuki files in stdlib/ and examples/ in place
make fmt-check            # Check all .kuki files are formatted (CI gate)
make generate             # Regenerate stdlib_registry_gen.go + all stdlib .go files
make genstdlibregistry    # Regenerate only internal/semantic/stdlib_registry_gen.go
make gengostdlib          # Regenerate only internal/semantic/go_stdlib_gen.go
kukicha check file.kuki   # Validate syntax without compiling
kukicha check myapp/      # Validate all .kuki files in a directory
kukicha check a.kuki b.kuki  # Check multiple targets
kukicha check --json file.kuki  # Emit structured JSON diagnostics
kukicha build file.kuki   # Transpile and compile to binary
kukicha build myapp/      # Build from directory (merges all .kuki files into main.go)
kukicha build --wasm file.kuki       # Build for WebAssembly
kukicha build --vulncheck file.kuki  # Build + check for vulnerabilities
kukicha run file.kuki     # Transpile, compile, and run
kukicha fmt -w file.kuki  # Format in place
kukicha fmt --check dir/  # Check formatting without modifying (exit 1 if unformatted)
kukicha audit             # Check dependencies for known vulnerabilities
kukicha-blend main.go             # Show Kukicha suggestions for Go file
kukicha-blend --diff ./pkg/       # Preview Go→Kukicha changes
kukicha-blend --apply main.go     # Convert main.go → main.kuki
kukicha-blend --patterns=onerr,operators main.go  # Selective patterns
```

## Multi-File Directory Builds

`kukicha build myapp/` merges all `.kuki` files in a directory into a single `main.go` and compiles it.

- Exactly **one file** defines `func main()` — the entry point
- Other files use `func init()` for startup code that runs before `main`
- All files must use the same `petiole` declaration (or all omit it)
- Imports are deduplicated; duplicate function names (except `init`) are rejected
- Test files (`*_test.kuki`) are excluded from the merge

## File Map

```
cmd/kukicha/              # CLI entry point
cmd/kukicha-blend/        # Go → Kukicha converter (separate binary)
cmd/genstdlibregistry/    # Generator: stdlib/*.kuki → stdlib_registry_gen.go
cmd/gengostdlib/          # Generator: Go stdlib signatures → go_stdlib_gen.go
internal/
  lexer/                  # Tokenization (INDENT/DEDENT handling)
  parser/                 # Recursive descent parser → AST
  ast/                    # AST node definitions
  semantic/               # Type checking, validation
    stdlib_registry_gen.go  # GENERATED — auto-updated by "make build" via go generate
    go_stdlib_gen.go        # GENERATED — auto-updated by "make build" via go generate
  ir/                     # Intermediate representation (Go-level imperative nodes)
  codegen/                # AST → IR (lower.go) → Go source (emit.go)
  blend/                  # Go → Kukicha transformation engine (patterns, apply, diff)
  formatter/              # Code formatting
stdlib/                   # Standard library (.kuki source files)
examples/                 # Example programs
docs/                     # Documentation
  SKILL.md                # Full language + stdlib reference (embedded into user projects as AGENTS.md)
```

## Imports

```kukicha
import "stdlib/slice"                   # standard package
import "stdlib/ctx" as ctxpkg          # alias — use when the package name conflicts with a local variable
import "github.com/jackc/pgx/v5" as pgx  # external package with alias
```

## Critical Rules

1. **Write Kukicha, not Go, in `.kuki` files** — use `and`/`or`/`not`, `equals`/`isnt`, `list of T`, `map of K to V`, `reference T`, `empty`, `onerr`, pipes. Reviewers reject PRs that leave `&&`, `==`, `[]T`, `*T`, `nil`, etc. in `.kuki` source when a Kukicha form exists.
2. **Always validate** — run `kukicha check` before committing `.kuki` changes
3. Use red/green TDD when adding new features. Update existing tests when required.
4. **4-space indentation only** — tabs are not allowed in Kukicha
5. **Explicit function signatures** — parameters and return types must be declared
6. **Test with `make test`** — run the full test suite
7. **Lint with `make lint`** — catch unused code, unchecked errors, and other issues
8. **Vet with `make vet`** — catch bugs in stdlib that golangci-lint excludes
9. **Modernize with `make modernize`** — ensure generated Go uses current patterns
10. **After adding a stdlib function or enum**, run `make genstdlibregistry`

## Skills

- `/kukicha` — full language syntax, error handling, pipes, lambdas, stdlib usage, troubleshooting
- `/compiler-internals` — lexer, parser, AST, semantic analysis, IR, codegen internals; adding new features
- `/stdlib` — stdlib authoring: package table, patterns, security checks, pitfalls, critical rules
- `/cmd` — CLI binary structure, subcommands, key functions, generators, test coverage
- `docs/SKILL.md` — full language + stdlib reference (the content embedded into user projects)
