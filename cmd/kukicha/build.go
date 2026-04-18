package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func buildMain(args []string) {
	buildFlags := flag.NewFlagSet("build", flag.ContinueOnError)
	buildFlags.SetOutput(os.Stderr)
	target := buildFlags.String("target", "", "Compile target")
	skipBuild := buildFlags.Bool("skip-build", false, "Skip go build step (for test files)")
	ifChanged := buildFlags.Bool("if-changed", false, "Skip writing output if Go body (excluding generated header) is unchanged")
	vulncheck := buildFlags.Bool("vulncheck", false, "Run govulncheck after successful build")
	wasm := buildFlags.Bool("wasm", false, "Build for WebAssembly (GOOS=js GOARCH=wasm)")
	noLineDirectives := buildFlags.Bool("no-line-directives", false, "Omit //line directives from generated Go (cleaner output for production builds)")
	if err := buildFlags.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "Usage: kukicha build [--target <target>] [--skip-build] [--if-changed] [--vulncheck] [--wasm] [--no-line-directives] <file.kuki>")
		os.Exit(1)
	}
	buildArgs := buildFlags.Args()
	if len(buildArgs) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: kukicha build [--target <target>] [--skip-build] [--if-changed] [--vulncheck] [--wasm] [--no-line-directives] <file.kuki>")
		os.Exit(1)
	}
	code := buildCommand(buildArgs[0], *target, *skipBuild, *ifChanged, *vulncheck, *wasm, *noLineDirectives)
	if code != 0 {
		os.Exit(code)
	}
}

func buildCommand(filename string, targetFlag string, skipBuild bool, ifChanged bool, vulncheck bool, wasm bool, noLineDirectives bool) int {
	cr, err := compile(filename, targetFlag, "", noLineDirectives)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	// Write Go file — for directory builds, use <dir>/main.go; for files, use <file>.go
	var outputFile string
	info, _ := os.Stat(filename)
	isDir := info != nil && info.IsDir()
	if isDir {
		outputFile = filepath.Join(cr.absFile, "main.go")
	} else {
		outputFile = strings.TrimSuffix(cr.absFile, ".kuki") + ".go"
	}

	if ifChanged {
		if existing, readErr := os.ReadFile(outputFile); readErr == nil {
			if bytes.Equal(stripFirstLine(existing), stripFirstLine(cr.formatted)) {
				// Body unchanged — touch the file so staleness checks pass,
				// but don't rewrite content (preserves old version comment).
				now := time.Now()
				os.Chtimes(outputFile, now, now)
				return 0
			}
		}
	}

	if err := os.WriteFile(outputFile, cr.formatted, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		return 1
	}

	if isDir {
		fmt.Printf("Successfully compiled %s/ to %s\n", cr.absFile, outputFile)
	} else {
		fmt.Printf("Successfully compiled %s to %s\n", cr.absFile, outputFile)
	}

	if err := ensureStdlibIfNeeded(cr.goCode, cr.projectDir); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	// Determine the output binary name. When cross-compiling for Windows
	// (GOOS=windows), append .exe so the binary is recognised as executable.
	// WASM builds produce a .wasm file instead.
	var binaryName string
	if isDir {
		binaryName = filepath.Base(cr.absFile)
	} else {
		binaryName = strings.TrimSuffix(filepath.Base(cr.absFile), ".kuki")
	}
	if wasm {
		binaryName += ".wasm"
	} else {
		targetOS := os.Getenv("GOOS")
		if targetOS == "" {
			targetOS = runtime.GOOS
		}
		if targetOS == "windows" {
			binaryName += ".exe"
		}
	}
	// Place binary in the current working directory, matching `go build` behavior.
	// If the resulting path collides with an existing directory (e.g., building
	// deploy/ from its parent), place the binary inside the directory instead.
	cwd, cwdErr := os.Getwd()
	if cwdErr != nil {
		fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", cwdErr)
		return 1
	}
	binaryPath := filepath.Join(cwd, binaryName)
	if info, err := os.Stat(binaryPath); err == nil && info.IsDir() {
		binaryPath = filepath.Join(binaryPath, binaryName)
	}

	// Run go build on the generated file. Use -mod=mod so go.sum is updated
	// automatically when stdlib transitive dependencies are not yet listed.
	if !skipBuild {
		cmd := exec.Command("go", "build", "-mod=mod", "-o", binaryPath, outputFile)
		cmd.Dir = cr.projectDir
		env := os.Environ()
		if wasm {
			env = setEnvVar(env, "GOOS", "js")
			env = setEnvVar(env, "GOARCH", "wasm")
		}
		cmd.Env = env
		cmd.Stdout = os.Stdout
		var stderrBuf bytes.Buffer
		cmd.Stderr = &stderrBuf
		err := cmd.Run()
		if stderrBuf.Len() > 0 {
			os.Stderr.Write(rewriteGoErrors(stderrBuf.Bytes(), outputFile, cr.absFile))
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: go build failed: %v\n", err)
			return 1
		}

		fmt.Printf("Successfully built binary: %s\n", binaryName)

		if wasm {
			wasmScaffold(cr.projectDir, strings.TrimSuffix(filepath.Base(cr.absFile), ".kuki"))
		}
	}

	if vulncheck {
		code := runAudit(AuditOptions{Dir: cr.projectDir})
		if code != 0 {
			return code
		}
	}
	return 0
}

// setEnvVar sets or replaces an environment variable in the env slice.
func setEnvVar(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

// wasmScaffold copies wasm_exec.js and generates index.html alongside the .wasm file.
func wasmScaffold(projectDir, name string) {
	// Copy wasm_exec.js from Go installation
	gorootBytes, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not determine GOROOT: %v\n", err)
		return
	}
	goroot := strings.TrimSpace(string(gorootBytes))
	wasmExecSrc := filepath.Join(goroot, "lib", "wasm", "wasm_exec.js")
	wasmExecDst := filepath.Join(projectDir, "wasm_exec.js")

	if data, err := os.ReadFile(wasmExecSrc); err == nil {
		if err := os.WriteFile(wasmExecDst, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not write wasm_exec.js: %v\n", err)
		} else {
			fmt.Printf("Copied wasm_exec.js\n")
		}
	} else {
		fmt.Fprintf(os.Stderr, "Warning: could not read wasm_exec.js from %s: %v\n", wasmExecSrc, err)
		fmt.Fprintf(os.Stderr, "You may need to copy it manually from $(go env GOROOT)/lib/wasm/wasm_exec.js\n")
	}

	// Generate index.html if it doesn't already exist
	indexPath := filepath.Join(projectDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		html := fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>%s</title>
<style>body { margin: 0; padding: 0; background: #000; }</style>
</head><body>
<script src="wasm_exec.js"></script>
<script>
const go = new Go();
WebAssembly.instantiateStreaming(fetch("%s.wasm"), go.importObject).then(r => go.run(r.instance));
</script>
</body></html>
`, name, name)
		if err := os.WriteFile(indexPath, []byte(html), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not write index.html: %v\n", err)
		} else {
			fmt.Printf("Generated index.html\n")
		}
	}
}
