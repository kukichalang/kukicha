package main

//go:generate sh -c "cd ../.. && go run ./cmd/genstdlibregistry"
//go:generate sh -c "cd ../.. && go run ./cmd/gengostdlib"

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/kukichalang/kukicha/internal/ast"
	"github.com/kukichalang/kukicha/internal/codegen"
	"github.com/kukichalang/kukicha/internal/parser"
	"github.com/kukichalang/kukicha/internal/semantic"
	"github.com/kukichalang/kukicha/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "build":
		buildFlags := flag.NewFlagSet("build", flag.ContinueOnError)
		buildFlags.SetOutput(os.Stderr)
		target := buildFlags.String("target", "", "Compile target")
		skipBuild := buildFlags.Bool("skip-build", false, "Skip go build step (for test files)")
		ifChanged := buildFlags.Bool("if-changed", false, "Skip writing output if Go body (excluding generated header) is unchanged")
		vulncheck := buildFlags.Bool("vulncheck", false, "Run govulncheck after successful build")
		wasm := buildFlags.Bool("wasm", false, "Build for WebAssembly (GOOS=js GOARCH=wasm)")
		if err := buildFlags.Parse(args); err != nil {
			fmt.Fprintln(os.Stderr, "Usage: kukicha build [--target <target>] [--skip-build] [--if-changed] [--vulncheck] [--wasm] <file.kuki>")
			os.Exit(1)
		}
		buildArgs := buildFlags.Args()
		if len(buildArgs) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: kukicha build [--target <target>] [--skip-build] [--if-changed] [--vulncheck] [--wasm] <file.kuki>")
			os.Exit(1)
		}
		buildCommand(buildArgs[0], *target, *skipBuild, *ifChanged, *vulncheck, *wasm)
	case "run":
		runFlags := flag.NewFlagSet("run", flag.ContinueOnError)
		runFlags.SetOutput(os.Stderr)
		target := runFlags.String("target", "", "Run target")
		if err := runFlags.Parse(args); err != nil {
			fmt.Fprintln(os.Stderr, "Usage: kukicha run [--target <target>] <file.kuki> [args...]")
			os.Exit(1)
		}
		runArgs := runFlags.Args()
		if len(runArgs) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: kukicha run [--target <target>] <file.kuki> [args...]")
			os.Exit(1)
		}
		runCommand(runArgs[0], *target, runArgs[1:])
	case "check":
		checkFlags := flag.NewFlagSet("check", flag.ContinueOnError)
		checkFlags.SetOutput(os.Stderr)
		strictOnerr := checkFlags.Bool("strict-onerr", false, "Treat onerr lint warnings as errors")
		if err := checkFlags.Parse(args); err != nil {
			fmt.Fprintln(os.Stderr, "Usage: kukicha check [--strict-onerr] <file.kuki>")
			os.Exit(1)
		}
		checkArgs := checkFlags.Args()
		if len(checkArgs) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: kukicha check [--strict-onerr] <file.kuki>")
			os.Exit(1)
		}
		checkCommand(checkArgs[0], *strictOnerr)
	case "fmt":
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: kukicha fmt [options] <files>")
			os.Exit(1)
		}
		fmtCommand(args)
	case "pack":
		packFlags := flag.NewFlagSet("pack", flag.ContinueOnError)
		packFlags.SetOutput(os.Stderr)
		outputDir := packFlags.String("output", "", "Output directory")
		if err := packFlags.Parse(args); err != nil {
			fmt.Fprintln(os.Stderr, "Usage: kukicha pack [--output <dir>] <skill.kuki>")
			os.Exit(1)
		}
		packArgs := packFlags.Args()
		if len(packArgs) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: kukicha pack [--output <dir>] <skill.kuki>")
			os.Exit(1)
		}
		packCommand(packArgs[0], *outputDir)
	case "audit":
		auditFlags := flag.NewFlagSet("audit", flag.ContinueOnError)
		auditFlags.SetOutput(os.Stderr)
		jsonFlag := auditFlags.Bool("json", false, "Output in JSON format")
		warnOnly := auditFlags.Bool("warn-only", false, "Exit 0 even if vulnerabilities are found")
		if err := auditFlags.Parse(args); err != nil {
			fmt.Fprintln(os.Stderr, "Usage: kukicha audit [--json] [--warn-only] [dir]")
			os.Exit(1)
		}
		auditCommand(auditFlags.Args(), *jsonFlag, *warnOnly)
	case "init":
		initCommand(args)
	case "version":
		fmt.Printf("kukicha version %s\n", version.Version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Kukicha - A transpiler that compiles Kukicha to Go")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  kukicha build [--target t] [--vulncheck] [--wasm] <file.kuki|dir>  Compile Kukicha to Go")
	fmt.Fprintln(os.Stderr, "  kukicha run [--target t] <file.kuki|dir>   Transpile and execute Kukicha")
	fmt.Fprintln(os.Stderr, "  kukicha check <file.kuki|dir>   Type check Kukicha")
	fmt.Fprintln(os.Stderr, "  kukicha audit [--json] [--warn-only] [dir]  Check dependencies for vulnerabilities")
	fmt.Fprintln(os.Stderr, "  kukicha fmt [options] <files>  Fix indentation and normalize style")
	fmt.Fprintln(os.Stderr, "    -w          Write result to file instead of stdout")
	fmt.Fprintln(os.Stderr, "    --check     Check if files are formatted (exit 1 if not)")
	fmt.Fprintln(os.Stderr, "  kukicha pack [--output dir] <skill.kuki>  Package skill for distribution")
	fmt.Fprintln(os.Stderr, "  kukicha init [module-name]  Initialize project (go mod init + extract stdlib)")
	fmt.Fprintln(os.Stderr, "  kukicha version             Show version information")
	fmt.Fprintln(os.Stderr, "  kukicha help                Show this help message")
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

	p, err := parser.New(string(source), filename)
	if err != nil {
		return nil, fmt.Errorf("lexer error in %s: %v", filename, err)
	}

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
func loadAndAnalyzeMulti(files []string) (*ast.Program, map[ast.Expression]int, map[ast.Expression]*semantic.TypeInfo, error) {
	programs := make([]*ast.Program, 0, len(files))
	for _, f := range files {
		absF, err := filepath.Abs(f)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("error resolving path %s: %v", f, err)
		}
		prog, err := parseFile(absF)
		if err != nil {
			return nil, nil, nil, err
		}
		programs = append(programs, prog)
	}

	merged, err := mergePrograms(programs, files)
	if err != nil {
		return nil, nil, nil, err
	}

	analyzer := semantic.NewWithFile(merged, files[0])
	semanticErrors := analyzer.Analyze()
	if len(semanticErrors) > 0 {
		var msgs []string
		for _, e := range semanticErrors {
			msgs = append(msgs, fmt.Sprintf("  %v", e))
		}
		return nil, nil, nil, fmt.Errorf("semantic errors:\n%s", strings.Join(msgs, "\n"))
	}

	return merged, analyzer.ReturnCounts(), analyzer.ExprTypes(), nil
}

