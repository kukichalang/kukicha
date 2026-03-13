# Source-Level Optimizations Plan

Inspired by Go's [source-level inliner](https://go.dev/blog/inliner), this plan
covers three improvements to Kukicha's transpiler and (future) LSP:

1. **Pipe chain tidiness** — cleaner generated Go from pipe expressions
2. **Shadowing analysis** — detect variable name collisions in onerr and temp vars
3. **`# kuki:fix` directives** — deprecation warnings and auto-migration

## Current State

| Area | Status |
|------|--------|
| Simple pipes (`a \|> b() \|> c()`) | Already collapse into nested calls `c(b(a))` |
| Onerr pipes | Always emit temp vars (`pipe_1`, `err_2`) at every step |
| Temp variable naming | Monotonic counter, no collision check against user vars |
| Onerr scoping | `renderHandler` saves/restores correctly, but no shadowing warnings |
| Directives | None — comments are non-semantic |
| Deprecation | Functions removed directly (acceptable at v0.0.x) |

---

## 1. Pipe Chain Tidiness

**Goal:** Reduce unnecessary temp variables in generated Go for onerr pipe chains.

### Phase 1A: Eliminate single-use final assignment

**Problem:** The last pipe step always assigns to a temp, then copies to the user variable.

```kukicha
result := fetchData() |> parse() onerr panic "fail"
```

Currently generates:
```go
pipe_1, err_2 := fetchData()
if err_2 != nil { panic("fail") }
pipe_3, err_4 := parse(pipe_1)
if err_4 != nil { panic("fail") }
result := pipe_3    // unnecessary copy
```

Should generate:
```go
pipe_1, err_2 := fetchData()
if err_2 != nil { panic("fail") }
result, err_3 := parse(pipe_1)
if err_3 != nil { panic("fail") }
```

**Approach:**
- In `lowerOnErrPipeChain` (lower.go), accept an optional target variable name
- When the pipe result feeds directly into a VarDecl, use that name for the last
  step's LHS instead of generating a temp
- The caller (`generateOnErrVarDecl` in codegen_onerr.go) passes the target name

**Files:** `lower.go`, `codegen_onerr.go`
**Risk:** Low — only changes the last assignment's LHS name
**Tests:** Update existing pipe chain tests to expect the optimized output

### Phase 1B: Collapse non-error-returning steps

**Problem:** Steps that don't return errors still get individual temp variables.

```kukicha
data |> string.ToLower() |> string.TrimSpace() |> parse() onerr panic "fail"
```

Currently generates:
```go
pipe_1 := strings.ToLower(data)
pipe_2 := strings.TrimSpace(pipe_1)
pipe_3, err_4 := parse(pipe_2)
if err_4 != nil { panic("fail") }
```

Should generate:
```go
pipe_1, err_2 := parse(strings.TrimSpace(strings.ToLower(data)))
if err_2 != nil { panic("fail") }
```

**Approach:**
- In `lowerOnErrPipeChain`, before emitting IR for each step, check the step's
  return count via `inferReturnCount`
- Consecutive steps with return count <= 1 (and not error-only) can be collapsed
  into a nested expression
- When we hit an error-returning step, flush the accumulated expression as its
  argument
- **Conservative rule:** Only collapse steps whose functions are in a known-pure
  set (stdlib string/slice operations). User functions always get their own temp
  to preserve observable side-effect ordering.

**Files:** `lower.go`, `codegen_onerr.go`, `codegen_stdlib.go` (pure function list)
**Risk:** Medium — requires hazard analysis. Wrong collapsing can reorder side effects.
**Tests:** New tests for mixed error/non-error pipe chains

---

## 2. Shadowing Analysis

**Goal:** Detect and prevent variable name collisions between generated temps and
user code, and between onerr bindings and outer scope variables.

### Phase 2A: Temp variable collision avoidance

**Problem:** `uniqueId("pipe")` generates `pipe_1`, `pipe_2`, etc. without checking
if those names exist in user code. A user variable named `pipe_1` would collide.

**Approach:**
- At `Lowerer` construction, collect all user-declared identifiers visible in the
  current scope into a `Set[string]`
- Modify `uniqueId` to skip names that appear in the set:
  ```go
  func (g *Generator) uniqueId(prefix string) string {
      for {
          g.tempCounter++
          name := fmt.Sprintf("%s_%d", prefix, g.tempCounter)
          if !g.userNames.Contains(name) {
              return name
          }
      }
  }
  ```
- Build the set by walking the AST's variable declarations before codegen, or by
  querying the semantic symbol table

**Files:** `codegen.go` (uniqueId), `lower.go` (constructor), `semantic/symbols.go`
**Risk:** Low — additive change, only affects naming
**Tests:** Test with user variable named `pipe_1` to verify skip behavior

### Phase 2B: Onerr shadowing warnings

**Problem:** If a user declares `error` as a variable in an outer scope and then
uses `{error}` in an onerr block, the onerr reference silently shadows the user
variable. This is confusing.

```kukicha
error := "previous error"
data := fetchData() onerr
    # Does {error} mean the onerr error or the "previous error" variable?
    print("got: {error}")
```

**Approach:**
- In `semantic_onerr.go`, when analyzing an onerr block:
  1. Check if the caught error name (`error` or the alias from `onerr as X`) is
     already declared in an enclosing scope
  2. If so, emit a warning diagnostic: "onerr variable 'error' shadows declaration
     at line N"
