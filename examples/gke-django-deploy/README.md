# GKE Django Deployer

Orchestrate OpenTofu stacks from Kukicha: create a GKE cluster, three
namespaces (`dev` / `test` / `prod`), and deploy a Django app with
environment-specific config into each namespace.

Uses OpenTofu's `-json-into` flag for dual human + machine-readable
output, and deploys namespaces concurrently.

## What it shows

- Multi-file directory build (`main.kuki`, `config.kuki`, `tofu.kuki`)
- CLI subcommands via `stdlib/cli` with `|>` builder chain
- Variant enums for deployment state
- `concurrent.MapWithLimit` for parallel namespace deploys
- `maps.Merge` / `maps.Pick` / `maps.Omit` for per-env config
- `set` operations for namespace diffing
- `stdlib/table` for status output

## Prerequisites

```bash
export GCP_PROJECT="my-gcp-project"
export DJANGO_IMAGE="gcr.io/my-project/django-app:v1.2.3"
```

## Running

```bash
kukicha run examples/gke-django-deploy/ deploy [--env dev|test|prod|all] [--auto-approve]
kukicha run examples/gke-django-deploy/ status
```
