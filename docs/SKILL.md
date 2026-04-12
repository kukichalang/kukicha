## Writing Kukicha

Kukicha is a strict superset of Go — all valid Go compiles as-is. Kukicha adds
pipes, `onerr`, enums, if-expressions, and readable operators on top. You can
use Go syntax or Kukicha syntax (or mix both) in `.kuki` files.

### Project Structure

**Single-file programs:** 

```kukicha
# hello.kuki — minimal working program
import "stdlib/string"

function main()
    name := "world"
    print("Hello {string.ToUpper(name)}!")
```

Run with: `kukicha run hello.kuki`

**Multi-file packages:** Build a directory with `kukicha build myapp/`. All `.kuki` files in the directory are merged into a single package. Rules:

- Exactly **one file** must define `func main()` — the entry point
- Other files use `func init()` for startup code (Go allows multiple `init` functions)
- All files must have the same `petiole` declaration (or all omit it)
- Imports are deduplicated across files
- Duplicate function names (except `init`) are rejected at compile time

```kukicha
# main.kuki — entry point
func main()
    startServer()

# routes.kuki — helper file in same directory
func init()
    print("routes loaded")

func startServer()
    print("listening on :8080")
```

Build with: `kukicha build myapp/` — merges all `.kuki` files into `myapp/main.go` and compiles.

### Syntax vs Go

| Kukicha | Go |
|---------|-----|
| `and`, `or`, `not` | `&&`, `\|\|`, `!` |
| `equals` | `==` |
| `empty` | `nil` |
| `list of string` (or `[]string`) | `[]string` |
| `map of string to int` (or `map[string]int`) | `map[string]int` |
| `reference User` | `*User` |
| `reference of x` | `&x` |
| `dereference ptr` | `*ptr` |
| `func Method on t T` (or `func (t T) Method()`) | `func (t T) Method()` |
| `many args` | `args...` |
| `make channel of T` | `make(chan T)` |
| `send val to ch` / `receive from ch` | `ch <- val` / `<-ch` |
| `defer f()` | `defer f()` (same keyword) |
| 4-space indentation | `{ }` braces |

### Keyword Aliases

`func`, `var`, and `const` have English-word aliases that compile identically:

```kukicha
func Add(a int, b int) int       # idiomatic
function Add(a int, b int) int   # beginner-friendly — same output

var AppName string = "myapp"
variable AppName string = "myapp"

const MaxRetries = 5
constant MaxRetries = 5
```

When generating beginner-facing code, prefer `function`, `variable`, and `constant`.

### Variables and Functions

```kukicha
count := 42           # inferred type
count = 100           # reassignment

func Add(a int, b int) int
    return a + b

func Divide(a int, b int) int, error
    if b equals 0
        return 0, error "division by zero"
    return a / b, empty

# Default parameter value
func Greet(name string, greeting string = "Hello") string
    return "{greeting}, {name}!"

# Named argument at call site
result := Greet("Alice", greeting: "Hi")
files.Copy(from: src, to: dst)
```

### Number Literals

```kukicha
count := 42              # decimal
mask := 0xFF             # hexadecimal (0x or 0X)
perms := 0o755           # octal (0o or 0O)
flags := 0b1010          # binary (0b or 0B)
legacy := 0755           # legacy octal (also supported)
pi := 3.14               # float
```

### Strings and Interpolation

```kukicha
greeting := "Hello {name}!"          # {expr} is interpolated
json := "key: \{value\}"             # \{ and \} produce literal braces
path := "{dir}\sep{file}"            # \sep → OS path separator at runtime

# Raw strings (backticks) — no escapes, no interpolation. Best for prompts,
# SQL, regex, or JSON templates that contain lots of literal braces:
prompt := `Reply JSON: {severity:1-5, kind, summary}`

# Escape sequences: \n \t \r \\ \" \' \xHH (hex) \0-\377 (octal)
esc  := "\033[0m"                    # octal escape (ESC character)
byte := "\x1b[31m"                   # hex escape (same ESC character)

# Interpolation converts any value to a string — replaces fmt.Sprintf
count := 42
label := "{count}"                   # "42" — no fmt.Sprintf("%d", count) needed
price := 9.99
msg   := "costs {price}"             # "costs 9.99"
```

### Types

```kukicha
type Repo
    name  string as "name"            # JSON field alias
    stars int    as "stargazers_count"
    tags  list of string
    meta  map of string to string
```

### Enums

```kukicha
# Integer enum
enum Status
    OK = 200
    NotFound = 404
    Error = 500

# String enum
enum LogLevel
    Debug = "debug"
    Info = "info"
    Warn = "warn"

# Dot access
status := Status.OK

# Exhaustiveness-checked switch
switch status
    when Status.OK
        print("ok")
    when Status.NotFound, Status.Error
        print("problem")
```

- Underlying type (int or string) inferred from values — all must match
- `Status.OK` transpiles to Go `StatusOK`
- Compiler warns if switch on enum misses cases (unless `otherwise` present)
- Integer enums warn if no case has value 0
- Auto-generated `String()` method (skipped if user defines one)

### Variant Enums (Tagged Unions)

```kukicha
enum Shape
    Circle
        radius float64
    Rectangle
        width  float64
        height float64
    Point

# Pattern matching with exhaustiveness checking
func area(s Shape) float64
    switch s as v
        when Circle
            return 3.14159 * v.radius * v.radius
        when Rectangle
            return v.width * v.height
        when Point
            return 0.0
```

- Cases without `=` are variant cases (data-carrying or unit)
- Each case becomes a Go struct implementing a sealed interface
- Variant cases are assignable to the parent enum type (struct fields, map values, function args)
- `switch s as v` + `when CaseName` for pattern matching
- Compiler warns if switch misses variant cases (unless `otherwise` present)
- Cannot mix value cases (`= literal`) and variant cases in the same enum

#### Single-case checks with `is`

Use the `is` operator to test a single variant case without a full switch:

