# Enum Research: Design Proposal for Kukicha

## What Are Enums?

An **enumeration** (enum) is a type that restricts a variable to a fixed set of named values. Instead of using a raw `int` or `string` and hoping the programmer remembers which values are valid, an enum makes the compiler enforce it.

For example, instead of:

```go
status := 200  // What does 200 mean? Is 999 valid?
```

You write:

```kukicha
enum Status
    OK = 200
    NotFound = 404
    Error = 500

status := Status.OK  # Clear, discoverable, type-safe
```

Enums are valuable because they:
- **Prevent invalid values** — the compiler rejects `Status(999)` in well-typed code
- **Make code self-documenting** — `Status.NotFound` is clearer than `404`
- **Enable exhaustiveness checking** — the compiler can warn if a `switch` misses a case
- **Improve IDE support** — autocomplete shows all valid values

## How Go Handles Enums Today

Go has no `enum` keyword. The idiomatic pattern uses `const` blocks with `iota`:

```go
type Status int

const (
    StatusOK       Status = 200
    StatusNotFound Status = 404
    StatusError    Status = 500
)
```

Or with `iota` for auto-incrementing:

```go
type Color int

const (
    Red   Color = iota  // 0
    Green               // 1
    Blue                // 2
)
```

### Limitations of Go's Pattern

1. **No exhaustiveness checking** — `switch` on a `Status` compiles fine with missing cases
2. **No namespace** — constants are package-level (`StatusOK`, not `Status.OK`)
3. **No enforcement** — `Status(999)` is perfectly legal Go
4. **Verbose naming** — the type name must be prefixed manually (`StatusOK`, `StatusNotFound`)
5. **`iota` is implicit** — beginners find it confusing that `Green` has no explicit `= 1`

## What Developers Want

The **2025 Go Developer Survey** found that 65% of Go developers' favorite languages have type-safe enums, and enums are the **#2 most-wanted feature** (after better error handling, which Kukicha already addresses with `onerr`).

Go's core team has acknowledged the demand but has not committed to any timeline or syntax. Their conservative approach ("stability over novelty") means Go may not get enums for years, if ever.

## Proposed Kukicha Syntax

### Basic Value Enums

```kukicha
enum Status
    OK = 200
    NotFound = 404
    Error = 500
```

