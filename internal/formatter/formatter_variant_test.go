package formatter

import "testing"

func TestFormatVariantEnum_WithFields(t *testing.T) {
	source := `enum Shape
    Circle
        radius float64
    Rectangle
        width  float64
        height float64
    Point
`
	expected := `enum Shape
    Circle
        radius float64
    Rectangle
        width float64
        height float64
    Point
`
	assertFormatted(t, source, expected)
}

func TestFormatVariantEnum_UnitOnly(t *testing.T) {
	source := `enum Direction
    North
    South
    East
    West
`
	expected := `enum Direction
    North
    South
    East
    West
`
	assertFormatted(t, source, expected)
}

func TestFormatIsExpression_NoBinding(t *testing.T) {
	source := `func check(s Shape) bool
    return s is Circle
`
	expected := `func check(s Shape) bool
    return s is Circle
`
	assertFormatted(t, source, expected)
}

func TestFormatIsExpression_WithBinding(t *testing.T) {
	source := `func area(s Shape) float64
    if s is Circle as c
        return c.radius * c.radius
    return 0.0
`
	expected := `func area(s Shape) float64
    if s is Circle as c
        return c.radius * c.radius
    return 0.0
`
	assertFormatted(t, source, expected)
}

func TestFormatIsExpression_NotPreservesParens(t *testing.T) {
	source := `func check(s Shape) bool
    if not (s is Circle)
        return false
    return true
`
	expected := `func check(s Shape) bool
    if not (s is Circle)
        return false
    return true
`
	assertFormatted(t, source, expected)
}
