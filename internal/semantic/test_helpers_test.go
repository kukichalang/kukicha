package semantic

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

func analyzeSource(t *testing.T, input string) (*Analyzer, []error) {
	t.Helper()
	return analyzeSourceWithFile(t, input, "test.kuki")
}

func analyzeSourceWithFile(t *testing.T, input, filename string) (*Analyzer, []error) {
	t.Helper()

	p, err := parser.New(input, filename)
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	analyzer := NewWithFile(program, filename)
	return analyzer, analyzer.Analyze()
}