- Warnings don't block compilation — they appear in `kukicha check` and LSP

**Files:** `semantic_onerr.go`, `symbols.go` (add warning infrastructure if needed)
**Risk:** Low — diagnostic only, no codegen changes
**Tests:** Test that shadowing produces a warning; test that non-shadowing is clean

### Phase 2C: Nested onerr correctness

**Problem:** `renderHandler` saves/restores `currentOnErrVar` correctly for linear
sequences, but nested onerr (onerr block body contains another onerr expression)
needs verification.

**Approach:**
- Write test cases for nested onerr patterns
- Verify the save/restore stack works correctly
- Fix any issues found

**Files:** `lower.go`, `codegen_test.go`
**Risk:** Low — testing and potential bug fixes
**Tests:** Nested onerr patterns

---

## 3. `# kuki:fix` Directives

**Goal:** Add a directive system for deprecation warnings and automated migration,
modeled on Go's `//go:fix inline`.

### Phase 3A: Lexer/parser — recognize directives

**Problem:** Comments are entirely non-semantic. We need a way to attach metadata
to declarations.

**Syntax:**
```kukicha
# kuki:deprecated "use newpkg.NewFunc instead"
func OldFunc(x int) int
    return newpkg.NewFunc(x)

# kuki:fix inline
func OldFunc(x int) int
    return newpkg.NewFunc(x)
```

**Approach:**
- **Lexer:** When tokenizing a comment that starts with `# kuki:`, emit
  `TOKEN_DIRECTIVE` with the full directive text (e.g., `deprecated "msg"` or
  `fix inline`)
- **Parser:** In `skipIgnoredTokens`, don't skip `TOKEN_DIRECTIVE`. Instead,
  collect directives and attach them to the next declaration node.
- **AST:** Add `Directives []Directive` field to `FuncDecl`, `TypeDecl`,
  `ConstDecl`. Define `Directive` struct:
  ```go
  type Directive struct {
      Name   string   // "deprecated", "fix"
      Args   []string // ["use newpkg.NewFunc instead"] or ["inline"]
      Token  Token
  }
  ```

**Files:** `lexer.go`, `parser.go`, `ast.go`
**Risk:** Medium — touches the lexer/parser pipeline. Must not break existing
comment handling.
**Tests:** Parse directives on functions, types, consts. Verify regular comments
still work.

### Phase 3B: Semantic — deprecation warnings

**Approach:**
- During call resolution in `semantic_calls.go`, if the target function has a
  `deprecated` directive, emit a warning with the directive's message
- Add a `Warnings` list to the semantic analyzer (alongside existing errors)
- `kukicha check` prints warnings to stderr but exits 0
- LSP surfaces them as `DiagnosticSeverityWarning`

**Files:** `semantic_calls.go`, `semantic.go` (warning infrastructure)
**Risk:** Low — additive
**Tests:** Call a deprecated function, verify warning is emitted

### Phase 3C: Registry integration

**Approach:**
- Extend `genstdlibregistry` to scan for `# kuki:deprecated` directives on
  exported functions
- Add deprecation info to `stdlib_registry_gen.go`:
  ```go
  var generatedStdlibDeprecations = map[string]string{
      "oldpkg.OldFunc": "use newpkg.NewFunc instead",
  }
  ```
- Semantic analyzer checks this map during call resolution (in addition to
  checking AST directives for user code)

**Files:** `cmd/genstdlibregistry/main.go`, `stdlib_registry_gen.go`,
`semantic_calls.go`
**Risk:** Low — generated code change
**Tests:** Add a deprecated stdlib function, verify it appears in generated registry

### Phase 3D: `kukicha fix` command (future)

**Approach:**
- New CLI subcommand that applies `# kuki:fix inline` transformations
- For stdlib wrappers: parse the wrapper body, verify it's a direct
  call-forwarding pattern, rewrite call sites
- Start with the simplest case: `OldFunc(args...) -> NewFunc(args...)` with no
  argument reordering
- More complex cases (argument reordering, type changes) can be added later

**Files:** `cmd/kukicha/main.go`, new `internal/fix/` package
**Risk:** Medium-high — source rewriting is inherently complex (as the Go blog
post details at length)
**Tests:** End-to-end tests with sample deprecated functions and expected rewrites

---

## Implementation Order

| Step | Item | Estimated scope | Dependencies |
|------|------|----------------|--------------|
| 1 | Phase 1A: Final assignment elimination | Small | None |
| 2 | Phase 2A: Temp collision avoidance | Small | None |
| 3 | Phase 1B: Non-error step collapsing | Medium | 1A |
| 4 | Phase 2B: Onerr shadowing warnings | Small | None |
| 5 | Phase 2C: Nested onerr tests | Small | None |
| 6 | Phase 3A: Directive lexing/parsing | Medium | None |
| 7 | Phase 3B: Deprecation warnings | Small | 3A |
| 8 | Phase 3C: Registry integration | Small | 3B |
| 9 | Phase 3D: `kukicha fix` command | Large | 3A-3C |

Steps 1-2 can be done in parallel. Steps 4-5 can be done in parallel with 3.
Step 9 is a stretch goal — the value of 3A-3C stands without it.
