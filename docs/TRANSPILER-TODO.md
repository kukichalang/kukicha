# Kukicha Roadmap

Kukicha (the tea) is made from the stems and twigs left over after
processing green tea leaves. The good stuff that got discarded.

Go (the language) keeps discarding good stuff too — better error handling,
enums, sum types, ternaries, pipe operators. The Go team's governance is
famously conservative; these proposals get rejected repeatedly. That's fine.
Kukicha is brewed from what Go leaves behind.

Strategic direction: make Kukicha a **strict superset of Go** with opt-in
ergonomic features. Kukicha's features have a durable moat (Go won't add
them), but only if adoption is frictionless.

The CoffeeScript trap isn't "host language catches up" (Go won't). It's
"adoption cliff too steep, ecosystem too small." The fix: let users rename
`.go` to `.kuki` and have it compile. Then blend in Kukicha features one
line at a time. Not sure? `kukicha brew` gives you standard Go back —
the stems dissolve, the tea remains.

---

## Completed Work

Transpiler improvements already shipped — context for why some areas are
in better shape than others.

### IR Layer (v0.0.25–v0.0.26)

All `onerr` code paths (except discard) now go through an intermediate
representation instead of direct string emission.

- `internal/ir/ir.go` — 9 immutable IR node types (Assign, VarDecl,
  IfErrCheck, Goto, Label, ScopedBlock, RawStmt, ReturnStmt, ExprStmt)
- `internal/codegen/lower.go` — Lowerer with 4 distinct phases:
  pipe chains, simple onerr, onerr pipe chains, piped switch (goto-label)
- `internal/codegen/emit.go` — mechanical IR walker (~100 lines)
- Lowerer shares Generator's `tempCounter` via `uniqueId()` for stable
  variable names (`pipe_1`, `err_2`) that tests depend on

### Pipe Robustness Audit (v0.0.25)

Systematic fixes from `docs/NEW-PIPE-ISSUES.md` (since removed):

- Bare identifier pipe targets (`data |> transform onerr ...`)
- Non-pipe base in piped switch (split multi-return into `val, err :=`)
- `pipeErr` threading through goto-label paths
- Unknown single-return detection with codegen warnings
- Multiple placeholder rejection (`data |> f(_, _)`) at compile time
- `warnPipeDiscardedErrors` for intermediate steps without `onerr`

### Lambda Parameter Type Inference (v0.0.20)

- `goStdlibEntry.ParamFuncParams` maps func-typed parameter positions to
  their inner parameter types
- `genstdlibregistry` populates it; qualifies named types with package prefix
- Import alias resolution (`importAliases` on Analyzer) for cross-package lambdas

### String Interpolation Tokenization (v0.0.28)

- Lexer emits `TOKEN_STRING_HEAD`, `TOKEN_STRING_MID`, `TOKEN_STRING_TAIL`
  instead of relying on codegen fallback parsing
- `interpStack []int` tracks brace depth within each interpolation level

### Other Notable Fixes

- `rewriteGoErrors()` — maps generated `.go` paths in `go build` stderr
  back to `.kuki` source paths
- `rewriteVarNames()` — appends variable hints mapping `pipe_N`/`err_N`
  to source descriptions
- `//line` directives in generated Go for source-mapped error messages
- `--no-line-directives` flag for clean production output
- Security directives (`# kuki:security`, `# kuki:deprecated`) with
  `genstdlibregistry` extraction
- Codegen warnings (`Generator.warn()`) matching semantic `Analyzer.warn()`

---

## 1. CLI Extraction (Clean House First)

**Problem:** `main.go` is a 939-line monolith — `build`, `run`, and `check`
subcommands are inlined with their flag parsing and orchestration. `fmt`,
`audit`, `pack`, and `init` already live in their own files. Before adding
`eject`, the CLI structure needs to be consistent so subcommands don't keep
piling into one file.

**Goal:** Each subcommand in its own file. `main.go` becomes a thin
dispatcher (~50 lines) that parses the subcommand name and delegates.

### Tasks

- [ ] Extract `build` into `build.go` — move flag parsing, `buildCommand()`,
  `wasmScaffold()`, `setEnvVar()`
- [ ] Extract `run` into `run.go` — move flag parsing and run logic
- [ ] Extract `check` into `check.go` — move flag parsing, JSON output,
  multi-target logic
- [ ] Extract `version` into `version.go` (trivial)
- [ ] Slim `main.go` to dispatcher: subcommand switch + `printUsage()`
- [ ] Keep shared functions in `compile.go`:
  - `compile()`, `loadAndAnalyze()`, `compileResult`
  - `rewriteGoErrors()`, `rewriteVarNames()`
  - `stripFirstLine()`
