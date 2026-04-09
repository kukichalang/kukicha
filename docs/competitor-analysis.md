# Competitor Analysis: Go Supersets / Transpilers

*Evaluated: April 2026 | Kukicha v0.1.3*

This document evaluates three Go-targeting transpiler projects against Kukicha across
the dimensions that matter to language adopters: Go compatibility, error handling,
type system extensions, tooling, and adoption trajectory.

---

## Projects Under Review

| Project | Repo | Stars | Status | License |
|---------|------|-------|--------|---------|
| **Kukicha** | kukichalang/kukicha | — | Active, v0.1.3 | — |
| **Dingo** | MadAppGang/dingo | ~1.9k | Active, targeting v1.0 | Apache 2.0 |
| **Soppo** | halcyonnouveau/soppo | ~61 | Active | BSD 3-Clause |
| **Gala** | martianoff/gala | ~7 | Early-stage | Apache 2.0 |

---

## 1. Go Compatibility

This is the sharpest differentiator between Kukicha and the competition.

| Project | Approach | All Go valid? |
|---------|----------|---------------|
| **Kukicha** | Strict superset — every valid `.go` file is valid `.kuki` | Yes |
| **Dingo** | Separate syntax that transpiles to Go | No |
| **Soppo** | Separate syntax (Rust-implemented compiler) | No |
| **Gala** | Entirely new syntax (ANTLR4 grammar, Bazel build) | No |

**Kukicha's advantage**: teams can adopt incrementally. Individual files or packages
can migrate from `.go` to `.kuki` without rewriting anything. The other three require
a complete rewrite into a new language at adoption time.

---

## 2. Error Handling

| Project | Mechanism | Notes |
|---------|-----------|-------|
| **Kukicha** | `onerr` block — inline fallback logic after any expression | Keeps error handling local; no operator magic |
| **Dingo** | `?` propagation + `Result<T, E>` / `Option<T>` types | Familiar from Rust; requires function return type to be `Result` |
| **Soppo** | `?` propagation + custom handling blocks | Similar to Dingo; requires changing function signatures |
| **Gala** | Monadic `Try[T]` / `Either[A,B]` with `.Map`/`.FlatMap` | Go multi-return auto-wraps into `Try[T]`; most functional, steepest learning curve |

Dingo and Soppo both adopt the Rust `?` idiom, which is ergonomic but forces callers
to use `Result` return types — a significant signature change. Kukicha's `onerr` works
with idiomatic Go `(T, error)` returns unchanged.

---

## 3. Enums and Sum Types

| Project | Feature | Data variants? | Exhaustive matching? |
|---------|---------|----------------|----------------------|
| **Kukicha** | `enum` (integer or string typed constants) | No (value enums only) | Yes — compiler warns on missing `when` cases; `String()` auto-generated |
| **Dingo** | `enum` with tagged union support | Yes (`Ok(value: int)`) | Yes (`match` expressions) |
| **Soppo** | Tagged unions | Yes | Yes |
| **Gala** | `sealed type` with `case` variants | Yes | Yes (compiler-enforced) |

Kukicha enums are exhaustiveness-checked (switch warns on missing cases unless `otherwise`
is present) and generate a `String()` method automatically — both features the competition
also claims. The real gap is **data-carrying variants**: Dingo, Soppo, and Gala support
tagged unions where enum cases carry associated values (e.g. `Ok(value: int)`).
Kukicha's enums are value-only (integer or string), which is correct for status codes and
flags but cannot express result/option types natively.

---

## 4. Pipes

| Project | Pipe support |
|---------|-------------|
| **Kukicha** | Yes — `\|>` operator, unique in this space |
| **Dingo** | No |
| **Soppo** | No |
| **Gala** | Via `.Map`/`.FlatMap` on monadic types only |

Kukicha is the only project in this group with a first-class pipe operator.
This is a genuine differentiator for data-transformation and middleware pipelines.

---

## 5. Syntax Philosophy

| Project | Philosophy |
|---------|-----------|
| **Kukicha** | English-friendly aliases, readable operators (`and`, `or`, `not`), both brace and indent accepted |
| **Dingo** | TypeScript-influenced: type annotations after name (`name: type`), `->` return type |
| **Soppo** | "If you know Go, you know most of Soppo" — minimal surface-area changes |
| **Gala** | Functional-first: `val`, `sealed`, expression functions, immutable by default |

