---
name: stdlib
description: Kukicha stdlib authoring guide — full package table, common usage patterns, security check patterns, common pitfalls, and rules for adding/modifying stdlib packages. Use when working in stdlib/ or reviewing stdlib usage.
---

# stdlib/ — Standard Library

Each package lives in `stdlib/<name>/` with:
- `<name>.kuki` — Kukicha source (types, enums, function signatures, inline implementations)
- `<name>.go` — **Generated** by `make generate` from the `.kuki` file. Never edit directly.

## Packages

| Package | Purpose | Key Functions |
|---------|---------|---------------|
| `stdlib/cast` | Smart type coercion (any → scalar) | SmartInt, SmartFloat64, SmartBool, SmartString, Atoi, ParseFloat |
| `stdlib/cli` | CLI argument parsing with subcommands | New, Description, Arg, AddFlag, Action, RunApp, Command, CommandFlag, CommandAction, GlobalFlag, CommandName, GetString, GetBool, GetInt, NewArgs, IsJSON, Error, Warn, Fatal |
| `stdlib/concurrent` | Parallel execution and concurrent map | Parallel, ParallelWithLimit, Map, MapWithLimit, Go |
| `stdlib/container` | Docker/Podman client via Docker SDK | Connect, ConnectRemote, New/Host/APIVersion/Open, ListContainers, ListImages, Pull, PullAuth, LoginFromConfig, Run, Stop, Remove, Build, Logs, LogsTail, Inspect, Wait/WaitCtx, Exec, Events/EventsCtx, CopyFrom, CopyTo |
| `stdlib/crypto` | Hashing, HMAC, and secure random (Go stdlib only) | SHA256, SHA256Bytes, HMAC, HMACBytes, RandomToken, RandomBytes, Equal |
| `stdlib/ctx` | Context timeout/cancellation helpers | Background, WithTimeout, WithTimeoutMs, WithDeadlineUnix, Cancel, Done, Err, Value |
| `stdlib/db` | SQL database (raw SQL + struct scanning, zero external deps) | Open, Close, Ping, Query, QueryRow, Exec, ScanAll, ScanOne, ScanRow, CloseRows, Transaction, TransactionWith, TxQuery, TxQueryRow, TxExec, Count, Exists; Types: Pool, Row, Rows, Tx, TxOptions |
| `stdlib/datetime` | Named formats, duration helpers, arithmetic, comparison | Format, Parse, Now, Today, AddDays, IsBefore, Unix, Sleep; Constants: ISO8601, RFC3339, Date, Time, DateTime |
| `stdlib/encoding` | Base64 and hex encoding/decoding | Base64Encode, Base64Decode, Base64URLEncode, Base64URLDecode, HexEncode, HexDecode |
| `stdlib/env` | Typed env vars with onerr | Get, GetOr, GetInt, GetIntOrDefault, GetBool, GetBoolOrDefault, GetFloat, GetList, Set, Unset, IsSet, All |
| `stdlib/errors` | Error wrapping and dual-message helpers | Wrap, Opaque, NewPublic, Public |
| `stdlib/fetch` | HTTP client (Builder, Auth, Sessions, Safe URL helpers, Retry) | Get, SafeGet, Post, Json, Decode, Text, Bytes, CheckStatus, URLTemplate, URLWithQuery, PathEscape, QueryEscape, New, NewExternal, Header/Timeout/Retry/MaxBodySize/Transport/Do, BearerAuth, BasicAuth, FormData, NewSession, DownloadTo |
| `stdlib/files` | File I/O operations | Read, ReadBytes, Write, WriteString, Append, AppendString, Exists, IsDir, IsFile, Copy, Move, Delete, DeleteAll, List, ListRecursive, MkDir, MkDirAll, TempFile, TempDir, Size, ModTime, Basename, Dirname, Extension, Join, Abs, UseWith, Watch |
| `stdlib/game` | 2D game library (kukichalang/game, Ebitengine wrapper, **WASM-only**) | Window, OnSetup, OnUpdate, OnDraw, Run, Clear, DrawRect, DrawCircle, DrawLine, DrawText, IsKeyDown, IsKeyPressed, MousePosition, MouseClicked, Overlaps, OverlapsCircle, CircleOverlapsRect, MakeColor, Random, RandomFloat, FrameCount |
| `stdlib/git` | Git/GitHub operations via gh CLI | ListTags, TagExists, DefaultBranch, CurrentBranch, ReleaseExists, CreateRelease, PreviewRelease, RepoExists, CurrentUser, Clone, CloneShallow |
| `stdlib/html` | Component-style HTML with auto-escaping | Render, Escape, Attr, Embed, WriteTo, WriteStatusTo, String, IsEmpty, Join, Map, When, WhenElse; Type: Fragment |
| `stdlib/http` | HTTP response/request helpers + security | JSON, JSONStatus, JSONCreated, JSONError, JSONBadRequest, JSONNotFound, Text, HTML, SafeHTML, ReadJSON, ReadJSONLimit, Redirect, SafeRedirect, SafeURL, SetSecureHeaders, SecureHeaders, WithCSRF, Serve, MethodNotAllowed, IsGet/IsPost/IsPut/IsDelete/IsPatch, GetQueryParam, GetHeader; Constants: StatusOK/NotFound/etc |
| `stdlib/input` | User input utilities | ReadLine, Prompt, Confirm, Choose |
| `stdlib/iterator` | Functional iteration (Go 1.23 iter.Seq) | Values, Filter, Map, FlatMap, Take, Skip, Enumerate, Chunk, Zip, Reduce, Collect, Any, All, Find |
| `stdlib/json` | encoding/json wrapper | Marshal, MarshalPretty, Unmarshal, MarshalWrite, UnmarshalRead, DecodeRead, NewEncoder, NewDecoder, Encode, Decode, WithDeterministic, WithIndent, WriteOutput |
| `stdlib/llm` | Large language model client (Chat Completions, OpenResponses, Anthropic; Retry) | New/Ask/Send/SendRaw/Complete, NewResponse/RAsk/RSend/Respond, NewMessages/MAsk/MSend/AnthropicComplete, Retry/RRetry/MRetry, Stream/RStream/MStream |
| `stdlib/maps` | Map utilities | Keys, Values, Contains, Has, Merge, SortedKeys |
| `stdlib/mcp` | Model Context Protocol server | New, Serve, Tool, Prop, Schema, Required, TextResult, ErrorResult |
| `stdlib/must` | Panic-on-error startup helpers | Do, DoMsg, Ok, OkMsg, Env, EnvOr, EnvInt, EnvIntOr, EnvBool, EnvBoolOr, EnvList, EnvListOr, True, False, NotEmpty, NotNil |
| `stdlib/netguard` | Network restriction & SSRF protection | NewSSRFGuard, NewAllow, NewBlock, Check, DialContext, HTTPTransport, HTTPClient |
| `stdlib/obs` | Structured observability helpers | New, Component, WithCorrelation, NewCorrelationID, Debug, Info, Warn, Error, Log, Start, Stop, Fail |
| `stdlib/parse` | Data format parsing | Json, JsonLines, JsonPretty, Csv, CsvWithHeader, Yaml, YamlPretty |
| `stdlib/random` | Random string and numeric generation | String, Alphanumeric, Int, Float |
| `stdlib/regex` | Regular expression matching and replacement | Match, Find, FindAll, FindGroups, FindAllGroups, Replace, ReplaceFunc, Split, IsValid, Compile, MustCompile + compiled variants |
| `stdlib/retry` | Retry with backoff | New, Attempts, Delay, Linear, Sleep |
| `stdlib/sandbox` | os.Root filesystem sandboxing | New, Close, Read, ReadString, Write, WriteString, Append, AppendString, MkDir, MkDirAll, List, Exists, IsDir, IsFile, Stat, Delete, DeleteAll, Rename, Path, FS |
| `stdlib/semver` | Semantic versioning (parse, bump, compare) | Parse, Bump, Format, Valid, Compare, Greater, Highest |
| `stdlib/shell` | Safe command execution | Run, Output, New/Dir/SetTimeout/Env/Execute, Args/FlagIf/Preview, Success, GetOutput, GetError, ExitCode, Which, Getenv, Setenv, Unsetenv, Environ |
| `stdlib/skills` | Runtime discovery of agent SKILL.md manifests | Discover, AgentSkills, ClaudeSkills |
| `stdlib/slice` | Slice operations (all generic) | Filter, Map, GroupBy, Sort, SortBy, First, Last, Drop, DropLast, Reverse, Unique, Chunk, Contains, IndexOf, Concat, Get, GetOr, FirstOne, FirstOr, LastOne, LastOr, Find, FindOr, FindIndex, FindLast, FindLastOr, IsEmpty, IsNotEmpty, Pop, Shift |
| `stdlib/sort` | Sorting slices (strings, ints, floats, custom) | Strings, Ints, Float64s, By, ByKey, Reverse |
| `stdlib/sqlite` | SQLite convenience layer over stdlib/db (ncruces/go-sqlite3) | Open, OpenMemory, OpenWith, Pragma, SetPragma, Tables, TableExists, IntegrityCheck, Vacuum, Backup, Version, BatchExec, CreateFunction, Dump |
| `stdlib/string` | String utilities | ToUpper, ToLower, Title, Trim, TrimSpace, TrimPrefix, TrimSuffix, TrimLeft, TrimRight, Split, SplitN, Join, Fields, Contains, HasPrefix, HasSuffix, Index, LastIndex, Count, Replace, ReplaceAll, Repeat, PadRight, PadLeft, Concat, EqualFold, Len, IsEmpty, IsBlank, Lines |
| `stdlib/table` | Terminal table rendering (plain, box, markdown) | New, AddRow, Print, PrintWithStyle, ToString, ToStringWithStyle |
| `stdlib/template` | Text templating (plain + HTML-safe) | New, Render, Parse, Data, WithContent, Execute, RenderSimple, HTMLExecute, HTMLRenderSimple, Must, Funcs |
| `stdlib/test` | Test assertion helpers (use in `*_test.kuki` only) | AssertEqual, AssertNotEqual, AssertTrue, AssertFalse, AssertNoError, AssertError, AssertNotEmpty, AssertNil, AssertNotNil |
| `stdlib/validate` | Input validation | NotEmpty, MinLength, MaxLength, Length, LengthBetween, Matches, Email, URL, Alpha, Alphanumeric, Numeric, NoWhitespace, StartsWith, EndsWith, Contains, OneOf, Positive, Negative, NonNegative, NonZero, InRange, Min, Max, PositiveFloat, InRangeFloat, ParseInt, ParsePositiveInt, ParseFloat, ParseBool, NotEmptyList, ListMinLength, ListMaxLength, WithMessage, Require, NoHTML, SafeFilename, NoNullBytes |

