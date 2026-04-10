package codegen

import (
	"strings"
	"testing"
)

func TestVariantEnum_Interface(t *testing.T) {
	input := `enum Shape
    Circle
        radius float64
    Rectangle
        width  float64
        height float64
    Point
`
	output := generateSource(t, input)

	if !strings.Contains(output, "type Shape interface") {
		t.Errorf("expected sealed interface 'type Shape interface', got:\n%s", output)
	}
	if !strings.Contains(output, "isShape()") {
		t.Errorf("expected unexported marker method 'isShape()', got:\n%s", output)
	}
}

func TestVariantEnum_CaseStructs(t *testing.T) {
	input := `enum Shape
    Circle
        radius float64
    Rectangle
        width  float64
        height float64
    Point
`
	output := generateSource(t, input)

	if !strings.Contains(output, "type Circle struct") {
		t.Errorf("expected 'type Circle struct', got:\n%s", output)
	}
	if !strings.Contains(output, "radius float64") {
		t.Errorf("expected field 'radius float64', got:\n%s", output)
	}
	if !strings.Contains(output, "type Rectangle struct") {
		t.Errorf("expected 'type Rectangle struct', got:\n%s", output)
	}
	if !strings.Contains(output, "width float64") {
		t.Errorf("expected field 'width float64', got:\n%s", output)
	}
	if !strings.Contains(output, "height float64") {
		t.Errorf("expected field 'height float64', got:\n%s", output)
	}
}

func TestVariantEnum_UnitVariant(t *testing.T) {
	input := `enum Shape
    Point
    Circle
        radius float64
`
	output := generateSource(t, input)

	if !strings.Contains(output, "type Point struct{}") {
		t.Errorf("expected 'type Point struct{}' for unit variant, got:\n%s", output)
	}
}

func TestVariantEnum_MarkerMethods(t *testing.T) {
	input := `enum Shape
    Circle
        radius float64
    Point
`
	output := generateSource(t, input)

	if !strings.Contains(output, "func (Circle) isShape()") {
		t.Errorf("expected marker method 'func (Circle) isShape()', got:\n%s", output)
	}
	if !strings.Contains(output, "func (Point) isShape()") {
		t.Errorf("expected marker method 'func (Point) isShape()', got:\n%s", output)
	}
}

func TestVariantEnum_UsedInTypedSwitch(t *testing.T) {
	input := `enum Shape
    Circle
        radius float64
    Point

func area(s Shape) float64
    switch s as v
        when Circle
            return v.radius * v.radius
        when Point
            return 0.0
`
	output := generateSource(t, input)

	if !strings.Contains(output, "switch") {
		t.Errorf("expected switch statement, got:\n%s", output)
	}
	if !strings.Contains(output, "case Circle:") {
		t.Errorf("expected 'case Circle:', got:\n%s", output)
	}
	if !strings.Contains(output, "case Point:") {
		t.Errorf("expected 'case Point:', got:\n%s", output)
	}
}

func TestVariantEnum_PipedSwitchIIFE_PanicUnreachable(t *testing.T) {
	// When a piped switch expression is used as a return value with no
	// otherwise clause, the generated IIFE must include panic("unreachable")
	// so Go's compiler doesn't report "missing return".
	input := `import "fmt"

enum Shape
    Circle
        radius float64
    Point

func describe(s Shape) string
    return s |> switch as v
        when Circle
            return fmt.Sprintf("circle r=%.1f", v.radius)
        when Point
            return "point"
`
	output := generateSource(t, input)

	if !strings.Contains(output, `panic("unreachable")`) {
		t.Errorf("expected panic(\"unreachable\") in IIFE without otherwise, got:\n%s", output)
	}
}

func TestVariantEnum_PipedSwitchIIFE_NoPanicWithOtherwise(t *testing.T) {
	// When otherwise is present, no panic should be injected
	input := `import "fmt"

enum Shape
    Circle
        radius float64
    Point

func describe(s Shape) string
    return s |> switch as v
        when Circle
            return fmt.Sprintf("circle r=%.1f", v.radius)
        otherwise
            return "other"
`
	output := generateSource(t, input)

	if strings.Contains(output, `panic("unreachable")`) {
		t.Errorf("should not have panic when otherwise is present, got:\n%s", output)
	}
}

func TestVariantEnum_NoConstBlock(t *testing.T) {
	// Variant enums must NOT emit a const block (that's for value enums only)
	input := `enum Shape
    Circle
        radius float64
    Point
`
	output := generateSource(t, input)

	if strings.Contains(output, "const (") {
		t.Errorf("variant enum must not emit a const block, got:\n%s", output)
	}
}