```kukicha
# Bool check — works in any expression position
if s is Circle
    return true

# With binding — `c` is typed as the matched case in the consequence block
func area(s Shape) float64
    if s is Circle as c
        return 3.14159 * c.radius * c.radius
    if s is Rectangle as r
        return r.width * r.height
    return 0.0
```

- `EXPR is CaseName` evaluates to `bool`
- `EXPR is CaseName as v` in an `if` condition binds `v` to the case's struct
  in the consequence block (scoped; not visible in `else`)
- The binding form is only valid as the **top-level** `if` condition — not
  nested inside `and`/`or` or used in other positions
- Left-hand side must be a variant enum value; case name must belong to that enum

### Methods

```kukicha
func Display on todo Todo string
    return "{todo.id}: {todo.title}"

func SetDone on todo reference Todo       # pointer receiver
    todo.done = true
```

### Error Handling (`onerr`)

The caught error is always `{error}` — never `{err}`. Using `{err}` is a compile-time error. To use a custom name in a block handler, write `onerr as e`.

```kukicha
data := fetch.Get(url) onerr panic "failed: {error}"        # stop with message
data := fetch.Get(url) onerr return                         # propagate (shorthand — raw error, zero values)
data := fetch.Get(url) onerr return empty, error "{error}"  # propagate (verbose, wraps error)
data := fetch.Get(url) onerr return {}, error "{error}"     # propagate with untyped zero struct
port := getPort()      onerr 8080                           # default value
_    := riskyOp()      onerr discard                        # ignore
v    := parse(item)    onerr continue                       # skip iteration (inside for loop)
v    := parse(item)    onerr break                          # exit loop (inside for loop)
data := fetch.Get(url) onerr explain "context hint"         # wrap and propagate

# Block form — multiple statements
users := parse() onerr
    print("failed: {error}")
    return

# Block form with named alias
users := parse() onerr as e
    print("failed: {e}")    # {e} and {error} both work
    return
```

### Pipes

```kukicha
result := data |> parse() |> transform()

# _ placeholder: pipe into a non-first argument position
todo |> json.MarshalWrite(w, _)   # → json.MarshalWrite(w, todo)

# Bare identifier as target
data |> print                     # → fmt.Println(data)

# Pipeline-level onerr — catches errors from any step
items := fetch.Get(url)
    |> fetch.CheckStatus()
    |> fetch.Json(list of Repo)
    onerr panic "{error}"

# Piped switch — pipe a value into a switch
user.Role |> switch
    when "admin"
        grantAccess()
    when "guest"
        denyAccess()
    otherwise
        checkPermissions()
```

### Control Flow

```kukicha
if count equals 0
    return "empty"
else if count < 10
    return "small"

for item in items
    process(item)

for i from 0 to 10        # 0..9 (exclusive)
for i from 0 through 10   # 0..10 (inclusive)
for i from 10 through 0   # 10..0 (descending)

for                        # bare loop (infinite — use break to exit)
    msg := receive from ch
    if msg equals "quit"
        break
    process(msg)

# If with init statement
if val, ok := cache[key]; ok
    return val

switch command
    when "fetch", "pull"
        fetchRepos()
    when "help"
        showHelp()
    otherwise
        print("Unknown: {command}")

# Bare switch (condition-based)
switch
    when stars >= 1000
        print("popular")
    otherwise
        print("new")

# Type switch
switch event as e
    when string
        print(e)
    when reference TaskEvent
        print(e.Status)
    otherwise
        print("unknown")
```

### Lambdas

Lambda parameter types are inferred from calling context — explicit annotations are optional.

```kukicha
# Inferred param type (preferred) — compiler resolves type from the list being piped
repos   |> slice.Filter(r => r.stars > 100)
entries |> sort.ByKey(e => e.name)

# Explicit param type (optional — only needed when inference can't determine the type)
repos |> slice.Filter((r Repo) => r.stars > 100)

# Zero params
button.OnClick(() => print("clicked"))

# Block lambda (multi-statement, explicit return)
repos |> slice.Filter(r =>
    name := r.name |> strpkg.ToLower()
    return name |> strpkg.Contains("go")
)

# sort.By — two params, both inferred
repos |> sort.By((a, b) => a.stars < b.stars)
```

### Collections

```kukicha
items  := list of string{"a", "b", "c"}
config := map of string to int{"port": 8080}
last   := items[-1]    # negative indexing
```

### Untyped Composite Literals

When the expected type is known from context, composite literal types can be omitted:

```kukicha
type Config
    host string
    port int

func makeConfig() Config
    return {host: "localhost", port: 8080}    # type inferred from return type

func applyConfig(c Config)
    print("{c.host}:{c.port}")

func main()
    applyConfig({host: "prod", port: 443})    # type inferred from parameter

    configs := list of Config{
        {host: "a", port: 1},                 # type inferred from list element type
        {host: "b", port: 2},
    }
```

Supported inference contexts:
- **Return statements** — type from function return signature
- **`onerr return`** — type from enclosing function's return signature
- **Function arguments** — type from parameter signature
- **Assignments** — type from left-hand side variable
- **Typed list elements** — type from `list of T`

Keyed `{key: val}` resolves to struct or map depending on the expected type.
Positional `{1, 2, 3}` resolves to a slice.
Empty `{}` without context defaults to `map[any]any{}`.

### Variadic Arguments (`many`)

```kukicha
func Sum(many numbers int) int
    total := 0
    for n in numbers
        total = total + n
    return total

result := Sum(1, 2, 3)
nums := list of int{1, 2, 3}
result := Sum(many nums)              # spread a slice
```

### Type Casts and Assertions

```kukicha
n := x as int                        # type conversion
f := n as float64
data = append(data, "\n" as byte)    # byte cast (emits rune literal)
result, ok := value.(string)          # safe type assertion (two-value)
s := value.(string)                   # panics if wrong type
```

### Multi-Value Destructuring

```kukicha
data, err := os.ReadFile(path)                  # 2-value
_, ipNet, err := net.ParseCIDR("192.168.0.0/16") # 3-value
```

### Concurrency