## Testing Stdlib Packages

Use the **table-driven pattern** for all `*_test.kuki` files:

```kukicha
petiole slice_test

import "stdlib/slice"
import "stdlib/test"
import "testing"

type FirstCase
    name    string
    n       int
    wantLen int

func TestFirst(t reference testing.T)
    items := list of string{"a", "b", "c", "d", "e"}
    cases := list of FirstCase{
        FirstCase{name: "3 elements", n: 3, wantLen: 3},
        FirstCase{name: "n > length", n: 10, wantLen: 5},
        FirstCase{name: "n=0", n: 0, wantLen: 0},
    }
    for tc in cases
        t.Run(tc.name, (t reference testing.T) =>
            result := slice.First(items, tc.n)
            test.AssertEqual(t, len(result), tc.wantLen)
        )
```

**Conventions:**
- Case types are declared at file scope, named `<FunctionName>Case`; `name string` is always the first field
- `t.Run(tc.name, (t reference testing.T) => ...)` wraps every assertion body
- Use `test.AssertEqual`, `test.AssertNoError`, `test.AssertError` in preference to bare `t.Errorf`
- A comment `# --- TestFoo ---` separates each function's table from the next
- Import `stdlib/test` only in `*_test.kuki` files, never in library code

## Security Patterns

The compiler enforces several security checks. Use the safe alternatives below to avoid compile errors.

