# Multi-line parenthesized calls

Surfaced during the `docs/tutorials/` review: long function calls split
across lines inside `()` don't parse. Users hit this constantly in
idiomatic code.

## The problem

This fails with `unexpected token in expression: NEWLINE`:

```kukicha
db.Exec(d.pool,
    "INSERT INTO links (code, url) VALUES (?, ?)",
    code, url)
```

So does every multi-line closure-list call:

```kukicha
concurrent.Parallel(
    () => processChunk(urlsA),
    () => processChunk(urlsB),
)

result.Match(
    fetchLinkResult(code),
    (link any) => httphelper.JSON(w, link),
    (cause error) => httphelper.JSONNotFound(w, "not found"),
)
```

Today the lexer only suppresses `TOKEN_NEWLINE` inside `[]` and literal
`{}` (`braceDepth > 0`). Parens are deliberately excluded — the comment
in `internal/lexer/lexer.go` explains why:

> `()` (parentheses) do NOT suppress newlines when inside a function
> literal body — closures need `INDENT/DEDENT` for their block structure.

That's the real constraint. Closure syntax uses `(params) =>` followed
by either an inline expression OR an indented block:

```kukicha
fn := (x int, y int) =>
    z := x + y
    return z
```

If the lexer blindly ate newlines between `(` and `)`, the closure body
would never get its `INDENT`/`DEDENT` tokens.

## The fix

Paren continuation is fine **until we see `=>`** — at that point the
block belongs to a closure and needs normal indent handling again. Two
workable designs:

### Option A: toggle on `=>` token

1. Track `parenDepth` separately from `braceDepth`.
2. Suppress `TOKEN_NEWLINE` while `parenDepth > 0`.
3. When the lexer emits `TOKEN_ARROW` (`=>`), decrement a "newline
   suppression" counter back to pre-paren behavior **for that closure's
   body only**. Once the closure body's matching structure closes (via
   DEDENT or the enclosing `)`), restore.

This is subtle because a single paren group can contain multiple
closures:

```kukicha
Match(x,
    (a) => a + 1,
    (b) => b * 2,
)
```

Each `=>` is followed by an expression, not a block. The rule becomes:
the `=>` only resumes indent processing if the next token is `NEWLINE`
+ `INDENT` (i.e. block-bodied closure). If `=>` is followed by an
expression, we stay in paren-continuation mode.

### Option B: explicit block closure marker

Require block-bodied closures inside paren calls to use `{ }`:

```kukicha
Parallel(
    () => { doA(); doB() },
    () => doC(),
)
```

Cleaner lexer but forces users into brace syntax for multi-statement
closures. Goes against the "indentation is primary" design.

**Recommendation: Option A.** Option B pushes the cost onto every
caller; Option A pushes it into the lexer once.

## Implementation sketch (Option A)

1. **`internal/lexer/lexer.go`:**
   - Add `parenDepth int` separate from `braceDepth`.
   - In `(` handler: increment `parenDepth`. In `)` handler: decrement.
   - In `TOKEN_NEWLINE` emission: if `parenDepth > 0 && closureBlockDepth == 0`,
     suppress the newline (same mechanism as `braceDepth`).
   - Add `closureBlockDepth int` — incremented when `=>` is followed by
     `NEWLINE` + leading whitespace greater than the enclosing line's
     indent (block-bodied closure), decremented on matching `DEDENT`.
   - Leading whitespace on continuation lines in paren mode: consume
     without emitting `INDENT`/`DEDENT` (mirror `braceDepth` logic).

2. **Trailing-comma tolerance:** Once continuation works, `Parallel(a,
   b,\n)` should accept the trailing comma before `)`. The parser
   already skips trailing commas in list/map literals; extend to call
   argument lists.

3. **Tests:**
   - `internal/lexer/lexer_test.go` — tokenize each of:
     - multi-line call with no closures
     - multi-line call with inline-expression closures
     - multi-line call with block-bodied closure
     - trailing comma before `)`
   - `internal/parser/parser_test.go` — parse the same forms into
     `CallExpr` with the right argument count.
   - End-to-end: add one or two tutorial snippets (from
     `production-patterns-tutorial.md` / `concurrent-url-health-checker.md`)
     to the codegen test corpus to lock in the multi-line call shape.

4. **Formatter:** `internal/formatter/` should emit multi-line calls
   when the single-line form exceeds the configured line width. Without
   this, `kukicha fmt` will re-flatten anything a user writes
   multi-line, making the feature invisible.

## Scope / blast radius

- Lexer change is additive — existing single-line calls tokenize the
  same way.
- No parser changes needed beyond trailing-comma tolerance in call
  args.
- Formatter update is the only user-visible behavior change. Omitting
  it leaves the feature usable but un-idiomatic.

## Why it matters

Every long SQL query, every `concurrent.Parallel(...)` fan-out, every
`http.NewServeMux()` wiring call, and every `Match`/`AndThen` chain
with closure arguments wants to wrap across lines. Today users either
write 140-column lines or the code doesn't compile. Fixing this
unblocks a large swathe of natural-looking code and removes a paper
cut that shows up in nearly every non-trivial program.

## Follow-up: after this lands

Sweep the tutorials:
- `docs/tutorials/production-patterns-tutorial.md` — restore multi-line
  `db.Exec`/`db.Query` formatting where it aids readability.
- `docs/tutorials/concurrent-url-health-checker.md` — the
  `concurrent.Parallel(\n    () => ...,\n)` block will just work.
- Audit `examples/` for flattened calls that would read better
  wrapped.
