# Deploy Status

Variant enums for a deployment pipeline — each state carries its own
fields, so there are no nullable catch-all columns and the compiler
enforces exhaustive handling.

## What it shows

- `enum DeployStatus` with five variants (`Queued`, `Building`,
  `Running`, `Failed`, `RolledBack`), each with distinct payload fields.
- `|> switch as v` to format each variant using its own fields.
- Plain `switch s as v` inside a predicate (`isHealthy`) that only
  cares about one variant.

## Running

```bash
kukicha run examples/deploy-status/main.kuki
```
