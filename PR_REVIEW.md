# PR #74 Review — Const Blocks, Triple-Quote Strings, stdlib/sort, stdlib/table, stdlib/crypto

## Overall Assessment

Solid PR that delivers Phase 1 (const blocks, multi-line strings) and Phase 2 (stdlib/crypto, stdlib/sort, stdlib/table) from the `NEW_STUFF.md` roadmap. The example program update is a compelling demonstration of the new features working together. A few issues and decisions worth discussing below.

---

## Decisions vs. NEW_STUFF.md Roadmap

### Implemented as planned
- **Const blocks** — syntax matches the spec exactly (`const Name = val` and grouped `const` + indent block)
- **Triple-quote strings** — `"""..."""` with dedent and interpolation, as proposed
- **stdlib/sort** — API matches spec (`Strings`, `Ints`, `By`, `Reverse`), plus added `Float64s`
- **stdlib/table** — API matches spec (`New`, `AddRow`, `Print`, `PrintWithStyle`, `ToString`), plus `ToStringWithStyle`

### Deviations from the roadmap

1. **`constant` keyword alias** — The roadmap asked "constant alias for new users like function and variable?" and this PR adds `constant` → `TOKEN_CONST` in the keyword table. Good decision — consistent with `function`/`variable` aliases. However, `CLAUDE.md` and the docs should be updated to mention `constant` as a keyword alias alongside `function`/`variable`.

2. **Guard clauses (Phase 1.3) skipped** — The roadmap marked this as "optional" and low priority. Reasonable to defer.

3. **crypto: `HashPassword`/`CheckPassword` omitted** — The roadmap proposed bcrypt functions wrapping `golang.org/x/crypto/bcrypt`. The PR intentionally avoids external dependencies ("All functions use Go standard library only"). This is a good call — keeping stdlib dependency-free is cleaner. The roadmap's own comment "I'm not sure we need this yet" supports this. Could be added later as a separate `stdlib/bcrypt` package if needed.

4. **`table.Print`/`PrintWithStyle` not registered in stdlib_registry** — Looking at the generated registry, `Print` and `PrintWithStyle` are missing. Only `New`, `AddRow`, `ToString`, and `ToStringWithStyle` are registered. This means `t |> table.Print()` and `t |> table.PrintWithStyle("box")` won't get return-type information from the semantic analyzer. Since these return void this might be fine, but it's worth verifying that calling them compiles without issues.

5. **`sort.ByKey` omitted** — The roadmap proposed `ByKey(items, keyFunc)` for sort-by-extracted-key. Only `By` (comparator-based) was implemented. This is the more useful one to have, but `ByKey` would be a natural addition for pipe-friendly usage like `repos |> sort.ByKey((r Repo) => r.Stars)`.

---

## Code Review: Issues & Suggestions

### Const Blocks

**Semantic analysis registers consts as `SymbolVariable`** (`semantic_declarations.go`):
```go
err := a.symbolTable.Define(&Symbol{
    Name:    spec.Name.Value,
    Kind:    SymbolVariable,  // ← Should this be a dedicated SymbolConst?
    Type:    &TypeInfo{Kind: TypeKindUnknown},
```
Constants are semantically different from variables — they can't be reassigned and must have compile-time-evaluable values. Registering them as `SymbolVariable` means the semantic analyzer won't catch `const X = 5; X = 10` as an error. Consider adding a `SymbolConst` kind and checking for reassignment in the assignment analyzer.

**No type annotation support** — Go supports `const MaxRetries int = 5`. The current parser only supports `const Name = value`. This is fine for v1 since Go infers const types, but the roadmap didn't mention this limitation.

**Const value validation** — There's no check that const values are actually constant expressions. `const X = someFunction()` would parse and generate Go code that won't compile. A semantic pass checking for literal/const-only expressions would catch this earlier, but Go's compiler will catch it regardless, so this is low priority.

### Triple-Quote Strings

**Source injection approach is clever but fragile** (`scanStringFromContent`):
The implementation extracts the raw content, processes dedent, then re-injects the content back into `l.source` at the current position with escape transformations. This works but has risks:

- **Memory**: Re-allocating `l.source` for every triple-quote string copies the entire remaining source. For large files with many triple-quoted strings, this could be significant.
- **Line number tracking**: After injection, the `l.line` and `l.column` fields may be off for tokens following the triple-quote string, since the injected content has different length than the original. This could affect error messages pointing to wrong lines.

**CR handling**: The `\r` skip in `scanStringFromContent` is good for Windows compatibility, but `dedentTripleQuote` doesn't handle `\r\n` line endings in the `strings.Split(raw, "\n")` call — lines would retain trailing `\r`. The leading `\r\n` stripping looks correct though.

### stdlib/crypto

**`error` as variable name** (`crypto.kuki`):
```kukicha
_, error := rand.Read(b)
if error != empty
```
This uses `error` as a variable name, which is valid in Kukicha (per CLAUDE.md: "'error' and 'empty' can be used as variable names"). Consistent with the language's design.

**No `SHA512` variant** — Minor, but SHA-512 is commonly needed. Easy to add later.

### stdlib/table

Well-structured with clean separation of rendering logic per style. The `AddRow` padding logic handles mismatched column counts gracefully.

**`renderTable` allocates heavily** — Every cell goes through `fmt.Sprintf("%-*s", ...)`. For very large tables this could be slow, but for CLI output this is perfectly fine.

### stdlib/sort

Clean and idiomatic. Using `slices.Clone` before sorting preserves immutability, which is the right default for a functional-style stdlib.

**`sort.By` uses `sort.SliceStable` but imports both `slices` and `sort`** — `slices.SortStableFunc` would be more modern (Go 1.21+), but `sort.SliceStable` with index-based access works fine.

---

## Example Program Update (gh-semver-release)

The example update is **the strongest part of this PR** — it demonstrates real, practical value from every new feature:

1. **Const block** extracts magic strings (`DefaultBump`, `InitialTagEnv`, etc.) into named constants at the top of the file — much more readable.

2. **Triple-quote strings** transform the unreadable single-line GraphQL query into a properly formatted, readable query. The jq filter similarly benefits. This is the killer use case.

3. **`sort.Strings()`** adds alphabetical ordering to the repo list — small but useful.

4. **`table.New`/`table.AddRow`/`table.Print`** replaces manual tab-separated output with proper formatted tables. The `list`→`entries` refactor to collect first, then render, is cleaner.

5. **`cmdStr` extraction** — The change from inline `{cmdArgs |> string.Join(\" \")}` to a separate `cmdStr` variable avoids escaped quotes in interpolation. Good simplification.

**One issue in the example**: The const `DefaultBump = "patch"` and `DefaultInitialTag = "v0.0.1"` are string constants, but the Go code generator will emit `const DefaultBump = "patch"` — Go can infer the type from the literal, so this works. Just noting that the generated Go is correct.

---

## Summary

| Category | Verdict |
|----------|---------|
| Const blocks | Good implementation, minor gap: no reassignment protection |
| Triple-quote strings | Works, but source-injection approach has edge-case risks |
| stdlib/crypto | Clean, wisely skips bcrypt dependency |
| stdlib/sort | Clean, missing `ByKey` from roadmap |
| stdlib/table | Well-structured, full-featured |
| Example update | Excellent showcase of all new features |
| Guard clauses | Not implemented (deferred) — fine |

**Recommendation**: The PR is in good shape. The const reassignment issue is the most impactful gap (it would silently generate invalid Go code), but Go's compiler catches it downstream. The triple-quote source injection is the main technical risk for future maintenance.