```kukicha
ch := make channel of string
send "message" to ch
msg := receive from ch
go doWork()

# Multi-statement goroutine
go
    mu.Lock()
    doWork()
    mu.Unlock()

# Select (channel multiplexing)
select
    when receive from done
        return
    when msg := receive from ch
        print(msg)
    when send "ping" to out
        print("sent")
    otherwise
        print("nothing ready")
```

### Defer

```kukicha
defer resource.Close()                # single call — runs when function exits

# Block form — multiple statements (emits defer func() { ... }())
defer
    if r := recover(); r != empty
        tx.Rollback()
        panic(r)
```

### Imports and Canonical Aliases

```kukicha
import "stdlib/slice"
import "stdlib/ctx"       as ctxpkg     # clashes with local 'ctx' variable
import "stdlib/errors"    as errs       # clashes with local 'err' / 'errors'
import "stdlib/json"      as jsonpkg    # clashes with 'encoding/json'
import "stdlib/string"    as strpkg     # clashes with 'string' type name
import "stdlib/container" as docker     # clashes with local 'container' variables
import "stdlib/http"      as httphelper # clashes with 'net/http'
import "stdlib/net"       as netutil    # clashes with 'net' package

import "github.com/jackc/pgx/v5" as pgx  # external package
```

Always use these aliases — clashes cause compile errors.

### Project Structure and Commands

```kukicha
petiole main                   # package declaration (Go's `package main`)
```

`petiole` is optional for single-file programs but required for multi-file packages and tests.

```bash
kukicha init [module]          # initialize project (go mod init + extract stdlib)
kukicha check file.kuki        # validate syntax without compiling
kukicha check --json file.kuki # structured JSON diagnostics
kukicha check a.kuki myapp/    # check multiple targets
kukicha run file.kuki          # transpile, compile, and run
kukicha build file.kuki        # transpile and compile to binary (named after source file stem)
kukicha build myapp/           # build directory — binary named after the directory
kukicha build .                # build current directory — binary named after working dir
kukicha fmt -w file.kuki       # format in place
kukicha fmt --check dir/       # check formatting without modifying (exit 1 if unformatted)
kukicha brew file.kuki         # convert .kuki to standalone Go (no header, no //line)
kukicha brew --stdout file.kuki  # print brewed Go to stdout
kukicha brew --remove-kuki dir/  # brew + delete .kuki sources (with confirmation)
kukicha pack skill.kuki        # package skill into directory with SKILL.md + binary
kukicha audit                  # check dependencies for known vulnerabilities
```

**Go → Kukicha conversion** (separate `kukicha-blend` binary):

```bash
kukicha-blend main.go                     # show Kukicha suggestions for Go code
kukicha-blend --diff ./pkg/               # preview changes as unified diff
kukicha-blend --apply main.go             # convert main.go → main.kuki
kukicha-blend --patterns=onerr main.go    # only error handling suggestions
kukicha-blend --patterns=operators,types main.go  # selective patterns
```

Available patterns: `operators` (`&&`→`and`, `||`→`or`, `!`→`not`), `comparisons` (`==`→`equals`, `!=`→`isnt`, `nil`→`empty`), `types` (`[]T`→`list of T`, `map[K]V`→`map of K to V`, `*T`→`reference T`), `onerr` (`if err != nil { return }` → `onerr return`), `package` (`package`→`petiole`).

**Build flags:**

```bash
kukicha build --wasm file.kuki                # WebAssembly output (GOOS=js GOARCH=wasm)
kukicha build --vulncheck file.kuki           # build + check for known vulnerabilities
kukicha build --no-line-directives file.kuki  # omit //line directives (cleaner output for production)
```

**Binary output location and name:**

The binary is placed in the current working directory, matching `go build` behavior. The name comes from:

- `kukicha build hello.kuki` → `./hello` (stem of the `.kuki` file)
- `kukicha build myapp/` → `./myapp` (base name of the directory)
- `kukicha build .` → `./myproject` (base name of the working directory)

If the binary name collides with an existing directory (e.g. `kukicha build deploy/` when `deploy/` exists in the cwd), the binary is placed inside the directory instead: `deploy/deploy`.

On Windows the binary gets a `.exe` suffix automatically. WASM builds produce a `.wasm` file instead.

---

### Stdlib Packages

#### Core & Collections

**stdlib/slice** — List operations

```kukicha
active  := slice.Filter(items, x => x.active)
names   := slice.Map(items, x => x.name)
byGroup := slice.GroupBy(items, x => x.category)
first   := slice.FirstOr(items, defaultVal)
val     := slice.GetOr(items, 0, defaultVal)
ok      := slice.Contains(items, value)
```

**stdlib/maps** — Map utilities

```kukicha
keys := maps.Keys(config)
vals := maps.Values(config)
ok   := maps.Contains(config, "port")

# Functional operations
active := maps.Filter(users, (k, v) => v != empty)
upper  := maps.MapValues(labels, v => strings.ToUpper(v as string))
subset := maps.Pick(config, list of any{"host" as any, "port" as any})
safe   := maps.Omit(config, list of any{"password" as any})
```

**stdlib/set** — Generic set operations (backed by `map[K]bool`)

```kukicha
import "stdlib/set"

s := set.From(list of string{"a", "b", "c"})
s2 := set.Add(s, "d")
set.AddIn(s, "d")            # in-place
ok := set.Contains(s, "a")

u := set.Union(s1, s2)
i := set.Intersect(s1, s2)
d := set.Difference(s1, s2)  # s1 minus s2
ok := set.IsSubset(small, large)
ok := set.Equal(s1, s2)
items := set.ToSlice(s)
```

**stdlib/sort** — Sorting slices (returns sorted copies, originals unchanged)

```kukicha
sorted := sort.Strings(names)                          # ascending, lexicographic
sorted := sort.Ints(scores)                            # ascending, numeric
sorted := sort.Float64s(values)                        # ascending, float64

# Custom comparator (stable sort)
sorted := sort.By(repos, (a, b) => a.stars < b.stars)

# Sort by extracted key (pipe-friendly)
sorted := repos |> sort.ByKey(r => r.name)

# Reverse sort
sorted := sort.Reverse(repos, (a, b) => a.stars < b.stars)
```

