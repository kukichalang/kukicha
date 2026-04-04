package blend

import (
	"strings"
	"testing"
)

func TestOperatorPatterns(t *testing.T) {
	src := []byte(`package main

func main() {
	if a && b || !c {
		return
	}
}
`)
	suggestions, err := BlendFile("test.go", src, PatternSet{Operators: true})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{"&& → and": false, "|| → or": false, "! → not": false}
	for _, s := range suggestions {
		want[s.Message] = true
	}
	for msg, found := range want {
		if !found {
			t.Errorf("missing suggestion: %s", msg)
		}
	}
}

func TestComparisonPatterns(t *testing.T) {
	src := []byte(`package main

func main() {
	if x == 1 && y != nil {
		return
	}
}
`)
	suggestions, err := BlendFile("test.go", src, PatternSet{Comparisons: true})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{"== → equals": false, "!= → isnt": false, "nil → empty": false}
	for _, s := range suggestions {
		want[s.Message] = true
	}
	for msg, found := range want {
		if !found {
			t.Errorf("missing suggestion: %s", msg)
		}
	}
}

func TestTypePatterns(t *testing.T) {
	src := []byte(`package main

func process(items []string, counts map[string]int, p *int) {
}
`)
	suggestions, err := BlendFile("test.go", src, PatternSet{Types: true})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{
		"[]string → list of string":           false,
		"map[string]int → map of string to int": false,
		"*int → reference int":                false,
	}
	for _, s := range suggestions {
		want[s.Message] = true
	}
	for msg, found := range want {
		if !found {
			t.Errorf("missing suggestion: %s", msg)
		}
	}
}

func TestAddressOfPattern(t *testing.T) {
	src := []byte(`package main

func main() {
	x := 42
	p := &x
	_ = p
}
`)
	suggestions, err := BlendFile("test.go", src, PatternSet{Types: true})
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, s := range suggestions {
		if s.Pattern == "types" && strings.Contains(s.Message, "&x → reference of x") {
			found = true
		}
	}
	if !found {
		t.Error("missing address-of suggestion: &x → reference of x")
	}
}

