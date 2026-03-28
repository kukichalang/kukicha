# Codegen TODO

## IIFE type inference falls back to `any` silently

**Severity:** Low
**Location:** `internal/codegen/lower.go` — `lowerPipeChain` and `lowerOnErrPipeChain`

When a multi-return pipe step is wrapped in an IIFE to extract the first value,
the return type is inferred via `inferExprReturnType`. If inference fails (returns
empty string), the fallback is `"any"`:

```go
retType := l.gen.inferExprReturnType(base)
if retType == "" {
    retType = "any"
}
```

This produces valid Go but loses type safety. The generated wrapper:

```go
func() any { val, _ := f(); return val }()
```

forces downstream code to work with `any`, requiring type assertions that
wouldn't be needed if the concrete type were known.

**Why this happens:** `inferExprReturnType` relies on `exprTypes` from semantic
analysis and a set of hardcoded literal/operator rules. When the expression is an
unresolved call (e.g., a user-defined function whose return type wasn't recorded
in `exprTypes`), inference returns empty.

**Impact:** The Go compiler still type-checks the final program, so this doesn't
cause incorrect code — only less precise types in intermediate pipe variables.
Users see this as needing explicit type assertions after certain pipe steps.

**Fix:** Requires propagating full return type information through the semantic
analyzer into `exprTypes` for all expressions, not just stdlib and known
externals. This is a broader type inference improvement tracked here for
visibility.
