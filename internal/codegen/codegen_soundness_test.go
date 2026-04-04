package codegen

import (
	goparser "go/parser"
	"go/token"
	"testing"
)

// TestCodegenStructuralSoundness verifies that for a broad set of valid Kukicha
// programs, the codegen output can be parsed as valid Go by go/parser.ParseFile.
// A failure here means codegen emitted syntactically broken Go, which is a compiler bug.
func TestCodegenStructuralSoundness(t *testing.T) {
	programs := []struct {
		name   string
		source string
	}{
		{
			name: "simple function",
			source: `func Add(a int, b int) int
    return a + b
`,
		},
		{
			name: "void function",
			source: `func Print(msg string)
    print(msg)
`,
		},
		{
			name: "type declaration",
			source: `type Point
    X int
    Y int
`,
		},
		{
			name: "method on type",
			source: `type Counter
    Value int

func Increment on c reference Counter()
    c.Value = c.Value + 1
`,
		},
		{
			name: "enum declaration",
			source: `enum Direction
    North = 0
    South = 1
    East = 2
    West = 3
`,
		},
		{
			name: "if else chain",
			source: `func Abs(x int) int
    if x < 0
        return -x
    return x
`,
		},
		{
			name: "for in loop",
			source: `func Max(items list of int) int
    m := 0
    for v in items
        if v > m
            m = v
    return m
`,
		},
		{
			name: "switch with when",
			source: `func Classify(n int) string
    switch n
        when 0
            return "zero"
        when 1
            return "one"
        otherwise
            return "many"
`,
		},
		{
			name: "string interpolation",
			source: `func Greet(name string) string
    return "Hello, {name}!"
`,
		},
		{
			name: "multi-return function",
			source: `func Divide(a int, b int) (int, error)
    if b equals 0
        return 0, error "division by zero"
    return a / b, empty
`,
		},
		{
			name: "pipe expression",
			source: `func Double(x int) int
    return x * 2

func Quad(x int) int
    return x |> Double() |> Double()
`,
		},
		{
			name: "onerr clause",
			source: `func MayFail(x int) (int, error)
    return x, empty

func Safe() int
    v := MayFail(5) onerr return 0
    return v
`,
		},
		{
			name: "interface declaration",
			source: `interface Stringer
    String() string

func UseStringer(s Stringer) string
    return s.String()
`,
		},
		{
			name: "const declaration",
			source: `const Pi = 3
const MaxSize = 100
`,
		},
		{
			name: "arrow lambda",
			source: `import "stdlib/slice"

func Doubles(nums list of int) list of int
    return slice.Map(nums, (x int) => x * 2)
`,
		},
		{
			name: "map operations",
			source: `func WordCount(words list of string) map of string to int
    counts := map[string]int{}
    for w in words
        counts[w] = counts[w] + 1
    return counts
`,
		},
		{
			name: "defer call",
			source: `func WithCleanup()
    defer print("done")
    print("working")
`,
		},
		{
			name: "nested functions and closures",
			source: `func Adder(x int) func(int) int
    return func(y int) int
        return x + y
`,
		},
		{
			name: "type alias",
			source: `type Handler func(string) string
`,
		},
		{
			name: "petiole package",
			source: `petiole utils

func Helper(s string) string
    return s
`,
		},
		{
			name: "struct literal",
			source: `type Point
    X int
    Y int

func Origin() Point
    return Point{X: 0, Y: 0}
`,
		},
		{
			name: "reference type",
			source: `type Node
    Value int

func NewNode(v int) reference Node
    n := Node{Value: v}
    return reference of n
`,
		},
		{
			name: "channel of type",
			source: `func ForwardChan(ch channel of int) channel of int
    return ch
`,
		},
		{
			name: "variadic function",
			source: `func Sum(many numbers int) int
    return 0
`,
		},
		{
			name: "goroutine block",
			source: `func doWork()
    print("working")

func Spawn()
    go
        doWork()
`,
		},
		{
			name: "piped switch",
			source: `func Describe(x int) string
    return x |> switch
        when 0
            "zero"
        otherwise
            "nonzero"
`,
		},
		{
			name: "if with init statement",
			source: `func LookupOrDefault(m map of string to int, key string) int
    if v, ok := m[key]; ok
        return v
    return 0
`,
		},
		{
			name: "numeric for loop",
			source: `func PrintN(n int)
    for i from 0 to n
        print(i)
`,
		},
	}

	for _, tc := range programs {
		t.Run(tc.name, func(t *testing.T) {
			output := generateAnalyzedSource(t, tc.source)

			// Verify output is valid Go
			fset := token.NewFileSet()
			if _, parseErr := goparser.ParseFile(fset, "generated.go", output, 0); parseErr != nil {
				t.Errorf("generated Go is not valid\nparse error: %v\noutput:\n%s", parseErr, output)
			}
		})
	}
}
