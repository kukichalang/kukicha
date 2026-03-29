# Verifier & Correctness Roadmap

Ideas extracted from the `formal-proof-verifier` branch (deleted — code was stale, ideas preserved here).

---

## 1. Fuzz Testing

Add Go native fuzz tests (`testing.F`) to find panics and crashes in the compiler.

### Lexer fuzz
```go
// internal/lexer/lexer_fuzz_test.go
func FuzzLexer(f *testing.F) {
    f.Add("func Add(a int, b int) int\n    return a + b\n")
    f.Fuzz(func(t *testing.T, data string) {
        // Must never panic — only return errors
        lexer.New(data, "fuzz.kuki")
    })
}
```

### Full pipeline fuzz
Feed random `.kuki` source through lex → parse → semantic → codegen. If it reaches codegen, verify the output is valid Go via `go/parser.ParseFile`. Seed corpus should cover: functions, types, methods, pipes, onerr, switch, lambdas, string interpolation, enums, channels, variadic, for loops.

**Target:** 0 panics after 10M iterations.

---

## 2. Property-Based Tests

### Formatter idempotency
`format(format(source)) == format(source)` — proves the formatter is stable.

### Codegen structural soundness
For a broad set of valid Kukicha programs, verify `go/parser.ParseFile(codegen(ast))` succeeds.

### Security check monotonicity
Adding code to a program should never remove a security error.

### Return count consistency
For all stdlib functions in both registries, `Count == len(Types)`, param names are non-empty, security functions exist in the registry.

---

## 3. Contracts (Design-by-Contract Directives)

Lightweight runtime specification checks via compiler directives. Stripped in release builds (`--release` flag).

### `# kuki:requires` — Preconditions
```kukicha
# kuki:requires "len(items) > 0"
func First(items list of string) string
    return items[0]
```
Generates at function entry: `if !(len(items) > 0) { panic("requires violated: len(items) > 0") }`

### `# kuki:ensures` — Postconditions
```kukicha
# kuki:ensures "result >= 0"
func Abs(x int) int
    if x < 0
        return -x
    return x
```
Generates named return vars + deferred check on all return paths.

### `# kuki:invariant` — Type invariants
```kukicha
# kuki:invariant "self.min <= self.max"
type Range
    min int
    max int
```
Generates a `Validate()` method that panics if the invariant is violated.

### Implementation notes
- Kukicha keywords in expressions (`and`, `or`, `not`, `equals`, `empty`) auto-translate to Go operators
- Semantic validation: rejects `ensures` on void functions, validates `self.field` references exist on struct
- `releaseMode` flag on Generator gates all contract codegen
- ~140 lines codegen (`codegen_contracts.go`) + ~100 lines semantic (`semantic_contracts.go`)

---

## 4. Structured Diagnostics for AI Agents

### JSON error output
`kukicha check --json` emits structured errors:
```json
{
  "file": "app.kuki",
  "line": 12,
  "col": 5,
  "severity": "error",
  "category": "security/sql-injection",
  "message": "SQL injection risk: ...",
  "suggestion": "use parameter placeholders ($1, $2, ...) instead of string interpolation"
}
```

### Batch check mode
`kukicha check [--json] file1.kuki file2.kuki ...` — processes multiple files, outputs single JSON array or grouped text. Makes the compiler usable as a "verifier kernel" in agent loops.

### Fix suggestions
Each security check category provides a concrete safe alternative in the suggestion field.

---

## 5. Agent-Specific Security Checks

Extend the existing 6 security checks with agent-relevant categories:

| Check | Category | What it catches |
|-------|----------|-----------------|
| Unbounded loops | `agent/unbounded-loop` | `for true` with no break/return inside HTTP handlers |
| Resource exhaustion | `agent/resource-exhaustion` | Goroutine spawning or channel creation inside loops in HTTP handlers |
| Privilege escalation | `agent/privilege-escalation` | `shell.Run`/`shell.Command` inside HTTP handlers |
| Data exfiltration | `agent/data-exfiltration` | `fetch.Post` with data from `files.Read` (requires taint tracking — deferred) |

---

## 6. Formal Verification (Long-term)

Prove properties of the compiler using Lean 4 or similar:

- **Type compatibility rules** — `typesCompatible()` has ~20 cases. Prove reflexivity, expected symmetry, no contradictions.
- **Security check soundness** — For a defined threat model, prove no false negatives.
- **Transpilation correctness** — Semantic preservation for pipes (`a |> b |> c` == `c(b(a))`), onerr desugaring, string interpolation.

---

## Priority Order

1. **Fuzz tests** — highest ROI, catches real crashes, zero new syntax needed
2. **Property tests** — extends confidence, catches regressions
3. **Structured diagnostics** — unlocks AI agent integration
4. **Contracts** — useful for stdlib and safety-critical code
5. **Agent security checks** — valuable once agent usage grows
6. **Formal verification** — long-term research
