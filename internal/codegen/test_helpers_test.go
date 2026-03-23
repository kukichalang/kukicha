package codegen

import (
	"testing"

	"github.com/kukichalang/kukicha/internal/ast"
	"github.com/kukichalang/kukicha/internal/parser"
)

func mustParseProgram(t *testing.T, input string) *ast.Program {
	t.Helper()

	p, err := parser.New(input, "test.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	return program
}

func generateSource(t *testing.T, input string) string {
	t.Helper()

	gen := New(mustParseProgram(t, input))
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}

	return output
}