- [ ] Verify no test regressions — tests already use `package main` so
  they'll see all exported/unexported symbols regardless of file layout
- [ ] Move `multifile_test.go` and `rewrite_errors_test.go` alongside
  their implementation files if it makes sense

### Resulting layout

```
cmd/kukicha/
  main.go            # ~50 lines: dispatcher + usage
  compile.go         # shared compile pipeline + helpers
  build.go           # kukicha build
  run.go             # kukicha run
  check.go           # kukicha check
  brew.go            # kukicha brew (new, item 2)
  fmt.go             # kukicha fmt (already exists)
  init.go            # kukicha init (already exists)
  audit.go           # kukicha audit (already exists)
  pack.go            # kukicha pack (already exists)
  version.go         # kukicha version
  stdlib.go          # ensureStdlib, ensureGoMod, extractAgentDocs
```

### Notes

- This is pure mechanical refactoring — no behavior changes, no new features
- All files stay `package main`, so nothing changes for tests or imports
- Do this first so `brew` lands in a clean structure

---

## 2. `kukicha brew` (Kukicha -> Go)

Brew the stems into tea. Convert `.kuki` source into pure, idiomatic Go
that stands on its own — no Kukicha dependency, no generated headers, no
source maps. The escape hatch that makes adoption safe.

**Goal:** `kukicha brew <file.kuki>` produces clean `.go` output. Thin
wrapper over the existing compile pipeline.

### Tasks

- [ ] Add `brew.go` to `cmd/kukicha/` (lands cleanly after item 1)
- [ ] Reuse `compile()` with `noLineDirectives: true`
- [ ] Strip the `// Generated by Kukicha` header from output
- [ ] Run `goimports` on the output for clean imports
- [ ] Run `gofmt` on the output for canonical Go formatting
- [ ] Write output to `<basename>.go` (or stdout with `--stdout`)
- [ ] For directory mode: brew all `.kuki` files, optionally remove originals
  with `--remove-kuki` flag (behind confirmation prompt)
- [ ] Add tests: brew example files, verify output compiles with `go build`

### Notes

- Mostly plumbing — the transpiler already produces valid Go;
  `goimports` + `gofmt` handle the rest
- Should fit in ~100 lines
- The `--no-line-directives` flag on `build` already does half the work
- The trust signal: "You're not locked in. Brew your Kukicha back to Go
  any time — the stems dissolve, the tea remains."

---

## 3. `kukicha-blend` (Go -> Kukicha, Separate Binary)

Blend Kukicha flavoring into existing Go code. Show developers what their
Go looks like with Kukicha idioms — `onerr` instead of `if err != nil`,
`and`/`or` instead of `&&`/`||`, pipes instead of nested calls.

**Goal:** `kukicha-blend <file.go>` suggests Kukicha patterns for existing
Go code. Separate binary because it uses `go/parser` (Go->Kukicha direction)
and shares almost no code with the compiler.

### Tasks

- [ ] Create `cmd/kukicha-blend/` as a separate binary
- [ ] Parse `.go` files with `go/parser`
- [ ] Suggest transformations as diagnostics (not auto-applied by default):
  - `if err != nil { return err }` -> `onerr return`
  - `if err != nil { return ..., err }` -> `onerr return ...`
  - `&&`/`||`/`!` -> `and`/`or`/`not`
  - `[]string` -> `list of string`
  - `map[string]int` -> `map of string to int`
  - `*T` -> `reference T`
  - Brace blocks -> indentation
- [ ] `--apply` flag to auto-convert and write `.kuki` output
- [ ] `--diff` flag to show changes without writing
- [ ] `--patterns` flag to select which transformations to blend
  (e.g., `--patterns=onerr,operators` for just error handling and operators)

### Notes

- Separate binary keeps the core `kukicha` CLI focused
- The marketing tool: "Run `kukicha-blend main.go` and see what Go
  discarded. 40% less error boilerplate, readable operators, real enums."
- With superset support, blending can be gradual — convert one pattern at a
  time, leave the rest as Go
- Could later be wired into `kukicha blend` as a subcommand that shells out
  to `kukicha-blend` if installed

---

## 4. Go Syntax Passthrough (Superset Foundation)

**Problem:** Kukicha currently requires its own syntax — braces are rejected,
`&&`/`||`/`!` aren't recognized, `nil` isn't valid, method declarations use
a different form. This means users must rewrite entire files to try Kukicha.
The adoption cliff kills interest before users see the value.

**Goal:** All valid Go is valid Kukicha. A renamed `.go` file compiles
unchanged. Kukicha features (`onerr`, pipes, `and`/`or`, indentation blocks,
`list of`, `enum`) are opt-in enhancements on top of standard Go.

### Phase 1: Operator and Keyword Aliases ✅

