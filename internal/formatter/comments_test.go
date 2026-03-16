package formatter

import (
	"testing"

	"github.com/duber000/kukicha/internal/lexer"
	"github.com/duber000/kukicha/internal/parser"
)

// helper: lex source, extract comments
func extractCommentsFromSource(t *testing.T, source string) []Comment {
	t.Helper()
	l := lexer.NewLexer(source, "test.kuki")
	tokens, err := l.ScanTokens()
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}
	return ExtractComments(tokens)
}

// helper: lex+parse source, attach comments, return map
func attachCommentsFromSource(t *testing.T, source string) CommentMap {
	t.Helper()
	l := lexer.NewLexer(source, "test.kuki")
	tokens, err := l.ScanTokens()
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}
	comments := ExtractComments(tokens)
	p := parser.NewFromTokens(tokens)
	program, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	return AttachComments(comments, program)
}

// --- ExtractComments tests ---

func TestExtractComments_NoComments(t *testing.T) {
	source := `func main()
    x := 1
`
	comments := extractCommentsFromSource(t, source)
	if len(comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(comments))
	}
}

func TestExtractComments_SingleComment(t *testing.T) {
	source := `# hello
func main()
    x := 1
`
	comments := extractCommentsFromSource(t, source)
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].Text != "# hello" {
		t.Errorf("expected '# hello', got %q", comments[0].Text)
	}
	if comments[0].Line != 1 {
		t.Errorf("expected line 1, got %d", comments[0].Line)
	}
}

func TestExtractComments_MultipleComments(t *testing.T) {
	source := `# first
# second
func main()
    # inside
    x := 1
`
	comments := extractCommentsFromSource(t, source)
	if len(comments) != 3 {
		t.Fatalf("expected 3 comments, got %d", len(comments))
	}
	if comments[0].Text != "# first" {
		t.Errorf("comment 0: expected '# first', got %q", comments[0].Text)
	}
	if comments[1].Text != "# second" {
		t.Errorf("comment 1: expected '# second', got %q", comments[1].Text)
	}
	if comments[2].Text != "# inside" {
		t.Errorf("comment 2: expected '# inside', got %q", comments[2].Text)
	}
}

func TestExtractComments_TrailingComment(t *testing.T) {
	source := `func main()
    x := 1 # trailing
`
	comments := extractCommentsFromSource(t, source)
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].Text != "# trailing" {
		t.Errorf("expected '# trailing', got %q", comments[0].Text)
	}
}

func TestExtractComments_DirectivesExcluded(t *testing.T) {
	// Directives (# kuki:...) should not appear as comments
	source := `# kuki:deprecated "use NewFunc"
func OldFunc()
    x := 1
`
	comments := extractCommentsFromSource(t, source)
	for _, c := range comments {
		if c.Text == `# kuki:deprecated "use NewFunc"` {
			t.Error("directive should not be extracted as a comment")
		}
	}
}

// --- AttachComments tests ---

func TestAttachComments_EmptyComments(t *testing.T) {
	source := `func main()
    x := 1
`
	cm := attachCommentsFromSource(t, source)
	if len(cm) != 0 {
		t.Errorf("expected empty comment map, got %d entries", len(cm))
	}
}

