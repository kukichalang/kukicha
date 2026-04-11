package semantic

import (
	"strings"
	"testing"

	"github.com/kukichalang/kukicha/internal/ast"
)

func TestUntypedLiteralVarDeclResolution(t *testing.T) {
	// var declarations with type annotation are top-level only in Kukicha;
	// inside functions, use return type context instead.
	input := `type Config
    host string
    port int

func makeConfig() Config
    return {host: "localhost", port: 8080}
`
	_, errs := analyzeSource(t, input)
	for _, e := range errs {
		t.Errorf("unexpected error: %v", e)
	}
}

func TestUntypedLiteralReturnResolution(t *testing.T) {
	input := `type Point
    x int
    y int

func origin() Point
    return {x: 0, y: 0}
`
	_, errs := analyzeSource(t, input)
	for _, e := range errs {
		t.Errorf("unexpected error: %v", e)
	}
}

func TestUntypedLiteralReturnMapResolution(t *testing.T) {
	input := `func headers() map of string to string
    return {"Content-Type": "text/html"}
`
	_, errs := analyzeSource(t, input)
	for _, e := range errs {
		t.Errorf("unexpected error: %v", e)
	}
}

func TestUntypedLiteralReturnSliceResolution(t *testing.T) {
	input := `func nums() list of int
    return {1, 2, 3}
`
	_, errs := analyzeSource(t, input)
	for _, e := range errs {
		t.Errorf("unexpected error: %v", e)
	}
}

func TestUntypedLiteralCallArgResolution(t *testing.T) {
	input := `type Point
    x int
    y int

func draw(p Point)
    print(p)

func main()
    draw({x: 1, y: 2})
`
	_, errs := analyzeSource(t, input)
	for _, e := range errs {
		t.Errorf("unexpected error: %v", e)
	}
}

func TestUntypedLiteralAssignResolution(t *testing.T) {
	input := `type Config
    host string
    port int

func main()
    c := Config{host: "", port: 0}
    c = {host: "prod", port: 443}
`
	_, errs := analyzeSource(t, input)
	for _, e := range errs {
		t.Errorf("unexpected error: %v", e)
	}
}

func TestUntypedLiteralInTypedListResolution(t *testing.T) {
	input := `type Point
    x int
    y int

func main()
    points := list of Point{{x: 1, y: 2}, {x: 3, y: 4}}
`
	_, errs := analyzeSource(t, input)
	for _, e := range errs {
		t.Errorf("unexpected error: %v", e)
	}
}

func TestUntypedLiteralResolvedTypeSet(t *testing.T) {
	input := `type Config
    host string
    port int

func makeConfig() Config
    return {host: "localhost", port: 8080}
`
	prog := mustParseProgram(t, input)
	analyzer := NewWithFile(prog, "test.kuki")
	analyzer.Analyze()

	// Find the UntypedCompositeLiteral in the return statement
	fn := prog.Declarations[1].(*ast.FunctionDecl)
	ret := fn.Body.Statements[0].(*ast.ReturnStmt)
	ucl, ok := ret.Values[0].(*ast.UntypedCompositeLiteral)
	if !ok {
		t.Fatalf("expected UntypedCompositeLiteral, got %T", ret.Values[0])
	}
	if ucl.ResolvedType == nil {
		t.Fatal("expected ResolvedType to be set after analysis")
	}
	namedType, ok := ucl.ResolvedType.(*ast.NamedType)
	if !ok {
		t.Fatalf("expected NamedType, got %T", ucl.ResolvedType)
	}
	if namedType.Name != "Config" {
		t.Errorf("expected resolved type 'Config', got '%s'", namedType.Name)
	}
}

func TestUntypedLiteralBadFieldName(t *testing.T) {
	input := `type Config
    host string
    port int

func makeConfig() Config
    return {hosst: "localhost", port: 8080}
`
	_, errs := analyzeSource(t, input)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "unknown field 'hosst'") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error about unknown field 'hosst', got: %v", errs)
	}
}

func TestUntypedLiteralBottomUpInference(t *testing.T) {
	input := `func main()
    x := {"key": "val"}
`
	_, errs := analyzeSource(t, input)
	for _, e := range errs {
		t.Errorf("unexpected error: %v", e)
	}
}
