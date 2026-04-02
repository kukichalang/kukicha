package semantic

import (
	"github.com/kukichalang/kukicha/internal/ast"
	"github.com/kukichalang/kukicha/internal/parser"
	"strings"
	"testing"
)

func TestStructLiteralValidFields(t *testing.T) {
	input := `type Person
    Name string
    Age int

func Test() Person
    return Person{Name: "Alice", Age: 30}
`

	p, err := parser.New(input, "test.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	analyzer := New(program)
	errors := analyzer.Analyze()

	if len(errors) > 0 {
		t.Fatalf("unexpected errors for valid struct literal: %v", errors)
	}
}

func TestStructLiteralUnknownField(t *testing.T) {
	input := `type Person
    Name string
    Age int

func Test() Person
    return Person{Name: "Alice", Score: 42}
`

	p, err := parser.New(input, "test.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	analyzer := New(program)
	errors := analyzer.Analyze()

	if len(errors) == 0 {
		t.Fatal("expected error for unknown struct field, got none")
	}

	found := false
	for _, e := range errors {
		if strings.Contains(e.Error(), "unknown field 'Score' on struct 'Person'") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'unknown field' error, got: %v", errors)
	}
}

func TestStructLiteralFieldTypeMismatch(t *testing.T) {
	input := `type Point
    X int
    Y int

func Test() Point
    return Point{X: "not-an-int", Y: 2}
`

	p, err := parser.New(input, "test.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	analyzer := New(program)
	errors := analyzer.Analyze()

	if len(errors) == 0 {
		t.Fatal("expected type mismatch error for struct field, got none")
	}

	found := false
	for _, e := range errors {
		if strings.Contains(e.Error(), "cannot use") && strings.Contains(e.Error(), "field 'X'") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected type mismatch error for field 'X', got: %v", errors)
	}
}

func TestMethodReturnTypeResolution(t *testing.T) {
	input := `type Counter
    value int

func GetValue on c Counter int
    return c.value

func main()
    c := Counter{value: 0}
    v := c.GetValue()
    print(v)
`

	analyzer, errors := analyzeSource(t, input)
	_ = analyzer

	for _, e := range errors {
		t.Errorf("unexpected error: %v", e)
	}

	// Verify the method return count was recorded
	for expr, count := range analyzer.ReturnCounts() {
		if mc, ok := expr.(*ast.MethodCallExpr); ok {
			if mc.Method.Value == "GetValue" {
				if count != 1 {
					t.Errorf("expected return count 1 for GetValue, got %d", count)
				}
				return
			}
		}
	}
}

func TestFieldAccessTypeResolution(t *testing.T) {
	input := `type User
    name string
    age int

func main()
    u := User{name: "Alice", age: 30}
    n := u.name
    print(n)
`

	analyzer, errors := analyzeSource(t, input)
	_ = analyzer

	for _, e := range errors {
		t.Errorf("unexpected error: %v", e)
	}

	// Verify field access type was recorded
	for expr, typeInfo := range analyzer.ExprTypes() {
		if field, ok := expr.(*ast.FieldAccessExpr); ok && field.Field.Value == "name" {
			if typeInfo.Kind != TypeKindString {
				t.Errorf("expected string type for field 'name', got %s", typeInfo.Kind)
			}
			return
		}
	}
}

func TestPipedFieldAccessTypeResolution(t *testing.T) {
	input := `type User
    name string

func main()
    u := User{name: "Alice"}
    n := u |> .name
    print(n)
`

	analyzer, errors := analyzeSource(t, input)
	_ = analyzer

	for _, e := range errors {
		t.Errorf("unexpected error: %v", e)
	}

	for expr, typeInfo := range analyzer.ExprTypes() {
		if field, ok := expr.(*ast.FieldAccessExpr); ok && field.Object == nil && field.Field.Value == "name" {
			if typeInfo.Kind != TypeKindString {
				t.Errorf("expected string type for piped field 'name', got %s", typeInfo.Kind)
			}
			return
		}
	}
}

func TestMethodWithMultipleReturns(t *testing.T) {
	input := `type Parser
    input string

func Parse on p Parser() (string, error)
    return p.input, empty

func main()
    p := Parser{input: "test"}
    result, err := p.Parse()
    print(result)
    print(err)
`

	analyzer, errors := analyzeSource(t, input)
	_ = analyzer

	for _, e := range errors {
		t.Errorf("unexpected error: %v", e)
	}

	// Verify 2-return method was correctly resolved
	for expr, count := range analyzer.ReturnCounts() {
		if mc, ok := expr.(*ast.MethodCallExpr); ok {
			if mc.Method.Value == "Parse" {
				if count != 2 {
					t.Errorf("expected return count 2 for Parse, got %d", count)
				}
				return
			}
		}
	}
}

func TestPointerReceiverMethodResolution(t *testing.T) {
	input := `type Counter
    value int

func Inc on c reference Counter
    c.value = c.value + 1

func GetValue on c Counter int
    return c.value

func main()
    c := Counter{value: 0}
    c.Inc()
    v := c.GetValue()
    print(v)
`

	analyzer, errors := analyzeSource(t, input)
	_ = analyzer

	for _, e := range errors {
		t.Errorf("unexpected error: %v", e)
	}
}

func TestReturnConcreteForUserInterface(t *testing.T) {
	// Returning a concrete reference type where a user-defined interface is
	// expected should not produce a type error — the Go compiler handles
	// interface satisfaction checking.
	input := `interface Greeter
    Greet() string

type Person
    Name string

func Greet on p Person string
    return "Hello"

func NewGreeter() Greeter
    p := Person{Name: "Alice"}
    return reference of p
`

	_, errors := analyzeSource(t, input)
	for _, e := range errors {
		t.Errorf("unexpected error: %v", e)
	}
}

func TestReturnConcreteForGoStdlibInterface(t *testing.T) {
	// Returning a concrete type where a Go stdlib interface (e.g., io.Reader)
	// is expected should not produce a type error.
	input := `import "io"

type MyReader
    data string

func Read on r MyReader (p list of byte) (int, error)
    return 0, empty

func MakeReader() io.Reader
    r := MyReader{data: "hello"}
    return reference of r
`

	_, errors := analyzeSource(t, input)
	for _, e := range errors {
		t.Errorf("unexpected error: %v", e)
	}
}

func TestReturnConcreteForExternalQualifiedType(t *testing.T) {
	// Returning a concrete type where a qualified external type is expected
	// should not produce a type error — we can't resolve external types at
	// Kukicha compile time, so we defer to the Go compiler.
	input := `import "net/http"

type MyHandler
    prefix string

func ServeHTTP on h MyHandler (w http.ResponseWriter, r reference http.Request)
    return

func NewHandler() http.Handler
    return reference of MyHandler{prefix: "/api"}
`

	_, errors := analyzeSource(t, input)
	for _, e := range errors {
		t.Errorf("unexpected error: %v", e)
	}
}
