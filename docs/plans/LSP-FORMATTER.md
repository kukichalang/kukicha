# Long-Term Plan: LSP & Formatter

> Written 2026-04-08. Based on analysis of Sky language's LSP/formatter
> (github.com/anzellai/sky) and Kukicha's current implementation gaps.

## Formatter: Wadler-Lindig Doc Algebra

**Problem:** `printer.go` is 38KB of hand-managed indentation and string concatenation. Every construct manages its own line-breaking logic. Hard to maintain, easy to get wrong.

**Fix:** Rewrite the printer around a Doc algebra (as Sky does, and as used by `gofmt`, `rustfmt`, Prettier, elm-format).

- **File:** `internal/formatter/doc.go` (new, ~200 lines)
  - Types: `DocText`, `DocLine` (soft break), `DocHardline`, `DocConcat`, `DocIndent`, `DocGroup` (try flat, break if >80 cols), `DocAlign`
  - Renderer: greedy width-fitting algorithm
- **File:** `internal/formatter/printer.go` (rewrite)
  - Each AST node type gets a `func nodeToDoc(node) Doc` function
  - Operator-specific strategies: `|>` always breaks, `and`/`or` chain with soft breaks, etc.
  - Comment attachment handled during Doc construction (attach leading/trailing comments from the comment map)
- **Benefit:** Layout decisions become declarative. Adding a new AST node means writing one small function. Width-fitting is automatic.

## Formatter: Self-Hosted in Kukicha

**Problem:** Go's type system can't enforce exhaustive switches on AST interfaces. Runtime exhaustiveness tests (see immediate fixes) are a band-aid, not a structural guarantee.

**Fix:** Write the formatter (or at minimum the AST→Doc translation layer) in Kukicha itself, using Kukicha enums and `case` for exhaustive matching.

- Kukicha's `case` on enums is exhaustive at compile time
- Adding a new AST variant would cause a compile error in the formatter
- This is the structural fix for formatter-compiler drift (same pattern Sky uses)
- **Prerequisite:** Kukicha would need to be able to import its own `internal/ast` types, or the AST→Doc layer would need a Kukicha-native AST representation

## LSP: Cross-File Symbol Resolution

**Problem:** Current LSP is single-file only. Can't go-to-definition across files, no workspace-wide completion, no rename across files.

**Fix:**
- **File:** `internal/lsp/workspace.go` (new)
- On `initialize`, scan the workspace for `.kuki` files. Parse and cache all of them. On `didChangeWatchedFiles`, invalidate and re-parse changed files. Build a workspace-wide symbol index (name → URI + position).
- Wire into hover, completion, and definition handlers as a fallback after per-document lookup.
- Resolve imports: for `import "stdlib/slice"`, look up stdlib registry; for local imports, look up workspace index.

## LSP: Multi-Level Hover Cascade

**Problem:** Hover currently only checks the AST/symbol table of the current document. No stdlib info, no Go stdlib info, no cross-file types.

**Fix:** Implement a cascade (inspired by Sky's 5-level fallback):
1. Current document's semantic analysis (already done)
2. Kukicha stdlib registry (`stdlib_registry_gen.go`) — show function signatures + docstrings
3. Go stdlib registry (`go_stdlib_gen.go`) — show Go stdlib function signatures
4. Workspace-wide symbol table (from cross-file resolution above)
5. Raw source heuristic fallback (grep for definition in imported files)

- **Files:** `internal/lsp/hover.go` (extend), new helper for registry lookups
- **Data:** Both registries are already generated and available as Go maps

## LSP: Semantic Tokens

**Problem:** Syntax highlighting currently relies on tree-sitter / TextMate grammars in the editor extensions. The LSP could provide more accurate semantic tokens.

**Fix:**
- **File:** `internal/lsp/semantic_tokens.go` (new)
- Walk the AST and emit semantic token spans with types (function, variable, type, keyword, enum, enumMember, etc.). Register `semanticTokensProvider` capability.
- **Benefit:** More accurate highlighting than regex-based grammars, especially for Kukicha-specific constructs like `onerr`, pipe operators, and `reference`/`dereference`.

## Priority Order

1. Cross-file resolution — biggest user-facing gap in the LSP
2. Hover cascade with registries — leverages existing generated data
3. Doc algebra — formatter maintainability
4. Semantic tokens — better highlighting
5. Self-hosted formatter — structural fix, but requires language maturity