Kukicha's dual-syntax approach (Go braces _and_ Python-style indentation) has no
equivalent in the competition. It is uniquely friendly to beginners while remaining
idiomatic to Go veterans.

---

## 6. Standard Library

| Project | Stdlib |
|---------|--------|
| **Kukicha** | Purpose-built stdlib in `.kuki` (slice, ctx, etc.) with security directives |
| **Dingo** | None beyond Go's stdlib; Result/Option are runtime types |
| **Soppo** | Unknown from public docs |
| **Gala** | Rich immutable collections (List, Array, HashMap, HashSet, TreeSet, TreeMap) + monads |

Gala's functional collection library is the richest. Kukicha's stdlib is growing and
has a unique security annotation system (`# kuki:security`) with no equivalent elsewhere.

---

## 7. Tooling

| Feature | Kukicha | Dingo | Soppo | Gala |
|---------|---------|-------|-------|------|
| LSP | Yes | Yes | Yes | Yes |
| Formatter | Yes | No explicit mention | Yes | No explicit mention |
| Linter | `make lint` (golangci-lint) | No | Yes | No |
| Vulnerability audit | `kukicha audit` | No | No | No |
| Playground | No | No | Yes | Yes |
| WASM target | Yes (`--wasm`) | No | No | No |
| Vuln check at build | `--vulncheck` | No | No | No |

Kukicha's `kukicha audit` and `--vulncheck` flag are unique. The WASM build target is
also not present in any competitor.

---

## 8. Implementation & Portability

| Project | Compiler language | Output |
|---------|------------------|--------|
| **Kukicha** | Go | Go source → binary |
| **Dingo** | Go (assumed) | Go source → binary |
| **Soppo** | **Rust** | Go source → binary |
| **Gala** | Go + ANTLR4, Bazel | Go source → binary |

Soppo's Rust dependency is a non-trivial barrier for Go teams. Gala's Bazel requirement
adds build-system complexity. Kukicha and Dingo install like any Go tool.

---

## Summary: Kukicha Strengths and Gaps

### Strengths (unique or best-in-class)

- **100% Go backward compatibility** — incremental adoption, no rewrites
- **Pipe operator** — not offered by any competitor
- **Security annotations** (`# kuki:security`, `--vulncheck`, `kukicha audit`)
- **WASM build target**
- **Dual indentation/brace syntax** — widest readability range
- **`kukicha-blend`** — automated Go→Kukicha migration, nothing like it elsewhere
- **Pure Go toolchain** — no Rust, no ANTLR, no Bazel

### Gaps relative to competition

| Gap | Who has it |
|-----|-----------|
| Data-carrying enum variants (tagged unions) | Dingo, Soppo, Gala |
| `?` / propagating error operator | Dingo, Soppo |
| Nil / Option safety | Dingo, Soppo |
| Immutable-by-default collections | Gala |
| Online playground | Soppo, Gala |

### Competitive Positioning

- **vs Dingo** (most dangerous competitor at 1.9k stars): Dingo wins on sum types and
  `Result` ergonomics; Kukicha wins on Go compatibility, pipes, security tooling, and
  migration story. Dingo is heading toward v1.0 — watch its release trajectory.
- **vs Soppo**: Soppo wins on nil safety and Rust-style diagnostics; Kukicha wins on
  everything else. Soppo's Rust dependency is a friction point.
- **vs Gala**: Gala is very early (7 stars) and Bazel-dependent. Not a near-term threat,
  but its functional collection library is worth tracking.

---

## Recommendations

1. **Prioritize data-carrying enums + `match`** — this is the biggest perceived gap and
   the feature most prominently advertised by Dingo, Soppo, and Gala.
2. **Build a playground** — both Soppo and Gala have one; it's a key top-of-funnel tool.
3. **Monitor Dingo's v1.0 release** — at 1.9k stars it has the largest existing audience
   and is the most direct threat once it stabilizes.
4. **Lean into the migration story** — `kukicha-blend` is genuinely unique; it should be
   front-and-center in marketing, since none of the competitors offer a migration path
   from existing Go codebases.
