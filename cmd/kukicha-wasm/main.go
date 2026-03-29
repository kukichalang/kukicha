//go:build js && wasm

// Package main is the WASM entrypoint for the Kukicha playground.
// It exposes a single JS function: kukichaTranspile(source) → {goSource, errors}.
//
// Build:
//
//	GOOS=js GOARCH=wasm go build -o kukicha.wasm ./cmd/kukicha-wasm
package main

import (
	"encoding/json"
	"go/format"
	"syscall/js"

	"github.com/kukichalang/kukicha/internal/codegen"
	"github.com/kukichalang/kukicha/internal/parser"
	"github.com/kukichalang/kukicha/internal/semantic"
)

// transpileResult is the JSON-serialisable response returned to JavaScript.
type transpileResult struct {
	GoSource string   `json:"goSource"`
	Errors   []string `json:"errors"`
}

// transpile runs the full Kukicha pipeline on source and returns Go source +
// any errors. It never panics — all error paths are collected and returned.
func transpile(source string) transpileResult {
	p, err := parser.New(source, "playground.kuki")
	if err != nil {
		return transpileResult{Errors: []string{err.Error()}}
	}

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		errs := make([]string, len(parseErrors))
		for i, e := range parseErrors {
			errs[i] = e.Error()
		}
		return transpileResult{Errors: errs}
	}

	analyzer := semantic.NewWithFile(program, "playground.kuki")
	semanticErrors := analyzer.Analyze()
	if len(semanticErrors) > 0 {
		errs := make([]string, len(semanticErrors))
		for i, e := range semanticErrors {
			errs[i] = e.Error()
		}
		return transpileResult{Errors: errs}
	}

	gen := codegen.New(program)
	gen.SetSourceFile("playground.kuki")
	gen.SetExprReturnCounts(analyzer.ReturnCounts())
	gen.SetExprTypes(analyzer.ExprTypes())
	goCode, err := gen.Generate()
	if err != nil {
		return transpileResult{Errors: []string{err.Error()}}
	}

	formatted, err := format.Source([]byte(goCode))
	if err != nil {
		// Return unformatted Go — still useful for debugging
		return transpileResult{GoSource: goCode}
	}

	return transpileResult{GoSource: string(formatted)}
}

func main() {
	js.Global().Set("kukichaTranspile", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) == 0 {
			return resultToJS(transpileResult{Errors: []string{"kukichaTranspile: missing source argument"}})
		}
		result := transpile(args[0].String())
		return resultToJS(result)
	}))

	// Block forever — the WASM module lives as long as the page.
	select {}
}

// resultToJS serialises result to a JSON string and parses it back as a JS
// object via JSON.parse. This avoids manually constructing js.Value maps and
// keeps the JS API surface simple: callers get a plain object with .goSource
// and .errors fields.
func resultToJS(result transpileResult) js.Value {
	b, _ := json.Marshal(result)
	return js.Global().Get("JSON").Call("parse", string(b))
}