Rules:
- Each case must have an explicit value (no hidden `iota` magic — matches Kukicha's "no magic" philosophy)
- The underlying type is inferred from the values (`int` for integers, `string` for strings)
- Cases are accessed with dot syntax: `Status.OK`

### String Enums

```kukicha
enum LogLevel
    Debug = "debug"
    Info = "info"
    Warn = "warn"
    Error = "error"
```

### Usage with Switch/When

```kukicha
func handleStatus(s Status) string
    return s |> switch
        when Status.OK
            return "Success"
        when Status.NotFound
            return "Not found"
        when Status.Error
            return "Server error"
```

### Usage in Types

```kukicha
type Response
    status Status
    body string

resp := Response{status: Status.OK, body: "hello"}
```

## Generated Go Code

### Input (Kukicha)

```kukicha
enum Status
    OK = 200
    NotFound = 404
    Error = 500
```

### Output (Go)

```go
type Status int

const (
    StatusOK       Status = 200
    StatusNotFound Status = 404
    StatusError    Status = 500
)
```

The generated Go follows the standard idiomatic pattern. The type name is prepended to each case name (`OK` becomes `StatusOK`). This is invisible to Kukicha users, who always write `Status.OK`.

### Dot Access Translation

In codegen, `Status.OK` is translated to `StatusOK`. This is a straightforward rewrite in `exprToString` — when the left side of a field access is a known enum type, concatenate the type name with the field name.

## Integration with Existing Features

### Pattern Matching (switch/when)

Kukicha's existing `switch`/`when` syntax works naturally with enums:

```kukicha
status |> switch
    when Status.OK
        print("ok")
    when Status.NotFound
        print("not found")
    otherwise
        print("other")
```

### Piped Switch

```kukicha
result := status |> switch
    when Status.OK
        return "ok"
    when Status.Error
        return "error"
    otherwise
        return "unknown"
```

### Comparison

```kukicha
if status equals Status.OK
    print("success")
```

### Function Parameters

```kukicha
func process(level LogLevel)
    if level equals LogLevel.Debug
        print("debug mode")
```

## AST Nodes

```go
// EnumDecl represents an enum declaration.
//
//  enum Status
//      OK = 200
//      NotFound = 404
type EnumDecl struct {
    Token      lexer.Token   // The 'enum' token
    Name       *Identifier
    Cases      []*EnumCase
    Directives []Directive
}

// EnumCase is a single named value in an enum.
type EnumCase struct {
    Name  *Identifier
    Value Expression  // Must be a literal (integer or string)
}
```

`EnumDecl` implements `Declaration` (like `TypeDecl` and `ConstDecl`).

## Risk Assessment

### Low Risk: Conflict with Go

Even if Go eventually adds enums, they will almost certainly generate `const`+`iota` patterns under the hood. Kukicha's generated Go code would just need codegen updates — the Kukicha syntax itself wouldn't need to change. Go has shown a strong preference for backward compatibility, so existing `const` patterns will remain valid.

### Medium Risk: Zero Value Safety

Integer enums always have `0` as the zero value of uninitialized variables. If no enum case maps to `0`, struct fields and function parameters of that enum type start in an invalid state with no compile-time detection. This is mitigated by warning at declaration time (see Open Design Question 4), but it cannot be fully eliminated without changing how Go initializes values.

### Medium Risk: Premature Design

If Go adds **data-carrying variants** (Rust-style `enum Result { Ok(T), Err(E) }`), Kukicha's simple value enums would need extension. However:

- Simple value enums are useful regardless — they cover the most common use case (status codes, log levels, directions, colors)
- Extension is additive — we can add data-carrying variants later without breaking existing enums
- Go's conservative approach makes complex enums unlikely in the near term

### Low Risk: Naming Collision

`enum` is not a Go keyword, so using it in generated code is not an issue. The generated output uses standard `type` + `const` patterns.

## Design Decisions

### 1. Auto-Increment vs Explicit Values

**Decision:** All values must be explicit.

```kukicha
enum Color
    Red = 0
    Green = 1
    Blue = 2
```

Kukicha's philosophy is "no magic" — auto-increment hides the actual values and makes insertion-order bugs possible. A beginner reading this code can see exactly what each name maps to. Auto-increment could be added later as a convenience feature if users request it.

### 2. Exhaustiveness Checking

**Decision:** Add exhaustiveness checking when there is no `otherwise` clause.

```kukicha
# This warns: "switch on Status is missing case: Error"
status |> switch
    when Status.OK
        print("ok")
    when Status.NotFound
        print("not found")
```

```kukicha
# This is fine — otherwise handles the rest
status |> switch
    when Status.OK
        print("ok")
    otherwise
        print("something else")
```

This matches Rust's approach and provides real safety without being annoying. If `otherwise` is present, the programmer has explicitly handled the remaining cases.

**Implementation:** In semantic analysis, when analyzing a `switch` where the expression type is a known enum:
- Collect all `when` case values
- Compare against all enum cases
- If missing cases and no `otherwise`, emit a compiler warning
- This runs in the semantic pass (not codegen) so the warning appears alongside other type errors

### 3. Underlying Type Specification

**Decision:** Infer from values. All integers → `int`, all strings → `string`.

```kukicha
# Underlying type is int (inferred from integer values)
enum Status
    OK = 200
    NotFound = 404

# Underlying type is string (inferred from string values)
enum LogLevel
    Debug = "debug"
    Info = "info"
```

Explicit types (`enum Port int`) can be added later if needed for `int32` or `uint8`.

### 4. Zero Value Safety

**Decision:** Warn when no integer enum case maps to `0`.

**The problem:** Since integer enums generate `type Status int`, any uninitialized variable or struct field has value `Status(0)`. If no case maps to `0`, this is a silently invalid state:

```kukicha
type Response
    status Status   # zero value is Status(0) — matches no case
    body string

resp := Response{body: "hello"}
# resp.status is Status(0), which matches no case
```

**The warning:**

```
Warning in api.kuki:2:5

    2 |    enum Status
      |    ^^^^ enum Status has no case with value 0 — uninitialized variables will hold an invalid state

    Help: Consider adding an explicit Unknown = 0 case:
          enum Status
              Unknown = 0
              OK = 200
```

This is a warning, not an error — legitimate enums like HTTP status codes intentionally have no `0` case. String enums have `""` as their zero value, which is also not a valid case, but at least obviously invalid in logs and JSON output.

### 5. String Representation

**Decision:** Defer `String()` generation to a follow-up. Ship enums without it.

Generating `String()` adds complexity: if the user also writes `func String on s Status string`, the generated Go has a duplicate method — a compile error. This requires a post-collection pass to detect user-defined `String` methods before deciding whether to generate one.

Users can write their own `String()` method in the meantime, and Go's `fmt.Stringer` works fine without it. This is additive — we can always add auto-generation later.

### 6. Enum Methods

**Decision:** Yes, users can define methods on enum types.

```kukicha
func IsClientError on s Status bool
    return s >= 400 and s < 500
```

Since enums generate a named type (`type Status int`), Go methods on that type work naturally. No special handling needed beyond what Kukicha already does for methods on named types.

### 7. Cross-Package Enum Usage

**Decision:** Defer to a follow-up release. Single-file and same-package enums ship first.

When cross-package support is added, the approach is:

- Add `EnumCases map[string]*TypeInfo` to the `TypeInfo` struct (alongside existing `Fields` and `Methods`)
- Populate it during `collectDeclarations()` so cross-file resolution works the same as struct field resolution
- Extend `genstdlibregistry` to emit enum case metadata when scanning `.kuki` files that contain enum declarations
- Consumer packages then resolve `Status.OK` through the registry the same way they resolve `slice.Filter` today

This is a natural extension of the existing registry pipeline but is not needed for the initial release where enums are used within a single package.

### 8. `EnumType.Case` vs Package Access Disambiguation

**Decision:** Use the existing symbol table. No ambiguity exists.

After reading the semantic analyzer, this concern is resolved by how the system already works:

1. **Package names** are registered as `SymbolVariable` in the symbol table during `collectDeclarations()` (line 96-104 of `semantic_declarations.go`)
2. **Enum type names** will be registered as `SymbolType` with a new `TypeKindEnum`
3. **`analyzeFieldAccessExpr`** already calls `analyzeExpression` on the object first, which resolves the identifier through the symbol table

The resolution order for `X.Y`:

| `X` resolves to | Meaning | Codegen |
|-----------------|---------|---------|
| `SymbolVariable` (import) | Package access | `X.Y` (unchanged) |
| `SymbolType` with `TypeKindEnum` | Enum case access | `XY` (concatenated) |
| `SymbolVariable`/`SymbolParameter` with enum type | Instance field access | Error: "enum cases are accessed on the type, not on values — use `Status.OK`, not `myStatus.OK`" |

**Concrete implementation in `analyzeFieldAccessExpr`:**

```go
// Before the existing objType resolution:
if ident, ok := expr.Object.(*ast.Identifier); ok {
    sym := a.symbolTable.Resolve(ident.Value)
    if sym != nil && sym.Kind == SymbolType && sym.Type != nil && sym.Type.Kind == TypeKindEnum {
        // X is an enum type name — validate Y is a known case
        if caseType, ok := sym.Type.EnumCases[expr.Field.Value]; ok {
            a.recordReturnCount(expr, 1)
            return caseType
        }
        a.error(expr.Field.Pos(), fmt.Sprintf(
            "'%s' is not a case of enum %s", expr.Field.Value, ident.Value))
        return &TypeInfo{Kind: TypeKindUnknown}
    }
}
```

This slots in at the top of the existing function, before the general field resolution logic. No new disambiguation mechanism needed.

**Codegen side** (`exprToString` for `FieldAccessExpr`):

```go
// When object is a known enum type, concatenate: Status.OK → StatusOK
if ident, ok := expr.Object.(*ast.Identifier); ok {
    if g.isEnumType(ident.Value) {
        return ident.Value + expr.Field.Value
    }
}
```

### 9. JSON Serialization

**Decision:** Serialize as the value (Go's default behavior). No special handling needed.

Since enums are typed constants, Go's `encoding/json` marshals them as their underlying value (`200` for `Status.OK`, `"debug"` for `LogLevel.Debug`). This is the expected behavior for APIs.

A `# kuki:json name` directive for name-based serialization could be added later as an opt-in.

---

## EBNF Grammar Addition

The following productions should be added to `docs/kukicha-grammar.ebnf.md`:

```ebnf
TopLevelDeclaration ::=
    | TypeDeclaration
    | InterfaceDeclaration
    | EnumDeclaration
    | FunctionDeclaration
    | MethodDeclaration

EnumDeclaration ::= "enum" IDENTIFIER NEWLINE INDENT EnumCaseList DEDENT

EnumCaseList ::= EnumCase { EnumCase }

EnumCase ::= IDENTIFIER "=" Literal NEWLINE
```

`enum` should be added to the reserved keywords list.

---

## Formatter Support

The formatter (`internal/formatter/printer.go`) needs a `case *ast.EnumDecl:` that prints:

```
enum Name
    Case1 = Value1
    Case2 = Value2
```

This follows the same pattern as `TypeDecl` formatting: print the keyword + name, then indent and print each child on its own line.

---

## Implementation Phases

### Phase 1: Core Enums (ship first)

Single-package enum declarations, dot access, codegen, and formatter.

| Step | File | Change |
|------|------|--------|
| 1 | `internal/lexer/token.go` | Add `TOKEN_ENUM`, add `"enum"` to keywords |
| 2 | `internal/ast/ast.go` | Add `EnumDecl`, `EnumCase` nodes |
| 3 | `internal/parser/parser_decl.go` | Add `parseEnumDecl()` |
| 4 | `internal/semantic/symbols.go` | Add `TypeKindEnum`, add `EnumCases` to `TypeInfo` |
| 5 | `internal/semantic/semantic_declarations.go` | `collectEnumDecl()` + `analyzeEnumDecl()` |
| 6 | `internal/semantic/semantic_calls.go` | Enum case resolution in `analyzeFieldAccessExpr` |
| 7 | `internal/codegen/codegen_decl.go` | `generateEnumDecl()` → `type X int` + `const (...)` |
| 8 | `internal/codegen/codegen_expr.go` | `Status.OK` → `StatusOK` in `exprToString` |
| 9 | `internal/formatter/printer.go` | Format enum declarations |
| 10 | `docs/kukicha-grammar.ebnf.md` | Add `EnumDeclaration` production |
| 11 | Tests in each package | Lexer, parser, semantic, codegen, formatter |

### Phase 2: Safety (follow-up)

| Step | File | Change |
|------|------|--------|
| 1 | `internal/semantic/semantic_declarations.go` | Zero-value warning for integer enums without `0` case |
| 2 | `internal/semantic/semantic_statements.go` | Exhaustiveness checking for switch/when on enum types |

### Phase 3: Polish (follow-up)

| Step | File | Change |
|------|------|--------|
| 1 | `internal/codegen/codegen_decl.go` | Auto-generate `String()` method (skip if user-defined) |
| 2 | `internal/semantic/stdlib_types.go` | Add `EnumCases` to registry types for cross-package support |
| 3 | `cmd/genstdlibregistry/main.go` | Emit enum case metadata from `.kuki` files |
