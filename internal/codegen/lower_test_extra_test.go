package codegen

import (
	"strings"
	"testing"

	"github.com/kukichalang/kukicha/internal/ast"
	"github.com/kukichalang/kukicha/internal/semantic"
)

func TestLowerPipedSwitchOnerrAsAlias(t *testing.T) {
	value := &ast.Identifier{Value: "value"}
	riskyCall := &ast.CallExpr{Function: &ast.Identifier{Value: "Risky"}, Arguments: []ast.Expression{}}
	pipe := &ast.PipeExpr{Left: value, Right: riskyCall}
	ps := &ast.PipedSwitchExpr{
		Left: pipe,
		Switch: &ast.SwitchStmt{
			Expression: &ast.Identifier{Value: "_"},
			Cases: []*ast.WhenCase{{
				Values: []ast.Expression{&ast.StringLiteral{Value: "a"}}, 
				Body: &ast.BlockStmt{Statements: []ast.Statement{&ast.ExpressionStmt{Expression: &ast.CallExpr{Function: &ast.Identifier{Value: "doA"}, Arguments: []ast.Expression{}}}}},
			}},
		},
	}
	clause := &ast.OnErrClause{
		Alias: "e",
		Handler: &ast.PanicExpr{Message: &ast.StringLiteral{
			Value:        "failed: {e} and {error}",
			Interpolated: true,
			Parts: []*ast.StringInterpolation{
				{IsLiteral: true, Literal: "failed: "},
				{Expr: &ast.Identifier{Value: "e"}},
				{IsLiteral: true, Literal: " and "},
				{Expr: &ast.Identifier{Value: "error"}},
			},
		}},
	}
	gen := New(&ast.Program{})
	gen.exprReturnCounts = map[ast.Expression]int{riskyCall: 2}
	l := newLowerer(gen)
	block := l.lowerPipedSwitchStmt(ps, clause)
	gen.emitIR(block)
	out := gen.output.String()
	if !strings.Contains(out, `panic(fmt.Sprintf("failed: %v and %v", pipeErr_1, pipeErr_1))`) && !strings.Contains(out, `panic(fmt.Sprintf("failed: %v and %v", pipeErr_3, pipeErr_3))`) {
		t.Errorf("expected interpolated panic handler with pipeErr, got:\n%s", out)
	}
}

func TestLowerPipedSwitchOnerrPanicError(t *testing.T) {
	value := &ast.Identifier{Value: "value"}
	riskyCall := &ast.CallExpr{Function: &ast.Identifier{Value: "Risky"}, Arguments: []ast.Expression{}}
	pipe := &ast.PipeExpr{Left: value, Right: riskyCall}
	ps := &ast.PipedSwitchExpr{
		Left: pipe,
		Switch: &ast.SwitchStmt{
			Expression: &ast.Identifier{Value: "_"},
			Cases: []*ast.WhenCase{{
				Values: []ast.Expression{&ast.StringLiteral{Value: "a"}}, 
				Body: &ast.BlockStmt{Statements: []ast.Statement{&ast.ExpressionStmt{Expression: &ast.CallExpr{Function: &ast.Identifier{Value: "doA"}, Arguments: []ast.Expression{}}}}},
			}},
		},
	}
	clause := &ast.OnErrClause{
		Handler: &ast.PanicExpr{Message: &ast.StringLiteral{
			Value:        "{error}",
			Interpolated: true,
			Parts: []*ast.StringInterpolation{
				{Expr: &ast.Identifier{Value: "error"}},
			},
		}},
	}
	gen := New(&ast.Program{})
	gen.exprReturnCounts = map[ast.Expression]int{riskyCall: 2}
	l := newLowerer(gen)
	block := l.lowerPipedSwitchStmt(ps, clause)
	gen.emitIR(block)
	out := gen.output.String()
	if !strings.Contains(out, `panic(fmt.Sprintf("%v", pipeErr_3))`) && !strings.Contains(out, `panic(fmt.Sprintf("%v", pipeErr_1))`) {
		t.Errorf("expected interpolated panic handler with pipeErr, got:\n%s", out)
	}
}

