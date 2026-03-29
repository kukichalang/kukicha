---
name: compiler-internals
description: Kukicha compiler internals — lexer, parser, AST, semantic analysis, IR, and codegen architecture. Use when adding language features, debugging compiler bugs, or modifying any package under internal/.
---

# internal/ — Kukicha Compiler Internals

Compiler internals reference. Read this when working in `internal/`. For language syntax and build commands see the root `CLAUDE.md`.

## Pipeline Overview

```
source (.kuki)
  → lexer/     — runes → []Token (INDENT/DEDENT injected)
  → parser/    — []Token → *ast.Program
  → semantic/  — validates AST, infers return counts, enforces security checks
  → codegen/   — *ast.Program → IR (via Lowerer) → Go source string (via emitIR)
```

Semantic analysis produces two maps passed to codegen:
- `exprReturnCounts map[ast.Expression]int` — passed via `generator.SetExprReturnCounts(...)`. Tells codegen how many values an expression returns so it can emit the right `val, err := f()` split for `onerr`.
- `exprTypes map[ast.Expression]*TypeInfo` — passed via `generator.SetExprTypes(...)`. Records inferred type of every analyzed expression. Used by codegen for: error-only pipe step detection (`isErrorOnlyReturn`), piped switch return type inference, `empty` keyword resolution, typed zero-value generation (`zeroValueForType`). In `analyzePipeExprMulti`, types are explicitly recorded on pipe step nodes via `recordType(right, types[0])` since steps bypass `analyzeExpression`. Pipe placeholder `_` identifiers get the piped value's type recorded when inside a call with a known function signature.

The formatter (`formatter/`) is a separate pipeline that re-parses and pretty-prints. The LSP (`lsp/`) wraps the compiler pipeline and is independent of the above.

## Package Overview

| Package | Role | Key entry point |
|---------|------|-----------------|
| `lexer/` | Tokenization (INDENT/DEDENT, string interpolation) | `NewLexer(source, file).ScanTokens()` |
| `parser/` | Recursive descent parser → AST | `New(source, file)` then `Parse()` |
| `ast/` | AST node definitions (no logic) | Node interfaces: `Declaration`, `Statement`, `Expression`, `TypeAnnotation` |
| `semantic/` | Type checking, symbol resolution, security checks | `New(program).Analyze()` |
| `ir/` | Intermediate representation (Go-level imperative nodes) | `Block`, `Assign`, `IfErrCheck`, `Goto`, `Label` |
| `codegen/` | AST → IR lowering → Go source emission | `New()` then `Generate()` |
| `formatter/` | Kukicha source code formatting | `Format(source, file, opts)` |
| `lsp/` | Language Server Protocol implementation | `NewServer(reader, writer).Run(ctx)` |
| `version/` | Single `const Version` for the compiler | `version.Version` |

---

## Lexer (`lexer/`)

**Key files:** `lexer.go`, `token.go`

### INDENT/DEDENT

Kukicha is indentation-sensitive. The lexer converts 4-space indentation changes into `TOKEN_INDENT` / `TOKEN_DEDENT` tokens using an `indentStack []int` (always starts at `[0]`).

- Indentation must be multiples of 4 spaces — tabs are rejected
- Each increase must be exactly +4 spaces
- Dedents can skip multiple levels (e.g., 8→0 emits two `TOKEN_DEDENT`)
- Blank lines and comment-only lines do not affect the indent stack
- Error messages include actionable detail (e.g., nearest valid indent level, valid dedent targets)

### Line continuation

`TOKEN_NEWLINE` is suppressed (continuation mode) in two ways:

**Inline (during tokenization):** Inside `[]` or `{}` (`braceDepth > 0`), `TOKEN_NEWLINE` is suppressed and `continuationLine` is set so the next line's indentation is consumed without emitting INDENT/DEDENT. `()` (parentheses) do NOT suppress newlines when inside a function literal body — closures need `INDENT/DEDENT` for their block structure.

**Post-pass (`mergeLineContinuations`):** Pipe continuation (`|>`) and `onerr` on continuation lines are handled after tokenization. The lexer emits NEWLINE/INDENT/DEDENT normally; the post-pass removes them around pipe chains. This decouples pipe handling from the indent stack. Three patterns are merged:
1. Trailing pipe: `PIPE [COMMENT*] NEWLINE [INDENT*]` → remove NEWLINE and INDENTs
2. Leading pipe: `NEWLINE [INDENT*] PIPE` → remove NEWLINE and INDENTs (no DEDENTs allowed)
3. Leading onerr: `NEWLINE [INDENT*] ONERR` → same as (2), only in pipe chain context

For each INDENT absorbed, a corresponding DEDENT is also absorbed later in the stream.

### Adding a new keyword

Add the keyword string → `TokenType` mapping in `token.go`'s `keywords` map and define the `TokenType` constant there.

### Directives (`TOKEN_DIRECTIVE`)

Comments starting with `# kuki:` are emitted as `TOKEN_DIRECTIVE` instead of `TOKEN_COMMENT`. The lexer's `scanComment` checks the prefix and selects the token type. `TOKEN_DIRECTIVE` is excluded from `lastTokenType` tracking (like `TOKEN_COMMENT`).

