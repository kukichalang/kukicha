---
name: release
description: Cut a new Kukicha release — bump version, regenerate stdlib, run tests, commit, tag, and push. Use when the user says "cut a release", "release a new version", or similar.
---

# Release Skill

Cuts a new Kukicha release following the process defined in `docs/contributing.md`.

## Steps

### 1. Determine the next version

Read the current version from `internal/version/version.go`. Increment the patch segment (e.g. `0.0.10` → `0.0.11`) unless the user specifies otherwise.

### 2. Bump version in source

Edit `internal/version/version.go`:
```
const Version = "0.0.X"   # new version
```

### 3. Update version references in docs

**README.md** — exactly two occurrences (grep for the old version to confirm):
- `go install github.com/kukichalang/kukicha/cmd/kukicha@vOLD`
- `**Version:** OLD — Ready for testing`

**CLAUDE.md** and **AGENTS.md** — one occurrence each near the top (must stay in sync):
- `Current version: **OLD**`

Replace all with the new version.

### 3b. Update version and WASM in kukicha.org

Update the three version strings in `~/repos/kukicha/kukicha.org`:

- `components/hero.kuki` — badge text: `v0.0.OLD` (two occurrences on one line: `title` attribute and link text)
- `components/layout.kuki` — footer text: `Kukicha v0.0.OLD`
- `Dockerfile` — `go install github.com/kukichalang/kukicha/cmd/kukicha@vOLD`

Then rebuild the playground WASM from the just-built compiler (the new version is now baked into the binary):

```bash
make build-wasm WASM_OUT=~/repos/kukicha/kukicha.org/static/wasm/kukicha.wasm
```

Commit and push in the kukicha.org repo:

```bash
cd ~/repos/kukicha/kukicha.org
git add components/hero.kuki components/layout.kuki Dockerfile static/wasm/kukicha.wasm
git commit -m "chore: bump version to vX.X.X"
git push origin main
```

> Note: `make build-wasm` must run **after** `make build` in step 4 so the WASM embeds the new version. Step ordering matters: 3b version strings → 4 rebuild → re-run `make build-wasm` if the WASM needs refreshing. In practice, run `make build-wasm` at the end of step 4.

> **Line directives:** The kukicha.org Dockerfile builds with `kukicha build --no-line-directives`, so the deployed site binary has clean Go output without `//line` comments. The playground WASM intentionally keeps line directives — they show up in the Generated Go pane and are explained to users. No action needed on either; both are handled automatically.

### 4. Regenerate, rebuild, and rebuild WASM

Generated `.go` headers no longer contain the version number, so a version-only bump does not require force-regenerating stdlib files. Just regenerate the registry files and rebuild:

```bash
make generate && make build
```

This regenerates `internal/semantic/stdlib_registry_gen.go` and `go_stdlib_gen.go`, then rebuilds the compiler.

Then rebuild the playground WASM so it embeds the new version (must happen after `make build`):

```bash
make build-wasm WASM_OUT=~/repos/kukicha/kukicha.org/static/wasm/kukicha.wasm
```

Stage the updated WASM in kukicha.org (the version-string commit in step 3b should already be done; amend it or add a second commit if needed):

```bash
cd ~/repos/kukicha/kukicha.org
git add static/wasm/kukicha.wasm
```

### 5. Run tests, lint, vet, and modernize

```bash
make test
make lint
make vet
make modernize
```

All tests must pass, lint/vet must be clean, and `make modernize` must find no outdated patterns before tagging. Do not proceed if any fails.

### 5b. Run fuzz tests and validate examples

Run the two fuzz targets briefly (10 seconds each) as a smoke test:

```bash
go test ./internal/lexer/ -fuzz=FuzzLexer -fuzztime=10s
go test ./internal/codegen/ -fuzz=FuzzPipeline -fuzztime=10s
```

Then validate all example files with structured diagnostics (use the locally-built binary from step 4):

```bash
./kukicha check --json examples/*.kuki
```

The fuzz tests must not find any crashes, and `kukicha check` must report no errors on the examples. Do not proceed if either fails.

### 6. Commit

Stage and commit in a single commit (per the contributing guide — do NOT split into multiple commits):

```bash
git add README.md CLAUDE.md AGENTS.md internal/version/version.go internal/semantic/stdlib_registry_gen.go
git commit -m "chore: release vX.X.X

Bump version constant and update doc install snippets."
```

### 7. Tag and confirm before pushing

Create the tag locally, then **ask the user to confirm** before pushing:

```bash
git tag vX.X.X
```

After user confirms:

```bash
git push origin main
git push origin vX.X.X
```

> Note: `git push --follow-tags` sometimes silently skips tags. Push the tag explicitly with `git push origin vX.X.X` and verify with `git ls-remote --tags origin | grep vX.X.X`.

## Checklist

- [ ] `internal/version/version.go` bumped
- [ ] `README.md` — both version strings updated
- [ ] `CLAUDE.md` — version line updated
- [ ] `AGENTS.md` — version line updated (must match CLAUDE.md)
- [ ] `make generate && make build` succeeded
- [ ] `make test` — all packages pass
- [ ] `make lint` — zero issues
- [ ] `make vet` — zero issues (covers stdlib, which golangci-lint excludes)
- [ ] `make modernize` — no outdated Go patterns in generated code
- [ ] Fuzz tests (`FuzzLexer`, `FuzzPipeline`) — no crashes in 10s runs
- [ ] `kukicha check --json examples/*.kuki` — no errors on examples
- [ ] Single commit with all changes
- [ ] Tag created and pushed
- [ ] `git ls-remote --tags origin` confirms tag is present
- [ ] `~/repos/kukicha/kukicha.org` — `components/hero.kuki` and `components/layout.kuki` updated
- [ ] `~/repos/kukicha/kukicha.org` — `Dockerfile` pinned to new version
- [ ] `~/repos/kukicha/kukicha.org` — `static/wasm/kukicha.wasm` rebuilt via `make build-wasm`
- [ ] `~/repos/kukicha/kukicha.org` — committed and pushed
