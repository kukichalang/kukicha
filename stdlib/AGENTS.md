# stdlib/ — Standard Library

For the full stdlib reference (package table, usage patterns, security checks, pitfalls), invoke the **`/stdlib`** skill.

**Always write stdlib packages in Kukicha, not Go** — we dogfood the language here. That means `.kuki` sources use `and`/`or`/`not`, `equals`/`isnt`, `list of T`, `map of K to V`, `reference T`, `empty`, `onerr`, pipes, and 4-space indentation. Raw Go forms (`&&`, `==`, `[]T`, `*T`, `nil`, braces) in a `.kuki` file are a style bug even though they compile — fix them before merging. See `../CLAUDE.md` for the full syntax table.

**Critical:** Never edit generated `*.go` files — edit `.kuki` source, then `make generate`.
After adding exported functions or enums to a `.kuki` file, run `make genstdlibregistry`.

## When to keep a stdlib wrapper vs drop it

Keep a wrapper when it hides awkward Go stdlib ceremony — receiver-on-package-var
patterns (`base64.StdEncoding.EncodeToString`), multi-step setup, or API shapes
a beginner wouldn't guess. The wrapper earns its keep by making the common case
one call.

Drop a wrapper when it's the same call shape as the underlying function
(`errors.Is(err, target)` → `goerrors.Is(err, target)`, `strconv.Atoi` →
`strconv.Atoi`). Passthroughs add an import without hiding friction, and mask
which package the caller actually depends on.

When Kukicha has a native construct (e.g. `error "msg"` replaces `errors.New`),
prefer the native form and drop the wrapper.