### String escape sequences and PUA sentinels

`scanString` handles escape sequences in the switch on the character after `\`. Two kinds of escapes exist:

**Compile-time (character substitution):** The escaped sequence maps directly to a Unicode Private Use Area (PUA) sentinel stored in the token value. Codegen's `escapeString` converts the sentinel back to the literal character when emitting Go string literals.

| Escape | PUA sentinel | Emitted as |
|--------|-------------|-----------|
| `\{`   | `\uE000`    | literal `{` |
| `\}`   | `\uE001`    | literal `}` |

**Runtime (expression injection):** The escape expands to a Go expression evaluated at runtime. The sentinel is included in literal token lexemes; codegen's `generateStringFromParts` detects `\uE002` and expands it.

| Escape  | PUA sentinel | Emitted as |
|---------|-------------|-----------|
| `\sep`  | `\uE002`    | `string(filepath.Separator)` — auto-imports `path/filepath` |

`\sep` is a multi-character escape: `scanStringEscape` checks `l.peek() == 'e' && l.peekNext() == 'p'` before consuming the `ep` suffix.

`generateStringLiteral` and `exprHasNonPrintfInterpolation` (in `codegen_walk.go`) both check `strings.ContainsRune(value, '\uE002')` to correctly handle strings that contain `\sep` but no `{expr}` interpolation.

### String interpolation tokenization

For interpolated strings (containing `{expr}`), the lexer emits multiple tokens instead of a single `TOKEN_STRING`:

| Token | Purpose | Example in `"Hello {name}, age {age}!"` |
|-------|---------|------------------------------------------|
| `TOKEN_STRING_HEAD` | Leading literal before first `{` | `"Hello "` |
| `TOKEN_STRING_MID` | Literal between two interpolations | `", age "` |
| `TOKEN_STRING_TAIL` | Trailing literal after last `}` | `"!"` |

Between HEAD→MID and MID→MID/TAIL, normal expression tokens are emitted. The parser calls `parseExpression()` directly on these tokens — no sub-parser needed.

**Brace depth tracking:** `interpStack []interpState` on the `Lexer` tracks nesting within each interpolation level. Each `interpState` stores `braceDepth int` and `quote rune` (either `'"'` or `'\''`). `{` inside an interpolation increments the top entry's `braceDepth`; `}` at depth 0 ends the interpolation and resumes string scanning via `scanStringContinuation(quote)`, which uses the stored quote to know which delimiter terminates the string. This correctly handles nested braces like `{MyStruct{field: 1}}` and interpolation in both double-quoted and single-quoted strings.

### Single-quote multi-line strings

`scanSingleQuoteString()` scans `'...'` strings. These are multi-line with auto-dedent (same as `"""..."""`), but use single quotes — ideal for HTML templating where double quotes are used for attributes. The raw content is collected, then passed to `dedentTripleQuote()` and `scanStringFromContent()` for interpolation tokenization. Escape `\'` for literal single quotes inside.

**Interpolation detection:** `isInterpStart()` checks if the character after `{` is alpha or `_`. Non-identifier starts like `{2,}` are treated as literal text.

Non-interpolated strings still emit a single `TOKEN_STRING`.

---

## Parser (`parser/`)

**Key files** (split from the original monolithic `parser.go`):

| File | Contents |
|------|---------|
| `parser.go` | Core struct, `New`, `Parse`, token helpers (`peekToken`, `consume`, `advance`, …) |
| `parser_type.go` | `parseTypeAnnotation` and all type sub-parsers |
| `parser_decl.go` | Declaration parsers (`parseFunctionDecl`, `parseTypeDecl`, `parseVarDeclaration`, …) |
| `parser_stmt.go` | Statement parsers (`parseBlock`, `parseStatement`, `parseIfStmt`, `parseForStmt`, `parseOnErrClause`, …) |
| `parser_expr.go` | Expression parsers (`parseExpression`, `parsePipeExpr`, `parseArrowLambda`, …) |

### Design

- Recursive descent
- **Error collection** (not fail-fast): errors are appended to `p.errors`, parsing continues. This allows multiple errors per compile.
- `peekToken()` calls `skipIgnoredTokens()` first, which skips `TOKEN_COMMENT` and `TOKEN_SEMICOLON`
- Context-sensitive keywords: `list`, `map`, `channel` are only keywords when followed by `of` in a type context — this allows them as variable names elsewhere. `empty` and `error` are context-sensitive too: `isIdentifierFollower()` checks if the next token indicates identifier usage (`:=`, `=`, `&`, `.`, `[`, `:`, `|>`, `)`, `,`, string interpolation mid/tail, etc.); if so, they parse as identifiers instead of `EmptyExpr`/`ErrorExpr`. This means `empty |> iterator.Values()`, `print(empty)`, and `empty.field` all work when `empty` is a user-defined variable.

### Operator precedence (lowest → highest)

or → pipe (`|>`) → and → bitwise or/and → comparison → additive → multiplicative → unary → postfix → primary

### Key helpers

