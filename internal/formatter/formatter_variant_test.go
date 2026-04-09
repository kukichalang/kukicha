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
