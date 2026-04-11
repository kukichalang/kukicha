# Untyped Composite Literals in `onerr` Handlers

## Problem

Untyped composite literals (`{field: val}`) cannot currently be used inside
`onerr` handler expressions. The following does **not** compile:

```kukicha
re := regexp.Compile(pattern) onerr return {}, error "invalid pattern: {error}"
```

It must instead be written with an explicit type:

```kukicha
re := regexp.Compile(pattern) onerr return Pattern{}, error "invalid pattern: {error}"
```

This affects all `onerr return TypeName{}, error "..."` patterns in the stdlib
(`llm`, `mcp`, `semver`, `regex`).

## Root Cause

The `onerr` handler is stored as a `ReturnExpr` in `OnErrClause.Handler` (see
`internal/ast/ast.go`). When `analyzeOnErrClause` in
`internal/semantic/semantic_onerr.go` (line 75) analyzes the handler, it calls
`a.analyzeExpression(clause.Handler)` directly — without first resolving any
untyped composite literals inside the return values against `a.currentFunc.Returns`.

The resolution hook (`resolveUntypedLiteral`) is called for normal return
statements in `semantic_statements.go` (line 591), but that path is never
reached for `onerr` handlers, which are expressions rather than statements.

## Fix

In `analyzeOnErrClause`, before calling `a.analyzeExpression`, check whether
the handler is a `*ast.ReturnExpr`. If so, and if `a.currentFunc` is non-nil,
call `resolveUntypedLiteral` on each return value against the corresponding
`a.currentFunc.Returns[i]`:

```go
// in analyzeOnErrClause, before the final analyzeExpression call:
if ret, ok := clause.Handler.(*ast.ReturnExpr); ok && a.currentFunc != nil {
    if len(ret.Values) == len(a.currentFunc.Returns) {
        for i, val := range ret.Values {
            a.resolveUntypedLiteral(val, a.currentFunc.Returns[i])
        }
    }
}
```

This mirrors the existing logic in the return-statement analyzer and requires
no changes to the AST, parser, or codegen.

## Affected Stdlib Sites (as of stdlib untyped-literal refactor)

```
stdlib/llm/llm.kuki       — Completion{}, Response{}, AnthropicResponse{}  (6 sites)
stdlib/mcp/mcp.kuki       — CallToolResult{}                                (1 site)
stdlib/semver/semver.kuki — Version{}                                       (3 sites)
stdlib/regex/regex.kuki   — Pattern{}                                       (1 site)
```

Once the fix is in place, all of these can be simplified to `return {}, error "..."`.
