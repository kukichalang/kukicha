# Formatter: comment misplacement near lambda-closing parens

## Bug

The formatter is not idempotent when a comment sits between two
statements whose arguments end with a multi-line lambda. On the first
pass the comment gets repositioned; on the second pass a blank line is
added (or removed), so `fmt | fmt` != `fmt`.

### Reproduction

```kukicha
func TestNesting(t reference testing.T)
    color.SetEnabled(true)

    t.Run("first", (t reference testing.T) =>
        x := 1
    )
    # This comment gets moved across passes
    t.Run("second", (t reference testing.T) =>
        y := 2
    )
```

Running `kukicha fmt` twice produces different output — caught by
`TestFormatterIdempotencyOnKukiFiles`.

## Root Cause

`maxExprLine()` in `internal/formatter/printer.go` (lines ~487-590)
determines the last line a statement occupies. It handles
`StructLiteralExpr`, `ListLiteralExpr`, `MapLiteralExpr`,
`UntypedCompositeLiteral`, `CallExpr`, `MethodCallExpr`, and `PipeExpr`
— but **omits `ArrowLambda` and `FunctionLiteral`**.

When a `CallExpr` has an `ArrowLambda` argument with a multi-line block
body, `maxExprLine` returns the line of the call's closing paren token
rather than the last line of the lambda's block. This feeds a wrong
`prevEndLine` into the blank-line detection in
`printBlockWithComments()` (`formatter.go`, lines ~327-344):

```go
if prevEndLine > 0 && stmtLine > prevEndLine+1 {
    p.writeLine("")
}
```

An incorrect `prevEndLine` means the formatter alternately inserts or
removes a blank line before the comment on each pass, breaking
idempotency.

## Affected Files

| File | Role |
|------|------|
| `internal/formatter/printer.go` ~L487-590 | `maxExprLine()` — missing lambda cases |
| `internal/formatter/formatter.go` ~L327-344 | `printBlockWithComments()` — blank-line insertion depends on accurate endLine |
| `internal/formatter/comments.go` ~L216-245 | `attachLeadingComments()` — comment attachment proximity uses endLine |
| `internal/ast/ast.go` ~L1042-1074 | `ArrowLambda` / `FunctionLiteral` node definitions |

## Fix

Add `ArrowLambda` and `FunctionLiteral` cases to `maxExprLine()`,
following the same pattern as `StructLiteralExpr` et al.: walk into the
block body and return the line of the last statement (plus one for the
implicit closing dedent), so `prevEndLine` is accurate.

Sketch:

```go
case *ast.ArrowLambda:
    if e.Block != nil && len(e.Block.Statements) > 0 {
        last := e.Block.Statements[len(e.Block.Statements)-1]
        if endLine := maxStmtLine(last); endLine > line {
            line = endLine
        }
    }

case *ast.FunctionLiteral:
    if e.Body != nil && len(e.Body) > 0 {
        last := e.Body[len(e.Body)-1]
        if endLine := maxStmtLine(last); endLine > line {
            line = endLine
        }
    }
```

The `maxStmtLine` helper (or inlined equivalent) should recursively
compute the last source line of a statement, mirroring what
`maxExprLine` does for expressions.

## Testing

1. Add a targeted test in `internal/formatter/` with the reproduction
   case above — format twice, assert output is identical.
2. Verify `TestFormatterIdempotencyOnKukiFiles` passes on the
   `stdlib/color/color_test.kuki` file that originally triggered this
   (the workaround — moving the comment above the first `t.Run` — can
   be reverted once the fix lands).
