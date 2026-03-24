# NPU Migration Plan: Extract Inference Packages to `kukichalang/infer`

## Overview

Extract the three inference packages from the `npu_support` / `npu-support-clean`
branches into a standalone `kukichalang/infer` repository, following the same
pattern used for `kukichalang/game`.

Source branches:
- `origin/npu_support` (kukichalang/kukicha) — base inference code
- `duber000/npu-support-clean` (duber000/kukicha, PR #87) — adds FastFlowLM + execution providers

## Rename Map

| Current name (on branch) | New stdlib name | Role |
|--------------------------|----------------|------|
| `stdlib/accel` | **`stdlib/infer`** | Smart orchestrator — auto-selects best backend. **User-facing entry point.** |
| `stdlib/infer` | **`stdlib/ort`** | Low-level native ONNX Runtime wrapper (CGo via `yalue/onnxruntime_go`) |
| `stdlib/webinfer` | **`stdlib/webinfer`** | Low-level browser-based ONNX (Playwright + onnxruntime-web CDN) |

Rationale: beginners just `import "stdlib/infer"` and get automatic backend
selection. Power users drop to `stdlib/ort` or `stdlib/webinfer` for control.

## What stays in `kukichalang/kukicha` (main repo)

### Registry stubs (`.kuki` files only)

- `stdlib/infer/infer.kuki` — type definitions + function signatures (was `accel`)
- `stdlib/ort/ort.kuki` — type definitions + function signatures (was `infer`)
- `stdlib/webinfer/webinfer.kuki` — type definitions + function signatures

These stubs enable type checking, LSP support, and error messages without
pulling in heavy dependencies.

### Import rewriting (`internal/codegen/codegen_imports.go`)

Add to the `externalStdlibPackages` map:

```go
var externalStdlibPackages = map[string]string{
    "game":     "github.com/kukichalang/game",
    "infer":    "github.com/kukichalang/infer",
    "ort":      "github.com/kukichalang/infer/ort",
    "webinfer": "github.com/kukichalang/infer/webinfer",
}
```

When user code writes `import "stdlib/infer"`, codegen rewrites it to
`github.com/kukichalang/infer`. Same for `stdlib/ort` → `.../infer/ort` and
`stdlib/webinfer` → `.../infer/webinfer`.

### FastFlowLM (no extraction needed)

The FastFlowLM provider stays in `stdlib/llm` — it's just two lines adding
`"fastflowlm"` / `"flm"` as a provider URL (`localhost:52625`). No ONNX or
heavy dependency involved. Cherry-pick from commit `6ed1a93`.

### `cmd/ku-accel` CLI

Moves to `kukichalang/infer` repo (it depends on `ort` for detection and
references FastFlowLM binary checks). The main repo does not need this CLI.

## External repo: `kukichalang/infer`

### Structure

```
github.com/kukichalang/infer/
├── infer.go              # Smart orchestrator (was accel.go)
├── infer.kuki            # Kukicha source (was accel.kuki)
├── infer_test.go
├── ort/
│   ├── ort.go            # Native ONNX Runtime (was infer.go)
│   ├── ort.kuki          # Kukicha source (was infer.kuki)
│   ├── ort_test.go
│   └── testdata/
│       └── tiny_linear.onnx
├── webinfer/
│   ├── webinfer.go       # Browser-based ONNX
│   ├── webinfer.kuki
│   └── webinfer_test.go
├── cmd/
│   └── ku-accel/
│       ├── main.go       # Hardware detection CLI
│       └── main.kuki
├── go.mod                # Heavy deps live here, not in main kukicha
├── go.sum
└── README.md
```

### Dependencies (isolated from main compiler)

- `github.com/yalue/onnxruntime_go v1.27.0` — CGo ONNX Runtime bindings
- `github.com/playwright-community/playwright-go v0.5700.1` — headless Chromium

These dependencies add ~150MB+ at install time and require native libraries.
Keeping them out of the main `kukichalang/kukicha` module is the primary
motivation for this split.

## Execution Steps

### Phase 1: Create `kukichalang/infer` repo

1. Create `kukichalang/infer` on GitHub
2. Initialize with `go.mod` (`module github.com/kukichalang/infer`)
3. Copy full implementations from `duber000/npu-support-clean`:
   - `stdlib/accel/` → repo root (`infer.go`, rename all `accel` references)
   - `stdlib/infer/` → `ort/` sub-package (rename all `infer` → `ort`)
   - `stdlib/webinfer/` → `webinfer/` sub-package
   - `cmd/ku-accel/` → `cmd/ku-accel/`
4. Update internal imports within the new repo
5. Add `go.sum` with heavy deps (onnxruntime_go, playwright-go)
6. Run tests, tag `v0.1.0`

### Phase 2: Update `kukichalang/kukicha` main repo

1. **Cherry-pick FastFlowLM** (commit `6ed1a93`) into `stdlib/llm/`
2. **Create stubs:**
   - `stdlib/infer/infer.kuki` — types + signatures from old `accel.kuki`
   - `stdlib/ort/ort.kuki` — types + signatures from old `infer.kuki`
   - `stdlib/webinfer/webinfer.kuki` — types + signatures from old `webinfer.kuki`
3. **Add import rewriting** in `internal/codegen/codegen_imports.go`
4. **Run `make genstdlibregistry`** to regenerate `stdlib_registry_gen.go`
5. **Remove heavy deps** from `go.mod` (onnxruntime_go, playwright-go)
6. **Update docs:**
   - `CLAUDE.md` file map: mark `infer/`, `ort/`, `webinfer/` as registry stubs
   - `stdlib/CLAUDE.md` package table: link to `kukichalang/infer`
   - `CHANGELOG.md`: document the extraction
7. **Run `make test && make lint && make vet`**

### Phase 3: Verify end-to-end

1. User writes `import "stdlib/infer"` in `.kuki` file
2. Compiler type-checks against stub signatures
3. Codegen rewrites to `github.com/kukichalang/infer`
4. `go build` fetches the external module
5. Binary runs with full inference support

## Not in scope

- `stdlib/kube/` and `stdlib/pg/` — separate extraction candidates, not part of this plan
- `cmd/ku-diag/`, `cmd/ku-k8s/`, `cmd/ku-sys/` — separate CLI tools, not inference-related
- `editors/vscode/`, `editors/zed/` — already have their own repos (`kukichalang/vscode-kukicha`, `kukichalang/zed-kukicha`)
- Module path fix (`duber000/kukicha` → `kukichalang/kukicha`) — should be done on `npu_support` branch regardless
