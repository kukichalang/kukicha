# Transpiler Known Limitations

Tracked limitations in the Kukicha compiler (`internal/`). Each entry includes the relevant source location(s) and a description of the gap. Items are grouped by compiler phase.

---

## Semantic Analysis / Type Inference

### Method return type resolution limited to hand-coded Go stdlib entries
**File:** `internal/semantic/semantic_calls.go`

User-defined methods are resolved via `registerMethod()` and `resolveMethodType()`. Kukicha stdlib methods are resolved via the generated registry. However, Go stdlib methods beyond a small hand-coded set (`time.Time.*`, `bufio.Scanner.*`, `regexp.Regexp.*`, `exec.ExitError.*`) still return `TypeKindUnknown`. Full resolution would require extending `cmd/gengostdlib/` to generate method entries from `go/types` package method sets.

---

## Summary

| Phase | Items |
|-------|-------|
| Semantic / type inference | 1 |
| **Total** | **1** |