**stdlib/iterator** — Lazy iteration (Go 1.23 iter.Seq)

```kukicha
import "stdlib/iterator"
names := repos
    |> iterator.Values()
    |> iterator.Filter((r Repo) => r.Stars > 100)
    |> iterator.Map((r Repo) => r.Name)
    |> iterator.Take(5)
    |> iterator.Collect()
```

Functions: `Values`, `Filter`, `Map`, `FlatMap`, `Take`, `Skip`, `Enumerate`, `Chunk`, `Zip`, `Reduce`, `Collect`, `Any`, `All`, `Find`.

**stdlib/string** (import as `strpkg`) — String utilities

```kukicha
import "stdlib/string" as strpkg
parts  := strpkg.Split(line, ",")
joined := strpkg.Join(parts, " | ")
lower  := strpkg.ToLower(name)
ok     := strpkg.Contains(text, "TODO")
clean  := strpkg.Replace(raw, "\t", " ")
```

**stdlib/regex** — Regular expressions

```kukicha
if regex.Match("\\d+", text)
    print("contains a number")

groups := regex.FindGroups("^(v?)(\\d+\\.\\d+\\.\\d+)$", tag) onerr panic "{error}"
cleaned := regex.Replace("\\s+", " ", messy)
parts   := regex.Split(",\\s*", line)

# Compiled patterns for hot paths
p    := regex.MustCompile("\\d+")
nums := regex.FindAllCompiled(p, "a1 b2 c3")
```

**stdlib/cast** — Type coercion

```kukicha
n := cast.SmartInt(value) onerr 0
s := cast.SmartString(value) onerr ""
```

#### Data & Encoding

**stdlib/json** (import as `jsonpkg`) — JSON encode/decode

```kukicha
import "stdlib/json" as jsonpkg
data   := jsonpkg.Marshal(value)                    onerr panic "{error}"
result := jsonpkg.Unmarshal(data, empty Repo)       onerr panic "{error}"
         jsonpkg.UnmarshalString(str, reference of v) onerr panic "{error}"
```

**stdlib/parse** — Data parsing

```kukicha
rows := csvData  |> parse.CsvWithHeader() onerr panic "{error}"
cfg  := yamlData |> parse.Yaml()          onerr panic "{error}"
```

**stdlib/encoding** — Base64 and hex

```kukicha
encoded := encoding.Base64Encode(data)
decoded := encoding.Base64Decode(encoded) onerr panic "{error}"
hex     := encoding.HexEncode(hashBytes)
```

**stdlib/template** — Templating

```kukicha
# text/template (no HTML escaping — for plain text only)
result := template.RenderSimple(src, data) onerr return

# html/template (auto-escapes {{ }} values — use for HTML responses)
html := template.HTMLRenderSimple(tmplStr, map of string to any{"name": username}) onerr return
```

#### I/O & Files

**stdlib/files** — File I/O

```kukicha
data := files.Read("path.txt")        onerr panic "{error}"
text := files.ReadString("cfg.json")  onerr panic "{error}"
       files.Write(data, "out.txt")   onerr panic "{error}"
       files.Append(line, "log.txt")  onerr discard
ok   := files.Exists("path.txt")
       files.Copy(from: src, to: dst) onerr panic "{error}"
```

**stdlib/sandbox** — Filesystem sandboxing (use in HTTP handlers)

```kukicha
box     := sandbox.New("/var/data") onerr return
content := sandbox.Read(box, userPath) onerr return   # can't escape root
sandbox.Write(box, "out.txt", data) onerr return
```

**stdlib/shell** — Run commands

```kukicha
# Run: for fixed string literals only (no variable interpolation)
diff := shell.Run("git diff --staged") onerr panic "{error}"

# Output: use when any argument is a variable — args passed directly to OS
out := shell.Output("git", "log", "--oneline", userBranch) onerr panic "{error}"

# Builder: add working directory, env vars, timeout
result := shell.New("npm", "test") |> shell.Dir(projectPath) |> shell.Env("CI", "true") |> shell.Execute()
if not shell.Success(result)
    print(shell.GetError(result) as string)
```

#### Networking & HTTP

**stdlib/fetch** — HTTP requests

```kukicha
# Simple GET with typed JSON decode
repos := fetch.Get(url)
    |> fetch.CheckStatus()
    |> fetch.Json(list of Repo) onerr panic "{error}"

# fetch.Json sample arg tells the compiler what to decode into:
#   fetch.Json(list of Repo)            → JSON array  → []Repo
#   fetch.Json(empty Repo)              → JSON object → Repo
#   fetch.Json(map of string to string) → JSON object → map[string]string

# Builder: auth, timeout, retry
resp := fetch.New(url)
    |> fetch.BearerAuth(token)
    |> fetch.Retry(3, 500)
    |> fetch.Do() onerr panic "{error}"
text := fetch.Text(resp) onerr panic "{error}"

# SSRF-protected GET — use inside HTTP handlers or server code
resp := fetch.SafeGet(url) onerr panic "{error}"

# Cap response body size (prevent OOM)
resp := fetch.New(url) |> fetch.MaxBodySize(1 << 20) |> fetch.Do() onerr panic "{error}"

# Safe URL construction
url := fetch.URLTemplate("https://api.example.com/users/{id}",
    map of string to string{"id": userID}) onerr panic "{error}"
url  = fetch.URLWithQuery(url,
    map of string to string{"per_page": "30"}) onerr panic "{error}"
```

**stdlib/http** (`import "stdlib/http" as httphelper`) — HTTP helpers + security

