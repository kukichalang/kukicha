package codegen

import (
	"fmt"
	"strings"
	"github.com/kukichalang/kukicha/internal/semantic"

	"github.com/kukichalang/kukicha/internal/ast"
)

// TypeParameter represents a type parameter for stdlib special transpilation
// This is internal to codegen and separate from the removed ast.TypeParameter
type TypeParameter struct {
	Name        string // Generated name: T, U, V, etc.
	Placeholder string // Original placeholder: "any", "any2", etc.
	Constraint  string // "any", "comparable", "cmp.Ordered"
}

// Generator generates Go code from an AST.
//
// ARCHITECTURE NOTE: Kukicha uses placeholders like "any" and "any2" in stdlib
// function signatures to represent generic type parameters. When generating Go code,
// we detect these placeholders and emit proper Go generics (e.g., [T any, K comparable]).
// This allows Kukicha users to write simple code while getting type-safe Go generics.
//
// The isStdlibIter and placeholderMap fields work together:
//   - isStdlibIter is true when generating stdlib/iterator or stdlib/slice code
//   - placeholderMap maps Kukicha placeholders ("any", "any2") to Go type params ("T", "K")
//   - During type annotation generation, we substitute placeholders with type params
//
// This design keeps Kukicha's "beginner-friendly" goal: users don't write generic syntax,
// but the generated Go code is fully type-safe with proper generic constraints.
// FuncDefaults stores information about a function's default parameter values
type FuncDefaults struct {
	ParamNames    []string         // Parameter names in order
	DefaultValues []ast.Expression // Default values (nil if no default)
	HasVariadic   bool             // Whether the last parameter is variadic
}

// defaultStdlibModuleBase is the module path prefix used to rewrite "stdlib/X"
// imports to their full Go module paths. Override with Generator.SetStdlibModule
// when the kukicha module is forked or vendored under a different path.
const defaultStdlibModuleBase = "github.com/kukichalang/kukicha"

type Generator struct {
	program              *ast.Program
	output               strings.Builder
	indent               int
	placeholderMap       map[string]string        // Maps placeholder names to type param names (e.g., "any" -> "T", "any2" -> "K")
	autoImports          map[string]bool          // Tracks auto-imports needed (e.g., "cmp" for generic constraints)
	pkgAliases           map[string]string        // Maps original package name -> alias when collision detected (e.g., "json" -> "kukijson")
	funcDefaults         map[string]*FuncDefaults // Maps function names to their default parameter info
	isStdlibIter         bool                     // True if generating stdlib/iterator code (enables iter-specific generic transpilation)
	sourceFile           string                   // Source file path for detecting stdlib
	currentFuncName      string                   // Current function being generated (for context-aware decisions)
	currentReturnTypes   []ast.TypeAnnotation     // Return types of current function (for type coercion in returns)
	processingReturnType bool                     // Whether we are currently generating return types
	tempCounter          int                      // Counter for generating unique temporary variable names
	exprReturnCounts     map[ast.Expression]int      // Semantic return counts passed from analyzer (drives onerr multi-value split)
	// exprTypes holds per-expression type info from semantic analysis.
	// Used by isErrorOnlyReturn, inferExprReturnType, inferExprType,
	// pipedSwitchReturnType, empty keyword resolution, and zeroValueForType.
	exprTypes            map[ast.Expression]*semantic.TypeInfo
	mcpTarget            bool                        // True if targeting MCP (Model Context Protocol)
	currentOnErrVar      string                   // Render-time context: set/restored only by renderHandler in lower.go
	currentOnErrAlias    string                   // Render-time context: set/restored only by renderHandler in lower.go
	currentReturnIndex   int                      // Index of return value being generated (-1 if not in return)
	stdlibModuleBase     string                   // Base module path for rewriting "stdlib/X" imports (default: defaultStdlibModuleBase)
	reservedNames        map[string]bool          // User-declared identifiers — uniqueId skips these to avoid collisions
	varMap               map[string]string        // Maps generated temp variable names to source descriptions (for debugging)
	warnings             []error                  // Non-fatal diagnostics collected during code generation
	enumTypes            map[string]bool          // Enum type names for Status.OK → StatusOK rewriting
	stripLineDirectives  bool                     // When true, omit //line directives from generated output
}

// New creates a new code generator
func New(program *ast.Program) *Generator {
	return &Generator{
		program:            program,
		indent:             0,
		autoImports:        make(map[string]bool),
		pkgAliases:         make(map[string]string),
		funcDefaults:       make(map[string]*FuncDefaults),
		stdlibModuleBase:   defaultStdlibModuleBase,
		currentReturnIndex: -1,
		varMap:             make(map[string]string),
		enumTypes:          make(map[string]bool),
	}
}

