package codegen

import (
	"regexp"
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

// mustContainPattern asserts that output matches the given regex pattern.
// Use `\d+` for temp variable suffixes to decouple tests from exact counter values.
func mustContainPattern(t *testing.T, output, pattern, msg string) {
	t.Helper()
	re := regexp.MustCompile(pattern)
	if !re.MatchString(output) {
		t.Errorf("%s\npattern: %s\ngot:\n%s", msg, pattern, output)
	}
}

// mustNotContainPattern asserts that output does NOT match the given regex pattern.
func mustNotContainPattern(t *testing.T, output, pattern, msg string) {
	t.Helper()
	re := regexp.MustCompile(pattern)
	if re.MatchString(output) {
		t.Errorf("%s\npattern: %s\ngot:\n%s", msg, pattern, output)
	}
}