```kukicha
httphelper.JSON(w, data)                        # 200 OK with JSON body
httphelper.JSONCreated(w, data)                 # 201 Created
httphelper.JSONNotFound(w, "not found")         # 404
httphelper.JSONBadRequest(w, "bad input")       # 400
httphelper.JSONError(w, "server error", 500)    # any status

httphelper.ReadJSONLimit(r, 1<<20, reference of input) onerr return   # parse + size cap
httphelper.SafeHTML(w, userContent)             # HTML-escape before write
httphelper.SafeRedirect(w, r, url, "myapp.com") onerr return  # host-allowlist redirect
httphelper.SetSecureHeaders(w)                  # per-handler security headers
http.ListenAndServe(":8080", httphelper.SecureHeaders(mux))   # middleware form
```

**stdlib/html** — Component-style HTML rendering with auto-escaping

```kukicha
# Render a fragment — use Escape() for user input, Embed() for child fragments
page := html.Render("<h1>{html.Escape(title)}</h1>")
nav  := html.Render("<nav>{html.Embed(links)}</nav>")

# Write to HTTP response
html.WriteTo(w, page) onerr discard
html.WriteStatusTo(w, errorPage, 404) onerr discard

# Compose fragments
full := html.Join(header, content, footer)

# Render a list
items := html.Map(users, (u User) =>
    return html.Render("<li>{html.Escape(u.Name)}</li>")
)

# Conditional rendering
badge := html.When(isAdmin, adminBadge)
nav   := html.WhenElse(loggedIn, userNav, guestNav)

# Attribute escaping
link := html.Render("<a href='{html.Attr(url)}'>click</a>")
```

**stdlib/net** (import as `netutil`) — IP/CIDR utilities

```kukicha
import "stdlib/net" as netutil
ip      := netutil.ParseIP("192.168.1.100")
network := netutil.ParseCIDR("192.168.0.0/16") onerr panic "{error}"
if netutil.Contains(network, ip) and netutil.IsPrivate(ip)
    print("private range")
```

**stdlib/netguard** — SSRF protection and network restriction

```kukicha
# Block all private/reserved IPs (standard SSRF protection)
guard := netguard.NewSSRFGuard()
client := netguard.HTTPClient(guard)

# Allow only specific CIDRs
guard := netguard.NewAllow(list of string{"93.184.216.0/24"}) onerr panic "{error}"

# Block specific CIDRs
guard := netguard.NewBlock(list of string{"10.0.0.0/8"}) onerr panic "{error}"

# Check a single IP
if netguard.Check(guard, "10.0.0.1")
    print("allowed")

# Use with fetch via guarded HTTP transport
transport := netguard.HTTPTransport(guard)
```

#### CLI & System

**stdlib/cli** — Command-line apps

```kukicha
func run(args cli.Args)
    name := cli.GetString(args, "name")
    port := cli.GetInt(args, "port")

func main()
    app := cli.New("myapp")
        |> cli.AddFlag("name", "Your name", "world")
        |> cli.AddFlag("port", "Port number", "8080")
        |> cli.Action(run)
    cli.RunApp(app) onerr panic "{error}"

# Fatal: print to stderr and exit 1 — ideal for onerr handlers
data := loadConfig() onerr cli.Fatal("config error: {error}")
stack := initStack(ctx) onerr cli.Fatal("init failed: {error}")
```

```kukicha
# Print to stderr (no exit — use for warnings, debug, non-fatal errors)
cli.Error("connection dropped: {error}")

# Prefixed warning
cli.Warn("disk space low")   # prints "warning: disk space low" to stderr
```

Use `cli.Fatal(msg)` when you want to print to stderr *and* exit. Use `cli.Error(msg)` when you want to print to stderr and keep running.

**stdlib/input** — Interactive CLI input

```kukicha
name := input.ReadLine("Enter name: ") onerr return
name := input.Prompt("Enter name: ")            # panics on stdin failure
ok   := input.Confirm("Proceed?") onerr return
idx  := input.Choose("Select:", options) onerr return
```

**stdlib/table** — Terminal tables (plain, box, markdown)

```kukicha
tbl := table.New(list of string{"Name", "Stars"})
tbl  = tbl |> table.AddRow(list of string{"go", "115000"})
tbl  = tbl |> table.AddRow(list of string{"rust", "97000"})
table.Print(tbl)                        # plain output
table.PrintWithStyle(tbl, "markdown")   # markdown table
table.PrintWithStyle(tbl, "box")        # box-drawing style
```

**stdlib/must** and **stdlib/env** — Config

```kukicha
import "stdlib/must"
# must: panics at startup if env var is missing (use for required config)
apiKey := must.Env("API_KEY")
port   := must.EnvIntOr("PORT", 8080)

import "stdlib/env"
# env: returns error via onerr (use for optional or runtime config)
debug  := env.GetBool("DEBUG") onerr false
token  := env.Get("TOKEN")     onerr panic "TOKEN required"
```

#### Concurrency & Resilience

**stdlib/concurrent** — Parallel execution and concurrent map

```kukicha
# Run zero-argument functions concurrently, wait for all to finish
concurrent.Parallel(
    () => processChunk(chunkA),
    () => processChunk(chunkB),
)

# Same with a concurrency limit
concurrent.ParallelWithLimit(4,
    () => processChunk(chunkA),
    () => processChunk(chunkB),
)

# Transform every element in parallel, results in original order
results := concurrent.Map(urls, url => check(url))

# Same with a concurrency cap (useful for rate-limited APIs)
results := concurrent.MapWithLimit(repos, 4, r => fetchDetails(r))
```

**stdlib/ctx** (import as `ctxpkg`) — Context helpers

```kukicha
import "stdlib/ctx" as ctxpkg
c := ctxpkg.Background() |> ctxpkg.WithTimeout(30)
defer ctxpkg.Cancel(c)
```

**stdlib/retry** — Retry with backoff

```kukicha
cfg := retry.New() |> retry.Attempts(5) |> retry.Delay(200)
for attempt from 0 to cfg.MaxAttempts
    result, err := doWork()
    if err equals empty
        break
    retry.Sleep(cfg, attempt)
```

**stdlib/datetime** — Time formatting and durations

```kukicha
formatted := datetime.Format(t, "iso8601")     # not Go's "2006-01-02"!
timeout   := datetime.Seconds(30)
```

