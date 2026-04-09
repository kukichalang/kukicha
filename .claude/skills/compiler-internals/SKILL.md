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

Semantic analysis produces an `*AnalysisResult` (via `analyzer.AnalyzeResult()`) containing errors, warnings, and two maps passed to codegen via `generator.SetAnalysisResult(result)`:
- `ExprReturnCounts map[ast.Expression]int` — tells codegen how many values an expression returns so it can emit the right `val, err := f()` split for `onerr`.
- `ExprTypes map[ast.Expression]*TypeInfo` — records inferred type of every analyzed expression. Used by codegen for: error-only pipe step detection (`isErrorOnlyReturn`), piped switch return type inference, `empty` keyword resolution, typed zero-value generation (`zeroValueForType`). In `analyzePipeExprMulti`, types are explicitly recorded on pipe step nodes via `recordType(right, types[0])` since steps bypass `analyzeExpression`. Pipe placeholder `_` identifiers get the piped value's type recorded when inside a call with a known function signature.
- `Warnings []error` — non-fatal diagnostics. Access via `result.Warnings`; `Analyzer` no longer exposes a `Warnings()` getter.

Tests that only need errors and warnings use `analyzeSourceResult(t, src)` which returns `*AnalysisResult`. Tests that need to inspect unexported `*Analyzer` fields (e.g. `exprTypes`, `symbolTable`) call `analyzeSource(t, src)` which returns `(*Analyzer, []error)`.

The formatter (`formatter/`) is a separate AST-based pipeline that re-parses and pretty-prints. It has exhaustiveness tests that parse `ast.go` to ensure every `Expression`, `Statement`, and `Declaration` type has a corresponding formatter case — adding a new AST node without updating the formatter will fail tests. Default cases emit `/* unhandled: %T */` instead of silent empty strings. The LSP (`lsp/`) wraps the compiler pipeline (lexer → parser → semantic) for diagnostics, hover, completion, definition, formatting, and signature help.

## Package Overview

| Package | Role | Key entry point |
|---------|------|-----------------|
| `lexer/` | Tokenization (INDENT/DEDENT, string interpolation) | `NewLexer(source, file).ScanTokens()` |
| `parser/` | Recursive descent parser → AST | `New(source, file)` then `Parse()` |
| `ast/` | AST node definitions (no logic) | Node interfaces: `Declaration`, `Statement`, `Expression`, `TypeAnnotation` |
| `semantic/` | Type checking, symbol resolution, security checks | `New(program).Analyze()` |
| `ir/` | Intermediate representation (Go-level imperative nodes) | `Block`, `Assign`, `IfErrCheck`, `Goto`, `Label` |
| `codegen/` | AST → IR lowering → Go source emission | `New()` then `Generate()` |
| `formatter/` | AST-based source formatting (exhaustiveness-tested against AST node types) | `Format(source, file, opts)` |
| `lsp/` | Language Server (hover, completion, definition, symbols, formatting, signature help) | `NewServer(reader, writer).Run(ctx)` |
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

### Brace block support (Go syntax passthrough)

Kukicha accepts Go-style `{ }` brace blocks as an alternative to indentation. The lexer converts block braces into `TOKEN_INDENT` / `TOKEN_DEDENT`, making them transparent to the parser (zero parser changes needed).

**Three new lexer fields:**
- `blockKeywordSeen bool` — set to `true` after a block keyword (`if`, `for`, `func`, `switch`, `else`, `select`, `go`, `defer`); cleared on `TOKEN_NEWLINE`, `TOKEN_INDENT`, or `TOKEN_OF`
- `braceStack []bool` — tracks whether each `{` was a block brace (`true`) or literal brace (`false`), so the matching `}` emits the correct token
- `braceBlockDepth int` — count of currently open block braces; suppresses indentation handling inside brace blocks

