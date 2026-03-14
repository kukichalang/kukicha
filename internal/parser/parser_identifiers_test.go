package parser

import (
	"github.com/duber000/kukicha/internal/ast"
	"testing"
)

func TestErrorAsVariableName(t *testing.T) {
	input := `func Main()
    error := 5
`
	p, err := New(input, "test.kuki")
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}
	program, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("expected no parse errors, got: %v", errs)
	}

	// Find the VarDeclStmt
	fn := program.Declarations[0].(*ast.FunctionDecl)
	varDecl, ok := fn.Body.Statements[0].(*ast.VarDeclStmt)
	if !ok {
		t.Fatalf("expected VarDeclStmt, got %T", fn.Body.Statements[0])
	}
	if varDecl.Names[0].Value != "error" {
		t.Errorf("expected name 'error', got %q", varDecl.Names[0].Value)
	}
}

func TestEmptyAsVariableName(t *testing.T) {
	input := `func Main()
    empty := 10
`
	p, err := New(input, "test.kuki")
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}
	program, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("expected no parse errors, got: %v", errs)
	}

	fn := program.Declarations[0].(*ast.FunctionDecl)
	varDecl, ok := fn.Body.Statements[0].(*ast.VarDeclStmt)
	if !ok {
		t.Fatalf("expected VarDeclStmt, got %T", fn.Body.Statements[0])
	}
	if varDecl.Names[0].Value != "empty" {
		t.Errorf("expected name 'empty', got %q", varDecl.Names[0].Value)
	}
}

func TestParseEmptyIdentifierInPipeExpr(t *testing.T) {
	input := `func Run()
    empty := 10
    empty |> print
`
	p, err := New(input, "test.kuki")
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}
	program, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("expected no parse errors, got: %v", errs)
	}

	fn := program.Declarations[0].(*ast.FunctionDecl)
	stmt, ok := fn.Body.Statements[1].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("expected ExpressionStmt, got %T", fn.Body.Statements[1])
	}
	pipeExpr, ok := stmt.Expression.(*ast.PipeExpr)
	if !ok {
		t.Fatalf("expected PipeExpr, got %T", stmt.Expression)
	}
	left, ok := pipeExpr.Left.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected pipe left side to be Identifier, got %T", pipeExpr.Left)
	}
	if left.Value != "empty" {
		t.Fatalf("expected identifier named empty, got %q", left.Value)
	}
}

func TestMultiValueAssignmentWithError(t *testing.T) {
	input := `func Main()
    val, error := f()
`
	p, err := New(input, "test.kuki")
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}
	program, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("expected no parse errors, got: %v", errs)
	}

	fn := program.Declarations[0].(*ast.FunctionDecl)
	varDecl, ok := fn.Body.Statements[0].(*ast.VarDeclStmt)
	if !ok {
		t.Fatalf("expected VarDeclStmt, got %T", fn.Body.Statements[0])
	}
	if len(varDecl.Names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(varDecl.Names))
	}
	if varDecl.Names[1].Value != "error" {
		t.Errorf("expected second name 'error', got %q", varDecl.Names[1].Value)
	}
}

func TestEmptyKeywordStillWorks(t *testing.T) {
	input := `func Main()
    x := empty string
`
	p, err := New(input, "test.kuki")
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}
	program, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("expected no parse errors, got: %v", errs)
	}

	fn := program.Declarations[0].(*ast.FunctionDecl)
	varDecl, ok := fn.Body.Statements[0].(*ast.VarDeclStmt)
	if !ok {
		t.Fatalf("expected VarDeclStmt, got %T", fn.Body.Statements[0])
	}
	// The value should be an EmptyExpr (keyword), not an Identifier
	_, ok = varDecl.Values[0].(*ast.EmptyExpr)
	if !ok {
		t.Fatalf("expected EmptyExpr (keyword), got %T", varDecl.Values[0])
	}
}

func TestErrorKeywordStillWorks(t *testing.T) {
	input := `func Main()
    x := error "msg"
`
	p, err := New(input, "test.kuki")
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}
	program, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("expected no parse errors, got: %v", errs)
	}

	fn := program.Declarations[0].(*ast.FunctionDecl)
	varDecl, ok := fn.Body.Statements[0].(*ast.VarDeclStmt)
	if !ok {
		t.Fatalf("expected VarDeclStmt, got %T", fn.Body.Statements[0])
	}
	// The value should be an ErrorExpr (keyword), not an Identifier
	_, ok = varDecl.Values[0].(*ast.ErrorExpr)
	if !ok {
		t.Fatalf("expected ErrorExpr (keyword), got %T", varDecl.Values[0])
	}
}

