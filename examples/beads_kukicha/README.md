# beads_kukicha

A local-first issue tracker stored in SQLite. Kukicha port of [beads_rust](https://github.com/Dicklesworthstone/beads_rust).

## What it shows

1. **Multi-file directory build** -- 8 `.kuki` files merged into one binary
2. **Pipe-chain CLI builder** -- `cli.New("bd") |> cli.Command(...) |> cli.RunApp()`
3. **JSON columns** -- labels stored as a JSON array, queried with `json_each()`, mutated with `json_insert()`/`json_remove()`
4. **Transactions** -- schema init and blocked-cache rebuild wrapped in `db.Transaction`
5. **Content hashing** -- SHA256 over canonical fields for change detection
6. **Materialized blocked cache** -- precomputed table rebuilt on dependency/status changes
7. **Struct scanning** -- `db.Query() |> db.ScanAll(empty list of Issue)` with `as` tag mapping
8. **`onerr` error handling** -- every DB call uses `onerr return` or `onerr panic`

## Building

```bash
kukicha build examples/beads_kukicha/
```

## Usage

```bash
# Initialize
bd init

# Create issues
bd create --title "Fix login timeout" --type bug --priority high --labels "backend,auth"
bd create --title "Add dark mode" --type feature --priority medium
bd create --title "Update README" --type chore --priority low --labels "docs"

# List / filter
bd list
bd list --status open --priority high
bd list --label backend

# Show details (includes deps + comments)
bd show --id bd-XXXXXXXX

# Update fields
bd update --id bd-XXXXXXXX --priority critical --assignee "alice"

# Close / reopen
bd close --id bd-XXXXXXXX
bd reopen --id bd-XXXXXXXX

# Dependencies
bd dep add --issue bd-AAA --depends-on bd-BBB
bd dep add --issue bd-AAA --depends-on bd-BBB --dep-type related
bd dep remove --issue bd-AAA --depends-on bd-BBB
bd dep list --issue bd-AAA

# Ready (open + not blocked) / blocked
bd ready
bd blocked

# Labels (JSON column)
bd label add --id bd-XXXXXXXX --label "frontend"
bd label remove --id bd-XXXXXXXX --label "frontend"

# Comments
bd comment add --id bd-XXXXXXXX --body "Found the root cause" --author "alice"
bd comment list --id bd-XXXXXXXX

# Search
bd search --query "login"

# Statistics
bd stats

# JSON output (any command)
bd list --json
bd stats --json
```

## File structure

| File | Purpose |
|------|---------|
| `main.kuki` | CLI dispatch via pipe-chain builder (`func main`) |
| `models.kuki` | Constants (status, priority, type, etc.) and struct types |
| `schema.kuki` | DDL constants and `initDB()` transaction |
| `ids.kuki` | ID generation (SHA256 + hex) and content hashing |
| `store.kuki` | Issue CRUD, labels, comments, stats |
| `deps.kuki` | Dependency management and blocked-issues cache |
| `commands.kuki` | CLI command handlers bridging args to store |
| `format.kuki` | ANSI colors, enum-to-string converters, time formatting |

## Schema

6 tables: `issues`, `dependencies`, `comments`, `events`, `blocked_issues_cache`, `config`. Labels live in a `TEXT NOT NULL DEFAULT '[]'` JSON array column on `issues`.