| Helper | Purpose |
|--------|---------|
| `peekToken()` | Look at next meaningful token (skips comments/semicolons) |
| `consume(type, msg)` | Advance and return token, or record error |
| `skipNewlines()` | Skip `TOKEN_NEWLINE` tokens |
| `parseBlock()` | Parse `INDENT … DEDENT` block into `*ast.BlockStmt` |
| `parseTypeAnnotation()` | Parse any Kukicha type (`list of T`, `map of K to V`, `reference T`, etc.) |
| `isIdentifierFollower()` | Returns true if next token indicates `empty`/`error` is being used as an identifier |

### Adding a new statement

1. Add a `TOKEN_*` keyword in `lexer/token.go`
2. In `parser_stmt.go`'s `parseStatement()` switch, add a `case TOKEN_*:` branch calling your new `parseXxxStmt()` method
3. Return the new `*ast.XxxStmt` node (defined in `ast/ast.go`)

### Adding a new expression

1. Hook into `parsePrimaryExpr()` in `parser_expr.go` for new literal/prefix forms, or the operator precedence chain for binary forms
2. Return a new `*ast.XxxExpr` node

**Public API:** `New(source, filename)`, `NewFromTokens(tokens)`, `Parse()`, `Errors()`

---

## AST (`ast/`)

**Key file:** `ast.go` (~1030 lines)

### Interface hierarchy

```
Node
├── Declaration  (declNode marker)  — FunctionDecl, TypeDecl, EnumDecl, ImportDecl, …
├── Statement    (stmtNode marker)  — IfStmt, ForRangeStmt, ReturnStmt, …
├── Expression   (exprNode marker)  — CallExpr, PipeExpr, ArrowLambda, …
└── TypeAnnotation (typeNode marker) — ListType, MapType, ReferenceType, …
```

### Convention for new nodes

Every node must implement `Node`:
```go
type XxxStmt struct {
    Token lexer.Token  // The keyword token (for position info)
    // ... fields
}

func (s *XxxStmt) TokenLiteral() string { return s.Token.Lexeme }
func (s *XxxStmt) Pos() Position {
    return Position{Line: s.Token.Line, Column: s.Token.Column, File: s.Token.File}
}
func (s *XxxStmt) stmtNode() {} // or declNode() / exprNode() / typeNode()
```

Always store the keyword's `lexer.Token` as the first field — it carries line/column for error messages.

**Notable:** `VarDeclStmt` implements both `Statement` and `Declaration`.

### EnumDecl and EnumCase

`EnumDecl` represents an `enum` declaration with named integer or string constants. `EnumCase` holds a single case name and its literal value. Directives (`# kuki:deprecated`) can annotate the enum.

### DeferStmt

`DeferStmt` has two mutually exclusive forms:
- **Call form:** `defer f()` — `Call` field is set, `Block` is nil
- **Block form:** `defer` + indented block — `Block` field is set, `Call` is nil. Codegen emits `defer func() { ... }()`.

### IfStmt

`IfStmt` supports optional init statements: `if x, ok := m[key]; ok`. The `Init` field holds the init statement (typically a `VarDeclStmt` or `AssignStmt`); it is nil for plain `if condition`. The parser uses a lookahead scan for `;` at the current nesting depth to decide whether an init statement is present. Semantic analysis enters a new scope before analyzing the init statement so that variables declared in the init are scoped to the if/else chain.

### OnErrClause

`OnErrClause` is **not** a standalone `Statement` or `Expression`. It is an optional field on `VarDeclStmt`, `AssignStmt`, and `ExpressionStmt`. The `Handler` field holds the parsed error handler expression (`PanicExpr`, `EmptyExpr`, `DiscardExpr`, `ReturnExpr`, or a default value expression). Shorthand forms use boolean flags instead of `Handler`: `ShorthandReturn`, `ShorthandContinue`, `ShorthandBreak`.

### PipedSwitchExpr

`PipedSwitchExpr` represents both regular and typed piped switches:

- Regular: `expr |> switch`
- Typed: `expr |> switch as v`

The AST stores the body as a `PipedSwitchBody`, implemented by both `*SwitchStmt` and `*TypeSwitchStmt`. `parsePipeExpr()` creates a `PipedSwitchExpr` when it sees `TOKEN_SWITCH` after `|>`, and dispatches to `parseSwitchBody()` or `parseTypeSwitchBody()` depending on whether `as binding` is present.

In codegen, value-producing piped switches are wrapped in an IIFE. Regular piped switches generate `switch left { ... }`; typed piped switches generate `switch v := left.(type) { ... }`. Return-type inference for typed piped switches special-cases `return v` so the IIFE can stay strongly typed instead of falling back to `any`.

### Directive

`Directive` represents a `# kuki:name args...` annotation. It has `Name string`, `Args []string`, and `Token lexer.Token`. `FunctionDecl`, `TypeDecl`, and `InterfaceDecl` all have a `Directives []Directive` field. The parser collects `TOKEN_DIRECTIVE` tokens in `skipIgnoredTokens` and attaches them to the next declaration via `drainDirectives()`.

Currently supported directives:
- `# kuki:deprecated "message"` — marks a function/type/interface as deprecated; semantic analysis warns at usage sites
- `# kuki:security "category"` — marks a function as security-sensitive (categories: `sql`, `html`, `fetch`, `files`, `redirect`, `shell`); drives compile-time security checks in `semantic_security.go`