### Security Check Table

| Category | Unsafe (compiler error) | Safe alternative |
|----------|------------------------|------------------|
| **SQL Injection** | `db.Query(pool, "SELECT * FROM t WHERE name = '{name}'")` | `db.Query(pool, "SELECT * FROM t WHERE name = $1", name)` |
| **XSS** | `http.HTML(w, userInput)` | `http.SafeHTML(w, userInput)`, `html.Render()` with `html.Escape()`, or `template.HTMLRenderSimple(...)` |
| **SSRF** | `fetch.Get(url)` (in HTTP handler) | `fetch.SafeGet(url)` |
| **Open Redirect** | `http.Redirect(w, r, userURL)` | `http.SafeRedirect(w, r, url, "example.com")` |
| **Path Traversal** | `files.Read(userInput)` (in HTTP handler) | `sandbox.New("/var/data")` + `sandbox.Read(box, userInput)` |
| **Command Injection** | `shell.Run("git log {branch}")` | `shell.Output("git", "log", branch)` |
| **Inline JS (XSS)** | `html.Render("<script>...</script>")` | Static `.js` file with `<script src="...">` |
| **Inline Event Handler (XSS)** | `html.Render("<button onclick='...'>")` | `addEventListener` in a static `.js` file |

HTTP handler detection: any function with an `http.ResponseWriter` parameter triggers the handler-context checks.

