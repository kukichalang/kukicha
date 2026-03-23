package parser

import (
	"testing"

	"github.com/kukichalang/kukicha/internal/ast"
)

func mustParseProgram(t *testing.T, input string) *ast.Program {
	t.Helper()

	p, err := New(input, "test.kuki")
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	program, errors := p.Parse()
	if len(errors) > 0 {
		t.Fatalf("parser errors: %v", errors)
	}

	return program
}
