package lexer

import (
	"testing"
)

func TestBraceBlockTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{
			name:  "func with brace block emits INDENT/DEDENT",
			input: "func main() {\n    x := 1\n}\n",
			expected: []TokenType{
				TOKEN_FUNC, TOKEN_IDENTIFIER, TOKEN_LPAREN, TOKEN_RPAREN,
				TOKEN_INDENT, TOKEN_NEWLINE, // { becomes INDENT
				TOKEN_IDENTIFIER, TOKEN_WALRUS, TOKEN_INTEGER, TOKEN_NEWLINE,
				TOKEN_DEDENT, TOKEN_NEWLINE, TOKEN_EOF, // } becomes DEDENT
			},
		},
		{
			name:  "if with brace block emits INDENT/DEDENT",
			input: "if x > 0 {\n    return x\n}\n",
			expected: []TokenType{
				TOKEN_IF, TOKEN_IDENTIFIER, TOKEN_GT, TOKEN_INTEGER,
				TOKEN_INDENT, TOKEN_NEWLINE, // {
				TOKEN_RETURN, TOKEN_IDENTIFIER, TOKEN_NEWLINE,
				TOKEN_DEDENT, TOKEN_NEWLINE, TOKEN_EOF, // }
			},
		},
		{
			name:  "single-line brace block",
			input: "if x > 0 { return x }\n",
			expected: []TokenType{
				TOKEN_IF, TOKEN_IDENTIFIER, TOKEN_GT, TOKEN_INTEGER,
				TOKEN_INDENT, TOKEN_RETURN, TOKEN_IDENTIFIER, TOKEN_DEDENT,
				TOKEN_NEWLINE, TOKEN_EOF,
			},
		},
		{
			name:  "else with brace block",
			input: "else {\n    y := 2\n}\n",
			expected: []TokenType{
				TOKEN_ELSE, TOKEN_INDENT, TOKEN_NEWLINE, // {
				TOKEN_IDENTIFIER, TOKEN_WALRUS, TOKEN_INTEGER, TOKEN_NEWLINE,
				TOKEN_DEDENT, TOKEN_NEWLINE, TOKEN_EOF, // }
			},
		},
		{
			name:  "struct literal stays LBRACE/RBRACE",
			input: "x := MyStruct{Name: \"foo\"}\n",
			expected: []TokenType{
				TOKEN_IDENTIFIER, TOKEN_WALRUS, TOKEN_IDENTIFIER,
				TOKEN_LBRACE, TOKEN_IDENTIFIER, TOKEN_COLON, TOKEN_STRING, TOKEN_RBRACE,
				TOKEN_NEWLINE, TOKEN_EOF,
			},
		},
		{
			name:  "no extra INDENT/DEDENT inside brace block",
			input: "func main() {\n    x := 1\n    y := 2\n}\n",
			expected: []TokenType{
				TOKEN_FUNC, TOKEN_IDENTIFIER, TOKEN_LPAREN, TOKEN_RPAREN,
				TOKEN_INDENT, TOKEN_NEWLINE,
				TOKEN_IDENTIFIER, TOKEN_WALRUS, TOKEN_INTEGER, TOKEN_NEWLINE,
				TOKEN_IDENTIFIER, TOKEN_WALRUS, TOKEN_INTEGER, TOKEN_NEWLINE,
				TOKEN_DEDENT, TOKEN_NEWLINE, TOKEN_EOF,
			},
		},
		{
			name:  "nested brace blocks",
			input: "func main() {\n    if true {\n        x := 1\n    }\n}\n",
			expected: []TokenType{
				TOKEN_FUNC, TOKEN_IDENTIFIER, TOKEN_LPAREN, TOKEN_RPAREN,
				TOKEN_INDENT, TOKEN_NEWLINE,
				TOKEN_IF, TOKEN_TRUE, TOKEN_INDENT, TOKEN_NEWLINE,
				TOKEN_IDENTIFIER, TOKEN_WALRUS, TOKEN_INTEGER, TOKEN_NEWLINE,
				TOKEN_DEDENT, TOKEN_NEWLINE,
				TOKEN_DEDENT, TOKEN_NEWLINE, TOKEN_EOF,
			},
		},
		{
			name:  "for condition with brace block",
			input: "for x < 10 {\n    print(x)\n}\n",
			expected: []TokenType{
				TOKEN_FOR, TOKEN_IDENTIFIER, TOKEN_LT, TOKEN_INTEGER,
				TOKEN_INDENT, TOKEN_NEWLINE,
				TOKEN_IDENTIFIER, TOKEN_LPAREN, TOKEN_IDENTIFIER, TOKEN_RPAREN, TOKEN_NEWLINE,
				TOKEN_DEDENT, TOKEN_NEWLINE, TOKEN_EOF,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, "test.kuki")
			tokens, err := lexer.ScanTokens()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(tokens) != len(tt.expected) {
				tokenTypes := make([]string, len(tokens))
				for i, tok := range tokens {
					tokenTypes[i] = tok.Type.String()
				}
				t.Fatalf("expected %d tokens, got %d: %v", len(tt.expected), len(tokens), tokenTypes)
			}

			for i, expected := range tt.expected {
				if tokens[i].Type != expected {
					t.Errorf("token %d: expected %s, got %s (%q)", i, expected, tokens[i].Type, tokens[i].Lexeme)
				}
			}
		})
	}
}

func TestBraceBlockNewlinesPreserved(t *testing.T) {
	input := "func main() {\n    x := 1\n    y := 2\n}\n"
	lexer := NewLexer(input, "test.kuki")
	tokens, err := lexer.ScanTokens()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newlineCount := 0
	for _, tok := range tokens {
		if tok.Type == TOKEN_NEWLINE {
			newlineCount++
		}
	}
	// Newlines: after {, after x:=1, after y:=2, after }
	if newlineCount != 4 {
		t.Errorf("expected 4 newlines inside brace block, got %d", newlineCount)
	}
}

func TestLiteralBraceNewlinesSuppressed(t *testing.T) {
	input := "x := MyStruct{\n    Name: \"foo\",\n}\n"
	lexer := NewLexer(input, "test.kuki")
	tokens, err := lexer.ScanTokens()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newlineCount := 0
	for _, tok := range tokens {
		if tok.Type == TOKEN_NEWLINE {
			newlineCount++
		}
	}
	if newlineCount != 1 {
		tokenTypes := make([]string, len(tokens))
		for i, tok := range tokens {
			tokenTypes[i] = tok.Type.String()
		}
		t.Errorf("expected 1 newline (only after }), got %d: %v", newlineCount, tokenTypes)
	}
}
