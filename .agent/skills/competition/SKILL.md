---
name: competition
description: Scan recent releases from competitor Go-superset/transpiler languages (soppo, dingo, gala, sky) for ideas useful in Kukicha's compiler, stdlib, LSP, or tooling. Tracks already-reviewed releases to avoid duplication. Use when the user says "check competitor releases", "what are competitors doing", "scan competition", or similar.
---

# Competition Release Tracker

Monitors four Go-adjacent language projects for new releases and extracts ideas
relevant to Kukicha. Uses `seen-releases.json` as a persistent deduplication log
so only genuinely new releases are analysed each run. Note: competition is used in a "tongue-in-cheek" manner, we are all scratching our own itch here.

---

## Tracked Repositories

| Repo | What it is | Relevance |
|------|-----------|-----------|
| `halcyonnouveau/soppo` | Go superset (compiled via Rust). Adds nil-safety, exhaustive pattern matching, `?` error propagation, LSP, formatter, linter. | Enum type-checking, nil-safety patterns, diagnostic quality |
| `MadAppGang/dingo` | Go meta-language transpiler (Go→Go). Result types, sum types, pattern matching, LSP, formatter, watch mode. | Error propagation UX, watch command, lambda inference, source maps |
| `martianoff/gala` | Go transpiler with Scala-like FP (sealed types, Option/Either/Try, ANTLR4 parser). Active LSP development. | LSP architecture, sealed-enum completion, chain completion, diagnostic line-number accuracy |
| `anzellai/sky` | Self-hosted Go-targeting language. Hindley-Milner inference, ADTs, Elm-style record constructors, fullstack. | Type system ideas, memoisation of zero-arity decls, Result/Task applicatives |

---

## How to Run a Scan

### Step 1 — Read the tracking file

Read `.claude/skills/competition/seen-releases.json`. Note:
- `last_scan` date
- Per-repo `seen` arrays (release tag names already reviewed)

### Step 2 — Fetch releases pages

For each repo, fetch `https://github.com/<owner>/<repo>/releases` and ask:
> "List every release tag and date on this page. I need the tag name, publish date, and a summary of the changelog."

Run all four fetches in parallel.

### Step 3 — Filter to new-only

Cross-reference each release tag against the `seen` list for that repo. Discard
anything already in `seen`. Only analyse releases **not** in `seen` that were
published within the last 30 days.

If nothing is new, report "No new releases since last scan on `<last_scan>`" and stop.

### Step 4 — Analyse each new release

For every new release, apply the Analysis Lens below and record findings.
Group findings under: **Compiler**, **LSP/Tooling**, **Language Features**, **Stdlib**, **DX/UX**.

### Step 5 — Update the tracking file

Append every newly examined tag to the `seen` array for its repo and update
`last_seen_at`. Update `last_scan` to today's date. Write the file back.

### Step 6 — Report

Produce a short markdown report:

```
## Competition Scan — <date>

### New releases reviewed
- soppo vX.Y.Z (date): ...
- sky vA.B.C (date): ...

### Ideas worth exploring for Kukicha
#### Compiler
- ...
#### LSP / Tooling
- ...
#### Language Features
- ...
#### Stdlib
- ...
#### DX / UX
- ...

### Skipped (already seen or outside 30-day window)
- dingo: no new releases
```

---

## Analysis Lens

When reading a release, look specifically for things Kukicha could adopt or
learn from. Map each finding to one of these buckets:

### Compiler
- Semantic error quality: accurate line/column numbers, zero-placeholder elimination
- Cross-file / multi-package type resolution
- Source maps and their impact on error UX
- Lambda / closure type inference strategies
- Memoisation of constant/zero-arity expressions at compile time
- Annotation-driven type propagation (load-bearing annotations)
- Enum exhaustiveness checking improvements
- Nil/nilable type tracking in multi-var assignment

### LSP / Tooling
- Cancel-and-restart diagnostics pattern (avoid stale errors after edits)
- Dot-completion for chained method returns
- Sibling-file discovery within packages
- Sealed / variant enum completion in switch branches
- Inlay hints (type annotations inline in editor)
- Document symbols
- Watch mode (`dingo watch`-style hot rebuild + restart)
- Cross-package method completion
- Grouped import generation

