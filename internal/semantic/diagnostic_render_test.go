package semantic

import (
	"strings"
	"testing"
)

func TestRenderPretty_Error(t *testing.T) {
	d := Diagnostic{
		File:     "app.kuki",
		Line:     10,
		Col:      5,
		Severity: "error",
		Message:  "undefined: foo",
	}
	source := strings.Split("line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\ndata |> foo onerr panic", "\n")
	got := d.RenderPretty(source, false)

	if !strings.Contains(got, "── ERROR ──") {
		t.Errorf("expected ERROR header, got:\n%s", got)
	}
	if !strings.Contains(got, "app.kuki:10:5") {
		t.Errorf("expected location in header, got:\n%s", got)
	}
	if !strings.Contains(got, "undefined: foo") {
		t.Errorf("expected message, got:\n%s", got)
	}
	if !strings.Contains(got, "10 │ data |> foo onerr panic") {
		t.Errorf("expected source line, got:\n%s", got)
	}
	if !strings.Contains(got, "    ^") {
		t.Errorf("expected caret at col 5, got:\n%s", got)
	}
}

func TestRenderPretty_WarningWithHint(t *testing.T) {
	d := Diagnostic{
		File:       "app.kuki",
		Line:       3,
		Col:        1,
		Severity:   "warning",
		Category:   "security/sql-injection",
		Message:    "SQL injection risk",
		Suggestion: "use parameter placeholders",
	}
	source := []string{"import \"db\"", "", "db.Query(pool, query)"}
	got := d.RenderPretty(source, false)

	if !strings.Contains(got, "── SECURITY/SQL-INJECTION ──") {
		t.Errorf("expected category in header, got:\n%s", got)
	}
	if !strings.Contains(got, "hint: use parameter placeholders") {
		t.Errorf("expected hint, got:\n%s", got)
	}
}

func TestRenderPretty_NoSourceFallback(t *testing.T) {
	d := Diagnostic{
		File:     "missing.kuki",
		Line:     99,
		Col:      1,
		Severity: "error",
		Message:  "some error",
	}
	got := d.RenderPretty(nil, false)

	if !strings.Contains(got, "── ERROR ──") {
		t.Errorf("expected header even without source, got:\n%s", got)
	}
	if !strings.Contains(got, "some error") {
		t.Errorf("expected message, got:\n%s", got)
	}
	// No source line or caret should appear
	if strings.Contains(got, " │ ") {
		t.Errorf("expected no source line without source, got:\n%s", got)
	}
}

func TestRenderPretty_MultiDigitLineNumber(t *testing.T) {
	d := Diagnostic{
		File:     "app.kuki",
		Line:     123,
		Col:      2,
		Severity: "error",
		Message:  "bad thing",
	}
	// Build 123 lines
	lines := make([]string, 123)
	lines[122] = "x := broken()"
	got := d.RenderPretty(lines, false)

	if !strings.Contains(got, "123 │ x := broken()") {
		t.Errorf("expected 3-digit line number, got:\n%s", got)
	}
	// Gutter alignment: "   " (3 spaces for "123") + " │ " + " " (col 2) + "^"
	if !strings.Contains(got, "    │  ^") {
		t.Errorf("expected properly aligned caret for col 2, got:\n%s", got)
	}
}

func TestRenderPretty_ColorError(t *testing.T) {
	d := Diagnostic{
		File:     "app.kuki",
		Line:     1,
		Col:      1,
		Severity: "error",
		Message:  "test",
	}
	source := []string{"hello"}
	got := d.RenderPretty(source, true)

	if !strings.Contains(got, "\033[1;31m") {
		t.Errorf("expected red ANSI code for error, got:\n%s", got)
	}
	if !strings.Contains(got, "\033[0m") {
		t.Errorf("expected ANSI reset, got:\n%s", got)
	}
}

func TestRenderPretty_ColorWarning(t *testing.T) {
	d := Diagnostic{
		File:     "app.kuki",
		Line:     1,
		Col:      1,
		Severity: "warning",
		Message:  "test",
	}
	source := []string{"hello"}
	got := d.RenderPretty(source, true)

	if !strings.Contains(got, "\033[1;33m") {
		t.Errorf("expected yellow ANSI code for warning, got:\n%s", got)
	}
}

func TestRenderCompact(t *testing.T) {
	d := Diagnostic{
		File:     "app.kuki",
		Line:     10,
		Col:      5,
		Severity: "error",
		Message:  "undefined: foo",
	}
	got := d.RenderCompact()
	want := "error: app.kuki:10:5: undefined: foo"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