**`{` handling:** If `blockKeywordSeen` is true, the `{` is a block brace → push `true` to `braceStack`, increment `braceBlockDepth`, emit `TOKEN_INDENT`. Otherwise it's a literal brace → push `false`, increment `braceDepth`, emit `TOKEN_LBRACE`.

**`}` handling:** Pop from `braceStack`. If the entry was `true` (block) → decrement `braceBlockDepth`, emit `TOKEN_DEDENT`. If `false` (literal) → decrement `braceDepth`, emit `TOKEN_RBRACE`.

**Indentation suppression:** When `braceBlockDepth > 0`, leading whitespace on new lines is consumed without emitting INDENT/DEDENT tokens.

**`TOKEN_OF` clearing:** `blockKeywordSeen` is cleared on `TOKEN_OF` because `for item in list of T{...}` is always a type+literal pattern, not a block brace.

**Known limitation (same as Go):** Composite literals inside `if`/`for`/`switch` conditions with brace blocks require parentheses to disambiguate: `if x == (MyStruct{}) { ... }`.

### Line continuation

`TOKEN_NEWLINE` is suppressed (continuation mode) in two ways:

**Inline (during tokenization):** Inside `[]` or literal `{}` (`braceDepth > 0`), `TOKEN_NEWLINE` is suppressed and `continuationLine` is set so the next line's indentation is consumed without emitting INDENT/DEDENT. Note: `braceDepth` only tracks `[]` and literal braces (struct/map literals), NOT block braces (which use `braceBlockDepth` instead). `()` (parentheses) do NOT suppress newlines when inside a function literal body — closures need `INDENT/DEDENT` for their block structure.

**Post-pass (`mergeLineContinuations`):** Pipe continuation (`|>`) and `onerr` on continuation lines are handled after tokenization. The lexer emits NEWLINE/INDENT/DEDENT normally; the post-pass removes them around pipe chains. This decouples pipe handling from the indent stack. Three patterns are merged:
1. Trailing pipe: `PIPE [COMMENT*] NEWLINE [INDENT*]` → remove NEWLINE and INDENTs
2. Leading pipe: `NEWLINE [INDENT*] PIPE` → remove NEWLINE and INDENTs (no DEDENTs allowed)
3. Leading onerr: `NEWLINE [INDENT*] ONERR` → same as (2), only in pipe chain context

For each INDENT absorbed, a corresponding DEDENT is also absorbed later in the stream.

### Number literal prefixes and underscore separators

`scanNumber()` supports decimal, hexadecimal (`0x`/`0X`), octal (`0o`/`0O`), and binary (`0b`/`0B`) integer literals. Legacy octal (`0755`) also works. Helper functions `isHexDigit()` and `isOctalDigit()` validate digits after the prefix. Invalid prefixes (e.g., `0o` with no digits) produce an error.

All digit-scanning loops (decimal, hex, octal, binary, float fractional) accept `_` as a visual separator (e.g., `1_000_000`, `0xFF_FF`, `3.141_592`). The underscore is included in the token lexeme and preserved through codegen and the formatter.

### NUL rejection in strings

`scanStringBody` rejects NUL (`\x00`) characters inside string literals with a clear error. This prevents invalid characters from propagating through to generated Go source.

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

**Byte-value escapes:** These produce raw bytes in the token value. Codegen's `escapeString` emits non-printable characters (< 0x20 or 0x7F) as `\xHH`.

| Escape     | Value     | Example          |
|------------|-----------|------------------|
| `\xHH`     | hex byte  | `\x1b` → ESC    |
| `\0`-`\377`| octal byte| `\033` → ESC     |