---

## Semantic Analysis (`semantic/`)

**Key files:**

| File | Contents |
|------|---------|
| `semantic.go` | Core `Analyzer` struct, `New`, `Analyze`, `Warnings`, `ReturnCounts`, error/warn helpers |
| `semantic_declarations.go` | Package name validation, skill validation, declaration collection/analysis, enum validation and exhaustiveness checking |
| `semantic_statements.go` | Statement analysis (`analyzeBlock`, `analyzeStatement`, `analyzeIfStmt`, …) |
| `semantic_expressions.go` | Expression analysis (`analyzeExpression`, `analyzeIdentifier`, `analyzeBinaryExpr`, `analyzePipeExprMulti`, …) |
| `semantic_onerr.go` | `onerr` clause analysis, `{error}` not `{err}` enforcement |
| `semantic_types.go` | Type annotation validation and conversion (`validateTypeAnnotation`, `typeAnnotationToTypeInfo`, `typesCompatible`) |
| `semantic_helpers.go` | Pure utilities (`isValidIdentifier`, `extractPackageName`, `isExported`, `isNumericType`) |
| `semantic_calls.go` | `analyzeCallExpr`, `analyzeMethodCallExpr`, `analyzeFieldAccessExpr` (incl. enum dot-access resolution) |
| `semantic_security.go` | Security checks (SQL injection, XSS, SSRF, path traversal, command injection, open redirect) |
| `symbols.go` | Symbol table and type info |
| `stdlib_types.go` | Shared `goStdlibType`/`goStdlibEntry` structs, `GetStdlibEnum`, `GetAllStdlibEnums` (not generated — edit directly) |
| `stdlib_registry_gen.go` | GENERATED — Kukicha stdlib signatures |
| `go_stdlib_gen.go` | GENERATED — Go stdlib signatures |

### Analysis passes

The `Analyze()` method runs three top-level passes in order:

1. **`collectDirectives()`** — scans all declarations for `# kuki:deprecated` and `# kuki:panics` directives, populating `deprecatedFuncs`/`deprecatedTypes`/`panickedFuncs` maps
2. **`collectDeclarations()`** — registers all top-level types, interfaces, and function signatures into the symbol table (so functions can call each other regardless of order); also validates package name (rejects Go stdlib names)
3. **`analyzeDeclarations()`** — validates function bodies, infers `exprReturnCounts`, enforces security checks, warns on deprecated calls

### TypeKindNil

The `empty` keyword has its own type kind (`TypeKindNil`) in `symbols.go`. This distinguishes `empty`-as-nil-literal from `empty`-as-variable-name. When semantic analysis encounters an `EmptyExpr` or an `Identifier` named `"empty"` that isn't shadowed by a user variable, it records `TypeKindNil`. Codegen checks this to decide whether to emit `nil` or preserve the variable name `empty`. The `isReferenceType()` helper determines which types are nil-compatible (references, lists, maps, channels, functions, interfaces), and `typesCompatible()` uses it so `TypeKindNil` is accepted where a reference type is expected.

### Struct literal validation

The semantic analyzer validates struct literal field names and types at compile time. During `collectDeclarations()`, each struct type's field names and types are stored in `TypeInfo.Fields`. When a `StructLiteralExpr` is analyzed, the analyzer resolves the struct's symbol and checks that every field name exists on the struct and that the value type is compatible with the declared field type.

### Method and field resolution

`TypeInfo.Methods` maps method names to their function `TypeInfo`. During `collectDeclarations()`, `registerMethod()` attaches each method's signature to its receiver type's symbol. At analysis time, `FieldAccessExpr` nodes resolve through `resolveFieldType()`, while `MethodCallExpr` nodes resolve through `resolveMethodType()`. Both handle pointer/reference receivers by dereferencing first.

### exprReturnCounts

The analyzer infers how many values an expression returns and stores it in `a.exprReturnCounts[expr]`. Codegen reads this to decide whether to emit `val, err := f()` (2-return) vs `val := f()` (1-return) for pipe + onerr chains.

When a new stdlib function is added to a `.kuki` file, run `make genstdlibregistry` to regenerate `stdlib_registry_gen.go` so the analyzer knows the function's return count.

### knownExternalReturns and registries

`knownExternalReturns` is a unified map of qualified function name → return count, built from two auto-generated sources:

1. **`generatedStdlibRegistry`** (`stdlib_registry_gen.go`) — return counts, per-position return types, and parameter names for Kukicha stdlib functions. Uses the shared `goStdlibEntry` struct. Contains five maps:
   - `generatedStdlibRegistry` — function name → `goStdlibEntry`
   - `generatedStdlibDeprecated` — function name → deprecation message
   - `generatedStdlibPanics` — function name → panic info (from `# kuki:panics` directives)
   - `generatedSecurityFunctions` — function name → security category
   - `generatedSliceGenericClass` — function name → generic class (`T`, `K`, `TK`, `O`, `TO`, `TR`)
   - `generatedStdlibInterfaces` — interface names
   - `generatedStdlibEnums` — enum type name → case names (for cross-package resolution)