Accept Go's native forms alongside Kukicha's English forms. Both compile
to identical output.

- [x] Lexer: accept `&&` as alias for `and`
- [x] Lexer: accept `||` as alias for `or`
- [x] Lexer: accept `!` as alias for `not`
- [x] Lexer: accept `nil` as alias for `empty`
- [x] Lexer: accept `==` as alias for `equals`
- [x] Lexer: accept `!=` as alias for `isnt`
- [x] Parser: accept `*T` as alias for `reference T`
- [x] Parser: accept `&x` as alias for `reference of x`
- [x] Parser: accept `[]T` as alias for `list of T`
- [x] Parser: accept `map[K]V` as alias for `map of K to V`
- [x] Tests: each alias produces identical codegen output as the Kukicha form

### Phase 2: Brace Blocks ✅

Accept `{ }` blocks as alternative to indentation. Mixed styles within a
file should work (indentation for Kukicha-style functions, braces for
Go-style).

- [x] Lexer: recognize `{` / `}` as block delimiters — block braces emit
  INDENT/DEDENT (transparent to parser); literal braces unchanged
- [x] Parser: brace blocks work everywhere indent blocks are accepted
  (no parser changes needed — lexer converts block braces to INDENT/DEDENT)
- [x] Handle mixed mode: braces suppress indent/dedent tracking within that block
- [ ] Ensure `kukicha fmt` normalizes to indentation style (configurable later)
- [x] Tests: Go-style `if err != nil { return err }` compiles as-is
- [x] Tests: nested brace blocks, single-line brace blocks, mixed files,
  if/else/for/defer with braces, Go-style operators + braces combined

### Phase 3: Go Method and Function Syntax ✅

Accept Go's `func (t T) Method()` alongside Kukicha's `func Method on t T`.

- [x] Parser: accept `func (receiver Type) Name(params) returns` form
- [x] Parser: accept `func name(params) (returns)` with parenthesized multi-return
- [x] Codegen: passthrough — if user wrote Go syntax, emit Go syntax unchanged
- [x] Tests: mixed Go-style and Kukicha-style methods in the same file

### Phase 4: Package and Import Compat ✅

- [x] Accept `package` as alias for `petiole`
- [x] Accept `import ( "fmt" )` grouped import syntax
- [x] Accept raw Go import paths without `import "stdlib/..."` prefix for Go stdlib

### Notes

- Incremental: each phase is independently valuable and shippable
- The lexer already handles `==`/`!=` — extend this pattern to `&&`/`||`/`!`
- Passthrough codegen means Go syntax costs nothing in the compiler; it's
  just an alternate parse path that produces the same AST nodes
- **Non-goal for v1:** full `go/parser` compatibility. Focus on the 90% of
  Go syntax that real projects use. Edge cases (labeled breaks, goto, naked
  returns, multi-value type switches) can wait.

---

## 5. If-Expressions ✅

Another stem Go discarded. The Go team rejected ternaries, and `if` is
statement-only — assigning a value based on a condition takes 5 lines.
Rust and Kotlin solved this years ago. Kukicha should too.

```kukicha
access := if age >= 18 then "Granted" else "Denied"

label := if count equals 1 then "item" else "items"

color := if status equals "ok" then green
    else if status equals "warn" then yellow
    else red
```

Codegen emits the verbose Go form (temp var + if/else assignment). The
`then` keyword keeps it consistent with Kukicha's indentation style and
avoids ambiguity with brace blocks once those land in item 4.

### Tasks

- [ ] Lexer: add `then` keyword token
- [ ] Parser: recognize `if` in expression position (right side of `:=`/`=`,
  function arguments, return values)
- [ ] AST: add `IfExpression` node (condition, then-expr, else-expr)
- [ ] Semantic: type-check that then/else branches return the same type
- [ ] Codegen: emit as temp variable + if/else block assignment
- [ ] Support chained `else if` in expression position
- [ ] Tests: basic, chained, nested, type mismatch error

### Notes

- `then` is required to disambiguate `if cond expr` from `if cond` + block
- Else is mandatory in if-expressions (unlike statement `if`) — must produce
  a value on all paths
- This is independent of the superset work and can ship at any time

---

## 6. Split the Semantic Analyzer

**Problem:** The `Analyzer` struct is ~2500 lines across 7 files. Directive
collection and security checks have been extracted; the struct is down to
16 fields grouped by lifecycle. Remaining coupling: lint warnings are
interleaved with type checking, and declaration collection is tightly
bound to the symbol table.

**Goal:** Separate passes with clear inputs/outputs so each can be tested,
reasoned about, and extended independently.

### Proposed passes

