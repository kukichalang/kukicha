package parser

import (
	"github.com/duber000/kukicha/internal/ast"
	"testing"
)

func TestParseVariadicParameter(t *testing.T) {
	input := `func Print(many values)
    return values
`

	program := mustParseProgram(t, input)

	fn := program.Declarations[0].(*ast.FunctionDecl)
	if len(fn.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(fn.Parameters))
	}

	param := fn.Parameters[0]
	if param.Name.Value != "values" {
		t.Errorf("expected parameter name 'values', got '%s'", param.Name.Value)
	}

	if !param.Variadic {
		t.Error("expected parameter to be variadic")
	}
}

func TestParseTypedVariadicParameter(t *testing.T) {
	input := `func Sum(many numbers int) int
    return 0
`

	program := mustParseProgram(t, input)

	fn := program.Declarations[0].(*ast.FunctionDecl)
	param := fn.Parameters[0]

	if !param.Variadic {
		t.Error("expected parameter to be variadic")
	}

	primType, ok := param.Type.(*ast.PrimitiveType)
	if !ok {
		t.Fatalf("expected PrimitiveType, got %T", param.Type)
	}

	if primType.Name != "int" {
		t.Errorf("expected type 'int', got '%s'", primType.Name)
	}
}

func TestParseGoBlockSyntax(t *testing.T) {
	input := `func main()
    go
        doSomething()
        doSomethingElse()
`

	program := mustParseProgram(t, input)

	fn := program.Declarations[0].(*ast.FunctionDecl)
	goStmt, ok := fn.Body.Statements[0].(*ast.GoStmt)
	if !ok {
		t.Fatalf("expected GoStmt, got %T", fn.Body.Statements[0])
	}

	if goStmt.Block == nil {
		t.Fatal("expected go block, got nil")
	}

	if goStmt.Call != nil {
		t.Fatal("expected nil Call for block form, got non-nil")
	}

	if len(goStmt.Block.Statements) != 2 {
		t.Fatalf("expected 2 statements in go block, got %d", len(goStmt.Block.Statements))
	}
}

func TestParseGoCallSyntaxStillWorks(t *testing.T) {
	input := `func main()
    go processItem(item)
`

	program := mustParseProgram(t, input)

	fn := program.Declarations[0].(*ast.FunctionDecl)
	goStmt, ok := fn.Body.Statements[0].(*ast.GoStmt)
	if !ok {
		t.Fatalf("expected GoStmt, got %T", fn.Body.Statements[0])
	}

	if goStmt.Call == nil {
		t.Fatal("expected Call for call form, got nil")
	}

	if goStmt.Block != nil {
		t.Fatal("expected nil Block for call form, got non-nil")
	}
}

func TestParseArrowLambdaTypedExpression(t *testing.T) {
	input := `func main()
    f := (r Repo) => r.Stars > 100
`

	program := mustParseProgram(t, input)

	fn := program.Declarations[0].(*ast.FunctionDecl)
	varDecl, ok := fn.Body.Statements[0].(*ast.VarDeclStmt)
	if !ok {
		t.Fatalf("expected VarDeclStmt, got %T", fn.Body.Statements[0])
	}

	lambda, ok := varDecl.Values[0].(*ast.ArrowLambda)
	if !ok {
		t.Fatalf("expected ArrowLambda, got %T", varDecl.Values[0])
	}

	if len(lambda.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(lambda.Parameters))
	}

	if lambda.Parameters[0].Name.Value != "r" {
		t.Errorf("expected param name 'r', got '%s'", lambda.Parameters[0].Name.Value)
	}

	if lambda.Parameters[0].Type == nil {
		t.Fatal("expected typed parameter, got nil type")
	}

	if lambda.Body == nil {
		t.Fatal("expected expression body, got nil")
	}

	if lambda.Block != nil {
		t.Fatal("expected nil block for expression lambda, got non-nil")
	}
}

func TestParseArrowLambdaUntypedSingleParam(t *testing.T) {
	input := `func main()
    f := r => r.Name
`

	program := mustParseProgram(t, input)

	fn := program.Declarations[0].(*ast.FunctionDecl)
	varDecl := fn.Body.Statements[0].(*ast.VarDeclStmt)
	lambda, ok := varDecl.Values[0].(*ast.ArrowLambda)
	if !ok {
		t.Fatalf("expected ArrowLambda, got %T", varDecl.Values[0])
	}

	if len(lambda.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(lambda.Parameters))
	}

	if lambda.Parameters[0].Name.Value != "r" {
		t.Errorf("expected param name 'r', got '%s'", lambda.Parameters[0].Name.Value)
	}

	if lambda.Parameters[0].Type != nil {
		t.Fatal("expected untyped parameter, got typed")
	}
}