func TestOnerrPattern(t *testing.T) {
	src := []byte(`package main

import "fmt"

func main() {
	result, err := fmt.Println("hello")
	if err != nil {
		return
	}
	_ = result
}
`)
	suggestions, err := BlendFile("test.go", src, PatternSet{Onerr: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(suggestions) == 0 {
		t.Fatal("expected at least one onerr suggestion")
	}
	s := suggestions[0]
	if s.Pattern != "onerr" {
		t.Errorf("expected pattern 'onerr', got %q", s.Pattern)
	}
	if s.Replacement != "onerr return" {
		t.Errorf("expected replacement 'onerr return', got %q", s.Replacement)
	}
}

func TestOnerrWithReturnValues(t *testing.T) {
	src := []byte(`package main

import "fmt"

func doSomething() (int, error) {
	n, err := fmt.Println("hello")
	if err != nil {
		return 0, err
	}
	return n, nil
}
`)
	suggestions, err := BlendFile("test.go", src, PatternSet{Onerr: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(suggestions) == 0 {
		t.Fatal("expected at least one onerr suggestion")
	}
	s := suggestions[0]
	if s.Replacement != "onerr return 0" {
		t.Errorf("expected replacement 'onerr return 0', got %q", s.Replacement)
	}
}

func TestPackagePattern(t *testing.T) {
	src := []byte(`package main

func main() {}
`)
	suggestions, err := BlendFile("test.go", src, PatternSet{Package: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}
	if suggestions[0].Replacement != "petiole" {
		t.Errorf("expected replacement 'petiole', got %q", suggestions[0].Replacement)
	}
}

func TestApply(t *testing.T) {
	src := []byte(`package main

func main() {
	if x == 1 && y != nil {
		return
	}
}
`)
	suggestions, err := BlendFile("test.go", src, AllPatterns())
	if err != nil {
		t.Fatal(err)
	}

	result := Apply(src, suggestions)
	output := string(result)

	if !strings.Contains(output, "petiole main") {
		t.Error("expected 'petiole main' in output")
	}
	if !strings.Contains(output, "equals") {
		t.Error("expected 'equals' in output")
	}
	if !strings.Contains(output, "and") {
		t.Error("expected 'and' in output")
	}
	if !strings.Contains(output, "isnt") {
		t.Error("expected 'isnt' in output")
	}
	if !strings.Contains(output, "empty") {
		t.Error("expected 'empty' in output")
	}
	// Original tokens should not remain
	if strings.Contains(output, "package main") {
		t.Error("'package main' should be replaced with 'petiole main'")
	}
}

func TestApplyOverlapping(t *testing.T) {
	// When onerr and comparisons both match, they overlap. The one with
	// the higher start offset (inner) is applied; the outer is skipped.
	src := []byte(`package main

func main() {
	err := doSomething()
	if err != nil {
		return
	}
}
`)
	suggestions, err := BlendFile("test.go", src, PatternSet{Onerr: true, Comparisons: true})
	if err != nil {
		t.Fatal(err)
	}

	result := Apply(src, suggestions)
	output := string(result)

	// Either onerr or comparison replacement should appear, not a garbled mix.
	hasOnerr := strings.Contains(output, "onerr return")
	hasIsnt := strings.Contains(output, "isnt")
	if !hasOnerr && !hasIsnt {
		t.Error("expected either onerr or comparison replacement in output")
	}
}

func TestDiff(t *testing.T) {
	original := []byte("package main\n\nfunc main() {\n\tif x == 1 {\n\t}\n}\n")
	suggestions, err := BlendFile("test.go", original, PatternSet{Comparisons: true, Package: true})
	if err != nil {
		t.Fatal(err)
	}

	blended := Apply(original, suggestions)
	d := Diff("test.go", "test.kuki", original, blended)

	if !strings.Contains(d, "--- test.go") {
		t.Error("diff missing original filename header")
	}
	if !strings.Contains(d, "+++ test.kuki") {
		t.Error("diff missing blended filename header")
	}
	if !strings.Contains(d, "-package main") {
		t.Error("diff missing deletion of 'package main'")
	}
	if !strings.Contains(d, "+petiole main") {
		t.Error("diff missing addition of 'petiole main'")
	}
}

func TestParsePatterns(t *testing.T) {
	tests := []struct {
		input string
		want  PatternSet
	}{
		{"", AllPatterns()},
		{"operators", PatternSet{Operators: true}},
		{"operators,onerr", PatternSet{Operators: true, Onerr: true}},
		{"types, comparisons", PatternSet{Types: true, Comparisons: true}},
	}
	for _, tt := range tests {
		got := ParsePatterns(tt.input)
		if got != tt.want {
			t.Errorf("ParsePatterns(%q) = %+v, want %+v", tt.input, got, tt.want)
		}
	}
}

func TestBlendFileParseError(t *testing.T) {
	src := []byte("this is not valid go")
	_, err := BlendFile("bad.go", src, AllPatterns())
	if err == nil {
		t.Error("expected parse error for invalid Go source")
	}
}

func TestNoSuggestions(t *testing.T) {
	src := []byte(`package main

func main() {
	x := 1 + 2
	_ = x
}
`)
	// Only check operators pattern — no &&, ||, ! in this code
	suggestions, err := BlendFile("test.go", src, PatternSet{Operators: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions, got %d", len(suggestions))
	}
}

func TestNestedTypes(t *testing.T) {
	src := []byte(`package main

func main() {
	var x [][]string
	_ = x
}
`)
	suggestions, err := BlendFile("test.go", src, PatternSet{Types: true})
	if err != nil {
		t.Fatal(err)
	}

	// Should get suggestions for both outer [][]string and inner []string
	if len(suggestions) < 2 {
		t.Errorf("expected at least 2 type suggestions for [][]string, got %d", len(suggestions))
	}
}