Octal escapes read 1–3 octal digits (max `\377` = 0xFF). Values > 255 are a compile error. The lexer uses `WriteByte` (not `WriteRune`) so high values like `\377` produce a single raw byte, not a multi-byte UTF-8 sequence.

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
- **Keywords as identifiers:** `parseIdentifier()` uses `isIdentifierToken()` to accept many keyword tokens in identifier position (method names, field names, function declaration names). This allows patterns like `obj.close()`, `db.select()`, `registry.list()`, `node.type`, `event.on()`. Only tokens with structural/control-flow meaning (e.g., `if`, `for`, `return`, `func`) are excluded. Previously only `close`, `empty`, and `error` were accepted; now ~25 keyword types are allowed.
- **Type declarations inside functions:** `parseStatement()` parses `TOKEN_TYPE` into a `TypeDeclStmt` (wrapping a `TypeDecl`) rather than rejecting it at parse time. The error is deferred to semantic analysis, which keeps all validation in a single pass and produces a clearer error message.

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
| `isIdentifierToken(t)` | Returns true if the token type can appear where an identifier is expected (includes ~25 keyword types) |

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
├── Declaration      (declNode marker, //sumtype:decl)  — FunctionDecl, TypeDecl, EnumDecl, ImportDecl, …
├── Statement        (stmtNode marker, //sumtype:decl)  — IfStmt, ForRangeStmt, ReturnStmt, …
├── Expression       (exprNode marker, //sumtype:decl)  — CallExpr, PipeExpr, ArrowLambda, …
├── TypeAnnotation   (typeNode marker, //sumtype:decl)  — ListType, MapType, ReferenceType, …
└── PipedSwitchBody  (pipedSwitchBodyNode, //sumtype:decl) — SwitchStmt, TypeSwitchStmt
```

The `//sumtype:decl` annotations enable `gochecksumtype` (via `make lint`) to enforce exhaustive type switches. Type switches without a `default:` branch must handle every type implementing the interface — adding a new AST node produces a linter error until all switches are updated.

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

**Notable:** `VarDeclStmt` implements both `Statement` and `Declaration`. `TypeDeclStmt` wraps a `TypeDecl` that appeared inside a function body — the parser accepts it syntactically so semantic analysis can reject it with a clear error.

### EnumDecl and EnumCase

`EnumDecl` represents an `enum` declaration with named integer or string constants, or a variant enum (tagged union). `EnumCase` holds a case name and either a literal `Value` (value enum) or `Fields` (variant enum). `IsVariant()` returns true when the first case has no value. Directives (`# kuki:deprecated`) can annotate the enum.

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
| `semantic.go` | Core `Analyzer` struct, `New`, `Analyze`, `AnalyzeResult`, `AnalysisResult` type, error/warn helpers |
| `semantic_directives.go` | `DirectiveResult` struct, `CollectDirectives()` — extracts deprecated/panics/todo from AST directives |
| `semantic_declarations.go` | Package name validation, skill validation, declaration collection/analysis, enum validation and exhaustiveness checking |
| `semantic_statements.go` | Statement analysis (`analyzeBlock`, `analyzeStatement`, `analyzeIfStmt`, …) |
| `semantic_expressions.go` | Expression analysis (`analyzeExpression`, `analyzeIdentifier`, `analyzeBinaryExpr`, `analyzePipeExprMulti`, …) |
| `semantic_onerr.go` | `onerr` clause analysis, `{error}` not `{err}` enforcement |
| `semantic_types.go` | Type annotation validation and conversion (`validateTypeAnnotation`, `typeAnnotationToTypeInfo`, `typesCompatible`) |
| `semantic_helpers.go` | Pure utilities (`isValidIdentifier`, `extractPackageName`, `isExported`, `isNumericType`) |
| `semantic_calls.go` | `analyzeCallExpr`, `analyzeMethodCallExpr`, `analyzeFieldAccessExpr` (incl. enum dot-access resolution) |
| `semantic_security.go` | Security checks (SQL injection, XSS, SSRF, path traversal, command injection, open redirect) |
| `semantic_lint.go` | Lint candidate collection (`LintCandidate`, `LintKind`) and deferred emission (`emitLintWarnings`) |
| `symbols.go` | Symbol table and type info |
| `stdlib_types.go` | Shared `goStdlibType`/`goStdlibEntry` structs, `GetStdlibEnum`, `GetAllStdlibEnums` (not generated — edit directly) |
| `stdlib_registry_gen.go` | GENERATED — Kukicha stdlib signatures |
| `go_stdlib_gen.go` | GENERATED — Go stdlib signatures |

### Analyzer struct layout

The `Analyzer` has 17 fields grouped by lifecycle phase:

| Group | Fields |
|-------|--------|
| Immutable inputs | `program`, `sourceFile` |
| Infrastructure | `symbolTable`, `security`, `errors`, `warnings` |
| Pre-pass output | `directives *DirectiveResult` (set once by `CollectDirectives`) |
| Pass 1 output | `importAliases` |
| Pass 2 transient | `currentFunc`, `loopDepth`, `switchDepth`, `inOnerr`, `currentOnerrAlias`, `inPipedSwitch` |
| Pass 2 output | `exprReturnCounts`, `exprTypes` |
| Lint candidates | `lintCandidates []LintCandidate` (collected during analysis, emitted in final pass) |

### Analysis passes

The `Analyze()` method runs four top-level passes in order:

1. **`CollectDirectives()`** — pure function; scans all declarations for `# kuki:deprecated`, `# kuki:panics`, and `# kuki:todo` directives, returning a `*DirectiveResult` stored as `a.directives`
2. **`collectDeclarations()`** — registers all top-level types, interfaces, and function signatures into the symbol table (so functions can call each other regardless of order); also validates package name (rejects Go stdlib names) and import paths (rejects `"`, `\`, and NUL characters)
3. **`analyzeDeclarations()`** — validates function bodies, infers `exprReturnCounts`, enforces security checks; collects lint candidates via `recordLint()` instead of emitting warnings directly
4. **`emitLintWarnings()`** — final pass that converts collected `LintCandidate` structs into warnings. Decouples lint detection from emission, enabling future filtering/configuration.

`AnalyzeResult()` wraps `Analyze()` and returns `*AnalysisResult` bundling errors, warnings, and both maps.

### Lint candidate system (`semantic_lint.go`)

All non-fatal diagnostics use the collect-then-emit pattern. During type checking, `recordLint(kind, pos, message)` appends a `LintCandidate` to `a.lintCandidates`. After all analysis completes, `emitLintWarnings()` converts them to warnings via `a.warn()`.

`LintKind` categories: `LintDeprecation`, `LintPanic`, `LintOnerr`, `LintPipe`, `LintEnum`, `LintTypeMismatch`, `LintSecurity`, `LintTodo`. These enable future per-category suppression (e.g., `--suppress-lint=deprecation`).

### Interface detection in typeAnnotationToTypeInfo

When `typeAnnotationToTypeInfo` processes a `NamedType`, it checks if the name refers to an interface — either a user-defined interface in the symbol table (`SymbolInterface`) or a known Go/Kukicha stdlib interface via `IsKnownInterface()`. If so, it returns `TypeKindInterface` instead of `TypeKindNamed`. This allows `typesCompatible()` to correctly accept concrete types (e.g., `*MyStruct`) where an interface return type is declared (e.g., `io.Reader`).

### Qualified named type compatibility

`typesCompatible()` defers to the Go compiler for qualified named types from external packages (e.g., `mypkg.Handler`). When either type contains a `.` in its name, the check returns `true` — Kukicha cannot resolve external type definitions at compile time, so the Go compiler handles the final check. This allows patterns like returning a concrete struct where an external interface is expected.

### ReturnExpr outside onerr

`ReturnExpr` (the expression form used in `onerr return ...`) is validated to only appear inside an `onerr` handler. The `inOnerr bool` field on the analyzer is checked in `analyzeExpression` for `*ast.ReturnExpr` — if false, an error is reported.

### Type declarations inside functions

`analyzeStatement()` handles `*ast.TypeDeclStmt` by emitting the error "type declarations must be at the top level, not inside a function". The parser accepts these syntactically (wrapping in `TypeDeclStmt`) so that the error comes from semantic analysis rather than the parser, providing a clearer message.

### List literal type precedence

In `analyzeListLiteral`, an explicitly declared element type (e.g., `list of Shape{...}`) takes priority over element-based inference. This allows heterogeneous interface lists where elements have different concrete types. When `expr.Type != nil`, it is used directly; otherwise the first element's type is inferred and subsequent elements are checked for compatibility.

### Switch scope per case

`analyzeSwitchStmt()` creates a new scope (`EnterScope`/`ExitScope`) around each `when` case body and the `otherwise` body. This prevents variable declarations in one branch from conflicting with declarations in other branches. This matches the behavior of `analyzeTypeSwitchStmt()`, which has always scoped per case.

### TypeKindNil

The `empty` keyword has its own type kind (`TypeKindNil`) in `symbols.go`. This distinguishes `empty`-as-nil-literal from `empty`-as-variable-name. When semantic analysis encounters an `EmptyExpr` or an `Identifier` named `"empty"` that isn't shadowed by a user variable, it records `TypeKindNil`. Codegen checks this to decide whether to emit `nil` or preserve the variable name `empty`. The `isReferenceType()` helper determines which types are nil-compatible (references, lists, maps, channels, functions, interfaces), and `typesCompatible()` uses it so `TypeKindNil` is accepted where a reference type is expected.

### Struct literal validation

The semantic analyzer validates struct literal field names and types at compile time. During `collectDeclarations()`, each struct type's field names and types are stored in `TypeInfo.Fields`. When a `StructLiteralExpr` is analyzed, the analyzer resolves the struct's symbol and checks that every field name exists on the struct and that the value type is compatible with the declared field type.

### Method and field resolution

`TypeInfo.Methods` maps method names to their function `TypeInfo`. During `collectDeclarations()`, `registerMethod()` attaches each method's signature to its receiver type's symbol. At analysis time, `FieldAccessExpr` nodes resolve through `resolveFieldType()`, while `MethodCallExpr` nodes resolve through `resolveMethodType()`. Both handle pointer/reference receivers by dereferencing first.

**Shorthand pipe syntax (`.Field` / `.Method()`):** The parser accepts dot-prefixed expressions without a left-hand side, producing AST nodes with `Object == nil`. These are only valid inside pipe expressions (e.g., `user |> .Name`), where `pipedArg` provides the receiver. Both `analyzeFieldAccessExpr` and `analyzeMethodCallExpr` reject shorthand syntax when `Object == nil && pipedArg == nil`, reporting a compile error.

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

In `analyzeMethodCallExpr`, the Go stdlib registry is checked first, then the Kukicha registry. When neither registry has an entry, no default return count is recorded — this lets codegen's `emitOnErrDiscard` use its bare-call fallback instead of assuming 1 return value.

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

### Variant enum analysis

Variant enums are detected by `EnumDecl.IsVariant()` (first case has no `Value`). `collectVariantEnumDecl()` registers the enum as `TypeKindVariant` with a `VariantCases` map (case name → struct `TypeInfo` with fields). Each case is also registered as a standalone struct type in the symbol table. Mixing value and variant cases is rejected. `checkVariantExhaustiveness()` in `semantic_statements.go` warns when a typed switch on a variant enum misses cases (unless `otherwise` is present).

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

### Variant enum codegen

`generateVariantEnumDecl()` (in `codegen_decl.go`) emits: (1) a sealed marker interface (`type Shape interface{ isShape() }`), (2) a struct per case (`type Circle struct { radius float64 }`), and (3) marker methods (`func (Circle) isShape() {}`). Unit variants get empty structs. The `variantCaseTypes` map tracks case name → parent enum name.

### onerr code generation (Lowerer + IR)

`onerr` is the most complex part of codegen. Split into two phases:

1. **Lowering** (`lower.go`): The `Lowerer` transforms pipe chains and onerr clauses into IR nodes. This makes the logic testable independently of string emission.
2. **Emission** (`emit.go`): `emitIR` walks the IR block and produces Go source text via `g.writeLine`.

Simple onerr cases (single call, no pipe) still use direct emission in `codegen_onerr.go`. Pipe chain and piped switch onerr delegate to the Lowerer. `currentOnErrVar` holds the generated error variable name so that `{error}` in string interpolation inside the block resolves to it.

`emitOnErrDiscard` handles `onerr discard` for all three forms (statement, var decl, assignment). When `inferReturnCount` succeeds and returns count >= 1, it emits the correct number of `_` blanks. When inference fails or returns 0, the fallback emits a bare function call without any assignment — Go allows discarding all return values from any call, so this is always valid.

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

### Integer/float literal codegen

`exprToString` for `IntegerLiteral` preserves the original lexeme when it has a prefix (`0x`, `0o`, `0b`, legacy `0...`) or contains underscore separators (`1_000`). Otherwise it formats via `%d`. `FloatLiteral` similarly preserves the lexeme when it contains underscores.

### For-range iterator detection

`generateForRangeStmt` uses single-variable form (`for v := range ...`) when the collection is a range-over-func iterator (`iter.Seq`/`iter.Seq2`). Detection uses `collectionIsIterator(expr)`, which checks `exprTypes` for `TypeKindFunction`. This complements the existing `isStdlibIter` flag (used when generating stdlib/iter itself).

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
- `TypeDeclStmt` is preserved in formatting output (even though semantic analysis rejects it) so the user sees the error
- Integer and float literals with underscore separators or prefixes preserve their original lexeme

### ⚠️ Dual printer architecture — keep in sync

`printer.go` defines `Printer` (base AST→text printer). `formatter.go` defines `PrinterWithComments` which **embeds `Printer` but duplicates key methods** to interleave comments. Any fix to `printer.go` must also be applied to the corresponding `WithComments` variant in `formatter.go`:

| `printer.go` method | `formatter.go` duplicate |
|---------------------|--------------------------|
| `Print()` (top-level dispatch) | `PrinterWithComments.Print()` |
| `printDeclaration()` | `printDeclarationWithComments()` |
| `printFunctionDecl()` | `printFunctionDeclWithComments()` |
| `printStatement()` | `printStatementWithComments()` |

Some base `Printer` methods are called directly by both paths (e.g., `printVarDeclStmt`, `printAssignStmt`, `printReturnStmt`, `exprToString`), so fixes there propagate automatically. But inline logic in the duplicated methods must be updated in both places.

### Parenthesis preservation

The parser discards grouping parentheses (`parseGroupedExpression` returns the inner expression). The formatter must re-add parens to preserve semantics:

- **`not`/`!` with lower-precedence operands:** `needsParensAfterNot` checks if the operand is a `BinaryExpr` or `PipeExpr` and wraps in `()` (e.g., `not (value == "")`)
- **Type casts as field access receivers:** `fieldAccessExprToString` wraps `TypeCastExpr` objects in `()` (e.g., `(val as Item).Id`)

### Multi-line closing paren placement

When a call argument spans multiple lines (function literals, arrow lambdas with blocks), the closing `)` must be on its own dedented line. For nested calls, if the last line of the joined args is already just closing parens (e.g., `)`), the outer `)` is appended directly to produce `))` on one line. This logic lives in `callExprToString` and `methodCallExprToString`.

### After modifying the formatter

1. `make build` — rebuild the compiler
2. `kukicha fmt -w stdlib/ examples/` — reformat all source files
3. `kukicha fmt --check stdlib/ examples/` — verify idempotency
4. `kukicha check` on all formatted files — verify they still parse
5. `make generate` — regenerate `.go` files from formatted `.kuki` sources
6. `make test` — run the full test suite
7. `make lint && make vet` — check for issues

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
6. **Lint** (`make lint`): `gochecksumtype` will flag any type switch without `default:` that's missing the new `ForRepeatStmt` case — fix all reported switches
7. **Tests**: add test cases in each package's `*_test.go`

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
