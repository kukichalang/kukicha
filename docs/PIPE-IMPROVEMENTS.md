# Pipe Operator — Improvement Plan

Status: **Draft**
Date: 2026-03-25

This document captures fragility points, user pain points, and proposed improvements for the Kukicha pipe operator (`|>`).

---

## Architecture Overview

The pipe operator flows through four compiler phases:

1. **Lexer** — tokenizes `|>`, handles line continuation (suppresses NEWLINE/INDENT/DEDENT when line ends with `|>`)
2. **Parser** — left-associative loop producing `PipeExpr` / `PipedSwitchExpr` AST nodes
3. **Semantic** — passes left-side type context to right-side analysis, records return counts and types on each step
4. **Codegen** — three paths depending on complexity:
   - Simple pipes: direct expression generation with placeholder substitution
   - Pipes with onerr: IR lowering with temp variables and error checks
   - Piped switches: IIFE wrapping with return type inference

---

## Fragility Points

### 1. Lexer Line Continuation (HIGH)

**Location:** `internal/lexer/lexer.go` — `continuationLine`, `isPipeAtStartOfNextLine()`

The indent-sensitive lexer must suppress `NEWLINE`/`INDENT`/`DEDENT` tokens when a line ends with `|>`. This couples pipe handling directly to the core indentation machinery. If the indent stack logic changes, multi-line pipes silently break.

The lookahead helpers (`isPipeAtStartOfNextLine`, `nextNonWhitespaceWithIndent`) add defensive complexity but are themselves fragile — they duplicate aspects of the main scanning logic.

**Mitigation ideas:**
- Add dedicated regression tests for multi-line pipes at varying indentation levels
- Consider a post-tokenization pass that merges continuation lines, decoupling it from the indent stack

### 2. Temp Variable Naming via Shared Counter (HIGH)

**Location:** `internal/codegen/lower.go` — `l.gen.uniqueId()`

The lowerer shares the generator's `tempCounter` to produce names like `pipe_1, err_2`. Tests in `codegen_pipes_test.go` hard-code these exact names. Any change to counter initialization, ordering, or unrelated codegen that increments the counter breaks every pipe test simultaneously.

This is primarily a testing fragility — the generated code is correct regardless of exact names — but it makes the lowerer risky to refactor.

**Mitigation ideas:**
- Use regex patterns or structural assertions in tests instead of exact string matching
- Alternatively, reset the counter per-test or use a deterministic counter scoped to lowering

### 3. Goto-Based Control Flow for Piped Switch + Onerr (MODERATE)

**Location:** `internal/codegen/lower.go` — `lowerPipedSwitchVarDecl()`, `lowerOnErrPipeChainWithLabels()`

When `|> switch` is combined with pipeline-level `onerr`, the lowerer emits goto labels and scoped blocks. This is the most complex codegen path and the hardest to reason about. The indent-bumping trick (`g.indent+1`) for IIFE generation is especially subtle.

**Mitigation ideas:**
- Add more targeted tests for piped switch + onerr combinations
- Document the goto-based pattern with a before/after example in code comments

### 4. Type Inference Fallbacks (MODERATE)

**Location:** `internal/codegen/codegen_expr.go` — `pipedSwitchReturnType()`, `inferPipedSwitchReturnType()`

Piped switch return type inference falls back to `"any"` when cases have mismatched types. Multi-return IIFE wrappers also fall back to `"any"`. This produces valid Go but silently loses type safety.

**Mitigation ideas:**
- Emit a compiler warning when falling back to `"any"`
- Prefer semantic analysis types over codegen-level inference (semantic already provides `exprTypes`)

### 5. Pipeline-Level Onerr Duplicates Handlers (MODERATE)

**Location:** `internal/codegen/lower.go` — `lowerOnErrPipeChain()`

In a chain like `data |> step1() |> step2() |> step3() onerr panic "failed"`, the same handler is emitted for *each* error-returning step. If the handler has side effects (logging, metrics), it fires multiple times. This behavior is currently undocumented.

**Mitigation ideas:**
- Document this behavior in the language reference
- Consider whether a "run handler once on first error" mode would be useful (would require goto-based lowering for all pipeline onerr, not just piped switch)

---

## User Pain Points

### 1. No Per-Step Error Context