| Pass | Input | Output |
|------|-------|--------|
| 1. Directive collection | AST | `[]Directive` (deprecated, security, panics) |
| 2. Declaration collection | AST | Symbol table (types, interfaces, functions) |
| 3. Type checking | AST + symbol table | `ExprTypes` map, return counts, type errors |
| 4. Security analysis | AST + directives | Security warnings/errors |
| 5. Lint warnings | AST + symbol table + ExprTypes | Unused vars, risky onerr, etc. |

### Tasks

- [x] Extract directive collection into its own pass/struct
  (`CollectDirectives()` in `semantic_directives.go`, returns `DirectiveResult`)
- [x] Extract security analysis (semantic_security.go already partially separate)
  (`SecurityChecker` struct with back-pointer to Analyzer, all `check*` methods moved)
- [x] Embed `*DirectiveResult` — replaced 3 separate fields (`deprecatedFuncs`,
  `deprecatedTypes`, `panickedFuncs`) with single `directives *DirectiveResult`
- [x] Add `AnalysisResult` type + `AnalyzeResult()` — bundles errors, warnings,
  `ExprReturnCounts`, and `ExprTypes` into one struct
- [x] Migrate callers — `compile.go`, `kukicha-wasm`, codegen use `SetAnalysisResult()`
  instead of separate `SetExprReturnCounts`/`SetExprTypes` calls
- [x] Reorder struct fields by lifecycle — 16 fields grouped into: immutable inputs,
  infrastructure, pre-pass output, pass 1 output, pass 2 transient state, pass 2 output
- [ ] Extract lint/warning pass (currently interleaved with type checking)
- [ ] Ensure existing tests pass after each extraction

### Notes

- `AnalyzeResult()` provides the bundled return; old `Analyze()` + individual
  getters remain for backward compat (53 test-file call sites)
- `SetAnalysisResult()` on codegen sets both maps in one call; old setters
  remain for test compat
- Declaration collection (`collectDeclarations` + helpers) is tightly coupled to
  the Analyzer — it needs `symbolTable`, `typeAnnotationToTypeInfo()`, `error()`,
  `importAliases`. Extracting with a back-pointer adds indirection without real
  decoupling. Needs a deeper redesign to pass typed results between phases.
- Lint warnings are deeply interleaved with type checking (e.g., deprecation
  checks fire during call analysis). Separating them requires collecting lint
  candidates during analysis and emitting them in a final pass — feasible but
  a larger change than the directive/security extractions.

---

## 7. Visitor Pattern for Codegen

**Problem:** Codegen uses ad-hoc type switches scattered across 4 files
(codegen_expr.go, codegen_stmt.go, codegen_decl.go, codegen_stdlib.go —
13K lines total). Adding a new AST node requires edits in multiple places
with no compile-time guarantee that all cases are handled.

**Goal:** A single `ast.Visitor` interface so that new node types produce a
compiler error until every pass handles them.

### Tasks

- [ ] Define `ast.Visitor` interface with one method per node type
- [ ] Implement `ast.Walk(visitor, node)` that dispatches to the correct method
- [ ] Refactor `Generator.exprToString()` type switches into visitor methods
- [ ] Refactor `generateDeclaration()` / `generateStatement()` the same way
- [ ] Add an `exhaustive` linter check (or build tag) to enforce all cases are covered
- [ ] Verify no test regressions after each file migration

### Notes

- Migrate incrementally — one codegen file at a time
- The IR layer (lower.go, emit.go) already uses a clean type switch over a
  small, stable set of IR nodes; leave it alone unless it grows
- With superset support, the number of AST node variants may grow (Go-style
  nodes alongside Kukicha-style nodes, or unified nodes with a syntax flag);
  the visitor pattern makes this manageable

---

## Priority Order

1. **CLI extraction** (1) — mechanical, zero risk, cleans house for everything else
2. **kukicha brew** (2) — trust signal, lands cleanly in the new CLI structure
3. **Go syntax passthrough** (4) — the strategic foundation; unlocks adoption
4. **If-expressions** (5) — new language feature, independent of superset work
5. **kukicha-blend** (3) — marketing tool, separate binary, can develop in parallel
6. **Split analyzer** (6) — maintainability for the long haul
7. **Visitor pattern** (7) — largest refactor, do last

## The Pitch

> Go is green tea. Kukicha is brewed from what's left over — the stems,
> the twigs, the features Go's governance keeps discarding. Better error
> handling. Enums. Readable operators. Pipes. They're good. Go just
> doesn't want them.
>
> Rename your `.go` files to `.kuki`. They still compile — Kukicha is a
> strict superset of Go. Then blend in the features Go left behind, one
> line at a time. Not sure? `kukicha brew` gives you standard Go back.
> The stems dissolve, the tea remains.
>
> Go's not getting friendlier. Kukicha is.