#### Data & Storage

**stdlib/db** — SQL database (raw SQL + struct scanning)

```kukicha
pool := db.Open("postgres", "postgres://localhost/mydb") onerr panic "{error}"
defer db.Close(pool)

# Query + typed scanning
rows := db.Query(pool, "SELECT id, name, email FROM users WHERE active = $1", true) onerr panic "{error}"
users := db.ScanAll(rows, list of User{}) onerr panic "{error}"

# Single row
row := db.QueryRow(pool, "SELECT id, name FROM users WHERE id = $1", userID)
user := db.ScanRow(row, User{}) onerr panic "{error}"

# INSERT/UPDATE/DELETE
affected := db.Exec(pool, "DELETE FROM sessions WHERE expired < $1", cutoff) onerr panic "{error}"

# Transactions (auto-commit on success, rollback on error)
db.Transaction(pool, transferFunds) onerr panic "transfer failed: {error}"

# Convenience
n     := db.Count(pool, "SELECT COUNT(*) FROM users") onerr panic "{error}"
found := db.Exists(pool, "SELECT 1 FROM users WHERE email = $1", email) onerr panic "{error}"
```

**stdlib/sqlite** — SQLite convenience layer (WAL, foreign keys, busy timeout by default)

```kukicha
import "stdlib/db"
import "stdlib/sqlite"

# Open with sensible defaults (WAL + foreign_keys=ON + busy_timeout=5000)
pool := sqlite.Open("/tmp/app.db") onerr panic "{error}"
defer db.Close(pool)

# In-memory (foreign keys enabled)
pool := sqlite.OpenMemory() onerr panic "{error}"

# Custom pragmas
pool := sqlite.OpenWith("/tmp/app.db", map of string to string{
    "cache_size": "-64000",
    "journal_mode": "WAL",
}) onerr panic "{error}"

# All queries use stdlib/db
db.Exec(pool, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)") onerr panic "{error}"
db.Exec(pool, "INSERT INTO users (name) VALUES (?)", "Alice") onerr panic "{error}"
users := db.Query(pool, "SELECT id, name FROM users")
    |> db.ScanAll(list of User{})
    onerr panic "{error}"

# Pragmas (injection-safe name and value validation)
mode := sqlite.Pragma(pool, "journal_mode") onerr panic "{error}"
sqlite.SetPragma(pool, "cache_size", "-64000") onerr panic "{error}"

# Utilities
tables  := sqlite.Tables(pool) onerr panic "{error}"
exists  := sqlite.TableExists(pool, "users") onerr panic "{error}"
sqlite.IntegrityCheck(pool) onerr panic "corrupt: {error}"
sqlite.Vacuum(pool) onerr panic "{error}"
sqlite.Backup(pool, "/tmp/backup.db") onerr panic "{error}"
v := sqlite.Version(pool) onerr panic "{error}"

# Batch insert (single transaction)
rows := list of list of any{list of any{"Alice"}, list of any{"Bob"}}
n := sqlite.BatchExec(pool, "INSERT INTO users (name) VALUES (?)", rows) onerr panic "{error}"

# Dump as SQL text
sql := sqlite.Dump(pool) onerr panic "{error}"
```

#### Security & Crypto

**stdlib/crypto** — Hashing, HMAC, and secure random generation

```kukicha
hash  := crypto.SHA256("hello world")                # hex-encoded SHA-256
mac   := crypto.HMAC("secret-key", "message-body")   # hex-encoded HMAC-SHA256
token := crypto.RandomToken(32) onerr panic "{error}" # 64-char hex string
bytes := crypto.RandomBytes(16) onerr panic "{error}" # raw random bytes

# Binary variants for byte-level pipelines
raw   := crypto.SHA256Bytes(data)
rawMac := crypto.HMACBytes(keyBytes, dataBytes)

# Constant-time comparison (prevents timing attacks)
if crypto.Equal(expected, actual)
    print("match")
```

**stdlib/validate** — Input validation

```kukicha
email |> validate.Email()          onerr return
age   |> validate.InRange(18, 120) onerr return
name  |> validate.NotEmpty()       onerr return
```

**stdlib/random** — Random generation

```kukicha
token := random.String(32)
dice  := random.Int(1, 7)        # [1, 7) — a d6 roll
angle := random.Float(0.0, 360.0)
```

**stdlib/errors** (import as `errs`) — Error wrapping

```kukicha
import "stdlib/errors" as errs
wrapped := errs.Wrap(err, "loading config")   # preserves Is/As chain
opaque  := errs.Opaque(originalErr, "db connect")   # breaks chain at boundaries
if errs.Is(err, io.EOF)
    return
# Dual-message: internal detail + user-safe message
e := errs.NewPublic("db: refused 10.0.0.1", "database unavailable")
print(errs.Public(e))   # "database unavailable"
```

#### DevOps & Infrastructure

**stdlib/container** (import as `docker`) — Docker/Podman client

```kukicha
import "stdlib/container" as docker
engine := docker.Connect() onerr panic "{error}"
defer docker.Close(engine)

images := engine |> docker.ListImages() onerr panic "{error}"
docker.Pull(engine, "alpine:latest") onerr panic "{error}"
id   := docker.Run(engine, "alpine:latest", list of string{"echo", "hello"}) onerr panic "{error}"
logs := docker.Logs(engine, id) onerr panic "{error}"
code := docker.Wait(engine, id, 60) onerr panic "{error}"
docker.Remove(engine, id) onerr discard
```

**stdlib/git** — Git/GitHub operations (requires `gh` CLI)

```kukicha
tags   := git.ListTags("owner/repo") onerr panic "{error}"
branch := git.DefaultBranch("owner/repo") onerr panic "{error}"
me     := git.CurrentUser() onerr panic "{error}"
exists, _ := git.TagExists("owner/repo", "v1.0.0")

# Create a release
opts := git.ReleaseOptions{Title: "v1.0.0", Target: "main", Draft: true, GenerateNotes: true}
git.CreateRelease("owner/repo", "v1.0.0", opts) onerr panic "{error}"

# Dry-run: preview command without executing
print("Would run: {git.PreviewRelease("owner/repo", "v1.0.0", opts)}")
```