// VarMap returns the mapping from generated temp variable names to source
// descriptions. Used by the CLI to annotate panic output and error messages.
func (g *Generator) VarMap() map[string]string {
	return g.varMap
}

// warn records a non-fatal diagnostic. Call Warnings() after Generate() to retrieve them.
func (g *Generator) warn(pos ast.Position, message string) {
	var w error
	if pos.File != "" && pos.Line > 0 {
		w = fmt.Errorf("%s:%d:%d: %s", pos.File, pos.Line, pos.Column, message)
	} else {
		w = fmt.Errorf("%s", message)
	}
	g.warnings = append(g.warnings, w)
}

// Warnings returns non-fatal diagnostics collected during code generation.
// Call after Generate().
func (g *Generator) Warnings() []error {
	return g.warnings
}

// childGenerator creates a temporary Generator that shares the parent's semantic
// state (program, auto-imports, aliases, type info) but writes to a fresh output
// buffer at an adjusted indent level. This replaces manual field-by-field copies
// when generating inline code blocks (function literals, arrow lambda bodies).
//
// The child shares the parent's autoImports map by reference, so auto-imports
// discovered during child generation are visible to the parent.
func (g *Generator) childGenerator(extraIndent int) *Generator {
	return &Generator{
		program:             g.program,
		indent:              g.indent + extraIndent,
		placeholderMap:      g.placeholderMap,
		autoImports:         g.autoImports,
		pkgAliases:          g.pkgAliases,
		funcDefaults:        g.funcDefaults,
		isStdlibIter:        g.isStdlibIter,
		sourceFile:          g.sourceFile,
		exprTypes:           g.exprTypes,
		exprReturnCounts:    g.exprReturnCounts,
		currentReturnIndex:  -1,
		stdlibModuleBase:    g.stdlibModuleBase,
		reservedNames:       g.reservedNames,
		enumTypes:           g.enumTypes,
		stripLineDirectives: g.stripLineDirectives,
	}
}

// SetStdlibModule overrides the base module path used when rewriting "stdlib/X"
// imports to full Go module paths. The default is "github.com/kukichalang/kukicha".
// Set this when building a fork or vendoring the kukicha stdlib under a different module name.
func (g *Generator) SetStdlibModule(base string) {
	g.stdlibModuleBase = base
}

// SetSourceFile sets the source file path and detects if special transpilation is needed
func (g *Generator) SetSourceFile(path string) {
	g.sourceFile = path
	// Enable special transpilation for stdlib/iterator files
	g.isStdlibIter = strings.Contains(path, "stdlib/iterator/") || strings.Contains(path, "stdlib\\iterator\\")
	// Note: stdlib/slice uses a different approach - type parameters are detected per-function
}

// isWasmOnlyPackage returns true if the source file belongs to a WASM-only external stdlib package,
// or if the program imports a WASM-only package.
func (g *Generator) isWasmOnlyPackage() bool {
	for pkg := range wasmOnlyPackages {
		if strings.Contains(g.sourceFile, "stdlib/"+pkg+"/") || strings.Contains(g.sourceFile, "stdlib\\"+pkg+"\\") {
			return true
		}
	}
	for _, imp := range g.program.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		if pkgName, ok := strings.CutPrefix(path, "stdlib/"); ok {
			if wasmOnlyPackages[pkgName] {
				return true
			}
		}
	}
	return false
}

// SetAnalysisResult passes all semantic analysis outputs to the generator at once.
func (g *Generator) SetAnalysisResult(r *semantic.AnalysisResult) {
	g.exprReturnCounts = r.ExprReturnCounts
	g.exprTypes = r.ExprTypes
}

// SetMCPTarget enables special codegen for MCP servers (e.g., print to stderr)
func (g *Generator) SetMCPTarget(v bool) {
	g.mcpTarget = v
}

// SetStripLineDirectives disables emission of //line directives in generated Go.
// Useful for production builds where .kuki source files are not present at
// runtime and smaller/cleaner output is preferred over source-mapped errors.
func (g *Generator) SetStripLineDirectives(v bool) {
	g.stripLineDirectives = v
}

