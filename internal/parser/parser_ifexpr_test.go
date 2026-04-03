package parser

import (
	"testing"

	"github.com/kukichalang/kukicha/internal/ast"
)

func TestParseIfExpression(t *testing.T) {
	input := `func Test() string
    x := if true then "yes" else "no"
    return x
`
	p, err := New(input, "test.kuki")
	if err != nil {
		t.Fatal(err)
	}
	program, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	fn := program.Declarations[0].(*ast.FunctionDecl)
	varDecl := fn.Body.Statements[0].(*ast.VarDeclStmt)
	ifExpr, ok := varDecl.Values[0].(*ast.IfExpression)
	if !ok {
		t.Fatalf("expected IfExpression, got %T", varDecl.Values[0])
	}
	if ifExpr.Then == nil || ifExpr.Else == nil {
		t.Fatal("IfExpression must have both then and else branches")
	}
}

func TestParseIfExpressionChained(t *testing.T) {
	input := `func Test() string
    x := if true then "a" else if false then "b" else "c"
    return x
`
	p, err := New(input, "test.kuki")
	if err != nil {
		t.Fatal(err)
	}
	program, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	fn := program.Declarations[0].(*ast.FunctionDecl)
	varDecl := fn.Body.Statements[0].(*ast.VarDeclStmt)
	ifExpr := varDecl.Values[0].(*ast.IfExpression)
	nested, ok := ifExpr.Else.(*ast.IfExpression)
	if !ok {
		t.Fatalf("expected nested IfExpression in else, got %T", ifExpr.Else)
	}
	if nested.Then == nil || nested.Else == nil {
		t.Fatal("nested IfExpression must have both branches")
	}
}

func TestParseIfExpressionMissingThen(t *testing.T) {
	input := `func Test() string
    x := if true "yes" else "no"
    return x
`
	p, err := New(input, "test.kuki")
	if err != nil {
		t.Fatal(err)
	}
	_, errs := p.Parse()
	if len(errs) == 0 {
		t.Fatal("expected error for missing 'then' keyword")
	}
}

func TestParseIfExpressionMissingElse(t *testing.T) {
	input := `func Test() string
    x := if true then "yes"
    return x
`
	p, err := New(input, "test.kuki")
	if err != nil {
		t.Fatal(err)
	}
	_, errs := p.Parse()
	if len(errs) == 0 {
		t.Fatal("expected error for missing 'else' branch")
	}
}