**stdlib/semver** — Semantic versioning

```kukicha
v    := semver.Parse("v1.2.3") onerr panic "{error}"
next := v |> semver.Bump("minor") |> semver.Format()   # "v1.3.0"
best := semver.Highest(tags) onerr panic "{error}"
if semver.Valid("v2.0.0")
    print("valid")
```

**stdlib/obs** — Structured logging

```kukicha
logger := obs.New("myapp", "prod") |> obs.Component("worker")
logger |> obs.Info("starting", map of string to any{"job": "build"})
logger |> obs.Error("failed",  map of string to any{"err": err})
```

#### AI & Agents

**stdlib/llm** — LLM calls

```kukicha
# OpenAI-compatible
reply := llm.New("openai:gpt-4o-mini")
    |> llm.Retry(3, 2000)
    |> llm.Ask("Hello!") onerr panic "{error}"

# Anthropic
reply := llm.NewMessages("claude-opus-4-6")
    |> llm.MRetry(3, 2000)
    |> llm.MAsk("Summarize this") onerr panic "{error}"
```

**stdlib/mcp** — MCP server and client

```kukicha
# Server
server := mcp.New("stock-tool", "1.0.0")
schema := mcp.Schema(list of mcp.SchemaProperty{
    mcp.Prop("symbol", "string", "Ticker symbol"),
}) |> mcp.Required(list of string{"symbol"})
mcp.Tool(server, "get_price", "Get stock price by ticker", schema, handler)
mcp.Serve(server) onerr panic "{error}"

# Client
session := mcp.Connect(ctx, "http://localhost:8000/mcp") onerr panic "{error}"
defer mcp.Close(session)
tools := mcp.ListTools(ctx, session) onerr panic "{error}"
result := mcp.CallTool(ctx, session, "get_price", args) onerr panic "{error}"
print(result.Text)

# Client with Bearer token authentication
session := mcp.BearerConnect(ctx, "https://mcp.example.com/mcp", apiKey) onerr panic "{error}"
```

---

### Security — Compiler-Enforced Checks

The compiler **rejects** these patterns as errors (not warnings):

| Pattern | Error | Fix |
|---------|-------|-----|
| `httphelper.HTML(w, nonLiteral)` | XSS risk | `httphelper.SafeHTML(w, content)` |
| `fetch.Get(url)` in HTTP handler | SSRF risk | `fetch.SafeGet(url)` |
| `files.Read(path)` in HTTP handler | Path traversal | `sandbox.Read(box, path)` |
| `shell.Run("cmd {var}")` | Command injection | `shell.Output("cmd", arg)` |
| `httphelper.Redirect(w, r, nonLiteral)` | Open redirect | `httphelper.SafeRedirect(w, r, url, "host")` |

HTTP handler detection: any function with an `http.ResponseWriter` parameter triggers the handler-context checks.

---

### Skills (Agent Tool Packaging)

The `skill` keyword declares a Kukicha package as a self-describing agent tool. `kukicha pack` compiles it into a distributable directory with a machine-readable manifest. `stdlib/skills` discovers these manifests at runtime.

#### Declaring a skill

```kukicha
# target: mcp
petiole weather

skill WeatherService
    description: "Provides weather forecasts."
    version: "1.0.0"

import "stdlib/mcp"

func GetForecast(city string) string
    return "sunny in {city}"

func main()
    server := mcp.New("weather", "1.0.0")
    schema := mcp.Schema(list of mcp.SchemaProperty{
        mcp.Prop("city", "string", "City name"),
    }) |> mcp.Required(list of string{"city"})
    mcp.Tool(server, "get_forecast", "Get weather forecast", schema, handleForecast)
    mcp.Serve(server) onerr panic "{error}"
```

Rules enforced by the compiler:
- Name must be exported (uppercase first letter)
- Requires a `petiole` declaration (skills are packages, not standalone programs)
- Must have a `description`
- `version` must be valid semver if present

#### Packaging with `kukicha pack`

```bash
kukicha pack weather.kuki
```

Produces a self-contained directory:

```
weather_service/
├── SKILL.md              # YAML manifest (name, description, version, exported functions + param types)
└── scripts/
    └── weather_service   # compiled MCP server binary
```

The generated `SKILL.md` contains a YAML frontmatter manifest describing the skill's API — function names, parameter types, and defaults — so orchestrators can discover what the tool offers without running it.

#### Discovering skills at runtime with `stdlib/skills`

```kukicha
import "stdlib/skills"

# Discover all SKILL.md manifests under a directory
tools := skills.Discover("./tools") onerr panic "{error}"
for tool in tools
    print("{tool.Name}: {tool.Content}")

# Convenience helpers for standard locations
agent := skills.AgentSkills() onerr panic "{error}"    # .agent/skills/
claude := skills.ClaudeSkills() onerr panic "{error}"  # .claude/skills/
```

An orchestrator written in Kukicha uses `skills.Discover()` to find packed skill manifests, reads their descriptions, and can feed them to an LLM or use the binary over MCP.

---

### Testing

Test files use the `*_test.kuki` suffix and the table-driven pattern:

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
    }
    for tc in cases
        t.Run(tc.name, (t reference testing.T) =>
            result := slice.First(items, tc.n)
            test.AssertEqual(t, len(result), tc.wantLen)
        )
```

Assertions: `test.AssertEqual`, `test.AssertNotEqual`, `test.AssertTrue`, `test.AssertFalse`, `test.AssertNoError`, `test.AssertError`, `test.AssertNotEmpty`, `test.AssertNil`, `test.AssertNotNil`.

---

**All available packages:** `cast`, `cli`, `concurrent`, `container`, `crypto`, `ctx`, `datetime`, `db`, `encoding`, `env`, `errors`, `fetch`, `files`, `game`, `git`, `html`, `http`, `input`, `iterator`, `json`, `llm`, `maps`, `mcp`, `must`, `net`, `netguard`, `obs`, `parse`, `random`, `regex`, `retry`, `sandbox`, `semver`, `shell`, `skills`, `slice`, `sort`, `sqlite`, `string`, `table`, `template`, `test`, `validate`

---

## Common Pitfalls

Patterns that look correct but introduce subtle bugs.

**WaitGroups in goroutines — always use `defer wg.Done()`**

```kukicha
# WRONG — hangs if task() panics, wg.Wait() blocks forever
go func()
    task()
    wg.Done()
