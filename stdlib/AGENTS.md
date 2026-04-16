# stdlib/ — Standard Library

For the full stdlib reference (package table, usage patterns, security checks, pitfalls), invoke the **`/stdlib`** skill.

**Always write stdlib packages in Kukicha, not Go** — we dogfood the language here. That means `.kuki` sources use `and`/`or`/`not`, `equals`/`isnt`, `list of T`, `map of K to V`, `reference T`, `empty`, `onerr`, pipes, and 4-space indentation. Raw Go forms (`&&`, `==`, `[]T`, `*T`, `nil`, braces) in a `.kuki` file are a style bug even though they compile — fix them before merging. See `../CLAUDE.md` for the full syntax table.

**Critical:** Never edit generated `*.go` files — edit `.kuki` source, then `make generate`.
After adding exported functions or enums to a `.kuki` file, run `make genstdlibregistry`.