func TestLowerPipeChainLong(t *testing.T) {
	calls := []ast.Expression{&ast.Identifier{Value: "start"}}
	for i := 0; i < 6; i++ {
		calls = append(calls, &ast.CallExpr{Function: &ast.Identifier{Value: "step"}, Arguments: []ast.Expression{}})
	}
	var pipe ast.Expression = calls[0]
	for i := 1; i < len(calls); i++ {
		pipe = &ast.PipeExpr{Left: pipe, Right: calls[i]}
	}
	
	gen := New(&ast.Program{})
	gen.exprReturnCounts = map[ast.Expression]int{}
	for i := 1; i < len(calls); i++ {
		gen.exprReturnCounts[calls[i]] = 2
	}
	clause := &ast.OnErrClause{Handler: &ast.PanicExpr{Message: &ast.StringLiteral{Value: "fail"}}}
	l := newLowerer(gen)
	block, _ := l.lowerOnErrPipeChain(pipe.(*ast.PipeExpr), clause, []string{"result"}, "")
	gen.emitIR(block)
	out := gen.output.String()
	if strings.Count(out, "err_") < 6 {
		t.Errorf("expected at least 6 err variables, got:\n%s", out)
	}
}

func TestLowerPipeChainConsecutiveErrorOnly(t *testing.T) {
	aCall := &ast.CallExpr{Function: &ast.Identifier{Value: "a"}, Arguments: []ast.Expression{}}
	err1 := &ast.CallExpr{Function: &ast.Identifier{Value: "errorOnly1"}, Arguments: []ast.Expression{}}
	err2 := &ast.CallExpr{Function: &ast.Identifier{Value: "errorOnly2"}, Arguments: []ast.Expression{}}
	bCall := &ast.CallExpr{Function: &ast.Identifier{Value: "b"}, Arguments: []ast.Expression{}}
	
	p1 := &ast.PipeExpr{Left: aCall, Right: err1}
	p2 := &ast.PipeExpr{Left: p1, Right: err2}
	p3 := &ast.PipeExpr{Left: p2, Right: bCall}
	
	errType := &semantic.TypeInfo{Kind: semantic.TypeKindNamed, Name: "error"}
	gen := New(&ast.Program{})
	gen.exprReturnCounts = map[ast.Expression]int{
		aCall: 2,
		err1:  1,
		err2:  1,
		bCall: 2,
	}
	gen.exprTypes = map[ast.Expression]*semantic.TypeInfo{
		err1: errType,
		err2: errType,
	}
	clause := &ast.OnErrClause{ShorthandReturn: true}
	l := newLowerer(gen)
	block, _ := l.lowerOnErrPipeChain(p3, clause, []string{"res"}, "")
	gen.emitIR(block)
	out := gen.output.String()
	
	if !strings.Contains(out, "errorOnly1(pipe_1)") {
		t.Errorf("missing errorOnly1 call, got:\n%s", out)
	}
	if !strings.Contains(out, "errorOnly2(pipe_1)") {
		t.Errorf("missing errorOnly2 call, got:\n%s", out)
	}
	// Both error-only steps should use the same pipe variable (current doesn't advance)
	if strings.Contains(out, "errorOnly1(pipe_3)") || strings.Contains(out, "errorOnly2(pipe_3)") {
		t.Errorf("error-only steps should not advance the pipe variable, got:\n%s", out)
	}
}

func TestPipedSwitchMismatchedTypesFallback(t *testing.T) {
	value := &ast.Identifier{Value: "value"}
	riskyCall := &ast.CallExpr{Function: &ast.Identifier{Value: "Risky"}, Arguments: []ast.Expression{}}
	pipe := &ast.PipeExpr{Left: value, Right: riskyCall}
	
	ps := &ast.PipedSwitchExpr{
		Left: pipe,
		Switch: &ast.SwitchStmt{
			Expression: &ast.Identifier{Value: "_"},
			Cases: []*ast.WhenCase{
				{
					Values: []ast.Expression{&ast.StringLiteral{Value: "a"}},
					Body: &ast.BlockStmt{Statements: []ast.Statement{&ast.ReturnStmt{Values: []ast.Expression{&ast.StringLiteral{Value: "str"}}}}},
				},
				{
					Values: []ast.Expression{&ast.StringLiteral{Value: "b"}},
					Body: &ast.BlockStmt{Statements: []ast.Statement{&ast.ReturnStmt{Values: []ast.Expression{&ast.IntegerLiteral{Value: 42}}}}},
				},
			},
		},
	}
	
	gen := New(&ast.Program{})
	gen.exprReturnCounts = map[ast.Expression]int{riskyCall: 2}
	l := newLowerer(gen)
	clause := &ast.OnErrClause{Handler: &ast.PanicExpr{Message: &ast.StringLiteral{Value: "fail"}}}
	
	block := l.lowerPipedSwitchVarDecl("result", ps, clause, []*ast.Identifier{{Value: "result"}})
	gen.emitIR(block)
	out := gen.output.String()
	
	if !strings.Contains(out, "func() any {") {
		t.Errorf("expected IIFE to return any, got:\n%s", out)
	}
}