()

# CORRECT — defer fires even on panic
go func()
    defer wg.Done()
    task()
()
```

**Cleanup goroutines — always provide a shutdown path**

Goroutines that loop on a ticker run forever if there's no stop signal. The parent exits, but the goroutine (and any connections it holds) leak.

```kukicha
# WRONG — cleanup runs forever, no way to stop it
go func()
    ticker := time.NewTicker(datetime.Seconds(60))
    for range ticker.C
        purgeExpired()
()

# CORRECT — select on a stop channel or context cancellation
go func()
    ticker := time.NewTicker(datetime.Seconds(60))
    defer ticker.Stop()
    for
        select
            when receive from stop
                return
            when receive from ticker.C
                purgeExpired()
()
```

Always pair background goroutines with a `Close()` method that signals them to exit. For context-based shutdown, use `ctxpkg.WithCancel()` and check `ctx.Done()` in the select.

**Context cancel lifetime — defer in the function that uses the resource**

```kukicha
# WRONG — cancel fires when buildCmd returns, before the resource is used
func buildCmd(cmd Command) reference exec.Cmd
    h := ctxpkg.WithTimeout(ctxpkg.Background(), 30)
    defer ctxpkg.Cancel(h)     # fires here — context already dead
    return exec.CommandContext(ctxpkg.Value(h), cmd.name, many cmd.args)

# CORRECT — defer in Execute, which owns the resource's lifetime
func Execute(cmd Command) Result
    h := ctxpkg.WithTimeout(ctxpkg.Background(), 30)
    defer ctxpkg.Cancel(h)     # fires after Run() completes
    execCmd := exec.CommandContext(ctxpkg.Value(h), cmd.name, many cmd.args)
    ...
```

**Never use `io.NopCloser` on a live response body**

`io.NopCloser` replaces `Close()` with a no-op — the TCP connection leaks. When capping reads with `io.LimitReader`, preserve the original closer:

```kukicha
# WRONG — Close() never reaches the connection
resp.Body = io.NopCloser(io.LimitReader(resp.Body, maxSize))

# CORRECT — wrap with a type that delegates both Read and Close
type limitReadCloser
    r io.Reader
    c io.Closer

func Read on b reference limitReadCloser (p list of byte) (int, error)
    return b.r.Read(p)

func Close on b reference limitReadCloser () error
    return b.c.Close()

resp.Body = reference of limitReadCloser{r: io.LimitReader(resp.Body, maxSize), c: resp.Body}
```

---

## Troubleshooting

| Error Message | Cause | Fix |
|---------------|-------|-----|
| `use {error} not {err} inside onerr` | Wrong error variable name in `onerr` block | Change `{err}` to `{error}` or use `onerr as e` |
| `undefined: {err}` | Referencing `err` inside `onerr` | The caught error is always named `error`. Use `{error}` in interpolation, or write `onerr as e` then use `{e}` |
| `variable 'data' not used` | Declared but never read | Use `_ := f()` to discard return values, or remove the variable |
| `function must declare return type` | Implicit return type | Function signatures require explicit return types: `func F() int` not `func F()` |
| `onerr return requires return type` | Using `onerr return` in function that doesn't return values | Use `onerr discard` to ignore the error, or add a return type to your function |
| `cannot take reference of ...` | Trying to get pointer of something | Use `reference of x` for variables, `empty T` for typed zero values |
| `SSRF risk: fetch.Get inside HTTP handler` | Security check triggering | Use `fetch.SafeGet(url)` inside HTTP handlers |
| `command injection risk` | Using `shell.Run` with variable interpolation | Use `shell.Output("cmd", arg1, arg2)` where args are separate parameters |
| `path traversal risk: files.Read inside HTTP handler` | Security check triggering | Use `sandbox.New(root)` + `sandbox.Read(box, path)` in handler code |
| `XSS risk: http.HTML with non-literal content` | Security check triggering | Use `httphelper.SafeHTML(w, content)` for user content |

---

### Common Mistakes

**1. Both Go and Kukicha syntax work**

Kukicha is a strict superset of Go — both styles compile. Use whichever you prefer, or mix them.

```kukicha
# Go style — works fine
if err != nil {
    return err
}

# Kukicha style — also works
if err isnt empty
    return err
```

**2. Wrong error variable in `onerr`**

```kukicha
# WRONG — {err} is not defined inside onerr
data := fetch.Get(url) onerr
    print("failed: {err}")    # compile error

# CORRECT — {error} is the always-available name
data := fetch.Get(url) onerr
    print("failed: {error}")

# ALSO CORRECT — rename if you prefer
data := fetch.Get(url) onerr as e
    print("failed: {e}")
```

**3. Forgetting `petiole` in multi-file projects**

```kukicha
# WRONG — files won't see each other
# (file 1: main.kuki — no petiole)
function main()
    serve()

# (file 2: serve.kuki — no petiole)
function serve()
    print("hello")

# CORRECT — same petiole name connects files
# main.kuki
petiole myapp
function main()
    serve()

# serve.kuki
petiole myapp
function serve()
    print("hello")
```

**4. Using `onerr return` in void functions**

```kukicha
# WRONG — no return type, can't use 'onerr return'
function logError(msg string)
    data := fetch(msg) onerr return    # compile error

# CORRECT — use 'onerr discard' or handle explicitly
function logError(msg string)
    _ := fetch(msg) onerr discard

# ALSO CORRECT — return an error type
function logError(msg string) error
    data := fetch(msg) onerr return empty, error "{error}"
    return empty
```