## Import Aliases

When a package's last path segment collides with a local variable name, use `as`. Always use these standard aliases:

| Package | Standard alias | Reason |
|---------|----------------|--------|
| `stdlib/ctx` | `ctxpkg` | Clashes with local `ctx` variable |
| `stdlib/db` | `dbpkg` | Clashes with local `db` variable |
| `stdlib/errors` | `errs` | Clashes with local `err` / `errors` |
| `stdlib/json` | `jsonpkg` | Clashes with `encoding/json` |
| `stdlib/string` | `strpkg` | Clashes with `string` type name |
| `stdlib/container` | `docker` | Clashes with local `container` variables |
| `stdlib/http` | `httphelper` | Clashes with `net/http` |

## Common Pitfalls

Patterns that look correct but introduce subtle bugs.

### WaitGroups in goroutines — use `defer wg.Done()`

Always make `defer wg.Done()` the **first statement** in a goroutine body. An explicit `wg.Done()` at the end is silently skipped if the task panics, hanging `wg.Wait()` forever.

```kukicha
# WRONG — hangs if t() panics
go func()
    t()
    wg.Done()
()

# CORRECT — defer fires even on panic
go func()
    defer wg.Done()
    t()
()
```

### Context/cancel lifetime — defer cancel in the function that owns the resource

Never call `defer ctxpkg.Cancel(h)` inside a helper that *returns* a resource built with that context. The cancel fires when the helper returns — before the resource is ever used — making the timeout dead on arrival. Defer cancel in the same function whose lifetime covers the resource's use.

```kukicha
# WRONG — cancel fires when buildCmd returns, context is already dead
func buildCmd(cmd Command) reference exec.Cmd
    h := ctxpkg.WithTimeout(ctxpkg.Background(), cmd.timeout as int64)
    defer ctxpkg.Cancel(h)   # fires here, before exec.Run
    return exec.CommandContext(ctxpkg.Value(h), cmd.name, many cmd.args)

# CORRECT — cancel deferred in Execute, fires after execCmd.Run() completes
func Execute(cmd Command) Result
    execCmd := exec.Command(cmd.name, many cmd.args)
    if cmd.timeout > 0
        h := ctxpkg.WithTimeout(ctxpkg.Background(), cmd.timeout as int64)
        defer ctxpkg.Cancel(h)   # fires after Run()
        execCmd = exec.CommandContext(ctxpkg.Value(h), cmd.name, many cmd.args)
    ...
```

