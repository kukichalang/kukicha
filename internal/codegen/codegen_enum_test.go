package codegen

import (
	"strings"
	"testing"
)

func TestEnumDecl_IntegerValues(t *testing.T) {
	input := `enum Status
    OK = 200
    NotFound = 404
    Error = 500

func main()
    s := Status.OK
`

	output := generateSource(t, input)

	if !strings.Contains(output, "type Status int") {
		t.Errorf("expected 'type Status int', got:\n%s", output)
	}
	if !strings.Contains(output, "StatusOK Status = 200") {
		t.Errorf("expected 'StatusOK Status = 200', got:\n%s", output)
	}
	if !strings.Contains(output, "StatusNotFound Status = 404") {
		t.Errorf("expected 'StatusNotFound Status = 404', got:\n%s", output)
	}
	if !strings.Contains(output, "StatusError Status = 500") {
		t.Errorf("expected 'StatusError Status = 500', got:\n%s", output)
	}
	// Dot access rewritten
	if !strings.Contains(output, "s := StatusOK") {
		t.Errorf("expected 'StatusOK' dot-access rewrite, got:\n%s", output)
	}
}

func TestEnumDecl_StringValues(t *testing.T) {
	input := `enum LogLevel
    Debug = "debug"
    Info = "info"
    Warn = "warn"
    Error = "error"

func main()
    l := LogLevel.Debug
`

	output := generateSource(t, input)

	if !strings.Contains(output, "type LogLevel string") {
		t.Errorf("expected 'type LogLevel string', got:\n%s", output)
	}
	if !strings.Contains(output, `LogLevelDebug LogLevel = "debug"`) {
		t.Errorf("expected LogLevelDebug const, got:\n%s", output)
	}
	if !strings.Contains(output, "l := LogLevelDebug") {
		t.Errorf("expected dot-access rewrite, got:\n%s", output)
	}
}

func TestEnumDecl_UsedInSwitch(t *testing.T) {
	input := `enum Color
    Red = 0
    Green = 1
    Blue = 2

func describe(c Color) string
    switch c
        when Color.Red
            return "red"
        when Color.Green
            return "green"
        when Color.Blue
            return "blue"
    return "unknown"
`

	output := generateSource(t, input)

	if !strings.Contains(output, "case ColorRed:") {
		t.Errorf("expected 'case ColorRed:', got:\n%s", output)
	}
	if !strings.Contains(output, "case ColorGreen:") {
		t.Errorf("expected 'case ColorGreen:', got:\n%s", output)
	}
	if !strings.Contains(output, "case ColorBlue:") {
		t.Errorf("expected 'case ColorBlue:', got:\n%s", output)
	}
}

func TestEnumDecl_UsedAsParam(t *testing.T) {
	input := `enum Direction
    Up = 0
    Down = 1

func move(d Direction)
    return

func main()
    move(Direction.Up)
`

	output := generateSource(t, input)

	if !strings.Contains(output, "move(DirectionUp)") {
		t.Errorf("expected 'move(DirectionUp)', got:\n%s", output)
	}
}

func TestEnumDecl_ConstBlock(t *testing.T) {
	input := `enum Status
    OK = 200
    NotFound = 404
`

	output := generateSource(t, input)

	if !strings.Contains(output, "const (") {
		t.Errorf("expected const block, got:\n%s", output)
	}
}

func TestEnumDecl_StringMethod_Integer(t *testing.T) {
	input := `enum Status
    OK = 200
    NotFound = 404
`

	output := generateSource(t, input)

	if !strings.Contains(output, "func (e Status) String() string {") {
		t.Errorf("expected auto-generated String() method, got:\n%s", output)
	}
	if !strings.Contains(output, `return "OK"`) {
		t.Errorf("expected return \"OK\" in String(), got:\n%s", output)
	}
	if !strings.Contains(output, `return "NotFound"`) {
		t.Errorf("expected return \"NotFound\" in String(), got:\n%s", output)
	}
	if !strings.Contains(output, `fmt.Sprintf("Status(%d)", int(e))`) {
		t.Errorf("expected fallback fmt.Sprintf for unknown int value, got:\n%s", output)
	}
}

func TestEnumDecl_StringMethod_String(t *testing.T) {
	input := `enum LogLevel
    Debug = "debug"
    Info = "info"
`

	output := generateSource(t, input)

	if !strings.Contains(output, "func (e LogLevel) String() string {") {
		t.Errorf("expected auto-generated String() method, got:\n%s", output)
	}
	if !strings.Contains(output, "return string(e)") {
		t.Errorf("expected fallback 'return string(e)' for string enum, got:\n%s", output)
	}
}

func TestEnumDecl_StringMethod_SkipsUserDefined(t *testing.T) {
	input := `enum Status
    OK = 200
    NotFound = 404

func String on s Status string
    return "custom"
`

	output := generateSource(t, input)

	// Should have the user's method
	if !strings.Contains(output, `return "custom"`) {
		t.Errorf("expected user-defined String() method, got:\n%s", output)
	}
	// Should NOT have auto-generated String()
	if strings.Contains(output, `case StatusOK:`) && strings.Contains(output, `return "OK"`) {
		t.Errorf("should not auto-generate String() when user defines one, got:\n%s", output)
	}
}

func TestEnumDecl_OrderIndependent(t *testing.T) {
	// Enum declared after function that uses it
	input := `func main()
    s := Status.OK

enum Status
    OK = 200
    NotFound = 404
`

	output := generateSource(t, input)

	if !strings.Contains(output, "s := StatusOK") {
		t.Errorf("expected dot-access rewrite even when enum is declared after usage, got:\n%s", output)
	}
}
