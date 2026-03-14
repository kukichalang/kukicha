package parser

import (
	"github.com/duber000/kukicha/internal/ast"
	"testing"
)

func TestParseSkillDeclSimple(t *testing.T) {
	input := `petiole weather

skill WeatherService
`

	program := mustParseProgram(t, input)

	if program.SkillDecl == nil {
		t.Fatal("expected SkillDecl, got nil")
	}

	if program.SkillDecl.Name.Value != "WeatherService" {
		t.Errorf("expected skill name 'WeatherService', got '%s'", program.SkillDecl.Name.Value)
	}

	if program.SkillDecl.Description != "" {
		t.Errorf("expected empty description, got '%s'", program.SkillDecl.Description)
	}

	if program.SkillDecl.Version != "" {
		t.Errorf("expected empty version, got '%s'", program.SkillDecl.Version)
	}
}

func TestParseSkillDeclWithBlock(t *testing.T) {
	input := `petiole weather

skill WeatherService
    description: "Provides real-time weather data."
    version: "2.1.0"
`

	program := mustParseProgram(t, input)

	if program.SkillDecl == nil {
		t.Fatal("expected SkillDecl, got nil")
	}

	skill := program.SkillDecl
	if skill.Name.Value != "WeatherService" {
		t.Errorf("expected skill name 'WeatherService', got '%s'", skill.Name.Value)
	}

	if skill.Description != "Provides real-time weather data." {
		t.Errorf("expected description 'Provides real-time weather data.', got '%s'", skill.Description)
	}

	if skill.Version != "2.1.0" {
		t.Errorf("expected version '2.1.0', got '%s'", skill.Version)
	}
}

func TestParseSkillDeclDescriptionOnly(t *testing.T) {
	input := `petiole myskill

skill MySkill
    description: "A test skill."
`

	program := mustParseProgram(t, input)

	skill := program.SkillDecl
	if skill == nil {
		t.Fatal("expected SkillDecl, got nil")
	}

	if skill.Description != "A test skill." {
		t.Errorf("expected description 'A test skill.', got '%s'", skill.Description)
	}

	if skill.Version != "" {
		t.Errorf("expected empty version, got '%s'", skill.Version)
	}
}

func TestParseOnErrExplainStandalone(t *testing.T) {
	input := `func Test() (string, error)
    x := foo() onerr explain "City names must be capitalized"
    return x, nil
`

	program := mustParseProgram(t, input)

	fn := program.Declarations[0].(*ast.FunctionDecl)
	varDecl, ok := fn.Body.Statements[0].(*ast.VarDeclStmt)
	if !ok {
		t.Fatalf("expected VarDeclStmt, got %T", fn.Body.Statements[0])
	}

	if varDecl.OnErr == nil {
		t.Fatal("expected OnErr clause, got nil")
	}

	if varDecl.OnErr.Explain != "City names must be capitalized" {
		t.Errorf("expected explain 'City names must be capitalized', got '%s'", varDecl.OnErr.Explain)
	}

	// Standalone explain has nil handler
	if varDecl.OnErr.Handler != nil {
		t.Errorf("expected nil handler for standalone explain, got %T", varDecl.OnErr.Handler)
	}
}

func TestParseOnErrWithHandlerAndExplain(t *testing.T) {
	input := `func Test()
    x := foo() onerr 0 explain "Expected a positive integer"
`

	program := mustParseProgram(t, input)

	fn := program.Declarations[0].(*ast.FunctionDecl)
	varDecl, ok := fn.Body.Statements[0].(*ast.VarDeclStmt)
	if !ok {
		t.Fatalf("expected VarDeclStmt, got %T", fn.Body.Statements[0])
	}

	if varDecl.OnErr == nil {
		t.Fatal("expected OnErr clause, got nil")
	}

	if varDecl.OnErr.Handler == nil {
		t.Fatal("expected handler, got nil")
	}

	// Handler should be the integer literal 0
	intLit, ok := varDecl.OnErr.Handler.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral handler, got %T", varDecl.OnErr.Handler)
	}
	if intLit.Value != 0 {
		t.Errorf("expected handler value 0, got %d", intLit.Value)
	}

	if varDecl.OnErr.Explain != "Expected a positive integer" {
		t.Errorf("expected explain 'Expected a positive integer', got '%s'", varDecl.OnErr.Explain)
	}
}

