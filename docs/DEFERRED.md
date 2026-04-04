# Deferred Work

Items from the transpiler roadmap that need more planning before
implementation. These aren't blocked — they need design decisions.

---

## Semantic Analyzer: Remaining Splits

**Context:** `DirectiveCollector` and `SecurityChecker` were extracted
(v0.0.31). The remaining extractions are blocked by tight coupling, not
complexity.

### 1. Typed Intermediate Results Between Passes

The three internal passes (`collectDirectives`, `collectDeclarations`,
`analyzeDeclarations`) communicate through shared mutable state on the
Analyzer struct (~18 fields). To make passes truly independent, each
needs typed inputs and outputs.

**Design questions:**
- Should `collectDeclarations` return a `SymbolTable` that
  `analyzeDeclarations` receives as input, or should both share a
  reference? The symbol table is mutated during analysis (new scopes
  pushed/popped), so passing by value doesn't work.
- `exprTypes` and `exprReturnCounts` are populated during analysis and
  consumed by codegen. Should they be part of an `AnalysisResult` return
  type? Changing `Analyze() []error` to `Analyze() *AnalysisResult`
  touches ~80 call sites across 20+ test files.
- The `importAliases` map is populated during `collectDeclarations` but
  read during `analyzeDeclarations` and by `typeAnnotationToTypeInfo()`.
  It crosses pass boundaries.

**Possible approach:** Keep `Analyze() []error` as the public API but
internally use a pipeline:

```go
func (a *Analyzer) Analyze() []error {
    directives := CollectDirectives(a.program)
    symbols := a.collectDeclarations(directives)
    a.analyzeDeclarations(symbols, directives)
    return a.errors
}
```

This makes data flow explicit without changing any callers.

### 2. Declaration Collector Extraction

`collectDeclarations()` and its ~10 helper methods use:
- `a.symbolTable` — to define symbols
- `a.typeAnnotationToTypeInfo()` — which itself reads `symbolTable`,
  `deprecatedTypes`, `importAliases`
- `a.error()` / `a.warn()` — for diagnostics
- `a.importAliases` — to track aliased imports
- `a.extractPackageName()` — import path parsing

Extracting with a back-pointer (like `SecurityChecker`) adds indirection
without reducing coupling. A real extraction needs either:
- A `DeclarationCollector` that owns its own error list and symbol table,
  returning both to the Analyzer when done
- Or the pipeline approach above, where `collectDeclarations` is a
  method that takes explicit parameters instead of reading struct fields

### 3. Lint Warning Extraction

Warnings are emitted at the point of detection, interleaved with type
checking:
- Deprecation warnings fire during `analyzeCallExpr` / `analyzeMethodCallExpr`
- Pipe discarded-error warnings fire during `analyzePipeExpr`
- Onerr lint (discard/panic in non-test) fires during `analyzeOnErrClause`
- Enum exhaustiveness warnings fire during `checkEnumExhaustiveness`

**Possible approach:** Collect lint candidates (structs with position +
category + context) during analysis, then run a `LintChecker` as a final
pass that filters and emits warnings. This separates "what to lint" from
"when to lint" but requires defining lint candidate types for each
category.

### 4. AnalysisResult Return Type

Changing `Analyze() []error` to return a struct bundles the scattered
getter methods (`ExprTypes()`, `ReturnCounts()`, `Warnings()`) into one
value. The change is mechanical but touches ~80 call sites.

**Possible approach:** Add `AnalyzeResult() *AnalysisResult` as a new
method, keep `Analyze() []error` as a wrapper for backward compat, then
migrate callers file-by-file.

---

## kukicha-blend (Go -> Kukicha Converter)

**Context:** Item 3 in the transpiler roadmap. Separate binary that
parses `.go` files with `go/parser` and suggests Kukicha transformations.

**Design questions:**
- Should it be a separate binary (`cmd/kukicha-blend/`) or a subcommand
  (`kukicha blend`) that shells out?
- Which transformations to prioritize? The high-value ones:
  1. `if err != nil { return err }` -> `onerr return`
  2. `&&`/`||`/`!` -> `and`/`or`/`not`
  3. Brace blocks -> indentation
- Should `--apply` write `.kuki` files directly or produce a diff?
- How to handle Go constructs that Kukicha doesn't support yet (labeled
  breaks, goto, naked returns)? Skip them? Warn?

**Not blocked on:** anything technical. This can start any time — it's
independent of the compiler.

---

## Visitor Pattern for Codegen

**Context:** Item 7 in the transpiler roadmap. Codegen uses ad-hoc type
switches across 4 files (13K lines). Adding a new AST node requires
edits in multiple places with no compile-time exhaustiveness check.

**Design questions:**
- Should the visitor interface have one method per AST node type, or use
  a smaller interface with `VisitExpression` / `VisitStatement` /
  `VisitDeclaration` groupings?
- How to handle the `exprToString` pattern where expression codegen
  returns a string rather than writing to a buffer?
- The IR layer (`lower.go`, `emit.go`) already uses clean type switches
  over a small set of IR nodes — should it also adopt the visitor, or is
  it stable enough to leave alone?
- Migration strategy: one codegen file at a time, or big-bang?

**Prerequisite:** The `exhaustive` linter (or equivalent) should be set
up first so that adding a new AST node produces a compile-time error in
every visitor implementation.