func TestParseArrowLambdaZeroParams(t *testing.T) {
	input := `func main()
    f := () => print("hello")
`

	program := mustParseProgram(t, input)

	fn := program.Declarations[0].(*ast.FunctionDecl)
	varDecl := fn.Body.Statements[0].(*ast.VarDeclStmt)
	lambda, ok := varDecl.Values[0].(*ast.ArrowLambda)
	if !ok {
		t.Fatalf("expected ArrowLambda, got %T", varDecl.Values[0])
	}

	if len(lambda.Parameters) != 0 {
		t.Fatalf("expected 0 parameters, got %d", len(lambda.Parameters))
	}
}

func TestParseArrowLambdaMultipleParams(t *testing.T) {
	input := `func main()
    f := (a string, b string) => a < b
`

	program := mustParseProgram(t, input)

	fn := program.Declarations[0].(*ast.FunctionDecl)
	varDecl := fn.Body.Statements[0].(*ast.VarDeclStmt)
	lambda, ok := varDecl.Values[0].(*ast.ArrowLambda)
	if !ok {
		t.Fatalf("expected ArrowLambda, got %T", varDecl.Values[0])
	}

	if len(lambda.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(lambda.Parameters))
	}

	if lambda.Parameters[0].Name.Value != "a" {
		t.Errorf("expected first param name 'a', got '%s'", lambda.Parameters[0].Name.Value)
	}

	if lambda.Parameters[1].Name.Value != "b" {
		t.Errorf("expected second param name 'b', got '%s'", lambda.Parameters[1].Name.Value)
	}
}

func TestParseArrowLambdaBlockForm(t *testing.T) {
	input := `func main()
    f := (r Repo) =>
        name := r.Name
        return name
`

	program := mustParseProgram(t, input)

	fn := program.Declarations[0].(*ast.FunctionDecl)
	varDecl := fn.Body.Statements[0].(*ast.VarDeclStmt)
	lambda, ok := varDecl.Values[0].(*ast.ArrowLambda)
	if !ok {
		t.Fatalf("expected ArrowLambda, got %T", varDecl.Values[0])
	}

	if lambda.Body != nil {
		t.Fatal("expected nil body for block lambda")
	}

	if lambda.Block == nil {
		t.Fatal("expected block body, got nil")
	}

	if len(lambda.Block.Statements) != 2 {
		t.Fatalf("expected 2 statements in block, got %d", len(lambda.Block.Statements))
	}
}

func TestParseArrowLambdaInPipe(t *testing.T) {
	input := `func main()
    result := repos |> slice.Filter((r Repo) => r.Stars > 100)
`

	p, err := New(input, "test.kuki")
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}
	_, errors := p.Parse()

	if len(errors) > 0 {
		t.Fatalf("parser errors: %v", errors)
	}
}

func TestParseBitwiseAndPrecedence(t *testing.T) {
	input := `func main()
    value := a | b & c
`

	program := mustParseProgram(t, input)

	fn := program.Declarations[0].(*ast.FunctionDecl)
	varDecl := fn.Body.Statements[0].(*ast.VarDeclStmt)
	root, ok := varDecl.Values[0].(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("expected BinaryExpr, got %T", varDecl.Values[0])
	}
	if root.Operator != "|" {
		t.Fatalf("expected root operator '|', got %q", root.Operator)
	}

	right, ok := root.Right.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("expected right-hand BinaryExpr, got %T", root.Right)
	}
	if right.Operator != "&" {
		t.Fatalf("expected nested operator '&', got %q", right.Operator)
	}
}

func TestParseBitwiseAndAssign(t *testing.T) {
	input := `func main()
    flags &= mask
`

	program := mustParseProgram(t, input)

	fn := program.Declarations[0].(*ast.FunctionDecl)
	assign, ok := fn.Body.Statements[0].(*ast.AssignStmt)
	if !ok {
		t.Fatalf("expected AssignStmt, got %T", fn.Body.Statements[0])
	}
	if assign.Token.Lexeme != "&=" {
		t.Fatalf("expected '&=' token, got %q", assign.Token.Lexeme)
	}
}

func TestParseGroupedExpressionStillWorks(t *testing.T) {
	// Ensure (x + y) still parses as grouped expression, not lambda
	input := `func main()
    result := (a + b) * c
`

	program := mustParseProgram(t, input)

	fn := program.Declarations[0].(*ast.FunctionDecl)
	varDecl := fn.Body.Statements[0].(*ast.VarDeclStmt)

	// Should NOT be an ArrowLambda
	_, isLambda := varDecl.Values[0].(*ast.ArrowLambda)
	if isLambda {
		t.Fatal("expected grouped expression, got ArrowLambda")
	}
}
