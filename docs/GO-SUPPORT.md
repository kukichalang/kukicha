# Bridging Kukicha to Go: Graduation Path

Kukicha's verbose English syntax is intentional — it's readable, auditable, and AI-friendly. But some constructs are genuinely painful to type repeatedly. Rather than introducing a dialect system or wholesale Go syntax support, we should be selective: add aliases that earn their keep and let tooling handle the rest.

## Done

### 1. Bracket Type Aliases (`[]T`, `map[K]V`)

Implemented in the parser — both syntaxes produce identical AST nodes, so semantic analysis and codegen required no changes.

**Type positions** (parameters, return types, variable declarations):
- `[]T` as an alias for `list of T`
- `map[K]V` as an alias for `map of K to V`
- Fully recursive: `[]map[string][]int` works

**Expression positions** (literals, typed-empty shorthand):
- `[]int{1, 2, 3}` — typed list literal
- `map[string]int{"a": 1}` — typed map literal
- `[]string` / `map[string]int` bare — typed-empty shorthand (zero value)

Note: we chose `map[K]V` over `[K]V` to avoid ambiguity with untyped list literals (`[expr, ...]`) in expression position. This matches Go's actual syntax.

### 2. Untyped Map Literals (`{key: val}`)

`{"a": 1, "b": 2}` now parses as a `MapLiteralExpr` with nil key/value types. Codegen defaults to `map[any]any`. Empty `{}` also works.

## TODO

### 3. Invest in `kukicha fmt` Expansion/Collapsing

Instead of a per-file dialect flag, let tooling bridge the gap:

- AI generates verbose Kukicha (easy to audit)
- `kukicha fmt --short` collapses to bracket aliases and shorthand literals
- `kukicha fmt --long` expands back to full English form

This keeps a single canonical language while letting humans work in whichever form they prefer. One language, one set of tools, no fragmentation.

## Deferred

### Symbolic Pointer Aliases (`*T`, `&x`)

Pointers are where Go beginners struggle the most. `reference` and `dereference` are long, but they're one of Kukicha's biggest educational wins — they make pointer semantics explicit and readable. Adding `*`/`&` as aliases would undermine that value and encourage people to skip the understanding step.

Revisit this once the formatter can round-trip between forms, so beginners always see the English version even if experienced users type the symbols.

### Syntax Pragma / Dialect Flag

File-level dialect modes (e.g., `syntax hybrid`) fragment the language. Every tool — formatter, LSP, linter — would need to understand which dialect a file is in, and a single project could contain files that look completely different. The per-keyword alias approach (like `==` for `equals`) is the right granularity. A per-file mode switch is not.

## Design Principle

Each alias must justify the testing, documentation, formatter, and LSP work it requires. "Just a few lines in the Lexer" undersells the real cost. Be selective — add the shortcuts that solve genuine pain, and let tooling handle the rest.
