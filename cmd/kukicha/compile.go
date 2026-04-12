package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/kukichalang/kukicha/internal/ast"
	"github.com/kukichalang/kukicha/internal/codegen"
	"github.com/kukichalang/kukicha/internal/parser"
	"github.com/kukichalang/kukicha/internal/semantic"
)

// compileResult holds the output of the shared compile pipeline.
type compileResult struct {
	absFile    string
	projectDir string
	program    *ast.Program
	goCode     string
	formatted  []byte
	varMap     map[string]string // Maps generated temp var names to source descriptions
}

// compile runs the shared pipeline: resolve path, parse, analyze, detect target,
// generate Go code, and format it. targetFlag overrides auto-detection when non-empty.
// defaultTarget is used when no flag is given and no target directive is found in source.
// stripLineDirectives suppresses //line directives in generated output.
func compile(filename, targetFlag, defaultTarget string, stripLineDirectives bool) compileResult {
	files, isDir, err := resolveKukiFiles(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// For directory mode, also include subdirectories
	if isDir {
		files, err = resolveKukiFilesRecursive(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	var absFile string
	if isDir {
		absFile, err = filepath.Abs(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving directory path: %v\n", err)
			os.Exit(1)
		}
	} else {
		absFile, err = filepath.Abs(files[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving file path: %v\n", err)
			os.Exit(1)
		}
	}
	projectDir := findProjectDir(absFile)

	var program *ast.Program
	var result *semantic.AnalysisResult
	if isDir {
		program, result, err = loadAndAnalyzeMulti(files)
	} else {
		program, result, err = loadAndAnalyze(absFile)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Detect target from source if not provided by flag
	if targetFlag != "" {
		program.Target = targetFlag
	} else if !isDir {
		t, readErr := detectTargetFromFile(absFile)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not read %s for target detection: %v\n", absFile, readErr)
		}
		if t != "" {
			program.Target = t
		} else if defaultTarget != "" {
			program.Target = defaultTarget
		}
	} else if defaultTarget != "" {
		program.Target = defaultTarget
	}

	// Generate Go code
	gen := codegen.New(program)
	gen.SetSourceFile(absFile)
	gen.SetProjectDir(projectDir)
	gen.SetAnalysisResult(result)
	if program.Target == "mcp" {
		gen.SetMCPTarget(true)
	}
	if stripLineDirectives {
		gen.SetStripLineDirectives(true)
	}
	goCode, err := gen.Generate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Code generation error: %v\n", err)
		os.Exit(1)
	}

	for _, w := range gen.Warnings() {
		fmt.Fprintf(os.Stderr, "warning: %v\n", w)
	}

	// Format with gofmt
	formatted, err := format.Source([]byte(goCode))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: gofmt failed, using unformatted output: %v\n", err)
		formatted = []byte(goCode)
	}

	return compileResult{
		absFile:    absFile,
		projectDir: projectDir,
		program:    program,
		goCode:     goCode,
		formatted:  formatted,
		varMap:     gen.VarMap(),
	}
}

// ensureStdlibIfNeeded checks if the generated Go code imports Kukicha stdlib
// packages and, if so, extracts the stdlib and configures go.mod.
func ensureStdlibIfNeeded(goCode, projectDir string) {
	if !needsStdlib(goCode, projectDir) {
		return
	}
	stdlibPath, err := ensureStdlib(projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting stdlib: %v\n", err)
		os.Exit(1)
	}
	if err := ensureGoMod(projectDir, stdlibPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating go.mod: %v\n", err)
		os.Exit(1)
	}
}

// resolveKukiFiles determines whether the argument is a single .kuki file or a
// directory. For directories it returns all *.kuki files (excluding *_test.kuki),
// sorted for deterministic output. The boolean indicates directory mode.
func resolveKukiFiles(arg string) ([]string, bool, error) {
	info, err := os.Stat(arg)
	if err != nil {
		return nil, false, err
	}
	if !info.IsDir() {
		return []string{arg}, false, nil
	}
	entries, err := os.ReadDir(arg)
	if err != nil {
		return nil, false, fmt.Errorf("error reading directory: %v", err)
	}
	var files []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".kuki") || strings.HasSuffix(name, "_test.kuki") {
			continue
		}
		files = append(files, filepath.Join(arg, name))
	}
	if len(files) == 0 {
		return nil, false, fmt.Errorf("no .kuki files found in %s", arg)
	}
	sort.Strings(files)
	return files, true, nil
}

