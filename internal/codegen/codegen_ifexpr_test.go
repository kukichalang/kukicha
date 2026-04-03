package codegen

import (
	"strings"
	"testing"
)

func TestIfExpressionSimple(t *testing.T) {
	input := `func Classify(age int) string
    access := if age >= 18 then "Granted" else "Denied"
    return access
`
	output := generateSource(t, input)

	if !strings.Contains(output, `func()`) {
		t.Errorf("expected IIFE for if-expression, got:\n%s", output)
	}
	if !strings.Contains(output, `if (age >= 18)`) {
		t.Errorf("expected condition in if-expression, got:\n%s", output)
	}
	if !strings.Contains(output, `return "Granted"`) {
		t.Errorf("expected then branch, got:\n%s", output)
	}
	if !strings.Contains(output, `return "Denied"`) {
		t.Errorf("expected else branch, got:\n%s", output)
	}
}

func TestIfExpressionChained(t *testing.T) {
	input := `func Grade(score int) string
    label := if score >= 90 then "A" else if score >= 80 then "B" else "C"
    return label
`
	output := generateSource(t, input)

	if !strings.Contains(output, `if (score >= 90)`) {
		t.Errorf("expected first condition, got:\n%s", output)
	}
	if !strings.Contains(output, `return "A"`) {
		t.Errorf("expected first then branch, got:\n%s", output)
	}
	if !strings.Contains(output, `if (score >= 80)`) {
		t.Errorf("expected chained condition, got:\n%s", output)
	}
	if !strings.Contains(output, `return "B"`) {
		t.Errorf("expected chained then branch, got:\n%s", output)
	}
	if !strings.Contains(output, `return "C"`) {
		t.Errorf("expected final else branch, got:\n%s", output)
	}
}

func TestIfExpressionInline(t *testing.T) {
	input := `func Max(a int, b int) int
    return if a > b then a else b
`
	output := generateSource(t, input)

	if !strings.Contains(output, `func()`) {
		t.Errorf("expected IIFE for inline if-expression, got:\n%s", output)
	}
}