2. **`generatedGoStdlib`** (`go_stdlib_gen.go`) — return counts and per-position type info for Go stdlib functions. Contains two maps:
   - `generatedGoStdlib` — function name → `goStdlibEntry`
   - `generatedGoInterfaces` — qualified interface type names (e.g., `io.Reader`)

Both registries use `goStdlibEntry` and `goStdlibType` types from `stdlib_types.go`. The Kukicha registry additionally populates `ParamNames` for named argument support and `DefaultValues` for default parameter filling.

In `analyzeMethodCallExpr`, the Go stdlib registry is checked first, then the Kukicha registry.

To add a new Go stdlib function: add it to the curated list in `cmd/gengostdlib/main.go` and run `make gengostdlib`.

### stdlib_types.go

Defines the shared `goStdlibType` and `goStdlibEntry` structs. Not auto-generated — edit directly when adding fields. Exports accessors for codegen: `GetStdlibEntry(name)`, `GetSliceGenericClass(name)`, `GetSecurityCategory(name)`, `IsKnownInterface(name)`, `GetStdlibEnum(name)`, `GetAllStdlibEnums()`.

`goStdlibType` carries nested type info for compound types: `ElementType *goStdlibType` for lists, `KeyType`/`ValueType *goStdlibType` for maps. This allows the semantic analyzer to propagate element types through pipe chains.

### Security checks

Security checks run during `analyzeDeclarations()`. The analyzer detects "inside an HTTP handler" by checking whether the enclosing `FunctionDecl` has an `http.ResponseWriter` parameter. The `inOnerr bool` field tracks whether the analyzer is currently inside an `onerr` block (used to enforce `{error}` not `{err}`).

### Enum analysis

Enums are registered during `collectDeclarations()` as `SymbolType` with `Kind == TypeKindEnum`. `TypeInfo.EnumCases` maps case names to their `TypeInfo` (with `Kind` of `TypeKindInt` or `TypeKindString`). In `collectEnumDecl()`, all case values must be the same literal type; mixed types are rejected. `analyzeEnumDecl()` warns if an integer enum has no case with value 0.

`checkEnumExhaustiveness()` verifies switch statements cover all cases of an enum type. If missing cases are found and no `otherwise` clause is present, the analyzer reports an error listing uncovered cases. Runs for both regular and piped switches.

Cross-package enums (from stdlib) are resolved via `GetStdlibEnum(qualifiedName)` in `analyzeFieldAccessExpr` — handles `pkg.EnumType.Case` patterns.

### Adding a new security check

1. Add `# kuki:security "category"` to the function in its `.kuki` file
2. Add a `checkXxx` method in `semantic_security.go`
3. Call it from `analyzeMethodCallExpr` in `semantic_calls.go`
4. Run `make genstdlibregistry`

---

## IR (`ir/`)

**Key file:** `ir.go`

The IR package defines Go-level imperative nodes used between AST lowering and code emission. The Lowerer (in `codegen/lower.go`) transforms high-level Kukicha constructs into IR node sequences, and the emitter (in `codegen/emit.go`) walks IR blocks to produce Go source text.

| Node | Purpose |
|------|---------|
| `Block` | Ordered sequence of IR nodes |
| `Assign` | `names := expr` or `names = expr` |
| `VarDecl` | `var name type [= value]` |
| `IfErrCheck` | `if errVar != nil { body }` |
| `Goto` | `goto Label` |
| `Label` | `LabelName:` |
| `ScopedBlock` | Bare `{ ... }` block for variable scoping |
| `RawStmt` | Pre-rendered Go statement (escape hatch) |
| `ReturnStmt` | `return val1, val2, ...` |
| `ExprStmt` | Standalone expression (`continue`, `break`, `panic(...)`) |
| `Comment` | `// text` |

The IR is intentionally thin — it models only the constructs needed by the onerr/pipe lowering passes. Other codegen paths still emit Go text directly.

### Source positions on IR nodes

`SourcePos{Line, File}` is an optional field on code-emitting IR nodes (`Assign`, `VarDecl`, `IfErrCheck`, `RawStmt`, `ReturnStmt`, `ExprStmt`). When populated (Line > 0, File non-empty), the emitter writes a `//line file.kuki:N` directive before the node's Go output. This maps generated pipe-chain and onerr code back to the original `.kuki` source for accurate stack traces, panics, and debugger breakpoints.

The Lowerer populates `Pos` using `posOf(expr)` (pipe step positions) and `clausePos(clause)` (onerr clause positions).

---

## Codegen (`codegen/`)

**Key files:**