// resolveKukiFilesRecursive globs *.kuki files recursively in a directory,
// excluding *_test.kuki. Subdirectory files with the same petiole are included.
func resolveKukiFilesRecursive(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(d.Name(), ".") && path != dir {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if strings.HasSuffix(name, ".kuki") && !strings.HasSuffix(name, "_test.kuki") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking directory: %v", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no .kuki files found in %s", dir)
	}
	sort.Strings(files)
	return files, nil
}

// parseFile reads and parses a single .kuki file, returning its AST.
func parseFile(filename string) (*ast.Program, error) {
	source, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %v", filename, err)
	}

	p, _ := parser.New(string(source), filename)
	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		var msgs []string
		for _, e := range parseErrors {
			msgs = append(msgs, fmt.Sprintf("  %v", e))
		}
		return nil, fmt.Errorf("parse errors:\n%s", strings.Join(msgs, "\n"))
	}
	return program, nil
}

// mergePrograms combines multiple parsed programs into one. All files must have
// the same petiole (or no petiole, defaulting to "main"). Imports are deduplicated.
func mergePrograms(programs []*ast.Program, files []string) (*ast.Program, error) {
	merged := &ast.Program{}

	// Validate petiole consistency
	var petioleName string
	var petioleFile string
	for i, prog := range programs {
		var name string
		if prog.PetioleDecl != nil {
			name = prog.PetioleDecl.Name.Value
		}
		if i == 0 {
			petioleName = name
			petioleFile = files[i]
			merged.PetioleDecl = prog.PetioleDecl
		} else if name != petioleName {
			return nil, fmt.Errorf("petiole mismatch: %s declares %q but %s declares %q",
				petioleFile, petioleName, files[i], name)
		}
	}

	// Deduplicate imports by path+alias
	seen := make(map[string]bool)
	for _, prog := range programs {
		for _, imp := range prog.Imports {
			key := imp.Path.Value
			if imp.Alias != nil {
				key += " as " + imp.Alias.Value
			}
			if !seen[key] {
				seen[key] = true
				merged.Imports = append(merged.Imports, imp)
			}
		}
	}

	// Check for duplicate function declarations and concatenate declarations.
	// We allow multiple "init" functions (as Go does) but ensure other
	// names (including "main") are unique within the merged package.
	seenFuncs := make(map[string]string) // func name -> file where first seen
	for i, prog := range programs {
		for _, decl := range prog.Declarations {
			if funcDecl, ok := decl.(*ast.FunctionDecl); ok {
				funcName := funcDecl.Name.Value
				// Go allows multiple init functions in the same package.
				if funcName != "init" {
					if existingFile, exists := seenFuncs[funcName]; exists {
						return nil, fmt.Errorf("function '%s' already declared in %s (first declared in %s)",
							funcName, files[i], existingFile)
					}
					seenFuncs[funcName] = files[i]
				}
			}
		}
		merged.Declarations = append(merged.Declarations, prog.Declarations...)
	}

	// Take first non-empty target and skill declaration
	for _, prog := range programs {
		if prog.Target != "" && merged.Target == "" {
			merged.Target = prog.Target
		}
		if prog.SkillDecl != nil && merged.SkillDecl == nil {
			merged.SkillDecl = prog.SkillDecl
		}
	}

	return merged, nil
}

