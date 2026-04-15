# mizuya

A small Kukicha example that turns SQLite into a knowledge graph with
**FTS5 full-text search** and **vector cosine similarity** — using only
`stdlib/sqlite`, `stdlib/db`, `stdlib/json`, and `stdlib/cli`. No graph
engine, no separate vector database, no extension binaries.

The name *mizuya* (水屋) is the preparation room in a tea house; a good
place to keep everything you've learned within reach.

## What the example demonstrates

- `stdlib/sqlite.Open` with WAL + foreign keys via the default pragmas
- Schema + FTS5 virtual tables + triggers defined with plain DDL constants
- JSON-in-SQL with `json_set`, `json_object`, `json_each`, and the `$[#]`
  append path for observations and tags
- BM25 ranking with per-column weights (`bm25(fts, 10, 5, 2, 1)`)
- 1-hop graph context lookups via the `relations` table
- `sqlite.CreateFunction` to register a pure-Kukicha cosine-distance UDF
- A deterministic hash-embedding so `similar` runs offline with zero setup

## Subcommands

| Command   | What it does                                                            |
|-----------|-------------------------------------------------------------------------|
| `init`    | Create the database file and schema                                     |
| `upsert`  | Create/update an entity (`--id --type --name --summary --tags`)         |
| `observe` | Append an observation (`--id --text [--source]`) and re-embed           |
| `relate`  | Create a typed edge (`--src --rel --dst`)                               |
| `search`  | BM25 full-text search (`--query [--limit]`)                             |
| `context` | Entity + outgoing/incoming 1-hop neighbors + observation log (`--id`)   |
| `similar` | Cosine similarity search (`--id` *or* `--text`, `[--limit]`)            |
| `list`    | List entities, optionally filtered by `--type`                          |

All commands accept the global `--db <path>` (default `.mizuya/mizuya.db`)
and `--json` for structured output.

## End-to-end walkthrough

```bash
kukicha run examples/mizuya/ -- init

kukicha run examples/mizuya/ -- upsert \
    --id kukicha --type project \
    --name "Kukicha Language" \
    --summary "Go superset with pipes, onerr, and enums" \
    --tags lang,compiler,go

kukicha run examples/mizuya/ -- upsert \
    --id sqlite --type tool \
    --name "SQLite" \
    --summary "Embedded SQL database with FTS5 and JSON" \
    --tags db,embedded,fts

kukicha run examples/mizuya/ -- upsert \
    --id vllm-h100 --type config \
    --name "vLLM on H100" \
    --summary "Inference server deployment notes" \
    --tags ml,gpu,inference

kukicha run examples/mizuya/ -- relate --src kukicha --rel uses --dst sqlite

kukicha run examples/mizuya/ -- observe \
    --id kukicha --text "Added pipe operator support in v0.0.29"

kukicha run examples/mizuya/ -- search --query "embedded database"
kukicha run examples/mizuya/ -- context --id kukicha
kukicha run examples/mizuya/ -- similar --text "fast database engine"
```

Expected: the `search` query ranks `sqlite` first; `context` shows the
`uses → sqlite` edge and the observation you just recorded; `similar`
puts `sqlite` ahead of `vllm-h100` for a database query, and vice versa
for an ML query.

## Scope

This is a teaching example, not the full Mizuya MCP server. It leaves
out the MCP/HTTP transports, real embedding providers (Ollama, LiteLLM),
and hybrid RRF search — the interesting storage and query primitives all
live in roughly 450 lines of Kukicha across six `.kuki` files.
