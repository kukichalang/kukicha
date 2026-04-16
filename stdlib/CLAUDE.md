# stdlib/ — Standard Library

For the full stdlib reference (package table, usage patterns, security checks, pitfalls), invoke the **`/stdlib`** skill.

stdlib packages should be written in kukicha not go(we need to dogfood).

**Critical:** Never edit generated `*.go` files — edit `.kuki` source, then `make generate`.
After adding exported functions or enums to a `.kuki` file, run `make genstdlibregistry`.
