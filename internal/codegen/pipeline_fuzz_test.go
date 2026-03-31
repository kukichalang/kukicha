package codegen

import (
	goparser "go/parser"
	"go/token"
	"testing"

	kukiparser "github.com/kukichalang/kukicha/internal/parser"
	"github.com/kukichalang/kukicha/internal/semantic"
)

// FuzzPipeline feeds random input through the full lex → parse → semantic → codegen
// pipeline and ensures:
//   - No stage panics, regardless of input.
//   - When codegen succeeds, the output is valid Go (verified with go/parser).
//
// Run with: go test -fuzz=FuzzPipeline ./internal/codegen/
// Target: 0 panics after 10M iterations.
func FuzzPipeline(f *testing.F) {
	// Seed corpus covering Kukicha constructs that exercise all codegen paths.
	seeds := []string{
		// Basic function
		"func Add(a int, b int) int\n    return a + b\n",
		// Types and methods
		"type Person\n    Name string\n    Age int\n\nfunc Greet on p Person() string\n    return p.Name\n",
		// Pipes
		`func Double(x int) int
    return x * 2

func Run() int
    result := 5 |> Double()
    return result
`,
		// onerr
		`func Safe(x int) (int, error)
    return x, empty

func UseOnerr() int
    v, err := Safe(1) onerr return 0
    _ = err
    return v
`,
		// String interpolation
		`func Greet(name string) string
    return "Hello {name}!"
`,
		// Enums
		`enum Color
    Red = 0
    Green = 1
    Blue = 2
`,
		// Lambdas
		`import "stdlib/slice"

func Filter(items list of int) list of int
    return slice.Filter(items, (x int) => x > 0)
`,
		// Switch
		`func Classify(x int) string
    switch x
        when 0
            return "zero"
        when 1
            return "one"
        otherwise
            return "other"
`,
		// For loops
		`func Sum(items list of int) int
    total := 0
    for v in items
        total = total + v
    return total
`,
		// Goroutines
		`func Fire()
    go
        doWork()
`,
		// Interfaces
		`interface Stringer
    String() string
`,
		// Error expressions
		`func MayFail(x int) (int, error)
    if x < 0
        return 0, error "negative value"
    return x, empty
`,
		// Reference types
		`type Node
    Value int
    Next reference Node

func NewNode(v int) reference Node
    n := Node{Value: v, Next: empty}
    return reference of n
`,
		// Variadic
		`func Concat(parts many string) string
    result := ""
    for _, p := range parts
        result = result + p
    return result
`,
		// Channels
		`func Producer() channel int
    ch := make(chan int, 1)
    return ch
`,
		// Defer
		`func WithDefer()
    defer print("done")
    print("working")
`,
		// Map literals
		`func MakeMap() map of string to int
    return map[string]int{"a": 1, "b": 2}
`,
		// Petiole
		"petiole mylib\n\nfunc Helper() string\n    return \"help\"\n",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data string) {
		// Stage 1: Parse. Invalid syntax is fine — just skip.
		p, err := kukiparser.New(data, "fuzz.kuki")
		if err != nil {
			return
		}
		program, parseErrors := p.Parse()
		if len(parseErrors) > 0 {
			return
		}

		// Stage 2: Semantic analysis. Type errors are fine — just skip.
		analyzer := semantic.New(program)
		semanticErrors := analyzer.Analyze()
		if len(semanticErrors) > 0 {
			return
		}

		// Stage 3: Codegen. Errors are acceptable — just skip.
		gen := New(program)
		gen.SetExprReturnCounts(analyzer.ReturnCounts())
		gen.SetExprTypes(analyzer.ExprTypes())
		output, err := gen.Generate()
		if err != nil {
			return
		}

		// Stage 4: Verify the generated output is valid Go source.
		// If codegen succeeds but produces invalid Go, that is a bug.
		fset := token.NewFileSet()
		if _, parseErr := goparser.ParseFile(fset, "generated.go", output, 0); parseErr != nil {
			t.Errorf("codegen produced invalid Go source\nparse error: %v\noutput:\n%s", parseErr, output)
		}
	})
}
