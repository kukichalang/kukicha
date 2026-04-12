# SQLite JSON Bookmark Tagger

Demonstrates SQLite 3.53.0+ JSON functions (`json_array_insert` and
`jsonb_array_insert`) used directly in SQL via Kukicha's `stdlib/db`
and `stdlib/sqlite`.

A small bookmarks table keeps per-row tags in a JSON array column.
All tag edits happen server-side in SQLite — no Go marshal/unmarshal
round-trip.

## What it shows

1. **Opening SQLite** with sensible defaults via `sqlite.Open` (WAL,
   foreign keys, busy timeout).
2. **A JSON column** (`tags TEXT NOT NULL DEFAULT '[]'`) storing a
   JSON array per bookmark.
3. **`json_array_insert`** to prepend, append, and bulk-insert tags
   at specific array positions within a single `UPDATE`.
4. **`jsonb_array_insert`** for the binary JSONB variant — identical
   semantics, faster on read-heavy workloads.
5. **`json_each`** to query bookmarks by tag and build a tag histogram.

## Running

```bash
kukicha run examples/bookmark-tags/main.kuki
```

## Expected output

```
=== SQLite JSON Bookmark Tagger (SQLite 3.53.0) ===

--- All bookmarks ---
  [1] SQLite — https://sqlite.org
       tags: ["fast","db","embedded"]
  [2] Go — https://go.dev
       tags: ["concurrent","lang","systems"]
  [3] Kukicha — https://kukicha.org
       tags: ["lang","open-source","tooling"]

--- Bookmarks tagged 'lang' ---
  Go: ["concurrent","lang","systems"]
  Kukicha: ["lang","open-source","tooling"]

--- Tag histogram ---
  lang: 2
  concurrent: 1
  db: 1
  embedded: 1
  fast: 1
  open-source: 1
  systems: 1
  tooling: 1

Done. Database: /tmp/bookmark-tags.db
```

## Why JSON in SQLite?

For semi-structured per-row data (tags, settings, labels, event
metadata), a JSON column is often simpler than a separate join table:

- One query reads the whole object.
- No fan-out cost on small arrays.
- `json_array_insert` / `json_each` keep mutation and traversal
  in SQL, which plays well with transactions.

For large arrays or heavy relational queries, prefer a proper
join table.

## Key pattern: appending with `$[#]`

SQLite supports a special `#` subscript that refers to "one past the
end of the array". It's the idiomatic way to append with
`json_array_insert` — no `json_array_length` round-trip needed:

```sql
UPDATE bookmarks
SET tags = json_array_insert(tags, '$[#]', ?)
WHERE id = ?
```

It composes cleanly with multiple path/value pairs for batch appends:

```sql
UPDATE bookmarks
SET tags = json_array_insert(tags, '$[#]', ?, '$[#]', ?)
WHERE id = ?
```