### Language Features
- Exhaustive pattern matching with sealed/variant enums
- Record constructors from type aliases (Elm-style)
- Applicative combinators (Result, Option, Task/Future)
- Error propagation operators (`?`, `onerr` cousins)
- Hindley-Milner inference (where applicable to Kukicha's typed model)
- Immutability-by-default for value types

### Stdlib

Look for packages or functions in competitor stdlibs that Kukicha's `stdlib/` is missing or could improve. Cross-reference against the existing Kukicha stdlib packages before flagging — avoid duplicating what's already there. When a finding is actionable, follow the `/stdlib` skill rules (security directives, `make genstdlibregistry`, etc.).

- **Collection helpers**: `map`, `filter`, `reduce`, `flatMap`, `zip`, `partition` on slices and maps — gap-fill for `stdlib/slice` and `stdlib/maps`
- **Result / Option types**: combinator functions (`map`, `flatMap`, `orElse`, `unwrapOr`) that wrap Go's `(T, error)` pattern into a chainable form
- **JSON ergonomics**: typed decode helpers, streaming, schema validation — improvements to `stdlib/json`
- **Concurrency primitives**: worker pools, fan-out/fan-in, rate limiters, cancellable tasks — additions to `stdlib/sync` or new packages
- **String utilities**: slugify, truncate, pad, word-wrap, template helpers — `stdlib/str` gaps
- **Math / numeric**: clamping, rounding modes, safe integer ops — `stdlib/math` gaps
- **Time / duration helpers**: humanise, parse natural language, truncate to boundary — `stdlib/time` gaps
- **Security-adjacent**: constant-time comparison, secure random, base64url — note any `# kuki:security` directives competitors attach to dangerous functions

### DX / UX
- CRLF / Windows line-ending handling in lexer
- Diagnostic grouping and formatting
- IDE plugin completeness (IntelliJ, VS Code)
- Self-hosting milestones (credibility signal)
- Binary size / startup time

---

## Mapping Findings to Kukicha Code

When a finding is actionable, point to the relevant Kukicha subsystem:

| Finding area | Kukicha subsystem |
|---|---|
| Lexer (CRLF, indentation) | `internal/lexer/` |
| Parser / AST | `internal/parser/`, `internal/ast/` |
| Type checking, enum exhaustiveness | `internal/semantic/` |
| Code generation | `internal/codegen/` |
| LSP server | `cmd/kukicha/` (LSP subcommand), `internal/` |
| Formatter | `internal/formatter/` |
| Stdlib additions | `stdlib/` — follow `/stdlib` skill rules |
| CLI commands (watch, audit) | `cmd/kukicha/` — follow `/cmd` skill rules |

---

## Initial Scan — 2026-04-10

_Baseline established on first skill invocation. All tags below are marked seen._

### soppo

**v0.10.1** (2026-01-28) — last 30-day window release
- Grouped import generation → **LSP / Tooling**: could inform `kukicha fmt` grouping heuristics
- CRLF line-ending fix → **Compiler/Lexer**: worth auditing `internal/lexer/` for `\r\n` handling
- Multi-var assignment for nilable types → **Compiler**: edge case for Kukicha's tuple/multi-return lowering

**v0.10.0** (2025-12-27) — outside 30-day window, logged for deduplication
- Attribute system, enum type checking, const field handling

### dingo

**v0.9.0** (2026-01-08) — outside 30-day window, logged for deduplication
- `dingo watch` hot-reload → **DX**: `kukicha run --watch` could be a useful addition
- Cross-file type resolution → **Compiler**: already partially covered by Kukicha's directory build merge
- Error propagation for external package methods → **Compiler**: `onerr` on external method chains

**v0.6.0** (2025-12-10) — outside window
- 4-layer lambda type inference, source map system, linter/formatter, Neovim plugin

**v0.5.0** (2025-12-09) — outside window  
- Adopted Go-native generic syntax `[T]`

**v0.3.0** (2025-11-18) — outside window  
- Result and Option types, error propagation operators

### gala

_(v0.26.0 – v0.29.6, all April 2026 — within 30-day window)_

**v0.29.6 / v0.29.5** (2026-04-10)
- Unified stdlib resolution across CLI and LSP → **LSP/Tooling**: Kukicha's LSP should share the same stdlib registry that `kukicha check` uses
- Sibling file discovery in packages → **LSP/Tooling**: helps LSP find context in directory builds
- Cancel-and-restart diagnostics pattern → **LSP/Tooling**: prevents stale red squiggles after fast edits; implement in kukicha-lsp

**v0.29.4** (2026-04-10)
- All 34 semantic errors now carry accurate line numbers (eliminated zero-line placeholders) → **Compiler**: audit `internal/semantic/` for any `Line: 0` in error construction

**v0.29.3** (2026-04-09)
- Chain completion for method returns → **LSP/Tooling**: dot-complete on pipe results
- `gala.mod` dependency resolution → lower priority (Kukicha uses Go modules directly)

**v0.29.2** (2026-04-09)
- Scoped type inference fix for same-name variables across functions → **Compiler**: scope isolation bug class; check `internal/semantic/` resolver scope stack
- Cross-package method completion → **LSP/Tooling**

**v0.29.1** (2026-04-08)
- Sealed case completion in switch → **LSP/Tooling**: variant enum branch completion for `switch` on enum values
- Parse error line numbers → **Compiler**: parser should always attach source position to errors

**v0.29.0** (2026-04-08) — major release
- `TransformForLSP` API: separate type-inference pass for LSP (doesn't need to emit Go) → **LSP/Tooling**: consider a lightweight `kukicha analyze` mode that skips codegen
- 84 LSP test functions, zero skips → **Testing**: model for building out kukicha-lsp test suite

**v0.28.0** (2026-04-06)
- `gala lsp` as CLI subcommand → **LSP/Tooling**: Kukicha already has `kukicha-lsp`; verify it's wired as subcommand too
- Built-in function recognition (print, Go interop) → **Compiler**: ensure Kukicha's semantic pass knows about `println`, `panic`, etc.

**v0.27.0** (2026-04-06)
- LSP 3.17 (inlay hints, document symbols) → **LSP/Tooling**: upgrade protocol version for inlay type hints
- IntelliJ plugin integration

**v0.26.0** (2026-04-06) — LSP launch
- ANTLR-based parser for LSP (incremental/error-tolerant) → **LSP/Tooling**: error-tolerant parsing is key for good IDE experience; Kukicha's recursive-descent parser currently fails hard on syntax errors

### sky

_(v0.7.24 – v0.7.33, April 2026 — within 30-day window)_

**v0.7.28** (recent)
- Type system overhaul: annotations are now load-bearing (drive inference) → **Compiler**: consider whether Kukicha type annotations could drive backward inference in pipes

**v0.7.26** (recent)
- Auto record constructors from type aliases (Elm-style) → **Language Feature**: not directly applicable but related to Kukicha struct literal ergonomics

**v0.7.25** (recent)
- Applicative combinators for Result and Task → **Stdlib**: Kukicha's stdlib/result could gain `map`, `flatMap`, `apply` if Result type is introduced

**v0.7.29** (recent)
- `Task.perform` returns Result uniformly → **Language Feature**: relates to Kukicha's `onerr` — uniform error-return from async-like operations

**v0.7.30** (recent)
- Memoize zero-arity decls (Ref bug fix) → **Compiler**: zero-arity function calls in const/top-level position could be memoised; check if Kukicha has analogous patterns

**v0.7.27** (recent)
- JSON pipeline runtime panic + Decoder type sigs → **Stdlib**: Kukicha's `stdlib/json` type signatures; ensure decode generics are correct

---

## Top Actionable Ideas (from initial scan)

Priority-ordered suggestions for Kukicha backlog consideration:

1. **Cancel-and-restart LSP diagnostics** (gala v0.29.5) — prevents stale errors during fast typing. High impact for IDE UX.
2. **Semantic errors always carry line numbers** (gala v0.29.4) — audit `internal/semantic/` for `Line: 0` constructions.
3. **Error-tolerant parser for LSP** (gala v0.26.0) — current hard-fail parser breaks completion mid-edit. Consider partial-parse fallback.
4. **Unified stdlib registry for CLI + LSP** (gala v0.29.5) — LSP should use same `stdlib_registry_gen.go` as `kukicha check`.
5. **`kukicha run --watch`** (dingo v0.9.0) — hot-rebuild for dev loop; maps to `cmd/kukicha/` watch subcommand.
6. **CRLF in lexer** (soppo v0.10.1) — small robustness fix for Windows users; `internal/lexer/`.
7. **Grouped import formatting** (soppo v0.10.1) — `kukicha fmt` could group stdlib / third-party / local imports.
8. **Variant enum branch completion** (gala v0.29.1) — LSP completes missing cases when cursor is inside `switch` on enum.
9. **Inlay hints (LSP 3.17)** (gala v0.27.0) — show inferred types inline; particularly useful for pipe chains.
10. **`TransformForLSP` lightweight analysis pass** (gala v0.29.0) — decouple type-check from codegen for faster LSP response.
