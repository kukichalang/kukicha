# CLAUDE.md

Kukicha is a beginner-friendly programming language that **transpiles to Go**.
Current version: **0.0.31**
When editing `.kuki` files, write **Kukicha syntax, NOT Go**.

## Kukicha vs Go Syntax (Common AI Mistakes)

| Go | Kukicha | Both accepted? |
|----|---------|----------------|
| `&&`, `\|\|`, `!` | `and`, `or`, `not` | Yes |
| `[]string` | `list of string` (or `[]string`) | Yes |
| `map[string]int` | `map of string to int` (or `map[string]int`) | Yes |
| `*User` | `reference User` (or `*User`) | Yes |
| `&user` | `reference of user` (or `&user`) | Yes |
| `*ptr` | `dereference ptr` | |
| `nil` | `empty` (or `nil`) | Yes |
| `{ }` braces | 4-space indentation | Yes |
| `==` | `equals` (or `==`) | Yes |
| `!=` | `isnt` (or `!=`) | Yes |
| `func (t T) Method()` | `func Method on t T` | Not yet |
| `func(x T) T { return expr }` | `(x T) => expr` | |
| `go func() { ... }()` | `go` + indented block | |
| `const (StatusOK = 200; ...)` | `enum Status` with `OK = 200` | |
| (no ternary) | `if COND then EXPR else EXPR` (if-expression) | |

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
make test                 # Run all tests
make lint                 # Run golangci-lint (errcheck, unused, staticcheck, etc.)
make vet                  # Run go vet on everything including stdlib
make modernize            # Check for outdated Go patterns (go fix -diff)
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
kukicha audit             # Check dependencies for known vulnerabilities
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

1. **Always validate** - Run `kukicha check` before committing `.kuki` changes
2. Use red/green TDD when adding new features. Update existing tests when required.
3. **4-space indentation only** — tabs are not allowed in Kukicha
4. **Explicit function signatures** — parameters and return types must be declared
5. **Test with `make test`** — run the full test suite
6. **Lint with `make lint`** — catch unused code, unchecked errors, and other issues
7. **Vet with `make vet`** — catch bugs in stdlib that golangci-lint excludes
8. **Modernize with `make modernize`** — ensure generated Go uses current patterns
9. **After adding a stdlib function or enum**, run `make genstdlibregistry`

## Skills

- `/kukicha` — full language syntax, error handling, pipes, lambdas, stdlib usage, troubleshooting
- `/compiler-internals` — lexer, parser, AST, semantic analysis, IR, codegen internals; adding new features
- `/stdlib` — stdlib authoring: package table, patterns, security checks, pitfalls, critical rules
- `/cmd` — CLI binary structure, subcommands, key functions, generators, test coverage
- `docs/SKILL.md` — full language + stdlib reference (the content embedded into user projects)
