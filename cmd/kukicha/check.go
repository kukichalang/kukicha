package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kukichalang/kukicha/internal/ast"
	"github.com/kukichalang/kukicha/internal/parser"
	"github.com/kukichalang/kukicha/internal/semantic"
	"golang.org/x/term"
)

func checkMain(args []string) {
	checkFlags := flag.NewFlagSet("check", flag.ContinueOnError)
	checkFlags.SetOutput(os.Stderr)
	strictOnerr := checkFlags.Bool("strict-onerr", false, "Treat onerr lint warnings as errors")
	jsonOutput := checkFlags.Bool("json", false, "Emit structured JSON diagnostics instead of plain text")
	if err := checkFlags.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "Usage: kukicha check [--strict-onerr] [--json] <file.kuki> [file2.kuki ...]")
		os.Exit(1)
	}
	checkArgs := checkFlags.Args()
	if len(checkArgs) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: kukicha check [--strict-onerr] [--json] <file.kuki> [file2.kuki ...]")
		os.Exit(1)
	}
	checkCommand(checkArgs, *strictOnerr, *jsonOutput)
}

// checkCommand runs semantic analysis on one or more files/directories and reports
// diagnostics. With --json it emits a JSON array; otherwise plain text.
// With --strict-onerr, warnings are promoted to errors (exit 1).
func checkCommand(targets []string, strictOnerr bool, jsonOutput bool) {
	allDiags := checkFileDiagnostics(targets)

	if jsonOutput {
		emitJSONDiagnostics(allDiags)
		// Exit 1 if any errors (or warnings when --strict-onerr)
		for _, d := range allDiags {
			if d.Severity == "error" || (strictOnerr && d.Severity == "warning") {
				os.Exit(1)
			}
		}
		return
	}

	// Pretty plain text output with source context
	color := term.IsTerminal(int(os.Stderr.Fd()))
	sourceCache := loadSourceCache(allDiags)
	hasErrors := false
	for _, d := range allDiags {
		if d.Severity == "error" {
			hasErrors = true
		}
		lines := sourceCache[d.File]
		fmt.Fprint(os.Stderr, d.RenderPretty(lines, color))
	}

	if hasErrors {
		os.Exit(1)
	}
	if strictOnerr {
		for _, d := range allDiags {
			if d.Severity == "warning" {
				fmt.Fprintln(os.Stderr, "onerr warnings promoted to errors (--strict-onerr)")
				os.Exit(1)
			}
		}
	}

	if len(targets) == 1 {
		fmt.Printf("✓ %s type checks successfully\n", targets[0])
	} else {
		fmt.Printf("✓ all %d targets type check successfully\n", len(targets))
	}
}

// checkFileDiagnostics resolves each target (file or directory), analyzes it, and
// returns all structured diagnostics. Each target is analyzed independently so that
// errors in one file do not suppress diagnostics from others.
func checkFileDiagnostics(targets []string) []semantic.Diagnostic {
	var all []semantic.Diagnostic
	for _, target := range targets {
		all = append(all, analyzeTarget(target)...)
	}
	return all
}

// analyzeTarget analyzes a single file-or-directory target and returns its diagnostics.
func analyzeTarget(target string) []semantic.Diagnostic {
	files, isDir, err := resolveKukiFiles(target)
	if err != nil {
		files = []string{target}
		isDir = false
	}
	if isDir {
		files, err = resolveKukiFilesRecursive(target)
		if err != nil {
			return []semantic.Diagnostic{{
				File:     target,
				Severity: "error",
				Message:  err.Error(),
			}}
		}
	}

	// Parse all files, collecting diagnostics from failures but continuing
	// so that one broken file doesn't suppress diagnostics from others.
	programs := make([]*ast.Program, 0, len(files))
	var parseDiags []semantic.Diagnostic
	for _, f := range files {
		absF, absErr := filepath.Abs(f)
		if absErr != nil {
			absF = f
		}
		source, readErr := os.ReadFile(absF)
		if readErr != nil {
			parseDiags = append(parseDiags, semantic.Diagnostic{File: absF, Severity: "error", Message: readErr.Error()})
			continue
		}
		p, lexErr := parser.New(string(source), absF)
		if lexErr != nil {
			parseDiags = append(parseDiags, semantic.Diagnostic{File: absF, Severity: "error", Message: lexErr.Error()})
			continue
		}
		program, parseErrors := p.Parse()
		if len(parseErrors) > 0 {
			for _, pe := range parseErrors {
				parseDiags = append(parseDiags, semantic.ParseErrorToDiagnostic(pe.Error()))
			}
			continue
		}
		programs = append(programs, program)
	}
	// If any file failed to parse, return all collected parse diagnostics.
	// We can't proceed to semantic analysis with an incomplete set of files.
	if len(parseDiags) > 0 {
		return parseDiags
	}

	// Merge if multi-file
	var program *ast.Program
	if len(programs) == 1 {
		program = programs[0]
	} else {
		merged, mergeErr := mergePrograms(programs, files)
		if mergeErr != nil {
			absF, _ := filepath.Abs(files[0])
			return []semantic.Diagnostic{{File: absF, Severity: "error", Message: mergeErr.Error()}}
		}
		program = merged
	}

	absFirst, _ := filepath.Abs(files[0])
	analyzer := semantic.NewWithFile(program, absFirst)
	analyzer.Analyze()
	return analyzer.Diagnostics()
}

// emitJSONDiagnostics writes the diagnostics slice as a JSON array to stdout.
// An empty slice produces [].
func emitJSONDiagnostics(diags []semantic.Diagnostic) {
	if len(diags) == 0 {
		fmt.Println("[]")
		return
	}
	b, err := jsonMarshalDiagnostics(diags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshalling diagnostics: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(b))
}

// jsonMarshalDiagnostics marshals diagnostics to indented JSON.
func jsonMarshalDiagnostics(diags []semantic.Diagnostic) ([]byte, error) {
	return json.MarshalIndent(diags, "", "  ")
}

// loadSourceCache reads each unique file referenced by diagnostics and returns
// a map of file path to pre-split source lines. Files that can't be read are
// silently skipped (RenderPretty handles nil gracefully).
func loadSourceCache(diags []semantic.Diagnostic) map[string][]string {
	cache := make(map[string][]string)
	for _, d := range diags {
		if d.File == "" {
			continue
		}
		if _, ok := cache[d.File]; ok {
			continue
		}
		data, err := os.ReadFile(d.File)
		if err != nil {
			cache[d.File] = nil
			continue
		}
		cache[d.File] = strings.Split(string(data), "\n")
	}
	return cache
}
