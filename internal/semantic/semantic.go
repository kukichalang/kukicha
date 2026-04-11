package semantic

import (
	"fmt"

	"github.com/kukichalang/kukicha/internal/ast"
)

// Analyzer performs semantic analysis on the AST
type Analyzer struct {
	// Immutable inputs
	program    *ast.Program
	sourceFile string // Source file path, used to detect stdlib context

	// Infrastructure (shared across passes)
	symbolTable *SymbolTable
	security    *SecurityChecker
	errors      []error
	warnings    []error // Non-fatal diagnostics (e.g. risky onerr handlers)

	// Pre-pass output (set once by CollectDirectives)
	directives *DirectiveResult

	// Pass 1 output (collectDeclarations → analyzeDeclarations)
	importAliases map[string]string // alias → base package name (e.g., "strpkg" → "string")

	// Pass 2 transient state (save/restore during tree walk)
	currentFunc        *ast.FunctionDecl // Track current function for return type checking
	loopDepth          int               // Track loop nesting for break/continue
	switchDepth        int               // Track switch nesting for break
	inOnerr            bool              // True while analyzing an onerr handler
	currentOnerrAlias string            // Named alias for caught error in current onerr block (e.g., "e" for "onerr as e")
	inPipedSwitch      bool              // True while analyzing piped switch case bodies (suppresses return-count checks)
	allowIsBinding     bool              // True while analyzing an if condition (permits `is CaseName as v`)

	// Pass 2 output (consumed by codegen)
	exprReturnCounts map[ast.Expression]int      // Inferred return counts for expressions
	exprTypes        map[ast.Expression]*TypeInfo // Inferred types for expressions

	// Lint candidates (collected during analysis, emitted in final pass)
	lintCandidates []LintCandidate
}

// New creates a new semantic analyzer
func New(program *ast.Program) *Analyzer {
	a := &Analyzer{
		program:     program,
		symbolTable: NewSymbolTable(),
		errors:      []error{},
	}
	a.security = &SecurityChecker{analyzer: a}
	return a
}

// NewWithFile creates a new semantic analyzer with the source file path.
// The file path is used to allow Kukicha stdlib packages to use Go stdlib names.
func NewWithFile(program *ast.Program, sourceFile string) *Analyzer {
	a := &Analyzer{
		program:     program,
		symbolTable: NewSymbolTable(),
		errors:      []error{},
		sourceFile:  sourceFile,
	}
	a.security = &SecurityChecker{analyzer: a}
	return a
}

// Analyze performs semantic analysis on the program
func (a *Analyzer) Analyze() []error {
	a.exprReturnCounts = make(map[ast.Expression]int)
	a.exprTypes = make(map[ast.Expression]*TypeInfo)

	// Check package name for collisions with Go stdlib
	a.checkPackageName()

	// Validate skill declaration if present
	a.checkSkillDecl()

	// Pre-pass: collect directives from declarations
	a.directives = CollectDirectives(a.program)
	for _, tw := range a.directives.TodoWarnings {
		a.recordLint(LintTodo, tw.Pos, tw.Message)
	}

	// First pass: Collect all type and interface declarations
	a.collectDeclarations()

	// Second pass: Analyze function bodies and validate
	a.analyzeDeclarations()

	// Final pass: emit lint warnings collected during analysis
	a.emitLintWarnings()

	return a.errors
}

// directiveMessage returns the message from a specific directive, or "".
func directiveMessage(dirs []ast.Directive, name string) string {
	for _, d := range dirs {
		if d.Name == name {
			if len(d.Args) > 0 {
				return d.Args[0]
			}
			return name
		}
	}
	return ""
}

func (a *Analyzer) recordReturnCount(expr ast.Expression, count int) {
	if expr == nil || count < 0 {
		return
	}
	a.exprReturnCounts[expr] = count
}

func (a *Analyzer) recordType(expr ast.Expression, info *TypeInfo) {
	if expr == nil || info == nil {
		return
	}
	// Do not overwrite TypeKindNil; we need it to identify the empty keyword in codegen.
	if existing, ok := a.exprTypes[expr]; ok && existing.Kind == TypeKindNil && info.Kind != TypeKindNil {
		return
	}
	a.exprTypes[expr] = info
}

func (a *Analyzer) error(pos ast.Position, message string) {
	err := fmt.Errorf("%s:%d:%d: %s", pos.File, pos.Line, pos.Column, message)
	a.errors = append(a.errors, err)
}

func (a *Analyzer) warn(pos ast.Position, message string) {
	w := fmt.Errorf("%s:%d:%d: %s", pos.File, pos.Line, pos.Column, message)
	a.warnings = append(a.warnings, w)
}

// SymbolTable returns the analyzer's symbol table (read-only after Analyze).
func (a *Analyzer) SymbolTable() *SymbolTable {
	return a.symbolTable
}

// AnalysisResult bundles all outputs from semantic analysis.
type AnalysisResult struct {
	Errors           []error
	Warnings         []error
	ExprReturnCounts map[ast.Expression]int
	ExprTypes        map[ast.Expression]*TypeInfo
}

// AnalyzeResult runs Analyze() and returns all outputs in a single struct.
func (a *Analyzer) AnalyzeResult() *AnalysisResult {
	errs := a.Analyze()
	return &AnalysisResult{
		Errors:           errs,
		Warnings:         a.warnings,
		ExprReturnCounts: a.exprReturnCounts,
		ExprTypes:        a.exprTypes,
	}
}
