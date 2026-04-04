package semantic

import (
	"fmt"

	"github.com/kukichalang/kukicha/internal/ast"
)

// DirectiveResult holds directive data collected from AST declarations.
type DirectiveResult struct {
	DeprecatedFuncs map[string]string // Function name → deprecation message
	DeprecatedTypes map[string]string // Type name → deprecation message
	PanickedFuncs   map[string]string // Function name → panic message
	TodoWarnings    []todoWarning     // TODO warnings to emit
}

type todoWarning struct {
	Pos     ast.Position
	Message string
}

// CollectDirectives scans all declarations for # kuki:deprecated, # kuki:panics,
// and # kuki:todo directives. Returns the collected results.
func CollectDirectives(program *ast.Program) *DirectiveResult {
	result := &DirectiveResult{
		DeprecatedFuncs: make(map[string]string),
		DeprecatedTypes: make(map[string]string),
		PanickedFuncs:   make(map[string]string),
	}

	for _, decl := range program.Declarations {
		switch d := decl.(type) {
		case *ast.FunctionDecl:
			if msg := directiveMessage(d.Directives, "todo"); msg != "" {
				result.TodoWarnings = append(result.TodoWarnings, todoWarning{
					Pos:     d.Pos(),
					Message: fmt.Sprintf("TODO: %q on %s", msg, d.Name.Value),
				})
			}
			if msg := directiveMessage(d.Directives, "deprecated"); msg != "" {
				result.DeprecatedFuncs[d.Name.Value] = msg
			}
			if msg := directiveMessage(d.Directives, "panics"); msg != "" {
				result.PanickedFuncs[d.Name.Value] = msg
			}
		case *ast.TypeDecl:
			if msg := directiveMessage(d.Directives, "todo"); msg != "" {
				result.TodoWarnings = append(result.TodoWarnings, todoWarning{
					Pos:     d.Pos(),
					Message: fmt.Sprintf("TODO: %q on %s", msg, d.Name.Value),
				})
			}
			if msg := directiveMessage(d.Directives, "deprecated"); msg != "" {
				result.DeprecatedTypes[d.Name.Value] = msg
			}
		case *ast.InterfaceDecl:
			if msg := directiveMessage(d.Directives, "todo"); msg != "" {
				result.TodoWarnings = append(result.TodoWarnings, todoWarning{
					Pos:     d.Pos(),
					Message: fmt.Sprintf("TODO: %q on %s", msg, d.Name.Value),
				})
			}
			if msg := directiveMessage(d.Directives, "deprecated"); msg != "" {
				result.DeprecatedTypes[d.Name.Value] = msg
			}
		case *ast.EnumDecl:
			if msg := directiveMessage(d.Directives, "todo"); msg != "" {
				result.TodoWarnings = append(result.TodoWarnings, todoWarning{
					Pos:     d.Pos(),
					Message: fmt.Sprintf("TODO: %q on %s", msg, d.Name.Value),
				})
			}
			if msg := directiveMessage(d.Directives, "deprecated"); msg != "" {
				result.DeprecatedTypes[d.Name.Value] = msg
			}
		}
	}

	return result
}