| File | Contents |
|------|---------|
| `codegen.go` | Core `Generator` struct, public API, `Generate`, output helpers (`write`, `writeLine`, `uniqueId`) |
| `codegen_decl.go` | Declaration generators (`generateTypeDecl`, `generateEnumDecl`, `generateFunctionDecl`, `generateArrowLambda`, …) |
| `codegen_stmt.go` | Statement generators (`generateBlock`, `generateVarDeclStmt`, `generateReturnStmt`, `generateIfStmt`, …) |
| `codegen_expr.go` | Expression generators (`exprToString`, `generatePipeExpr`, `generateCallExpr`, string interpolation, …) |
| `codegen_onerr.go` | `onerr` code generation; delegates pipe-chain and piped-switch onerr to Lowerer |
| `codegen_types.go` | Type annotation generation |
| `lower.go` | `Lowerer` struct — transforms pipe chains, onerr clauses, and piped switches into IR nodes |
| `emit.go` | `emitIR` — walks IR blocks and emits Go source via `g.writeLine` |
| `codegen_imports.go` | Import generation and auto-import scanning |
| `codegen_stdlib.go` | Stdlib/generics type inference (`inferStdlibTypeParameters`, `zeroValueForType`, `returnTypeForFunctionName`, …) |
| `codegen_walk.go` | Unified AST visitor and `needsXxx` helpers; `collectReservedNames` |

### Generator state

| Field | Purpose |
|-------|---------|
| `output strings.Builder` | Accumulates generated Go source |
| `indent int` | Current indentation level (each level = 1 tab in output) |
| `autoImports map[string]bool` | Packages auto-imported by codegen (e.g., `fmt`, `errors`) |
| `pkgAliases map[string]string` | Collision aliases (e.g., `json` → `kukijson`) |
| `funcDefaults map[string]*FuncDefaults` | Default parameter info for wrapper generation |
| `placeholderMap map[string]string` | Generic placeholder substitution (`"any"→"T"`, `"any2"→"K"`) |
| `sourceFile string` | Source file path for detecting stdlib packages |
| `currentFuncName string` | Current function being generated |
| `currentReturnTypes []ast.TypeAnnotation` | Return types of current function (for `onerr` zero-value generation) |
| `currentOnErrVar string` | Error variable name in active `onerr` block (for `{error}` interpolation) |
| `currentOnErrAlias string` | User-specified alias in `onerr as e` blocks |
| `currentReturnIndex int` | Index of return value being generated (-1 if not in return); resolves placeholder type for bare `empty` |
| `tempCounter int` | Counter for unique temp variable names via `uniqueId()` |
| `exprReturnCounts map[ast.Expression]int` | From semantic — drives `onerr` multi-value split |
| `exprTypes map[ast.Expression]*TypeInfo` | From semantic — used by `isErrorOnlyReturn()` and `empty` resolution |
| `reservedNames map[string]bool` | User-declared identifiers — `uniqueId` skips these |
| `stdlibModuleBase string` | Base module path for rewriting `"stdlib/X"` imports |
| `mcpTarget bool` | True if targeting MCP (Model Context Protocol) — affects main function generation |
| `processingReturnType bool` | True while processing a return type annotation (prevents placeholder expansion loops) |
| `varMap map[string]string` | Maps generated temp variable names (e.g., `pipe_1`) to source descriptions for debugging |
| `enumTypes map[string]bool` | Known enum type names (local + cross-package); drives dot-access rewriting |
| `warnings []error` | Non-fatal diagnostics collected during codegen (retrieved via `Warnings()`) |

### Enum codegen

`generateEnumDecl()` emits: (1) `type X int` or `type X string`, (2) a `const (...)` block with prefixed names (`StatusOK`, `StatusNotFound`), and (3) an auto-generated `String()` method (switch-based for int enums, `return string(e)` for string enums). The `String()` method is skipped if `hasMethodOnType()` finds a user-defined one.

**Dot-access rewriting:** `generateFieldAccessExpr` checks `g.enumTypes[object]` — if the object is an enum name, `Status.OK` becomes `StatusOK`. The `enumTypes` map is populated in two pre-scans in `Generate()`: (1) local enums from the program's declarations, (2) cross-package enums by checking each imported package against `GetAllStdlibEnums()`.

### onerr code generation (Lowerer + IR)

`onerr` is the most complex part of codegen. Split into two phases:

1. **Lowering** (`lower.go`): The `Lowerer` transforms pipe chains and onerr clauses into IR nodes. This makes the logic testable independently of string emission.
2. **Emission** (`emit.go`): `emitIR` walks the IR block and produces Go source text via `g.writeLine`.

Simple onerr cases (single call, no pipe) still use direct emission in `codegen_onerr.go`. Pipe chain and piped switch onerr delegate to the Lowerer. `currentOnErrVar` holds the generated error variable name so that `{error}` in string interpolation inside the block resolves to it.

The Lowerer handles three cases per pipe step:
1. **Multi-return** (count >= 2): split into `val, err := call()`; check err
2. **Error-only** (count == 1 and type is `error`): `err := call()`; check err; keep current pipe variable unchanged
3. **Single value** (count == 1, non-error): `pipe := call()`; advance pipe variable

Error-only detection uses `isErrorOnlyReturn()` which checks both `exprReturnCounts` (count == 1) and `exprTypes` (type is `error`). When neither map has an entry for a single-return step, `isUnknownSingleReturn()` returns true and the Lowerer emits a warning via `g.warn()`. Codegen warnings are retrieved after `Generate()` via `g.Warnings()` and printed by the CLI alongside semantic warnings.

`lowerOnErrWithExplicitErr` handles multi-return cases where the user provides the error variable as the last LHS name (e.g., `a, b, err := f() onerr ...`). If the last name is `_`, it replaces it with a generated unique error variable, since Go's blank identifier cannot be read in `if _ != nil`.

