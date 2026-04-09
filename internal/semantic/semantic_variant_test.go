package semantic

import (
	"strings"
	"testing"
)

func TestVariantEnum_Valid(t *testing.T) {
	input := `enum Shape
    Circle
        radius float64
    Rectangle
        width  float64
        height float64
    Point

func main()
    c := Circle{radius: 5.0}
    _ = c
`
	_, errs := analyzeSource(t, input)
	if len(errs) > 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestVariantEnum_UnitVariant(t *testing.T) {
	input := `enum Direction
    North
    South
    East
    West
`
	_, errs := analyzeSource(t, input)
	if len(errs) > 0 {
		t.Errorf("expected no errors for unit variants, got: %v", errs)
	}
}

func TestVariantEnum_MixedWithValueCases_Error(t *testing.T) {
	input := `enum Bad
    OK = 200
    Circle
        radius float64
`
	_, errs := analyzeSource(t, input)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "cannot mix") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'cannot mix' error for mixed enum, got: %v", errs)
	}
}

func TestVariantEnum_ExhaustivenessWarning(t *testing.T) {
	input := `enum Shape
    Circle
        radius float64
    Rectangle
        width  float64
        height float64
    Point

func area(s Shape) float64
    switch s as v
        when Circle
            return v.radius * v.radius
        when Rectangle
            return v.width * v.height
`
	result := analyzeSourceResult(t, input)
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w.Error(), "missing cases") && strings.Contains(w.Error(), "Point") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected exhaustiveness warning mentioning Point, got: %v", result.Warnings)
	}
}

func TestVariantEnum_ExhaustivenessNoWarningWithOtherwise(t *testing.T) {
	input := `enum Shape
    Circle
        radius float64
    Point

func area(s Shape) float64
    switch s as v
        when Circle
            return v.radius * v.radius
        otherwise
            return 0.0
`
	result := analyzeSourceResult(t, input)
	for _, w := range result.Warnings {
		if strings.Contains(w.Error(), "missing cases") {
			t.Errorf("unexpected exhaustiveness warning with otherwise: %v", w)
		}
	}
}

func TestVariantEnum_ExhaustivenessAllCovered(t *testing.T) {
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
	result := analyzeSourceResult(t, input)
	for _, w := range result.Warnings {
		if strings.Contains(w.Error(), "missing cases") {
			t.Errorf("unexpected exhaustiveness warning when all covered: %v", w)
		}
	}
}
