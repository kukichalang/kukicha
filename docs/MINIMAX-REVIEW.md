# Minimax Code Review

**Date:** March 19, 2026
**Reviewer:** Automated Analysis
**Scope:** Full codebase review

---

## Critical Issues

### 1. Security gaps - `files.Copy` and `files.Move` lack path traversal checks

**Location:** `stdlib/files/files.kuki:144` and `stdlib/files/files.kuki:157`

`files.Copy` and `files.Move` have NO `# kuki:security "files"` directive. The security registry at `stdlib_registry_gen.go:799-808` only includes:
- `files.Append`, `files.AppendString`, `files.Delete`, `files.DeleteAll`
- `files.List`, `files.ListRecursive`, `files.Read`, `files.ReadBytes`
- `files.Write`, `files.WriteString`

**Impact:** Path traversal attacks via user-controlled paths won't be caught at compile time:

```kukicha
func Handle(w http.ResponseWriter, r reference http.Request)
    userPath := r.URL.Query().Get("path")
    files.Copy(userPath, "/safe/dest")  # NO WARNING - but this is a path traversal!
```

**Fix:** Add `# kuki:security "files"` directive to both functions.

---

### 2. IIFE allocation for every multi-return pipe step

**Location:** `lower.go:43-50` and `lower.go:61-67`

Every pipe step that returns multiple values (e.g., `(data, error)`) gets wrapped in an anonymous closure:

```go
if count, ok := l.gen.inferReturnCount(base); ok && count >= 2 {
    blanks := make([]string, count-1)
    for i := range blanks {
        blanks[i] = "_"
    }
    baseExpr = fmt.Sprintf("func() any { val, %s := %s; return val }()", strings.Join(blanks, ", "), baseExpr)
}
```

This generates code like:

```go
pipe_1 := func() any { val, _ := fetch.Get(); return val }()
pipe_2 := process(pipe_1)
```

**Impact:** Every runtime execution allocates a closure on the heap. For frequently used pipe chains, this adds GC pressure.

**Fix:** Consider restructuring to avoid IIFE when possible, or document this as a known trade-off.

---

## Medium Severity

### 3. `typesCompatible()` is overly permissive

**Location:** `semantic_types.go:210-215`

```go
// Nil is compatible with reference types
if t1.Kind == TypeKindNil {
    return a.isReferenceType(t2)
}
if t2.Kind == TypeKindNil {
    return a.isReferenceType(t1)
}
```

And `isReferenceType()` returns `true` for `TypeKindUnknown` (`semantic_types.go:164`):

```go
case TypeKindUnknown:
    return true // Allow leniently
```

This means `empty` (nil) is compatible with anything unknown, deferring validation to the Go compiler.

---

### 4. Arrow lambda return type inference uses only first return

**Location:** `codegen_decl.go:73-82`

```go
func (g *Generator) inferBlockReturnType(block *ast.BlockStmt) string {
    for _, stmt := range block.Statements {
        if ret, ok := stmt.(*ast.ReturnStmt); ok {
            if len(ret.Values) == 1 {
                return g.inferExprReturnType(ret.Values[0])
            }
        }
    }
    return ""
}
```

Early-return patterns with different types won't be inferred correctly:

```kukicha
func example(cond bool) string
    if cond
        return "string"
    return 42  # This type is never seen by the inference
```

---

### 5. SQL injection check only catches interpolated strings

**Location:** `semantic_security.go:63`

```go
if strLit, ok := sqlArg.(*ast.StringLiteral); ok && strLit.Interpolated {
    a.error(strLit.Pos(), ...)
}
```

Concatenation-based SQL building would slip through, though this is partially defensible given Kukicha's string interpolation is the primary string-building mechanism.

---

### 6. HTTP handler detection relies on exact type name match

**Location:** `semantic_security.go:30-34`

```go
for _, param := range a.currentFunc.Parameters {
    if named, ok := param.Type.(*ast.NamedType); ok {
        if named.Name == "http.ResponseWriter" {
            return true
        }
    }
}
```

Only `http.ResponseWriter` (exact name) is detected. Aliased imports or compatible interfaces are missed.

