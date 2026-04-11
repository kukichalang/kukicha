package codegen

import (
	"strings"
	"testing"
)

func TestBracketListTypeCodegen(t *testing.T) {
	input := `func f(items []string) int
    return len(items)
`
	output := generateSource(t, input)
	if !strings.Contains(output, "func f(items []string) int") {
		t.Errorf("expected []string param, got: %s", output)
	}
}

func TestBracketMapTypeCodegen(t *testing.T) {
	input := `func f(m map[string]int) int
    return len(m)
`
	output := generateSource(t, input)
	if !strings.Contains(output, "func f(m map[string]int) int") {
		t.Errorf("expected map[string]int param, got: %s", output)
	}
}

func TestBracketNestedTypeCodegen(t *testing.T) {
	// []map[string][]int should produce identical Go to list of map of string to list of int
	bracketInput := `func f(x []map[string][]int)
    y := x
`
	englishInput := `func f(x list of map of string to list of int)
    y := x
`
	bracketOutput := generateSource(t, bracketInput)
	englishOutput := generateSource(t, englishInput)

	if !strings.Contains(bracketOutput, "[]map[string][]int") {
		t.Errorf("bracket: expected []map[string][]int, got: %s", bracketOutput)
	}
	if !strings.Contains(englishOutput, "[]map[string][]int") {
		t.Errorf("english: expected []map[string][]int, got: %s", englishOutput)
	}
}

func TestBracketListLiteralCodegen(t *testing.T) {
	input := `func main()
    x := []int{1, 2, 3}
`
	output := generateSource(t, input)
	if !strings.Contains(output, "[]int{1, 2, 3}") {
		t.Errorf("expected []int{1, 2, 3}, got: %s", output)
	}
}

func TestBracketMapLiteralCodegen(t *testing.T) {
	input := `func main()
    x := map[string]int{"a": 1, "b": 2}
`
	output := generateSource(t, input)
	if !strings.Contains(output, `map[string]int{"a": 1, "b": 2}`) {
		t.Errorf("expected map literal, got: %s", output)
	}
}

func TestBracketReturnTypeCodegen(t *testing.T) {
	input := `func f() []string
    return []string{"a", "b"}
`
	output := generateSource(t, input)
	if !strings.Contains(output, "func f() []string") {
		t.Errorf("expected []string return type, got: %s", output)
	}
	if !strings.Contains(output, `[]string{"a", "b"}`) {
		t.Errorf("expected []string literal, got: %s", output)
	}
}

func TestBracketEmptyShorthandCodegen(t *testing.T) {
	input := `func main()
    x := []string
    y := map[string]int
`
	output := generateSource(t, input)
	// []string typed-empty should produce the zero value
	if !strings.Contains(output, "[]string{}") {
		t.Errorf("expected []string{} zero value, got: %s", output)
	}
	if !strings.Contains(output, "map[string]int{}") {
		t.Errorf("expected map[string]int{} zero value, got: %s", output)
	}
}

// --- Feature 2: Inferred collection literals ---

func TestUntypedMapLiteralCodegen(t *testing.T) {
	input := `func main()
    x := {"name": "Alice", "role": "admin"}
`
	output := generateSource(t, input)
	if !strings.Contains(output, `map[any]any{"name": "Alice", "role": "admin"}`) {
		t.Errorf("expected map[any]any literal, got: %s", output)
	}
}

func TestUntypedMapLiteralEmptyCodegen(t *testing.T) {
	input := `func main()
    x := {}
`
	output := generateSource(t, input)
	if !strings.Contains(output, "map[any]any{}") {
		t.Errorf("expected empty map[any]any{}, got: %s", output)
	}
}

// --- Untyped composite literal resolved codegen ---

func TestUntypedLiteralVarDeclStructCodegen(t *testing.T) {
	input := `type Config
    host string
    port int

func makeConfig() Config
    return {host: "localhost", port: 8080}
`
	output := generateAnalyzedSource(t, input)
	if !strings.Contains(output, `Config{host: "localhost", port: 8080}`) {
		t.Errorf("expected resolved struct literal, got: %s", output)
	}
}

func TestUntypedLiteralReturnStructCodegen(t *testing.T) {
	input := `type Point
    x int
    y int

func origin() Point
    return {x: 0, y: 0}
`
	output := generateAnalyzedSource(t, input)
	if !strings.Contains(output, "return Point{x: 0, y: 0}") {
		t.Errorf("expected resolved struct return, got: %s", output)
	}
}

func TestUntypedLiteralReturnMapCodegen(t *testing.T) {
	input := `func headers() map of string to string
    return {"Content-Type": "text/html"}
`
	output := generateAnalyzedSource(t, input)
	if !strings.Contains(output, `return map[string]string{"Content-Type": "text/html"}`) {
		t.Errorf("expected resolved map return, got: %s", output)
	}
}

func TestUntypedLiteralReturnSliceCodegen(t *testing.T) {
	input := `func nums() list of int
    return {1, 2, 3}
`
	output := generateAnalyzedSource(t, input)
	if !strings.Contains(output, "return []int{1, 2, 3}") {
		t.Errorf("expected resolved slice return, got: %s", output)
	}
}

func TestUntypedLiteralCallArgCodegen(t *testing.T) {
	input := `type Point
    x int
    y int

func draw(p Point)
    print(p)

func main()
    draw({x: 1, y: 2})
`
	output := generateAnalyzedSource(t, input)
	if !strings.Contains(output, "draw(Point{x: 1, y: 2})") {
		t.Errorf("expected resolved struct arg, got: %s", output)
	}
}

func TestUntypedPositionalCodegen(t *testing.T) {
	input := `func main()
    x := {1, 2, 3}
`
	output := generateSource(t, input)
	if !strings.Contains(output, "[]any{1, 2, 3}") {
		t.Errorf("expected positional fallback, got: %s", output)
	}
}

func TestUntypedLiteralAssignCodegen(t *testing.T) {
	input := `type Config
    host string
    port int

func main()
    c := Config{host: "", port: 0}
    c = {host: "prod", port: 443}
`
	output := generateAnalyzedSource(t, input)
	if !strings.Contains(output, `Config{host: "prod", port: 443}`) {
		t.Errorf("expected resolved struct assignment, got: %s", output)
	}
}

func TestUntypedLiteralInTypedListCodegen(t *testing.T) {
	input := `type Point
    x int
    y int

func main()
    points := list of Point{{x: 1, y: 2}, {x: 3, y: 4}}
`
	output := generateAnalyzedSource(t, input)
	if !strings.Contains(output, "Point{x: 1, y: 2}") {
		t.Errorf("expected resolved struct in list, got: %s", output)
	}
	if !strings.Contains(output, "Point{x: 3, y: 4}") {
		t.Errorf("expected resolved struct in list (2nd), got: %s", output)
	}
}

func TestMixedBracketAndEnglishCodegen(t *testing.T) {
	input := `func f(a list of string, b []int) map[string]bool
    x := a
    return map[string]bool
`
	output := generateSource(t, input)
	if !strings.Contains(output, "func f(a []string, b []int) map[string]bool") {
		t.Errorf("expected mixed params to produce same Go output, got: %s", output)
	}
}