func loadAndAnalyze(filename string) (*ast.Program, map[ast.Expression]int, map[ast.Expression]*semantic.TypeInfo, error) {
	source, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error reading file: %v", err)
	}

	p, err := parser.New(string(source), filename)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("lexer error: %v", err)
	}

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		var msgs []string
		for _, e := range parseErrors {
			msgs = append(msgs, fmt.Sprintf("  %v", e))
		}
		return nil, nil, nil, fmt.Errorf("parse errors:\n%s", strings.Join(msgs, "\n"))
	}

	analyzer := semantic.NewWithFile(program, filename)
	semanticErrors := analyzer.Analyze()
	if len(semanticErrors) > 0 {
		var msgs []string
		for _, e := range semanticErrors {
			msgs = append(msgs, fmt.Sprintf("  %v", e))
		}
		return nil, nil, nil, fmt.Errorf("semantic errors:\n%s", strings.Join(msgs, "\n"))
	}

	return program, analyzer.ReturnCounts(), analyzer.ExprTypes(), nil
}

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
func compile(filename, targetFlag, defaultTarget string) compileResult {
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
	var projectDir string
	if isDir {
		// In directory mode, the directory itself is the project root
		projectDir = findProjectDir(filepath.Join(absFile, "main.kuki"))
	} else {
		projectDir = findProjectDir(absFile)
	}

	var program *ast.Program
	var returnCounts map[ast.Expression]int
	var exprTypes map[ast.Expression]*semantic.TypeInfo
	if isDir {
		program, returnCounts, exprTypes, err = loadAndAnalyzeMulti(files)
	} else {
		program, returnCounts, exprTypes, err = loadAndAnalyze(absFile)
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
	gen.SetExprReturnCounts(returnCounts)
	gen.SetExprTypes(exprTypes)
	if program.Target == "mcp" {
		gen.SetMCPTarget(true)
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

// rewriteGoErrors replaces references to the generated .go file path in stderr
// output with the original .kuki source path. While //line directives handle
// most source mapping, some Go compiler errors reference the physical file path
// directly (e.g., package-level errors, import failures, syntax errors in
// generated code). This function catches those residual references.
//
// The replacement is intentionally simple (strings.ReplaceAll) because Go error
// formats vary across versions and tools (go build, go vet, etc.). A regex
// approach would need to track multiple formats and could miss edge cases.
// The broad replacement is safe because goFile is a unique temp/output path
// that won't appear in error messages for any other reason.
func rewriteGoErrors(stderr []byte, goFile, kukiFile string) []byte {
	if len(stderr) == 0 {
		return stderr
	}
	result := strings.ReplaceAll(string(stderr), goFile, kukiFile)
	return []byte(result)
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

func buildCommand(filename string, targetFlag string, skipBuild bool, ifChanged bool, vulncheck bool, wasm bool) {
	cr := compile(filename, targetFlag, "")

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
				return // body unchanged — preserve old version comment, skip write+build
			}
		}
	}

	if err := os.WriteFile(outputFile, cr.formatted, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	if isDir {
		fmt.Printf("Successfully compiled %s/ to %s\n", cr.absFile, outputFile)
	} else {
		fmt.Printf("Successfully compiled %s to %s\n", cr.absFile, outputFile)
	}

	ensureStdlibIfNeeded(cr.goCode, cr.projectDir)

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
	binaryPath := filepath.Join(cr.projectDir, binaryName)

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
			os.Exit(1)
		}

		fmt.Printf("Successfully built binary: %s\n", binaryName)

		if wasm {
			wasmScaffold(cr.projectDir, strings.TrimSuffix(filepath.Base(cr.absFile), ".kuki"))
		}
	}

	if vulncheck {
		code := runAudit(AuditOptions{Dir: cr.projectDir})
		if code != 0 {
			os.Exit(code)
		}
	}
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

func runCommand(filename string, targetFlag string, scriptArgs []string) {
	cr := compile(filename, targetFlag, "")

	// If stdlib is needed, extract it and ensure go.mod is configured.
	// Keep temp source in project context so local replace directives resolve.
	var tmp *os.File
	var err error
	if needsStdlib(cr.goCode, cr.projectDir) {
		ensureStdlibIfNeeded(cr.goCode, cr.projectDir)
		tmp, err = os.CreateTemp(filepath.Join(cr.projectDir, ".kukicha"), "run-*.go")
	} else {
		tmp, err = os.CreateTemp("", "kukicha-run-*.go")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temporary file: %v\n", err)
		os.Exit(1)
	}
	tmpFile := tmp.Name()
	defer os.Remove(tmpFile)

	if _, err := tmp.Write(cr.formatted); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing temporary file: %v\n", err)
		os.Exit(1)
	}
	if err := tmp.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Error closing temporary file: %v\n", err)
		os.Exit(1)
	}

	// Run with go run. Use -mod=mod so Go updates go.sum automatically when
	// stdlib transitive dependencies (e.g. gopkg.in/yaml.v3) are not yet listed.
	goArgs := append([]string{"run", "-mod=mod", tmpFile}, scriptArgs...)
	cmd := exec.Command("go", goArgs...)
	cmd.Dir = cr.projectDir
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if stderrBuf.Len() > 0 {
		rewritten := rewriteGoErrors(stderrBuf.Bytes(), tmpFile, cr.absFile)
		rewritten = rewriteVarNames(rewritten, cr.varMap)
		os.Stderr.Write(rewritten)
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

func checkCommand(filename string, strictOnerr bool) {
	files, isDir, err := resolveKukiFiles(filename)
	if err != nil {
		// Fall back to treating as a single file for better error messages
		files = []string{filename}
		isDir = false
	}

	if isDir {
		files, err = resolveKukiFilesRecursive(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	var program *ast.Program
	if isDir || len(files) > 1 {
		program, _, _, err = loadAndAnalyzeMulti(files)
	} else {
		program, _, _, err = loadAndAnalyze(files[0])
	}
	_ = program // check only — no codegen needed
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Re-run analysis to get warnings (loadAndAnalyze* already validated)
	analyzer := semantic.NewWithFile(program, files[0])
	_ = analyzer.Analyze() // errors already checked above
	warnings := analyzer.Warnings()
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %v\n", w)
	}
	if strictOnerr && len(warnings) > 0 {
		fmt.Fprintln(os.Stderr, "onerr warnings promoted to errors (--strict-onerr)")
		os.Exit(1)
	}

	fmt.Printf("✓ %s type checks successfully\n", filename)
}
