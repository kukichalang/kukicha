package codegen

import (
	"strings"
	"testing"
)

func TestBraceBlockFunction(t *testing.T) {
	input := "func main() {\n    x := 1\n    print(x)\n}\n"
	output := generateSource(t, input)
	if !strings.Contains(output, "func main()") {
		t.Errorf("expected func main(), got: %s", output)
	}
	if !strings.Contains(output, "x := 1") {
		t.Errorf("expected x := 1, got: %s", output)
	}
}

func TestBraceBlockIfElse(t *testing.T) {
	input := "func main() {\n    x := 1\n    if x > 0 {\n        print(\"positive\")\n    } else {\n        print(\"negative\")\n    }\n}\n"
	output := generateSource(t, input)
	if !strings.Contains(output, "if (x > 0)") {
		t.Errorf("expected if (x > 0), got: %s", output)
	}
	if !strings.Contains(output, "} else {") {
		t.Errorf("expected } else {, got: %s", output)
	}
}

func TestBraceBlockForCondition(t *testing.T) {
	// Kukicha-style for-condition with brace block
	input := "func main() {\n    x := 0\n    for x < 10 {\n        x = x + 1\n    }\n}\n"
	output := generateSource(t, input)
	if !strings.Contains(output, "for (x < 10)") {
		t.Errorf("expected for (x < 10), got: %s", output)
	}
}

func TestBraceBlockBareFor(t *testing.T) {
	// Bare for loop with braces (infinite loop)
	input := "func main() {\n    for {\n        break\n    }\n}\n"
	output := generateSource(t, input)
	if !strings.Contains(output, "for {") {
		t.Errorf("expected for {, got: %s", output)
	}
}

func TestBraceBlockSingleLine(t *testing.T) {
	input := "func Test(x int) int {\n    if x > 0 { return x }\n    return 0\n}\n"
	output := generateSource(t, input)
	if !strings.Contains(output, "return x") {
		t.Errorf("expected return x, got: %s", output)
	}
	if !strings.Contains(output, "return 0") {
		t.Errorf("expected return 0, got: %s", output)
	}
}

func TestBraceBlockMatchesIndentBlock(t *testing.T) {
	// Brace and indent versions should produce identical Go output
	braceInput := "func Add(a int, b int) int {\n    return a + b\n}\n"
	indentInput := "func Add(a int, b int) int\n    return a + b\n"

	braceOutput := generateSource(t, braceInput)
	indentOutput := generateSource(t, indentInput)

	if !strings.Contains(braceOutput, "return (a + b)") {
		t.Errorf("brace: expected return (a + b), got: %s", braceOutput)
	}
	if !strings.Contains(indentOutput, "return (a + b)") {
		t.Errorf("indent: expected return (a + b), got: %s", indentOutput)
	}
}

func TestBraceBlockNestedIf(t *testing.T) {
	input := `func main() {
    x := 5
    if x > 2 {
        if x < 10 {
            print(x)
        }
    }
}
`
	output := generateSource(t, input)
	if !strings.Contains(output, "if (x > 2)") {
		t.Errorf("expected if (x > 2), got: %s", output)
	}
	if !strings.Contains(output, "if (x < 10)") {
		t.Errorf("expected if (x < 10), got: %s", output)
	}
}

func TestBraceBlockMixedFile(t *testing.T) {
	// One function with braces, another with indentation in the same file
	input := `func BraceFunc() int {
    return 42
}

func IndentFunc() int
    return 99
`
	output := generateSource(t, input)
	if !strings.Contains(output, "return 42") {
		t.Errorf("expected return 42 from brace func, got: %s", output)
	}
	if !strings.Contains(output, "return 99") {
		t.Errorf("expected return 99 from indent func, got: %s", output)
	}
}

func TestBraceBlockIfErrNilReturn(t *testing.T) {
	// The canonical Go error handling pattern (single-line)
	input := `func main() {
    if err != nil { return }
}
`
	output := generateSource(t, input)
	if !strings.Contains(output, "if (err != nil)") {
		t.Errorf("expected if (err != nil), got: %s", output)
	}
	if !strings.Contains(output, "return") {
		t.Errorf("expected return, got: %s", output)
	}
}

func TestBraceBlockWithReturnType(t *testing.T) {
	input := "func Double(x int) int {\n    return x * 2\n}\n"
	output := generateSource(t, input)
	if !strings.Contains(output, "func Double(x int) int") {
		t.Errorf("expected func signature with return type, got: %s", output)
	}
	if !strings.Contains(output, "return (x * 2)") {
		t.Errorf("expected return (x * 2), got: %s", output)
	}
}

func TestBraceBlockElseIf(t *testing.T) {
	input := `func classify(x int) string {
    if x > 0 {
        return "positive"
    } else if x < 0 {
        return "negative"
    } else {
        return "zero"
    }
}
`
	output := generateSource(t, input)
	if !strings.Contains(output, `"positive"`) {
		t.Errorf("expected positive, got: %s", output)
	}
	if !strings.Contains(output, `"negative"`) {
		t.Errorf("expected negative, got: %s", output)
	}
	if !strings.Contains(output, `"zero"`) {
		t.Errorf("expected zero, got: %s", output)
	}
}

func TestBraceBlockDefer(t *testing.T) {
	input := `func main() {
    defer print("done")
    print("working")
}
`
	output := generateSource(t, input)
	if !strings.Contains(output, "defer") {
		t.Errorf("expected defer, got: %s", output)
	}
}

func TestBraceBlockForRange(t *testing.T) {
	input := `func main() {
    items := []string{"a", "b", "c"}
    for item in items {
        print(item)
    }
}
`
	output := generateSource(t, input)
	if !strings.Contains(output, "for _, item := range items") {
		t.Errorf("expected for range, got: %s", output)
	}
}

func TestBraceBlockGoSyntaxOperators(t *testing.T) {
	// Combine brace blocks with Go-style operators from Phase 1
	input := `func check(a bool, b bool) bool {
    if a && b {
        return !a || b
    }
    return false
}
`
	output := generateSource(t, input)
	if !strings.Contains(output, "&&") {
		t.Errorf("expected && operator, got: %s", output)
	}
	if !strings.Contains(output, "||") {
		t.Errorf("expected || operator, got: %s", output)
	}
}