// Generate generates Go code from the AST
func (g *Generator) Generate() (string, error) {
	g.output.Reset()

	// Generate header comment
	g.writeLine("// Generated by Kukicha (requires Go 1.26+)")
	g.writeLine("")

	// Emit build constraint for WASM-only packages (e.g., game uses Ebitengine which requires X11 on native Linux)
	if g.isWasmOnlyPackage() {
		g.writeLine("//go:build js")
		g.writeLine("")
	}

	// Generate package declaration
	g.generatePackage()

	// Generate skill metadata comment if present
	g.generateSkillComment()

	// Collect user-declared identifiers so uniqueId can avoid collisions
	g.collectReservedNames()

	// Pre-scan for auto-imports (e.g. net/http for fetch wrappers)
	g.scanForAutoImports()

	// Pre-scan for function defaults (needed for named arguments and default parameter values)
	g.scanForFunctionDefaults()

	// Generate imports (including auto-imports like fmt for string interpolation, print builtin, and onerr explain)
	needsFmt := g.needsStringInterpolation() || g.needsPrintBuiltin() || g.needsExplain()
	needsErrors := g.needsErrorsPackage()
	if len(g.program.Imports) > 0 || needsFmt || needsErrors || len(g.autoImports) > 0 {
		g.writeLine("")
		g.generateImports()
	}

	// Pre-scan enum declarations so dot-access rewriting works regardless of declaration order
	for _, decl := range g.program.Declarations {
		if ed, ok := decl.(*ast.EnumDecl); ok {
			g.enumTypes[ed.Name.Value] = true
		}
	}

	// Register cross-package enum types from stdlib registry so dot-access rewriting
	// works for imported enums (e.g., http.Status.OK → http.StatusOK)
	for _, imp := range g.program.Imports {
		canonicalPkg := extractPkgName(strings.Trim(imp.Path.Value, "\""))
		localPkg := canonicalPkg
		if imp.Alias != nil {
			localPkg = imp.Alias.Value
		}
		// Check all known stdlib enums for this package prefix
		for qualifiedName := range semantic.GetAllStdlibEnums() {
			if strings.HasPrefix(qualifiedName, canonicalPkg+".") {
				// Register as "localPkg.EnumName" so generateFieldAccessExpr can match
				suffix := qualifiedName[len(canonicalPkg)+1:]
				g.enumTypes[localPkg+"."+suffix] = true
			}
		}
	}

	// Generate declarations
	for _, decl := range g.program.Declarations {
		g.writeLine("")
		g.generateDeclaration(decl)
	}

	return g.output.String(), nil
}

func (g *Generator) generatePackage() {
	packageName := "main"
	if g.program.PetioleDecl != nil {
		packageName = g.program.PetioleDecl.Name.Value
	}
	g.writeLine(fmt.Sprintf("package %s", packageName))
}

func (g *Generator) generateSkillComment() {
	skill := g.program.SkillDecl
	if skill == nil {
		return
	}
	g.writeLine("")
	g.writeLine(fmt.Sprintf("// Skill: %s", skill.Name.Value))
	if skill.Description != "" {
		g.writeLine(fmt.Sprintf("// Description: %s", skill.Description))
	}
	if skill.Version != "" {
		g.writeLine(fmt.Sprintf("// Version: %s", skill.Version))
	}
}

func (g *Generator) generateDeclaration(decl ast.Declaration) {
	g.emitLineDirective(decl.Pos())
	switch d := decl.(type) {
	case *ast.TypeDecl:
		g.generateTypeDecl(d)
	case *ast.InterfaceDecl:
		g.generateInterfaceDecl(d)
	case *ast.FunctionDecl:
		g.generateFunctionDecl(d)
	case *ast.VarDeclStmt:
		g.generateGlobalVarDecl(d)
	case *ast.ConstDecl:
		g.generateConstDecl(d)
	case *ast.EnumDecl:
		g.generateEnumDecl(d)
	}
}

func (g *Generator) write(s string) {
	g.output.WriteString(s)
}

func (g *Generator) writeLine(s string) {
	if s != "" {
		g.output.WriteString(g.indentStr() + s)
	}
	g.output.WriteString("\n")
}

func (g *Generator) indentStr() string {
	return strings.Repeat("\t", g.indent)
}

// emitLineDirective writes a //line directive that maps the generated Go code
// back to the original .kuki source file. The Go compiler and runtime honor
// these directives, so compile errors, panics, and stack traces will reference
// the .kuki file instead of the generated .go file.
func (g *Generator) emitLineDirective(pos ast.Position) {
	if !g.stripLineDirectives && pos.Line > 0 && pos.File != "" {
		fmt.Fprintf(&g.output, "//line %s:%d\n", pos.File, pos.Line)
	}
}

// uniqueId generates unique identifiers to prevent variable shadowing.
// It skips names that collide with user-declared variables in reservedNames.
func (g *Generator) uniqueId(prefix string) string {
	for {
		g.tempCounter++
		name := fmt.Sprintf("%s_%d", prefix, g.tempCounter)
		if !g.reservedNames[name] {
			return name
		}
	}
}
