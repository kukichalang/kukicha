package codegen

import (
	"strings"
	"testing"
)

// Tests for Go syntax passthrough (Phase 1: Operator and Keyword Aliases).
// Each test verifies that Go-native syntax produces identical codegen output
// as the equivalent Kukicha syntax.

func TestGoAndAndOperator(t *testing.T) {
	input := `func Test(a bool, b bool) bool
    return a && b
`
	output := generateSource(t, input)
	if !strings.Contains(output, "&&") {
		t.Errorf("expected && operator, got: %s", output)
	}
}

func TestGoOrOrOperator(t *testing.T) {
	input := `func Test(a bool, b bool) bool
    return a || b
`
	output := generateSource(t, input)
	if !strings.Contains(output, "||") {
		t.Errorf("expected || operator, got: %s", output)
	}
}

func TestGoAndAndOrOrMixed(t *testing.T) {
	// Go-style && and || should produce identical output to Kukicha and/or
	kukichaInput := `func Test(a bool, b bool) bool
    return a and b or a
`
	goInput := `func Test(a bool, b bool) bool
    return a && b || a
`
	kukichaOutput := generateSource(t, kukichaInput)
	goOutput := generateSource(t, goInput)

	// Both should contain && and ||
	if !strings.Contains(goOutput, "&&") {
		t.Errorf("Go syntax: expected && operator, got: %s", goOutput)
	}
	if !strings.Contains(goOutput, "||") {
		t.Errorf("Go syntax: expected || operator, got: %s", goOutput)
	}
	if !strings.Contains(kukichaOutput, "&&") {
		t.Errorf("Kukicha syntax: expected && operator, got: %s", kukichaOutput)
	}
	if !strings.Contains(kukichaOutput, "||") {
		t.Errorf("Kukicha syntax: expected || operator, got: %s", kukichaOutput)
	}
}

func TestGoBangOperator(t *testing.T) {
	input := `func Test(a bool) bool
    return !a
`
	output := generateSource(t, input)
	if !strings.Contains(output, "!a") {
		t.Errorf("expected !a, got: %s", output)
	}
}

func TestGoNilKeyword(t *testing.T) {
	input := `func Test(s reference string) bool
    return s == nil
`
	output := generateSource(t, input)
	if !strings.Contains(output, "nil") {
		t.Errorf("expected nil, got: %s", output)
	}
}

func TestGoStarTypeAnnotation(t *testing.T) {
	input := `func Test(s *string) *string
    return s
`
	output := generateSource(t, input)
	if !strings.Contains(output, "func Test(s *string) *string") {
		t.Errorf("expected *string type annotations, got: %s", output)
	}
}

func TestGoStarTypeMatchesReference(t *testing.T) {
	// *string should produce identical output to "reference string"
	goInput := `func Test(s *string) *string
    return s
`
	kukichaInput := `func Test(s reference string) reference string
    return s
`
	goOutput := generateSource(t, goInput)
	kukichaOutput := generateSource(t, kukichaInput)

	if !strings.Contains(goOutput, "*string") {
		t.Errorf("Go syntax: expected *string, got: %s", goOutput)
	}
	if !strings.Contains(kukichaOutput, "*string") {
		t.Errorf("Kukicha syntax: expected *string, got: %s", kukichaOutput)
	}
}

func TestGoAmpersandAddressOf(t *testing.T) {
	input := `func Test() *int
    x := 42
    return &x
`
	output := generateSource(t, input)
	if !strings.Contains(output, "&x") {
		t.Errorf("expected &x, got: %s", output)
	}
}

func TestGoAmpersandMatchesReferenceOf(t *testing.T) {
	// &x should produce identical output to "reference of x"
	goInput := `func Test() *int
    x := 42
    return &x
`
	kukichaInput := `func Test() reference int
    x := 42
    return reference of x
`
	goOutput := generateSource(t, goInput)
	kukichaOutput := generateSource(t, kukichaInput)

	if !strings.Contains(goOutput, "&x") {
		t.Errorf("Go syntax: expected &x, got: %s", goOutput)
	}
	if !strings.Contains(kukichaOutput, "&x") {
		t.Errorf("Kukicha syntax: expected &x, got: %s", kukichaOutput)
	}
}

func TestGoStarNestedTypes(t *testing.T) {
	// *[]string — pointer to a slice
	input := `func Test(s *[]string)
    print("ok")
`
	output := generateSource(t, input)
	if !strings.Contains(output, "*[]string") {
		t.Errorf("expected *[]string, got: %s", output)
	}
}

func TestGoDoublePointer(t *testing.T) {
	// **int — pointer to pointer
	input := `func Test(s **int)
    print("ok")
`
	output := generateSource(t, input)
	if !strings.Contains(output, "**int") {
		t.Errorf("expected **int, got: %s", output)
	}
}
