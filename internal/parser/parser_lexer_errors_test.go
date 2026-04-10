package parser

import (
	"strings"
	"testing"
)

// TestLexerErrorsSurfaceViaParseWithPosition verifies that lexer errors
// (invalid characters, bad indentation, etc.) are returned by Parse() as
// individually positioned "file:line:col: message" errors rather than a
// single opaque "lexer errors: [...]" string.
func TestLexerErrorsSurfaceViaParseWithPosition(t *testing.T) {
	// \t (tab) is a lexer error in Kukicha — only 4-space indentation is allowed.
	input := "func Foo() int\n\treturn 1\n"

	p, err := New(input, "test.kuki")
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}

	_, parseErrors := p.Parse()
	if len(parseErrors) == 0 {
		t.Fatal("expected at least one parse error from lexer, got none")
	}

	for _, pe := range parseErrors {
		msg := pe.Error()
		// Each error must be in "file:line:col: message" format.
		if !strings.Contains(msg, ":") {
			t.Errorf("error lacks position info: %q", msg)
			continue
		}
		// Must NOT be the old opaque wrapper.
		if strings.HasPrefix(msg, "lexer errors:") {
			t.Errorf("error is the opaque wrapper rather than an individual positioned error: %q", msg)
		}
		// Must contain the filename.
		if !strings.Contains(msg, "test.kuki") {
			t.Errorf("error does not mention filename: %q", msg)
		}
	}
}

// TestLexerMultipleErrorsAllSurfaceViaParseWithPosition verifies that when
// the lexer encounters several errors, each one is returned individually.
func TestLexerMultipleErrorsAllSurfaceViaParseWithPosition(t *testing.T) {
	// Two lines with illegal characters.
	input := "x := \x01\ny := \x02\n"

	p, err := New(input, "src.kuki")
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}

	_, parseErrors := p.Parse()
	if len(parseErrors) < 2 {
		t.Fatalf("expected at least 2 parse errors, got %d: %v", len(parseErrors), parseErrors)
	}

	for i, pe := range parseErrors {
		msg := pe.Error()
		if strings.HasPrefix(msg, "lexer errors:") {
			t.Errorf("error[%d] is the opaque wrapper: %q", i, msg)
		}
		if !strings.Contains(msg, "src.kuki") {
			t.Errorf("error[%d] does not mention filename: %q", i, msg)
		}
	}
}

// TestNewAlwaysReturnsNilError verifies that parser.New() never returns a
// non-nil error — even when the input contains lexer errors — so callers can
// unconditionally use the returned parser and check errors via Parse().
func TestNewAlwaysReturnsNilError(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"valid source", "func Foo() int\n    return 1\n"},
		{"tab indentation (lexer error)", "func Foo() int\n\treturn 1\n"},
		{"invalid character", "x := \x01\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(tc.input, "test.kuki")
			if err != nil {
				t.Errorf("New() returned non-nil error for %q: %v", tc.name, err)
			}
		})
	}
}
