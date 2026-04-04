package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kukichalang/kukicha/internal/blend"
)

func main() {
	fs := flag.NewFlagSet("kukicha-blend", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	apply := fs.Bool("apply", false, "Convert .go files to .kuki (writes new files)")
	diff := fs.Bool("diff", false, "Show diff of what would change (no files written)")
	patterns := fs.String("patterns", "", "Comma-separated patterns to blend (default: all)\n  operators   &&, ||, ! → and, or, not\n  comparisons == , !=, nil → equals, isnt, empty\n  types       []T, map[K]V, *T → list of, map of, reference\n  onerr       if err != nil { return } → onerr return\n  package     package → petiole")

	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	args := fs.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	ps := blend.ParsePatterns(*patterns)
	exitCode := 0

	for _, arg := range args {
		files, err := resolveGoFiles(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			exitCode = 1
			continue
		}

		for _, f := range files {
			code := processFile(f, ps, *apply, *diff)
			if code > exitCode {
				exitCode = code
			}
		}
	}

	os.Exit(exitCode)
}

func processFile(filename string, ps blend.PatternSet, applyMode, diffMode bool) int {
	src, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", filename, err)
		return 1
	}

	suggestions, err := blend.BlendFile(filename, src, ps)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", filename, err)
		return 1
	}

	if len(suggestions) == 0 {
		if !applyMode && !diffMode {
			fmt.Printf("%s: no suggestions\n", filename)
		}
		return 0
	}

	if applyMode {
		return applyFile(filename, src, suggestions)
	}

	if diffMode {
		return showDiff(filename, src, suggestions)
	}

	// Default: diagnostic mode — print suggestions
	return printSuggestions(filename, suggestions)
}

func printSuggestions(filename string, suggestions []blend.Suggestion) int {
	fmt.Printf("%s: %d suggestion(s)\n", filename, len(suggestions))
	for _, s := range suggestions {
		fmt.Printf("  %s:%d:%d [%s] %s\n", s.File, s.Line, s.Col, s.Pattern, s.Message)
	}
	return 0
}

func applyFile(filename string, src []byte, suggestions []blend.Suggestion) int {
	result := blend.Apply(src, suggestions)
	outFile := strings.TrimSuffix(filename, ".go") + ".kuki"
	if err := os.WriteFile(outFile, result, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outFile, err)
		return 1
	}
	fmt.Printf("Blended %s → %s (%d changes)\n", filename, outFile, len(suggestions))
	return 0
}

func showDiff(filename string, src []byte, suggestions []blend.Suggestion) int {
	result := blend.Apply(src, suggestions)
	kukiName := strings.TrimSuffix(filename, ".go") + ".kuki"
	d := blend.Diff(filename, kukiName, src, result)
	if d == "" {
		return 0
	}
	fmt.Print(d)
	return 0
}

func resolveGoFiles(arg string) ([]string, error) {
	info, err := os.Stat(arg)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if !strings.HasSuffix(arg, ".go") {
			return nil, fmt.Errorf("%s is not a .go file", arg)
		}
		if strings.HasSuffix(arg, "_test.go") {
			return nil, fmt.Errorf("%s is a test file (skipped)", arg)
		}
		return []string{arg}, nil
	}

	entries, err := os.ReadDir(arg)
	if err != nil {
		return nil, fmt.Errorf("error reading directory: %v", err)
	}
	var files []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, filepath.Join(arg, name))
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no .go files found in %s", arg)
	}
	return files, nil
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "kukicha-blend — Blend Kukicha idioms into existing Go code")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  kukicha-blend [flags] <file.go|dir> [...]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  --apply              Convert .go files to .kuki (writes new files)")
	fmt.Fprintln(os.Stderr, "  --diff               Show unified diff of what would change")
	fmt.Fprintln(os.Stderr, "  --patterns <list>    Comma-separated patterns (default: all)")
	fmt.Fprintln(os.Stderr, "                         operators, comparisons, types, onerr, package")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  kukicha-blend main.go                     # Show suggestions")
	fmt.Fprintln(os.Stderr, "  kukicha-blend --diff ./pkg/               # Preview changes")
	fmt.Fprintln(os.Stderr, "  kukicha-blend --apply main.go             # Write main.kuki")
	fmt.Fprintln(os.Stderr, "  kukicha-blend --patterns=onerr main.go    # Only error handling")
}