func TestDirectiveAttachedToFunction(t *testing.T) {
	input := `# kuki:deprecated "Use NewFunc instead"
func OldFunc() string
    return "old"
`
	p, err := New(input, "test.kuki")
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}
	program, errors := p.Parse()
	if len(errors) > 0 {
		t.Fatalf("parser errors: %v", errors)
	}
	if len(program.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(program.Declarations))
	}
	fn, ok := program.Declarations[0].(*ast.FunctionDecl)
	if !ok {
		t.Fatalf("expected FunctionDecl, got %T", program.Declarations[0])
	}
	if len(fn.Directives) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(fn.Directives))
	}
	d := fn.Directives[0]
	if d.Name != "deprecated" {
		t.Errorf("expected directive name 'deprecated', got %q", d.Name)
	}
	if len(d.Args) != 1 || d.Args[0] != "Use NewFunc instead" {
		t.Errorf("expected args [\"Use NewFunc instead\"], got %v", d.Args)
	}
}

func TestDirectiveAttachedToType(t *testing.T) {
	input := `# kuki:deprecated "Use NewUser instead"
type OldUser
    name string
`
	p, err := New(input, "test.kuki")
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}
	program, errors := p.Parse()
	if len(errors) > 0 {
		t.Fatalf("parser errors: %v", errors)
	}
	if len(program.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(program.Declarations))
	}
	td, ok := program.Declarations[0].(*ast.TypeDecl)
	if !ok {
		t.Fatalf("expected TypeDecl, got %T", program.Declarations[0])
	}
	if len(td.Directives) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(td.Directives))
	}
	if td.Directives[0].Name != "deprecated" {
		t.Errorf("expected directive name 'deprecated', got %q", td.Directives[0].Name)
	}
}

func TestMultipleDirectivesOnFunction(t *testing.T) {
	input := `# kuki:deprecated "Use NewFunc"
# kuki:fix inline
func OldFunc() string
    return "old"
`
	p, err := New(input, "test.kuki")
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}
	program, errors := p.Parse()
	if len(errors) > 0 {
		t.Fatalf("parser errors: %v", errors)
	}
	fn := program.Declarations[0].(*ast.FunctionDecl)
	if len(fn.Directives) != 2 {
		t.Fatalf("expected 2 directives, got %d", len(fn.Directives))
	}
	if fn.Directives[0].Name != "deprecated" {
		t.Errorf("expected first directive 'deprecated', got %q", fn.Directives[0].Name)
	}
	if fn.Directives[1].Name != "fix" {
		t.Errorf("expected second directive 'fix', got %q", fn.Directives[1].Name)
	}
	if len(fn.Directives[1].Args) != 1 || fn.Directives[1].Args[0] != "inline" {
		t.Errorf("expected fix args [\"inline\"], got %v", fn.Directives[1].Args)
	}
}

func TestCommentBeforeDirectiveNotAttached(t *testing.T) {
	input := `# regular comment
# kuki:deprecated "msg"
func Foo() string
    return "foo"
`
	p, err := New(input, "test.kuki")
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}
	program, errors := p.Parse()
	if len(errors) > 0 {
		t.Fatalf("parser errors: %v", errors)
	}
	fn := program.Declarations[0].(*ast.FunctionDecl)
	// Only the directive should attach, not the plain comment
	if len(fn.Directives) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(fn.Directives))
	}
	if fn.Directives[0].Name != "deprecated" {
		t.Errorf("expected directive name 'deprecated', got %q", fn.Directives[0].Name)
	}
}

func TestDirectiveAttachedToInterface(t *testing.T) {
	input := `# kuki:deprecated "Use NewFoo instead"
interface Foo
    Bar() string
`
	p, err := New(input, "test.kuki")
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}
	program, errors := p.Parse()
	if len(errors) > 0 {
		t.Fatalf("parser errors: %v", errors)
	}
	if len(program.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(program.Declarations))
	}
	iface, ok := program.Declarations[0].(*ast.InterfaceDecl)
	if !ok {
		t.Fatalf("expected InterfaceDecl, got %T", program.Declarations[0])
	}
	if len(iface.Directives) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(iface.Directives))
	}
	d := iface.Directives[0]
	if d.Name != "deprecated" {
		t.Errorf("expected directive name 'deprecated', got %q", d.Name)
	}
	if len(d.Args) != 1 || d.Args[0] != "Use NewFoo instead" {
		t.Errorf("expected args [\"Use NewFoo instead\"], got %v", d.Args)
	}
}
