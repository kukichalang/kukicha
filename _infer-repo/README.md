# kukichalang/infer

Smart inference orchestrator for Kukicha — ONNX Runtime with NPU/GPU/browser fallback.

## Overview

This module provides three packages:

- **`infer`** — Smart fallback orchestrator. Tries native ONNX Runtime first, falls back to browser-based inference via Playwright + onnxruntime-web.
- **`ort`** — Native ONNX Runtime wrapper via [onnxruntime_go](https://github.com/yalue/onnxruntime_go). Supports CPU and hardware-accelerated execution providers (CUDA, TensorRT, CoreML, OpenVINO, DirectML, QNN, etc.).
- **`webinfer`** — Browser-based inference via headless Chromium + [onnxruntime-web](https://www.npmjs.com/package/onnxruntime-web). Enables WebNN and WebGPU execution providers without native libraries.

## Structure

```
go.mod                      # Module: github.com/kukichalang/infer
infer.go                    # Smart orchestrator (auto-detects backend)
infer.kuki                  # Kukicha stub source
infer_test.go               # Tests for orchestrator
internal/errors/errors.go   # Internal error helpers
ort/
  ort.go                    # Native ONNX Runtime wrapper
  ort.kuki                  # Kukicha stub source
  ort_test.go               # Tests for native backend
webinfer/
  webinfer.go               # Browser-based inference backend
  webinfer.kuki             # Kukicha stub source
```

## Usage (Kukicha)

```kukicha
import "stdlib/infer"

env := infer.Init() onerr panic "no inference: {error}"
defer infer.Cleanup(env)

print("Backend: {infer.Backend(env)}")

input := infer.NewFloat32(env, infer.Shape(1, 10), data) onerr panic "{error}"
output := infer.ZeroFloat32(env, infer.Shape(1, 5)) onerr panic "{error}"

model := infer.New()
    |> infer.Threads(4)
    |> infer.EP("webnn")
    |> infer.Load(env, "model.onnx",
        list of string{"input"}, list of string{"output"},
        list of Tensor{input}, list of Tensor{output})
    onerr panic "{error}"
defer infer.Close(model)

infer.Run(model) onerr panic "{error}"
results := infer.GetFloat32(output)
```

## Dependencies

- `github.com/yalue/onnxruntime_go v1.27.0` — ONNX Runtime Go bindings
- `github.com/playwright-community/playwright-go v0.5700.1` — Headless browser automation

## Migrated from

Originally `stdlib/accel` → `infer`, `stdlib/infer` → `ort`, `stdlib/webinfer` → `webinfer` in the kukichalang/kukicha repo.
