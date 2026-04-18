# Stdlib Review TODO

Ongoing audit of `stdlib/` packages against the project's "newbie-friendly,
AI/devops core, pipe/onerr friendly" design. See `stdlib/CLAUDE.md` for the
wrapper keep/drop rubric that drives these decisions.

## Focus

- **Drop passthrough wrappers** whose call shape matches the underlying Go
  stdlib function (`cast.Atoi` → `strconv.Atoi`, `errors.Is` → `goerrors.Is`).
- **Keep wrappers** that hide Go stdlib ceremony (e.g. `base64.StdEncoding.EncodeToString`),
  add pipe/onerr ergonomics, or provide Kukicha-novel patterns (sample-driven
  generics, `error "msg"` native syntax).
- **Remove dead knobs** left behind by past migrations (e.g. jsonv2 revert).
- Dogfood Kukicha syntax everywhere — no raw Go operators in `.kuki` sources.

## Aftermath

Before you change or remove any stdlib function review it's usage in 5 repos in ~/repos/kukicha, our examples/.kuki and our tutorials

## Reviewed

- `stdlib/cast` — removed `Atoi`, `ParseFloat` passthroughs; kept `Smart*` real
  coercion helpers.
- `stdlib/errors` — removed `Is`, `Unwrap`, `New`, `Join` passthroughs; kept
  `Wrap`, `Opaque`, `NewPublic`/`Public`/`PublicError` (dual-message). Native
  `error "msg"` syntax is the preferred replacement for `errors.New`.
- `stdlib/encoding` — removed `Base64RawEncode`, `Base64RawURLEncode` (advanced
  no-padding variants with no decode counterparts); kept `Base64*`/`Hex*`
  friendly wrappers that hide `StdEncoding.EncodeToString` ceremony.
- `stdlib/http` — reviewed, kept as-is (pipe-friendly server helpers).
- `stdlib/fetch`, `stdlib/net`, `stdlib/parse` — reviewed in the NewExternal /
  SSRF-guard pass; kept.
- `stdlib/json` — removed dead `WithDeterministic` knob left over from the
  jsonv2 → `encoding/json` revert (commit `cbbb4e8`). Kept Marshal/Unmarshal
  passthroughs since callers typically already import `stdlib/json` for
  `MarshalPretty`/`Parse`/`Encode`/etc.; forcing a dual import would be net
  friction. Sample-pattern helpers (`Parse`, `ParseString`, `DecodeRead`) are
  Kukicha-novel and stay.
- `stdlib/html` — removed `Attr` (literal duplicate of `Escape`; docstring
  already admitted "identical to Escape() but communicates intent"). Kept the
  Fragment component API (`Render`, `Embed`, `Join`, `Map`, `When`, `WhenElse`,
  `WriteTo`, `WriteStatusTo`, `String`, `IsEmpty`) — all Kukicha-novel or
  hide real HTTP header/write ceremony. Kept `Escape` since callers already
  import `stdlib/html` for Fragment composition; forcing a dual Go-`html`
  import just to escape would be friction. Cross-repo migration: updated
  `~/repos/kukicha/kukicha.org` (`components/cards.kuki`, `components/layout.kuki`
  — 5 call sites) to use `html.Escape`. Other 4 repos (`owui`, `mizuya`,
  `game`, `infer`) + `examples/` + `docs/tutorials/` clean.
- `stdlib/template` — removed `Funcs` (broken stub — doc claimed it registered
  functions but body silently returned input unchanged, a footgun), `Render`
  (literal duplicate of `Parse`; both set Content + empty map), `New` +
  `WithContent` (verbose builder pattern only used in own test; `Parse(content)`
  is one call), and `Must` (Kukicha's native `onerr panic`/`onerr fatal` covers
  this). Kept `Parse`, `Data`, `Execute`, `HTMLExecute`, `RenderSimple`,
  `HTMLRenderSimple`, `TemplateData` — pipe-builder and one-shot wrappers that
  hide `text/template` + `html/template` ceremony. Fixed pre-existing tutorial
  bug (`docs/tutorials/web-app-tutorial.md` Step 8) that called
  `template.New("home") |> .Parse(html)` — Go-stdlib shape against the
  `stdlib/template` wrapper, would not compile — migrated to
  `template.HTMLRenderSimple` + `httphelper.HTML` so user-supplied data is
  auto-escaped. Cross-repo: no calls in owui/mizuya/game/infer/kukicha.org
  (only a package-name string literal in `kukicha.org/main.kuki:69`).

- `stdlib/regex` — reviewed, all wrappers earn their keep. One-shots
  (`Match`, `Find`, `FindAll`, `FindGroups`, `FindAllGroups`, `Replace`,
  `ReplaceFunc`, `Split`) hide `regexp.MustCompile(p)` + method-call
  ceremony. `*Compiled` variants are required because `Pattern.re` is a
  private field; callers can't bypass with `p.re.MatchString(text)`.
  `Compile`/`MustCompile`/`IsValid` are genuine helpers. **One behavior
  fix:** `FindAllGroups` previously returned `(empty, error "no matches
  found")` on zero hits, asymmetric with `FindAll` (returns empty list,
  no error). Zero-hit is a normal multi-match outcome, not an error —
  changed signature to `list of list of string` (dropped the error
  return). No cross-repo callers to migrate. **Deferred follow-ups**
  (out of scope for keep/drop): asymmetric compiled set (no
  `FindAllGroupsCompiled`, no `ReplaceFuncCompiled`); `regex.Match`
  with dynamic patterns can panic via `MustCompile` — a
  `(bool, error)` variant for untrusted input would help.

- `stdlib/string` — applied the html/import-cohesion precedent: callers
  already import `stdlib/string` for the genuine Kukicha-novel helpers
  (`Title` hides `cases.Title(language.Und).String` + `golang.org/x/text`
  dep, `PadLeft`/`PadRight` have no Go equivalent, `IsBlank` is a
  non-obvious compound), so forcing dual `strings` imports for the cohesive
  passthroughs (`Contains`, `HasPrefix`, `ToLower`, `TrimSpace`, etc. —
  ~150 call sites) would be friction. Kept those. Dropped only the
  redundant-within-the-package entries: `Len(s)` (shadows the Go builtin
  `len()`), `Concat(parts)` (just `Join(parts, "")`), `IsEmpty(s)` (just
  `s equals ""`), and the dead "Builder Functions" comment block
  documenting a function that never existed. All call sites were inside
  the package's own tests — no cross-codebase migration needed. Deferred
  follow-up: `Lines` uses a hardcoded `"\n"` separator and won't handle
  `\r\n` on Windows-authored inputs; could migrate to `bufio.Scanner`.

## Not yet reviewed

### Data / format
- `stdlib/datetime`
- `stdlib/color`

### Collections
- `stdlib/slice`
- `stdlib/maps`
- `stdlib/set`
- `stdlib/sort`
- `stdlib/iterator`
- `stdlib/container`
- `stdlib/table`

### I/O & system
- `stdlib/files`
- `stdlib/shell`
- `stdlib/sandbox`
- `stdlib/git`
- `stdlib/input`
- `stdlib/random`
- `stdlib/crypto`

### Concurrency / context
- `stdlib/concurrent`
- `stdlib/ctx`
- `stdlib/retry`

### Storage
- `stdlib/db`
- `stdlib/sqlite`

### AI / agent
- `stdlib/llm`
- `stdlib/mcp`
- `stdlib/skills`
- `stdlib/obs`

### Testing
- `stdlib/test`