Pipeline-level `onerr` gives the same error message regardless of which step failed. Users who want `"step 2 failed: {error}"` have to break the chain apart manually:

```kukicha
# User wants different messages per step — must abandon pipeline onerr
parsed := rawData |> parse() onerr return empty, error "parse failed: {error}"
result := parsed |> transform() onerr return empty, error "transform failed: {error}"
```

**Possible improvement:** Annotate pipeline errors with step index or function name automatically, e.g. `"pipeline step 2 (transform) failed: original error"`.

### 2. Placeholder / Discard Ambiguity

The `_` token serves double duty as both "pipe placeholder" and "discard value." The parser handles this by treating both `Identifier("_")` and `DiscardExpr` as placeholders in pipe context, but it's a conceptual overlap that could confuse users reading generated code.

**Possible improvement:** No immediate action needed — the current behavior is correct. Document the dual role explicitly in the language reference.

### 3. Piped Switch Type Inference Is Opaque

When piped switch return type inference falls back to `"any"`, the user gets no warning. They just get a less-typed result that may cause issues downstream.

**Possible improvement:** Emit a compiler note/warning when type inference falls back to `"any"` in a piped switch.

### 4. Multi-Return Left Side Silently Discards Errors

When the left side of a pipe returns `(T, error)` and there's no `onerr`, the codegen wraps it in an IIFE that extracts the first value. The error is silently discarded.

```kukicha
# getResult returns (Data, error) — error silently dropped
result := getResult() |> transform()
```

**Possible improvement:** Emit a compiler warning when a multi-return pipe step discards an error without explicit `onerr` or `onerr discard`. This is arguably the most important improvement — it's a real footgun.

### 5. Limited Pipe Strategies

Only two strategies exist: data-first (default) and placeholder (`_`). There's no "data-last" or positional markers like `$1`, `$2`. For APIs that consistently take the "interesting" argument in a non-first position, users must always write `_`.

**Possible improvement:** Low priority. The placeholder strategy handles all cases. Adding more strategies would increase complexity without proportional benefit. If a pattern emerges where specific packages always need `_`, consider per-package defaults as a future feature.

---

## Proposed Changes (Prioritized)

### P0 — Warn on silent error discard in pipes (DONE)

When a pipe step returns `(T, error)` and there's no `onerr`, emit a compiler warning. This catches real bugs with minimal effort.

**Files:** `internal/semantic/semantic_expressions.go` (`warnPipeDiscardedErrors`), `internal/semantic/semantic_statements.go` (call sites), `internal/semantic/semantic_pipe_warning_test.go`

### P1 — Document pipeline onerr handler duplication (DONE)

Add a note to the language reference explaining that pipeline-level `onerr` handlers run once per failing step, not once per chain.

**Files:** `CLAUDE.md` (pipe section comment clarifying per-step semantics)

### P1 — Decouple test assertions from exact temp names (DONE)

Refactor `codegen_pipes_test.go` to use pattern-based assertions. This unblocks future lowerer refactoring.

**Files:** `internal/codegen/codegen_pipes_test.go` (8 assertions converted), `internal/codegen/test_helpers_test.go` (`mustContainPattern`, `mustNotContainPattern`, `extractMatch`)

### P2 — Warn on piped switch `"any"` fallback (DONE)

Emit a compiler warning when piped switch return type inference detects conflicting types across cases. The warning fires during semantic analysis when `mergePipedSwitchReturnType` produces `TypeKindUnknown` from two concrete types.

**Files:** `internal/semantic/semantic_statements.go` (`analyzePipedSwitchBody`), `internal/semantic/semantic_piped_switch_test.go`

### P2 — Add step context to pipeline onerr (DONE)

Each error-returning step in a pipeline onerr chain now gets a `// pipe step N: func(...)` comment in the generated Go code, identifying the source function for each error check.

**Files:** `internal/codegen/lower.go` (`lowerOnErrPipeChain`, `lowerOnErrPipeChainWithLabels`), `internal/codegen/codegen_pipes_test.go`

### P3 — Decouple lexer line continuation from indent stack

Refactor line continuation to a post-tokenization pass or separate state machine, reducing coupling to indent tracking.

**Files:** `internal/lexer/lexer.go`