// TestUnknownSingleReturnWarnsInOnerr verifies that a pipe step with count==1
// and TypeKindUnknown emits a warning (error may be silently dropped).
func TestUnknownSingleReturnWarnsInOnerr(t *testing.T) {
	unknownCall := &ast.CallExpr{Function: &ast.Identifier{Value: "externalFn"}, Arguments: []ast.Expression{}}
	nextCall := &ast.CallExpr{Function: &ast.Identifier{Value: "finalStep"}, Arguments: []ast.Expression{}}
	pipe := &ast.PipeExpr{
		Left:  &ast.PipeExpr{Left: &ast.Identifier{Value: "data"}, Right: unknownCall},
		Right: nextCall,
	}

	gen := New(&ast.Program{})
	gen.exprReturnCounts = map[ast.Expression]int{
		unknownCall: 1, // single return, type unknown → should warn
		nextCall:    2, // error-returning final step
	}
	gen.exprTypes = map[ast.Expression]*semantic.TypeInfo{
		unknownCall: {Kind: semantic.TypeKindUnknown},
	}

	clause := &ast.OnErrClause{Handler: &ast.PanicExpr{Message: &ast.StringLiteral{Value: "fail"}}}
	l := newLowerer(gen)
	block, _ := l.lowerOnErrPipeChain(pipe, clause, []string{"result"}, "")
	gen.emitIR(block)

	found := false
	for _, w := range gen.Warnings() {
		if strings.Contains(w.Error(), "return type of 'externalFn(...)' is unknown") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected unknown-return warning, got warnings: %v", gen.Warnings())
	}
}

// TestKnownSingleReturnNoWarnInOnerr verifies that a step with a known
// non-error single return type does NOT emit the unknown-return warning.
func TestKnownSingleReturnNoWarnInOnerr(t *testing.T) {
	knownCall := &ast.CallExpr{Function: &ast.Identifier{Value: "stringify"}, Arguments: []ast.Expression{}}
	nextCall := &ast.CallExpr{Function: &ast.Identifier{Value: "finalStep"}, Arguments: []ast.Expression{}}
	pipe := &ast.PipeExpr{
		Left:  &ast.PipeExpr{Left: &ast.Identifier{Value: "data"}, Right: knownCall},
		Right: nextCall,
	}

	gen := New(&ast.Program{})
	gen.exprReturnCounts = map[ast.Expression]int{
		knownCall: 1,
		nextCall:  2,
	}
	gen.exprTypes = map[ast.Expression]*semantic.TypeInfo{
		knownCall: {Kind: semantic.TypeKindNamed, Name: "string"}, // known, non-error
	}

	clause := &ast.OnErrClause{Handler: &ast.PanicExpr{Message: &ast.StringLiteral{Value: "fail"}}}
	l := newLowerer(gen)
	block, _ := l.lowerOnErrPipeChain(pipe, clause, []string{"result"}, "")
	gen.emitIR(block)

	if len(gen.Warnings()) > 0 {
		t.Errorf("expected no warning for known-type step, got warnings: %v", gen.Warnings())
	}
}

// TestBareIdentifierPipeTargetInOnerr verifies that a bare identifier as a pipe
// step in an onerr chain is treated as a zero-argument call (consistent with
// the non-onerr path in generatePipeExpr).
func TestBareIdentifierPipeTargetInOnerr(t *testing.T) {
	bareIdent := &ast.Identifier{Value: "transform"}
	finalCall := &ast.CallExpr{Function: &ast.Identifier{Value: "finalStep"}, Arguments: []ast.Expression{}}
	pipe := &ast.PipeExpr{
		Left:  &ast.PipeExpr{Left: &ast.Identifier{Value: "data"}, Right: bareIdent},
		Right: finalCall,
	}

	gen := New(&ast.Program{})
	gen.exprReturnCounts = map[ast.Expression]int{finalCall: 2}

	clause := &ast.OnErrClause{Handler: &ast.PanicExpr{Message: &ast.StringLiteral{Value: "fail"}}}
	l := newLowerer(gen)
	block, _ := l.lowerOnErrPipeChain(pipe, clause, []string{"result"}, "")

	if block == nil {
		t.Fatal("expected non-nil block; bare identifier pipe target should not abort lowering")
	}
	gen.emitIR(block)
	out := gen.output.String()

	if !strings.Contains(out, "transform(data)") {
		t.Errorf("expected bare identifier to emit transform(data), got:\n%s", out)
	}
	if len(gen.Warnings()) > 0 {
		t.Errorf("expected no warning for bare identifier pipe target, got warnings: %v", gen.Warnings())
	}
}
