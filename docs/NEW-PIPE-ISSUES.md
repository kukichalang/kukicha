# New Pipeline Issues

Actionable bugs found during review of the pipe + onerr implementation.
Each entry is confirmed against the source — no speculative issues.

---

## Issue 1: Error-only step detection silently collapses when `exprTypes` is missing

**Severity:** High (silent wrong behavior — errors swallowed at runtime)
**File:** `internal/codegen/codegen_stdlib.go:401-412`

### Problem

`isErrorOnlyReturn` requires two independent conditions to both be satisfied:

```go
func (g *Generator) isErrorOnlyReturn(expr ast.Expression) bool {
    count, ok := g.inferReturnCount(expr)
    if !ok || count != 1 {
        return false
    }
    if g.exprTypes != nil {
        if ti, ok := g.exprTypes[expr]; ok && ti != nil {
            return ti.Kind == semantic.TypeKindNamed && ti.Name == "error"
        }
    }
    return false  // <-- falls here if exprTypes has no entry for expr
}
```

If `exprReturnCounts[expr] == 1` but `exprTypes` has no entry for `expr`, the function
returns `false`. The Lowerer then falls through to the non-error branch and collapses
the step into a nested expression — its error return is never checked.

This silently swallows errors at runtime. No warning is emitted, and the generated Go
compiles without issue.

### When does this happen?

Semantic analysis only records `exprTypes` for expressions it recognizes. Gaps include:

- External library functions that aren't in `generatedGoStdlib` or `generatedStdlibRegistry`
- Functions defined in the same file that semantic analysis processes after the call site
  (if declaration order creates a gap in the analysis pass)
- Any pipe step where `analyzeExpression` returns but `recordType` is not called on the step node

### Affected syntax

```kukicha
# If validate.Check is not in the registry and returns only error:
result := data
    |> parse.Json(list of User)
    |> validate.Check()    # ← count==1, exprTypes missing → silently collapsed
    onerr panic "pipeline failed"
```

The generated Go becomes:
```go
result, err_1 := parse.Json[...](...) // error checked
...
result2, err_2 := b(validate.Check(result))  // validate.Check error NEVER checked
```

### Proposed fix

When `count == 1` and `exprTypes` has no entry, treat the step as potentially
error-returning and emit a named error variable with a check anyway. The worst case is
a spurious `if err != nil` on a non-error step, which is a compile error the user can
diagnose — far better than a silent swallow.

Alternatively, emit a warning:

```
warning: return type of 'validate.Check' is unknown; error may be unchecked in pipe chain
```

---

## Issue 2: Bare identifier pipe targets in onerr chains produce empty output silently

**Severity:** Medium (silent codegen failure — no output, no diagnostic)
**Files:** `internal/codegen/codegen_onerr.go:285-286`, `internal/codegen/lower.go:113-115`, `lower.go:308-310`, `lower.go:379-381`

### Problem

`generatePipedStepCall` handles three right-hand-side types:

```go
func (g *Generator) generatePipedStepCall(right ast.Expression, leftExpr string) (string, bool) {
    if call, ok := right.(*ast.CallExpr); ok { ... }
    else if method, ok := right.(*ast.MethodCallExpr); ok { ... }
    else if field, ok := right.(*ast.FieldAccessExpr); ok { ... }
    else {
        return "", false  // ← bare identifier, PipedSwitchExpr, anything else
    }
    ...
}
```

Every caller in the Lowerer treats `false` by returning `nil, ""`:

```go
callExpr, ok := l.gen.generatePipedStepCall(step, current)
if !ok {
    return nil, ""  // no error, no warning
}
```

A bare identifier is a valid pipe target in the non-onerr path (`data |> print` → `fmt.Println(data)`),
and the parser accepts it. But when the same expression appears in an onerr chain,
the Lowerer silently produces a nil block, and the calling codegen path emits nothing.

### Affected syntax

```kukicha
# Non-onerr: works fine (handled by generatePipeExpr)
data |> print

# onerr chain: bare identifier step silently erases the entire pipe chain
result := data
    |> transform         # ← bare identifier, not a CallExpr
    |> parse.Json(Todo)
    onerr panic "failed"
```

The generated Go for the entire statement is empty — no assignment, no error check,
no `result` variable declared.

### Proposed fix

Add a diagnostic in the `!ok` branch of each Lowerer caller, or in
`generatePipedStepCall` itself:

```go
} else {
    g.addWarning(posOf(right), fmt.Sprintf(
        "unsupported pipe target type %T in onerr chain; wrap in a call: f()",
        right,
    ))
    return "", false
}
```

For bare identifiers specifically, `generatePipedStepCall` could handle them the same
way `generatePipeExpr` does — emit `ident(leftExpr)` — so behavior is consistent
between the onerr and non-onerr paths.

---

## Summary

| # | Issue | Severity | Status |
|---|-------|----------|--------|
| 1 | Error-only step silently collapsed when `exprTypes` missing | High | Open |
| 2 | Bare identifier pipe target in onerr chain produces empty output | Medium | Open |