---

## Low Severity

### 7. Registry `returnCount` only increases, never decreases

**Location:** `cmd/genstdlibregistry/main.go:245-253`

```go
if existing, exists := result.registry[key]; !exists || returnCount > existing.count {
    result.registry[key] = registryEntry{...}
}
```

If a stdlib function is refactored from 2→1 return values, the registry won't update.

---

### 8. printf method detection is name-only

**Location:** `codegen_expr.go:916-934`

```go
var printfMethods = map[string]bool{
    "Errorf":  true,
    "Fatalf":  true,
    "Logf":    true,
    // ...
}
```

No signature validation; any method named `Errorf`/`Fatalf`/`Logf` passes even if the first arg isn't a format string.

---

### 9. Walrus flag not validated

**Location:** `lower.go:85-92`

The `walrus` flag is blindly trusted. If `walrus=true` but the RHS doesn't actually return multiple values, Go compilation fails.

---

### 10. `make generate` runs `genstdlibregistry` twice

**Location:** `Makefile:35` vs `Makefile:18-19`

```makefile
generate: genstdlibregistry build
# ...
build:
    go generate ./...
    go build -o $(KUKICHA) ./cmd/kukicha
```

`go generate` in `build` calls `genstdlibregistry` again.

---

### 11. `go_stdlib_gen.go` missing header comment

**Location:** `internal/semantic/go_stdlib_gen.go:1-7`

Unlike `stdlib_registry_gen.go`, it doesn't list which Go packages were scanned.

---

### 12. No staleness check for main `.kuki` → `.go` files

**Location:** `Makefile:55-70`

Only tests are checked (`check-test-staleness`). If `stdlib/*.kuki` is edited without `make generate`, the `.go` file silently becomes stale.

---


### 13. `json.Encode` comment misleading

**Location:** `stdlib/json/json.kuki:54-59`

Says "indent/prefix options not yet supported" but `WithIndent`/`WithPrefix` functions exist above it.

---

### 14. `slice.First`/`Last` lose type info

**Location:** `stdlib/slice/slice.kuki:11-26`

Returns `list of any` instead of preserving the element type. A fundamental limitation of the current placeholder system.

---

### 15. Missing common stdlib functions

- `slice.Partition` - split slice into two based on predicate
- `slice.Flatten` - flatten `list of list of T` into `list of T`
- `maps.Map` - transform map values

---

### 16. `not!=` not handled

**Location:** `parser_expr.go:146-149`

`not equals` works but `not!=` doesn't parse correctly.

---

### 17. `reference` keyword has no short alias

Unlike `func`/`function`, `var`/`variable`, there's no `ref` alias.

---

## What's Well Done

- **Clean separation:** lexer → parser → ast → semantic → codegen with no circular dependencies
- **IR is appropriately minimal:** Models exactly what's needed for pipe/onerr lowering
- **Security directive system:** Elegant and extensible via `# kuki:security`
- **Test staleness checks:** Catches drift between `.kuki` and `_test.kuki` files
- **Comprehensive internal docs:** `internal/AGENTS.md` (700+ lines) and `internal/CLAUDE.md`
- **Good use of directives:** `# kuki:deprecated`, `# kuki:panics`, `# kuki:security`

---

## Priority Fixes

1. **Add `# kuki:security "files"` to `files.Copy` and `files.Move`** (`stdlib/files/files.kuki:144,157`)
2. **Audit other stdlib functions for missing security directives**
3. **Fix the double-run of `genstdlibregistry`** in `make generate`
4. **Add main `.kuki` → `.go` staleness check** (analogous to `check-test-staleness`)
5. **Remove or flesh out empty `cmd/ku-*` directories**
6. **Document the IIFE allocation as a known trade-off** or investigate avoiding it

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 2 |
| Medium | 4 |
| Low | 16 |

The codebase is generally well-architected with clean separation of concerns. The most pressing issue is the missing security directives on `files.Copy` and `files.Move`, which creates a path traversal detection gap. Secondary concerns are the IIFE allocation pattern and registry staleness handling.