func TestParseThreeValueAssignment(t *testing.T) {
	input := `func Test()
    _, ipNet, err := net.ParseCIDR("192.168.0.0/16")
`

	program := mustParseProgram(t, input)

	fn := program.Declarations[0].(*ast.FunctionDecl)
	varDecl, ok := fn.Body.Statements[0].(*ast.VarDeclStmt)
	if !ok {
		t.Fatalf("expected VarDeclStmt, got %T", fn.Body.Statements[0])
	}

	if len(varDecl.Names) != 3 {
		t.Errorf("expected 3 names, got %d", len(varDecl.Names))
	}

	if varDecl.Names[0].Value != "_" {
		t.Errorf("expected first name '_', got %s", varDecl.Names[0].Value)
	}
	if varDecl.Names[1].Value != "ipNet" {
		t.Errorf("expected second name 'ipNet', got %s", varDecl.Names[1].Value)
	}
	if varDecl.Names[2].Value != "err" {
		t.Errorf("expected third name 'err', got %s", varDecl.Names[2].Value)
	}
}

func TestParseTypeSwitchStatement(t *testing.T) {
	input := `func Handle(event any)
    switch event as e
        when reference a2a.TaskStatusUpdateEvent
            print(e.Status.State)
        when reference a2a.Task
            result := taskFromA2A(e)
        when string
            print(e)
        otherwise
            print("unknown")
`

	program := mustParseProgram(t, input)

	fn := program.Declarations[0].(*ast.FunctionDecl)
	tsStmt, ok := fn.Body.Statements[0].(*ast.TypeSwitchStmt)
	if !ok {
		t.Fatalf("expected TypeSwitchStmt, got %T", fn.Body.Statements[0])
	}

	// Check expression
	ident, ok := tsStmt.Expression.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected Identifier expression, got %T", tsStmt.Expression)
	}
	if ident.Value != "event" {
		t.Errorf("expected expression 'event', got %s", ident.Value)
	}

	// Check binding
	if tsStmt.Binding.Value != "e" {
		t.Errorf("expected binding 'e', got %s", tsStmt.Binding.Value)
	}

	// Check cases
	if len(tsStmt.Cases) != 3 {
		t.Fatalf("expected 3 type cases, got %d", len(tsStmt.Cases))
	}

	// First case: reference a2a.TaskStatusUpdateEvent
	refType, ok := tsStmt.Cases[0].Type.(*ast.ReferenceType)
	if !ok {
		t.Fatalf("expected ReferenceType for case 0, got %T", tsStmt.Cases[0].Type)
	}
	named, ok := refType.ElementType.(*ast.NamedType)
	if !ok {
		t.Fatalf("expected NamedType inside ReferenceType, got %T", refType.ElementType)
	}
	if named.Name != "a2a.TaskStatusUpdateEvent" {
		t.Errorf("expected type 'a2a.TaskStatusUpdateEvent', got %s", named.Name)
	}

	// Third case: plain type (string)
	primType, ok := tsStmt.Cases[2].Type.(*ast.PrimitiveType)
	if !ok {
		t.Fatalf("expected PrimitiveType for case 2, got %T", tsStmt.Cases[2].Type)
	}
	if primType.Name != "string" {
		t.Errorf("expected type 'string', got %s", primType.Name)
	}

	// Check otherwise
	if tsStmt.Otherwise == nil {
		t.Fatal("expected otherwise branch, got nil")
	}
}

