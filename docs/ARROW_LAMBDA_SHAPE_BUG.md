# Arrow lambda shape bug

## Summary

Some Kukicha arrow lambdas currently transpile to Go closures without the expected return type.

When that happens, valid Kukicha such as:

```kukicha
items |> slice.Filter((t string) => semverPattern.MatchString(t))
```

is emitted as Go like:

```go
slice.Filter(items, func(t string) { return semverPattern.MatchString(t) })
```

instead of:

```go
slice.Filter(items, func(t string) bool { return semverPattern.MatchString(t) })
```

## Symptoms

The generated Go fails to compile with errors like:

```text
in call to slice.Filter, type func(t string) of func(t string) {…} does not match inferred type func(string) bool for func(T) bool
too many return values
    have (bool)
    want ()
```

## Reproduction

Observed in `examples/gh-semver-release/main.kuki` with:

```kukicha
tags := raw
    |> string.Lines()
    |> slice.Filter((t string) => semverPattern.MatchString(t))
```

The same file contains other arrow lambdas that transpile correctly, so the issue appears to depend on the expression shape rather than arrow lambdas in general.

## Expected behavior

Expression lambdas should preserve their inferred return type in generated Go regardless of whether the body is a plain comparison, a negated call, or a method call such as `semverPattern.MatchString(t)`.

## Fix

The root cause was a two-layer miss:

1. `go_stdlib_gen.go` emitted `TypeKindReference` for pointer return types without preserving the named element type, so `regexp.MustCompile` returned an anonymous reference.
2. Because `semverPattern`'s type had no name, `analyzeMethodCallExpr` couldn't match it against any of the hand-coded instance method tables, leaving the lambda body typed as `TypeKindUnknown`.
3. `inferExprReturnType` in codegen discards `TypeKindUnknown` and the `MethodCallExpr` case fell back to the narrow `boolMethods` allowlist, which didn't include `MatchString`.

**Applied fixes (all three layers):**
- `cmd/gengostdlib/main.go`: `goTypeToRepr` now preserves the element type name for pointer returns (e.g., `*regexp.Regexp`).
- `internal/semantic/go_stdlib_gen.go`: regenerated — `regexp.MustCompile` now carries `Name: "*regexp.Regexp"`.
- `internal/semantic/semantic_calls.go`: `analyzeMethodCallExpr` now resolves `*regexp.Regexp` instance methods (`MatchString`, `Match`, `MatchReader`, `FindString`, `ReplaceAllString`, etc.) with correct return types.

The inline form now compiles correctly:

```kukicha
tags := raw
    |> string.Lines()
    |> slice.Filter((t string) => semverPattern.MatchString(t))
```

## Status

Fixed. Regression test added: `TestArrowLambdaMethodCallReturnType` in `internal/codegen/codegen_test.go`.
