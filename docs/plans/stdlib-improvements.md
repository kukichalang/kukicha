# Stdlib Improvements

Follow-up items surfaced during the `examples/` review pass.

## `db.ScanAll` / `db.ScanOne` — type-parameter inference

Current signature forces every caller to cast:

```kukicha
all := db.Query(pool, listSQL) |> db.ScanAll(empty list of Bookmark) onerr panic "scan: {error}"
for b in all.(list of Bookmark)
    ...
```

The `.(list of Bookmark)` assertion is only there because `ScanAll` is
typed `(rows Rows, sample any) (any, error)`. The `sample` is already a
typed empty slice — the information is present, it's just erased at the
function boundary.

### Proposal

Make `ScanAll` / `ScanOne` generic the same way `stdlib/slice` and
`stdlib/set` are. In `.kuki` source that means:

```kukicha
func ScanAll(rows Rows, sample list of any) (list of any, error)
func ScanOne(rows Rows, sample any) (any, error)
```

Generated Go should become:

```go
func ScanAll[T any](rows Rows, sample []T) ([]T, error)
func ScanOne[T any](rows Rows, sample T) (T, error)
```

Callers then write:

```kukicha
bookmarks := db.Query(pool, listSQL) |> db.ScanAll(empty list of Bookmark) onerr panic "{error}"
for b in bookmarks
    ...
```

No cast. No `.(list of Bookmark)` ceremony.

### What's needed

1. **Compiler opt-in for `stdlib/db`.** Placeholder resolution (`any` →
   `T`) is gated per-package in `internal/codegen/codegen_decl.go`:
   `isStdlibIter`, `isStdlibSlice`, `isStdlibSort`,
   `isStdlibConcurrent`, `isStdlibFetch`, `isStdlibJSON`,
   `isStdlibSet`. Add `isStdlibDB` + `inferDBTypeParameters` and wire
   both into `generateFunctionDecl`.
2. **Body adjustment.** `ScanAll`'s implementation builds the slice via
   `reflect.MakeSlice(sliceType, …)` and returns `resultSlice.Interface()`.
   Under generics the return type is `[]T`, so the last line must be
   `resultSlice.Interface().([]T), nil`. Likewise for `ScanOne`.
3. **Registry regeneration.** Run `make genstdlibregistry` after
   changing the signatures so semantic analysis infers the right
   return type at each call site.
4. **Tests.** Add a codegen test that `db.ScanAll(rows, empty list of User)`
   emits `[]User` (not `[]any`) in the generated Go, and a semantic
   test that downstream `for u in result` typechecks without a cast.

### Scope / blast radius

Every caller of `db.ScanAll` / `db.ScanOne` currently appends
`.(list of T)` or `.(T)`. After this change those casts become dead
code — they still compile (redundant type assertions are legal in Go)
but should be removed in a follow-up sweep across:

- `examples/bookmark-tags/main.kuki`
- `stdlib/sqlite/sqlite_test.kuki`
- Any new examples that land in the meantime

### Why it matters

The cast is the single biggest readability leak in database-shaped
examples. Fixing it brings `stdlib/db` in line with `stdlib/slice` and
`stdlib/set`, which already read fluently through pipe chains without
any type-assertion tax.