func TestAttachComments_LeadingCommentOnFunction(t *testing.T) {
	source := `# This is main
func main()
    x := 1
`
	cm := attachCommentsFromSource(t, source)

	// Should have at least one attachment with a leading comment
	found := false
	for _, attachment := range cm {
		if len(attachment.Leading) > 0 {
			for _, c := range attachment.Leading {
				if c.Text == "# This is main" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected leading comment '# This is main' attached to a node")
	}
}

func TestAttachComments_MultipleLeadingComments(t *testing.T) {
	source := `# Comment A
# Comment B
func main()
    x := 1
`
	cm := attachCommentsFromSource(t, source)

	found := 0
	for _, attachment := range cm {
		found += len(attachment.Leading)
	}
	if found < 2 {
		t.Errorf("expected at least 2 leading comments attached, got %d", found)
	}
}

func TestAttachComments_TrailingCommentOnStatement(t *testing.T) {
	source := `func main()
    x := 1 # the value
`
	cm := attachCommentsFromSource(t, source)

	foundTrailing := false
	for _, attachment := range cm {
		if attachment.Trailing != nil && attachment.Trailing.Text == "# the value" {
			foundTrailing = true
		}
	}
	if !foundTrailing {
		t.Error("expected trailing comment '# the value' attached to a node")
	}
}

func TestAttachComments_CommentInsideIfBlock(t *testing.T) {
	source := `func main()
    if x == 1
        # inside if
        y := 2
`
	cm := attachCommentsFromSource(t, source)

	found := false
	for _, attachment := range cm {
		for _, c := range attachment.Leading {
			if c.Text == "# inside if" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected comment '# inside if' attached inside if block")
	}
}

func TestAttachComments_CommentInsideForLoop(t *testing.T) {
	source := `func main()
    for i from 0 to 10
        # loop body
        print(i)
`
	cm := attachCommentsFromSource(t, source)

	found := false
	for _, attachment := range cm {
		for _, c := range attachment.Leading {
			if c.Text == "# loop body" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected comment '# loop body' attached inside for loop")
	}
}

func TestAttachComments_CommentInsideSwitchCase(t *testing.T) {
	source := `func main()
    switch x
        when 1
            # case 1
            doA()
        otherwise
            # default
            doB()
`
	cm := attachCommentsFromSource(t, source)

	foundCase := false
	foundDefault := false
	for _, attachment := range cm {
		for _, c := range attachment.Leading {
			if c.Text == "# case 1" {
				foundCase = true
			}
			if c.Text == "# default" {
				foundDefault = true
			}
		}
	}
	if !foundCase {
		t.Error("expected comment '# case 1' attached inside switch case")
	}
	if !foundDefault {
		t.Error("expected comment '# default' attached inside otherwise block")
	}
}

func TestAttachComments_CommentBetweenFunctions(t *testing.T) {
	source := `func first()
    x := 1

# Between functions
func second()
    y := 2
`
	cm := attachCommentsFromSource(t, source)

	found := false
	for _, attachment := range cm {
		for _, c := range attachment.Leading {
			if c.Text == "# Between functions" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected comment '# Between functions' attached as leading to second function")
	}
}

func TestAttachComments_CommentOnImport(t *testing.T) {
	source := `# Import fmt
import "fmt"

func main()
    fmt.Println("hi")
`
	cm := attachCommentsFromSource(t, source)

	found := false
	for _, attachment := range cm {
		for _, c := range attachment.Leading {
			if c.Text == "# Import fmt" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected comment '# Import fmt' attached to import")
	}
}

func TestAttachComments_CommentOnTypeField(t *testing.T) {
	source := `type User
    # User's name
    name string
    age int
`
	cm := attachCommentsFromSource(t, source)

	found := false
	for _, attachment := range cm {
		for _, c := range attachment.Leading {
			if c.Text == "# User's name" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected comment '# User's name' attached to name field")
	}
}

// --- Formatter round-trip integration tests ---

func TestFormatPreservesLeadingComment(t *testing.T) {
	source := `# Main function
func main()
    x := 1
`
	opts := DefaultOptions()
	result, err := Format(source, "test.kuki", opts)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	// The comment should be preserved in the output
	if !containsString(result, "# Main function") {
		t.Errorf("leading comment lost in formatted output:\n%s", result)
	}

	// Round-trip should be stable
	result2, err := Format(result, "test.kuki", opts)
	if err != nil {
		t.Fatalf("Round-trip format error: %v", err)
	}
	if result != result2 {
		t.Errorf("format is not idempotent:\n--- first ---\n%s--- second ---\n%s", result, result2)
	}
}

func TestFormatPreservesTrailingComment(t *testing.T) {
	source := `func main()
    x := 1 # important value
`
	opts := DefaultOptions()
	result, err := Format(source, "test.kuki", opts)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	if !containsString(result, "# important value") {
		t.Errorf("trailing comment lost in formatted output:\n%s", result)
	}
}

func TestFormatPreservesMultipleComments(t *testing.T) {
	source := `# Comment 1
# Comment 2
func main()
    # Comment 3
    x := 1
`
	opts := DefaultOptions()
	result, err := Format(source, "test.kuki", opts)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	for _, expected := range []string{"# Comment 1", "# Comment 2", "# Comment 3"} {
		if !containsString(result, expected) {
			t.Errorf("comment %q lost in formatted output:\n%s", expected, result)
		}
	}
}

func TestFormatPreservesCommentInNestedBlock(t *testing.T) {
	source := `func main()
    if true
        # nested comment
        x := 1
`
	opts := DefaultOptions()
	result, err := Format(source, "test.kuki", opts)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	if !containsString(result, "# nested comment") {
		t.Errorf("nested comment lost in formatted output:\n%s", result)
	}
}

func TestFormatPreservesCommentBetweenDecls(t *testing.T) {
	source := `func first()
    x := 1

# separator
func second()
    y := 2
`
	opts := DefaultOptions()
	result, err := Format(source, "test.kuki", opts)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	if !containsString(result, "# separator") {
		t.Errorf("comment between decls lost in formatted output:\n%s", result)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchString(s, substr)))
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