**Piped switch base handling:** `lowerPipedSwitchVarDecl` and `lowerPipedSwitchStmt` both handle two shapes of `ps.Left`: a `PipeExpr` (delegated to `lowerOnErrPipeChainWithLabels`) or a single expression (non-pipe base). For single multi-return bases (e.g., `getValue() |> switch`), the base is split into `val, err :=` and the error is checked with a goto to the onerr label, populating `pipeErrVar` so `{error}` resolves in the handler.

### Pipe operator design notes (non-onerr path)

The non-onerr `generatePipeExpr` wraps multi-return **Left** sides in an IIFE to extract the first value (discarding trailing returns). The **Right** (final) side is NOT wrapped — its full return signature becomes the pipe expression's result, so `val, err := data |> parse()` works naturally. `warnPipeDiscardedErrors` in semantic analysis warns when intermediate steps discard errors without `onerr`. This is intentional: the Go compiler catches type mismatches if the final step's multi-return is used in a single-value context. IIFE return types are inferred via `inferExprReturnType`, which resolves user-defined function return types through `returnTypeForFunctionName` (scanning program declarations) — avoiding the `any` fallback for known functions.

**Shorthand `.Method()` and placeholders:** In `data |> .Method(args)`, the piped value becomes the **receiver** (not an argument), so `_` placeholders in the argument list are not meaningful and are treated as literal underscore identifiers. This differs from `data |> pkg.Func(_, x)` where `_` marks the insertion point for the piped value as a function argument.

### `empty` keyword in codegen

When codegen encounters an `Identifier` with value `"empty"`, it consults `exprTypes` to decide what to emit:

- **`TypeKindNil`** (not shadowed) → emit `nil`. In generic stdlib context with a placeholder return type, `exprToString` returns `*new(T)` or `*new(K)` as an intermediate marker; `replaceGenericZeroExprs` (called from `generateReturnStmt`) converts this to `var _zeroN T; return _zeroN`.
- **Not `TypeKindNil`** (shadowed by a user variable) → emit `empty` as-is, preserving the variable name.

### Arrow lambda parameter type inference

Arrow lambdas do **not** support an implicit `it` parameter. Lambdas must declare their parameters explicitly.

When arrow lambda parameters have no type annotation, the semantic analyzer infers them from the calling function's signature and records the result in `exprTypes[param.Name]`. `generateArrowLambda` checks `exprTypes` before emitting a bare parameter name.

Three inference cases handled in `semantic_calls.go`:

| Case | Trigger | Source |
|------|---------|--------|
| A — user-defined | `funcType.Kind == TypeKindFunction` | `funcType.Params[paramIdx].Params[j]` |
| B — generic stdlib | `ParamFuncParams[paramIdx]` contains `"any"` | element type of the piped/first list argument |
| C — non-generic stdlib | `ParamFuncParams[paramIdx]` with concrete type | `goStdlibEntry.ParamFuncParams` (e.g. `cli.Args`) |

`goStdlibEntry.ParamFuncParams map[int][]goStdlibType` is populated by `genstdlibregistry`. Unqualified named types are prefixed with the package name (`"Args"` → `"cli.Args"`); placeholder names (`any`, `any2`, `ordered`, `error`) are left as-is for runtime substitution.

`goStdlibEntry.ParamFuncReturns map[int][]goStdlibType` stores the return types of func-typed parameters. This enables lambda return scoping — when a block lambda is passed to a function, the semantic analyzer uses these return types to validate `return` statements inside the lambda against the lambda's expected return type, not the enclosing function's.

`resolveExpectedLambdaSignature` (in `semantic_calls.go`) returns a full `*TypeInfo` with both `Params` and `Returns` for the expected lambda signature. The signature is recorded on the lambda node in `exprTypes`, and used by both semantic analysis (for return validation) and codegen (for block lambda return type annotation).

`inferLambdaParamTypes` is called in `analyzeCallExpr`; `inferLambdaParamTypesMethod` in `analyzeMethodCallExpr`. Both record inferred types in `a.exprTypes` so codegen can emit fully typed Go func literals.

**Lambda return scoping:** When analyzing a block lambda that has an expected signature (from `exprTypes`), `analyzeExpression` temporarily swaps `a.currentFunc` to a synthetic `FunctionDecl` with the lambda's return types. This makes `analyzeReturnStmt` validate against the lambda's returns, not the enclosing function's. The original `currentFunc` is restored after the lambda body is analyzed.

**Import alias resolution:** Registry keys use base package names (e.g., `string.Split`), but user code may use aliases (e.g., `strpkg.Split`). The `importAliases map[string]string` field (populated during `collectDeclarations`) maps alias → base name. `resolveQualifiedName()` in `semantic_helpers.go` rewrites aliased qualified names before registry lookups in both `analyzeMethodCallExpr` and `inferLambdaParamTypesMethod`.

**Analysis ordering:** Non-lambda arguments are analyzed first, then lambda param types are inferred, then lambda bodies are analyzed. This ensures lambda parameters have their inferred types in the symbol table when the body is analyzed.