### io.ReadCloser wrapping — never use `io.NopCloser` on a live body

`io.NopCloser` replaces `Close()` with a no-op. Wrapping a response body with it means `Close()` never reaches the underlying connection, leaking it. When you need to cap reads with `io.LimitReader`, preserve the original closer with a wrapper type that delegates both `Read` and `Close`.

```kukicha
# WRONG — NopCloser silences Close(), TCP connection is never released
resp.Body = io.NopCloser(io.LimitReader(resp.Body, maxSize))

# CORRECT — limitReadCloser delegates Read to LimitReader, Close to original body
type limitReadCloser
    r io.Reader
    c io.Closer

func Read on b reference limitReadCloser (p list of byte) (int, error)
    return b.r.Read(p)

func Close on b reference limitReadCloser () error
    return b.c.Close()

resp.Body = reference of limitReadCloser{r: io.LimitReader(resp.Body, maxSize), c: resp.Body}
```

## Module Structure

Every stdlib module is **pure Kukicha**: `<name>.kuki` source + `<name>.go` generated output. There are no hand-written Go implementation files anywhere in `stdlib/`.

To re-export an external Go type (so callers can use it in type assertions without importing the original package), use a **transparent type alias** in the `.kuki` source:

```kukicha
type TextContent = mcp.TextContent     # re-exports external type; generates: type TextContent = mcp.TextContent
```

This generates Go's `type X = Y` form — the types are identical, enabling cross-package type assertions like `result.(*mcp.TextContent)`.

### WASM-only packages

Some external stdlib packages (e.g., `game`) depend on libraries with native platform requirements. The codegen automatically emits a `//go:build js` constraint for these packages and any user code that imports them.

The list of WASM-only packages is defined in `wasmOnlyPackages` in `internal/codegen/codegen_imports.go`.

### External stdlib packages

`externalStdlibPackages` map in `internal/codegen/codegen_imports.go` maps stdlib names to external module paths:
- `"game"` → `"github.com/kukichalang/game"`
- `"infer"` → `"github.com/kukichalang/infer"`
- `"ort"` → `"github.com/kukichalang/infer/ort"`
- `"webinfer"` → `"github.com/kukichalang/infer/webinfer"`

Registry stubs (`.kuki` only, no `.go`) kept in main repo: `stdlib/game/`, `stdlib/infer/`, `stdlib/ort/`, `stdlib/webinfer/`.

## Vulnerability Auditing

The stdlib is extracted as a standalone Go module (`.kukicha/stdlib/`) with its own `go.mod` (embedded in `cmd/kukicha/stdlib.go`). When users run `kukicha audit` in their project, govulncheck follows the `replace` directive into the extracted stdlib and checks its dependencies transitively.

When updating stdlib dependencies (the `stdlibGoMod` constant in `cmd/kukicha/stdlib.go`), always run `kukicha audit` afterward to verify no new vulnerabilities were introduced.

## Critical Rules

1. **Never edit generated `*.go` files in stdlib** — edit `.kuki` source, then `make generate`
2. **Never edit `internal/semantic/stdlib_registry_gen.go` or `go_stdlib_gen.go`** — both are auto-generated
3. **Types must be defined in `.kuki`** — so the Kukicha compiler knows about them
4. **After adding an exported function or enum to a stdlib `.kuki` file**, run `make genstdlibregistry` so `onerr`, pipe expressions, and cross-package enum resolution work correctly
5. **To deprecate a stdlib function**, add `# kuki:deprecated "Use NewFunc instead"` above it in the `.kuki` source, then run `make genstdlibregistry`
6. **To mark a function as security-sensitive**, add `# kuki:security "category"` above it, then run `make genstdlibregistry`
7. **Every stdlib package must have a `*_test.kuki` file** using the table-driven pattern
8. **`stdlib/test` is test-only** — import it only in `*_test.kuki` files, never in library `.kuki` files
