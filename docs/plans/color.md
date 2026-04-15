# stdlib/color — ANSI terminal colors

## Motivation

Every CLI tool and example in the repo reinvents the same ~15 lines of
ANSI escape constants and one-line wrappers. Current offenders:

- `examples/mizuya/format.kuki` — 8 constants + 7 wrappers
  (`bold`, `dim`, `red`, `green`, `yellow`, `blue`, `cyan`)
- `examples/gh-semver-release/main.kuki` — similar pattern (spot-check)
- `examples/triage.kuki`, `examples/feature_sampler.kuki` — smaller but
  still hand-rolled

A tiny `stdlib/color` package eliminates this duplication and gives
tutorial-grade examples a one-liner for coloured output.

## Scope

Keep it **small, zero-dep, pure-ANSI**. No truecolor, no 256-color
palettes, no Windows console API shim (modern terminals handle ANSI;
`NO_COLOR`-aware users can opt out — see below). Anything fancier
(hyperlinks, progress bars, spinners) belongs in a separate package.

## Proposed API

```kukicha
petiole color

import "os"
import "stdlib/string" as strpkg

# Enabled reports whether ANSI output should be emitted. Honors the
# NO_COLOR convention (https://no-color.org) and falls back to isatty
# detection on stdout. Callable from user code to gate custom escapes.
func Enabled() bool

# SetEnabled forces the enabled/disabled state, overriding auto-detect.
# Useful in tests or when the caller knows the output sink.
func SetEnabled(on bool)

# Style wrappers — return s unchanged when Enabled() is false.
func Bold(s string) string
func Dim(s string) string
func Italic(s string) string
func Underline(s string) string

# Foreground colors (basic 8 + bright red for errors).
func Red(s string) string
func Green(s string) string
func Yellow(s string) string
func Blue(s string) string
func Magenta(s string) string
func Cyan(s string) string
func Gray(s string) string
func BrightRed(s string) string

# Semantic helpers — opinionated shorthands for CLI UX.
func Error(s string) string    # BrightRed + Bold
func Warn(s string) string     # Yellow
func Success(s string) string  # Green
func Info(s string) string     # Cyan
func Muted(s string) string    # Dim
```

All wrappers are pure string → string; they wrap with the ANSI
sequence and a reset suffix. When `Enabled()` is false they return
the input untouched, so callers never have to branch.

## Behavior details

- **`NO_COLOR` detection:** if `os.Getenv("NO_COLOR") != ""`, disable.
- **isatty fallback:** when not overridden, check whether
  `os.Stdout` is a terminal. Reuse whatever Go stdlib path we already
  pull in; add `golang.org/x/term` only if nothing cheaper works.
- **`FORCE_COLOR` override:** if set, enable regardless of tty.
  Common in CI log collectors that still render ANSI.
- **State:** a single package-level `bool` guarded by `sync.Once` for
  first-read auto-detect, plus `SetEnabled` for manual override.
  Keep it simple — no per-writer state.

## Non-goals

- 256-color / truecolor palettes — adds API surface for marginal gain;
  revisit if a concrete user shows up.
- Background colors — almost never used in CLI tools; skip until asked.
- Per-writer styling (colored `io.Writer` wrappers) — belongs in a
  logging/printing library, not here.
- Windows legacy console support — modern Windows Terminal + PowerShell
  handle ANSI natively. If a user complains, consider `x/sys/windows`
  shim then.

## Migration

After landing, update the four known call sites to drop their hand-
rolled helpers:

1. `examples/mizuya/format.kuki` — remove constants + wrappers, import
   `stdlib/color`, swap `bold(x)` → `color.Bold(x)`, etc. Net: ~30
   lines deleted.
2. `examples/gh-semver-release/main.kuki`
3. `examples/triage.kuki`
4. `examples/feature_sampler.kuki`

Also worth auditing `stdlib/cli` — `cli.Error`/`cli.Warn`/`cli.Fatal`
currently print with hard-coded escapes; they should route through
`stdlib/color` so `NO_COLOR` works uniformly.

## Tests

- Round-trip: `Bold("x")` with `SetEnabled(true)` contains the expected
  escape codes; with `SetEnabled(false)` equals `"x"`.
- `NO_COLOR` env var disables output.
- `Enabled()` after `SetEnabled` matches the forced value.
- Golden strings for each color — use `\x1b[...m` literals so a
  reader can eyeball the codes.

## Generator notes

- Add exports to `stdlib/color/color.kuki`; run `make genstdlibregistry`.
- No security directives needed — these are pure string helpers.
- No external module deps unless `x/term` ends up required for isatty.

## Open questions

- **Package name vs. identifiers:** `color.Red("x")` reads well; no
  conflict with a likely user variable unless someone has a local
  `color`. `as colorpkg` import alias is fine if so.
- **Should `stdlib/cli` depend on `stdlib/color`?** Probably yes —
  gives `NO_COLOR` uniform effect across the stdlib. Verify no import
  cycle first.
- **Reset strategy:** wrap every call with a full reset (`\x1b[0m`), or
  use `\x1b[22m`/`\x1b[39m` scoped resets? Full reset is simpler and
  matches what hand-rolled code already does; stick with that.