// loadAndAnalyzeMulti parses multiple .kuki files, merges them, and runs semantic analysis.
func loadAndAnalyzeMulti(files []string) (*ast.Program, *semantic.AnalysisResult, error) {
	programs := make([]*ast.Program, 0, len(files))
	for _, f := range files {
		absF, err := filepath.Abs(f)
		if err != nil {
			return nil, nil, fmt.Errorf("error resolving path %s: %v", f, err)
		}
		prog, err := parseFile(absF)
		if err != nil {
			return nil, nil, err
		}
		programs = append(programs, prog)
	}

	merged, err := mergePrograms(programs, files)
	if err != nil {
		return nil, nil, err
	}

	analyzer := semantic.NewWithFile(merged, files[0])
	result := analyzer.AnalyzeResult()
	if len(result.Errors) > 0 {
		var msgs []string
		for _, e := range result.Errors {
			msgs = append(msgs, fmt.Sprintf("  %v", e))
		}
		return nil, nil, fmt.Errorf("semantic errors:\n%s", strings.Join(msgs, "\n"))
	}

	return merged, result, nil
}

func loadAndAnalyze(filename string) (*ast.Program, *semantic.AnalysisResult, error) {
	source, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading file: %v", err)
	}

	p, _ := parser.New(string(source), filename)
	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		var msgs []string
		for _, e := range parseErrors {
			msgs = append(msgs, fmt.Sprintf("  %v", e))
		}
		return nil, nil, fmt.Errorf("parse errors:\n%s", strings.Join(msgs, "\n"))
	}

	analyzer := semantic.NewWithFile(program, filename)
	result := analyzer.AnalyzeResult()
	if len(result.Errors) > 0 {
		var msgs []string
		for _, e := range result.Errors {
			msgs = append(msgs, fmt.Sprintf("  %v", e))
		}
		return nil, nil, fmt.Errorf("semantic errors:\n%s", strings.Join(msgs, "\n"))
	}

	return program, result, nil
}

func detectTarget(source string) string {
	lines := strings.Split(source, "\n")
	for i, line := range lines {
		if i >= 10 { // Only look at first 10 lines
			break
		}
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "# target:"); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

func detectTargetFromFile(filename string) (string, error) {
	source, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return detectTarget(string(source)), nil
}

// rewriteGoErrors rewrites residual references to the generated .go file path
// in Go toolchain stderr output. It is reached when //line directives did NOT
// translate a position — typically package-level errors (import failures,
// "declared but not used"), linker errors, go.mod resolution errors, and
// panic frames outside //line-covered statements.
//
// In those cases any :line:col following the .go path refers to the GENERATED
// file, not the .kuki source, so we strip them. Pointing the user at
// "app.kuki:247: undefined: Foo" when line 247 is unrelated would be worse
// than "app.kuki: undefined: Foo". Positions that DID flow through //line
// already reference the .kuki path and are left alone.
func rewriteGoErrors(stderr []byte, goFile, kukiFile string) []byte {
	if len(stderr) == 0 {
		return stderr
	}
	pattern := regexp.QuoteMeta(goFile) + `(?::\d+(?::\d+)?)?`
	re := regexp.MustCompile(pattern)
	return re.ReplaceAll(stderr, []byte(kukiFile))
}

// rewriteVarNames scans stderr for generated temp variable names (pipe_N, err_N)
// and appends a "Variable hints" section mapping them to their source descriptions.
func rewriteVarNames(stderr []byte, varMap map[string]string) []byte {
	if len(stderr) == 0 || len(varMap) == 0 {
		return stderr
	}
	text := string(stderr)

	// Collect temp vars that actually appear in the output.
	var hints []string
	for name, desc := range varMap {
		if strings.Contains(text, name) {
			hints = append(hints, fmt.Sprintf("  %s = %s", name, desc))
		}
	}
	if len(hints) == 0 {
		return stderr
	}

	// Sort for stable output.
	sort.Strings(hints)

	var b strings.Builder
	b.WriteString(text)
	if !strings.HasSuffix(text, "\n") {
		b.WriteByte('\n')
	}
	b.WriteString("\nkukicha: variable hints:\n")
	for _, h := range hints {
		b.WriteString(h)
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

// stripFirstLine removes the first line (including its newline) from b.
// Used to compare generated Go files while ignoring the version header.
func stripFirstLine(b []byte) []byte {
	if _, after, ok := bytes.Cut(b, []byte{'\n'}); ok {
		return after
	}
	return b
}