func TestParseTypeSwitchNoOtherwise(t *testing.T) {
	input := `func Handle(event any)
    switch event as e
        when int
            print(e)
        when string
            print(e)
`

	program := mustParseProgram(t, input)

	fn := program.Declarations[0].(*ast.FunctionDecl)
	tsStmt, ok := fn.Body.Statements[0].(*ast.TypeSwitchStmt)
	if !ok {
		t.Fatalf("expected TypeSwitchStmt, got %T", fn.Body.Statements[0])
	}

	if len(tsStmt.Cases) != 2 {
		t.Fatalf("expected 2 type cases, got %d", len(tsStmt.Cases))
	}

	if tsStmt.Otherwise != nil {
		t.Error("expected no otherwise branch")
	}
}

func TestParseTypedPipedSwitchExpr(t *testing.T) {
	input := `func Convert(value any) string
    result := value |> switch as v
        when string
            return v
        when int
            return "number"
        otherwise
            return "other"
    return result
`

	program := mustParseProgram(t, input)

	fn := program.Declarations[0].(*ast.FunctionDecl)
	varDecl, ok := fn.Body.Statements[0].(*ast.VarDeclStmt)
	if !ok {
		t.Fatalf("expected VarDeclStmt, got %T", fn.Body.Statements[0])
	}

	ps, ok := varDecl.Values[0].(*ast.PipedSwitchExpr)
	if !ok {
		t.Fatalf("expected PipedSwitchExpr, got %T", varDecl.Values[0])
	}

	ts, ok := ps.Switch.(*ast.TypeSwitchStmt)
	if !ok {
		t.Fatalf("expected TypeSwitchStmt, got %T", ps.Switch)
	}

	if ts.Binding.Value != "v" {
		t.Fatalf("expected binding 'v', got %s", ts.Binding.Value)
	}
	if len(ts.Cases) != 2 {
		t.Fatalf("expected 2 type cases, got %d", len(ts.Cases))
	}
	if ts.Otherwise == nil {
		t.Fatal("expected otherwise branch, got nil")
	}
}

func TestParseSelectStatement(t *testing.T) {
	input := `func Run(ch channel of string, done channel of string, out channel of string)
    select
        when receive from done
            return
        when msg := receive from ch
            return
        when msg, ok := receive from ch
            return
        when send "ping" to out
            return
        otherwise
            return
`

	program := mustParseProgram(t, input)

	fn, ok := program.Declarations[0].(*ast.FunctionDecl)
	if !ok {
		t.Fatalf("expected FunctionDecl, got %T", program.Declarations[0])
	}

	selectStmt, ok := fn.Body.Statements[0].(*ast.SelectStmt)
	if !ok {
		t.Fatalf("expected SelectStmt, got %T", fn.Body.Statements[0])
	}

	if len(selectStmt.Cases) != 4 {
		t.Fatalf("expected 4 when cases, got %d", len(selectStmt.Cases))
	}

	// Case 0: bare receive
	c0 := selectStmt.Cases[0]
	if c0.Recv == nil {
		t.Fatal("case 0: expected Recv, got nil")
	}
	if len(c0.Bindings) != 0 {
		t.Errorf("case 0: expected 0 bindings, got %d", len(c0.Bindings))
	}

	// Case 1: 1-var binding receive
	c1 := selectStmt.Cases[1]
	if c1.Recv == nil {
		t.Fatal("case 1: expected Recv, got nil")
	}
	if len(c1.Bindings) != 1 || c1.Bindings[0] != "msg" {
		t.Errorf("case 1: expected bindings [msg], got %v", c1.Bindings)
	}

	// Case 2: 2-var binding receive
	c2 := selectStmt.Cases[2]
	if c2.Recv == nil {
		t.Fatal("case 2: expected Recv, got nil")
	}
	if len(c2.Bindings) != 2 || c2.Bindings[0] != "msg" || c2.Bindings[1] != "ok" {
		t.Errorf("case 2: expected bindings [msg ok], got %v", c2.Bindings)
	}

	// Case 3: send case
	c3 := selectStmt.Cases[3]
	if c3.Send == nil {
		t.Fatal("case 3: expected Send, got nil")
	}
	if c3.Recv != nil {
		t.Error("case 3: expected Recv nil")
	}

	// Otherwise
	if selectStmt.Otherwise == nil {
		t.Fatal("expected otherwise branch")
	}
}
