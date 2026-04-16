package codegen

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// lineOfSubstring returns the 1-indexed line number in src where sub first appears.
// Fails the test if sub is not found or appears more than once.
func lineOfSubstring(t *testing.T, src, sub string) int {
	t.Helper()
	lines := strings.Split(src, "\n")
	hit := 0
	for i, ln := range lines {
		if strings.Contains(ln, sub) {
			hit = i + 1
		}
	}
	if hit == 0 {
		t.Fatalf("substring %q not found in source:\n%s", sub, src)
	}
	return hit
}

// lineDirectiveFor returns the line number from the most recent
// `//line test.kuki:N` directive that appears before the first occurrence
// of the given target fragment in the generated Go output.
func lineDirectiveFor(t *testing.T, generated, target string) int {
	t.Helper()
	before, _, found := strings.Cut(generated, target)
	if !found {
		t.Fatalf("target %q not found in generated output:\n%s", target, generated)
	}
	re := regexp.MustCompile(`//line\s+test\.kuki:(\d+)`)
	matches := re.FindAllStringSubmatch(before, -1)
	if len(matches) == 0 {
		t.Fatalf("no //line directive before %q in generated output:\n%s", target, generated)
	}
	last := matches[len(matches)-1][1]
	var n int
	if _, err := fmt.Sscanf(last, "%d", &n); err != nil {
		t.Fatalf("could not parse line from directive %q: %v", last, err)
	}
	return n
}

// Regression test for the //line off-by-one bug reported in
// docs/plans/codegen-fix.md: a statement following a `go` block + blank
// line was getting the blank line's position rather than its own.
func TestLineDirectiveAfterGoBlockWithBlankLine(t *testing.T) {
	input := `func Go(fn func())
    go
        fn()

    return
`

	out := generateSource(t, input)

	srcLine := lineOfSubstring(t, input, "return")
	dirLine := lineDirectiveFor(t, out, "return\n")
	if dirLine != srcLine {
		t.Errorf("return stmt: expected //line test.kuki:%d, got :%d\nsource:\n%s\ngenerated:\n%s",
			srcLine, dirLine, input, out)
	}
}

// The `Parallel` counterexample from the bug report: `go` inside a
// for-loop, followed by a blank line, then a same-scope statement.
func TestLineDirectiveAfterGoBlockInsideForLoop(t *testing.T) {
	input := `func Parallel(many tasks func())
    for task in tasks
        t := task
        go
            t()

    finish()
`

	out := generateSource(t, input)

	srcLine := lineOfSubstring(t, input, "finish()")
	dirLine := lineDirectiveFor(t, out, "finish()")
	if dirLine != srcLine {
		t.Errorf("finish() call: expected //line test.kuki:%d, got :%d\nsource:\n%s\ngenerated:\n%s",
			srcLine, dirLine, input, out)
	}
}

// Multiple blank lines between the `go` block and the trailing
// statement should still attribute the trailing statement to its
// own source line.
func TestLineDirectiveAfterGoBlockWithMultipleBlankLines(t *testing.T) {
	input := `func Go(fn func())
    go
        fn()



    return
`

	out := generateSource(t, input)

	srcLine := lineOfSubstring(t, input, "return")
	dirLine := lineDirectiveFor(t, out, "return\n")
	if dirLine != srcLine {
		t.Errorf("return stmt: expected //line test.kuki:%d, got :%d\nsource:\n%s\ngenerated:\n%s",
			srcLine, dirLine, input, out)
	}
}