### Generics via placeholders

When generating stdlib code (`isStdlibIter`, or per-function for `stdlib/slice`, `stdlib/sort`, `stdlib/concurrent`), the generator detects `any`/`any2`/`ordered`/`result` placeholders in type annotations and:
1. Builds a `placeholderMap` mapping placeholder → Go type param name (`T`, `K`, `R`)
2. Emits `[T any, K comparable]`, `[T any, K cmp.Ordered]`, or `[T any, R any]` on the function signature
3. Substitutes placeholders throughout parameter and return types
4. `exprToString` returns `*new(T)` as intermediate marker for bare `empty` in generic return position; `replaceGenericZeroExprs` rewrites these to `var _zeroN T; return _zeroN`

The generic classification (`T`, `K`, `TK`, `O`, `TO`, `TR`) is auto-derived from placeholder usage in `.kuki` function signatures and stored in `generatedSliceGenericClass`. Application code never sees this.

### Error expression codegen (`codegen_expr.go`)

`generateErrorExpr(strLit)` for `error "..."` expressions:
- **Plain string** → `errors.New("literal")` (auto-imports `errors`)
- **Interpolated string** (Parts populated) → `fmt.Errorf("format", args...)` (auto-imports `fmt`, no `errors` import)
- **`\sep`-only string** (no Parts) → `errors.New(fmt.Sprintf(...))` fallback

`needsErrorsPackage()` in `codegen_walk.go` skips interpolated `ErrorExpr` nodes (which use `fmt.Errorf`) so the `errors` import is only added when actually needed.

### String literal codegen (`codegen_expr.go`)

`generateStringLiteral` routes to one of three paths:
- **Plain string** (`Interpolated == false`, no `\uE002`): emits `"escaped"` via `escapeString`
- **Sep-only string** (`Parts` empty but `\uE002` present): calls `generateSepOnlyString` — splits on `\uE002` and emits `string(filepath.Separator)` concatenation
- **Interpolated string** (`Parts` populated): calls `generateStringFromParts` — builds `fmt.Sprintf` with `%v` placeholders for expressions and escaped literals

### Child generators for inline code blocks

Use `g.childGenerator(extraIndent)` when generating inline function bodies (function literals, arrow lambda blocks). The child shares the parent's semantic state by reference and writes to a fresh `strings.Builder`.

### Writing to output

Use `g.write(str)` (no indent) or `g.writeLine(str)` (with current indent + newline). Do not write to `g.output` directly.

---

## Formatter (`formatter/`)

**Files:** `formatter.go`, `printer.go`, `comments.go`, `preprocessor.go`

- `Format(source, filename, opts)` — format Kukicha source
- `FormatCheck(source, filename, opts)` — check if already formatted
- Supports Go-style preprocessing (braces/semicolons → indentation)
- Comment preservation: extracts from tokens, attaches to AST nodes, emits during printing
- `printEnumDecl()` formats enum declarations with proper indentation

## LSP (`lsp/`)

**Files:** `server.go`, `document.go`, `completion.go`, `diagnostics.go`, `hover.go`, `definition.go`, `builtins.go`

- JSON-RPC 2.0 server over stdio
- Supported methods: hover, definition, completion, documentSymbol, diagnostics
- `DocumentStore` manages open documents with cached AST/symbol table/errors
- Thread-safe with RWMutex

---

## Adding a Feature: End-to-End Example

**Example: add `repeat N times` loop (`for i repeat 5`)**

1. **Lexer** (`lexer/token.go`): add `TOKEN_REPEAT`, add `"repeat"` to `keywords` map
2. **AST** (`ast/ast.go`): add `ForRepeatStmt { Token, Count Expression, Body *BlockStmt }`
3. **Parser** (`parser/parser_stmt.go`): in `parseStatement()` add `case lexer.TOKEN_REPEAT:` → `parseForRepeatStmt()`
4. **Semantic** (`semantic/semantic_statements.go`): in `analyzeStatement()` add `case *ast.ForRepeatStmt:` → validate `Count` is numeric
5. **Codegen** (`codegen/codegen_stmt.go`): in `generateStatement()` add `case *ast.ForRepeatStmt:` → emit `for _i := 0; _i < N; _i++ { ... }`
6. **Tests**: add test cases in each package's `*_test.go`

---

## Test Patterns

Each package has its own `*_test.go`. The pattern is:
- **Lexer tests**: feed source string → check token types/lexemes
- **Parser tests**: feed source string → check AST structure
- **Codegen tests**: feed source string → check generated Go string (often with `strings.Contains`)
- **Semantic tests**: feed source string → check error messages

Some tests check exact temp variable names (`pipe_1`, `err_2`) — the lowerer must share the generator's counter.

Test helpers: `mustParse` in codegen_test package, `test_helpers_test.go` in parser/semantic.

```bash
make test    # Run all tests
make lint    # Run linter
```

## Generated Files

**Do not edit directly.** Regenerated automatically by `make build` via `go generate ./...`.

- `make genstdlibregistry` → `semantic/stdlib_registry_gen.go`
- `make gengostdlib` → `semantic/go_stdlib_gen.go`
- `make generate` → both
