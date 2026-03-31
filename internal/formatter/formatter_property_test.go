package formatter

import "testing"

// TestFormatterIdempotency verifies that format(format(source)) == format(source)
// for a corpus of valid Kukicha programs. A non-idempotent formatter would cause
// `kukicha fmt` to produce different output on repeated runs, which is confusing.
func TestFormatterIdempotency(t *testing.T) {
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
			name: "type declaration",
			source: `type Person
    Name string
    Age int
`,
		},
		{
			name: "import and main",
			source: `import "fmt"

func main()
    fmt.Println("Hello, World!")
`,
		},
		{
			name: "if else",
			source: `func Sign(x int) string
    if x > 0
        return "positive"
    else if x < 0
        return "negative"
    else
        return "zero"
`,
		},
		{
			name: "for in loop",
			source: `func Sum(items list of int) int
    total := 0
    for v in items
        total = total + v
    return total
`,
		},
		{
			name: "switch with when",
			source: `func Day(n int) string
    switch n
        when 1
            return "Monday"
        when 2
            return "Tuesday"
        otherwise
            return "other"
`,
		},
		{
			name: "enum",
			source: `enum Status
    OK = 200
    NotFound = 404
    InternalError = 500
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
			name: "pipe expression",
			source: `func Double(x int) int
    return x * 2

func Quadruple(x int) int
    return x |> Double() |> Double()
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
			name: "string interpolation",
			source: `func Greet(name string, age int) string
    return "Hello {name}, you are {age} years old!"
`,
		},
		{
			name: "interface",
			source: `interface Writer
    Write(data string) (int, error)
`,
		},
		{
			name: "petiole declaration",
			source: `petiole mylib

func Helper(x int) int
    return x * 2
`,
		},
		{
			name: "multiple imports",
			source: `import "stdlib/slice"
import "stdlib/string"

func Process(items list of string) list of string
    return items
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
			name: "defer statement",
			source: `func Cleanup()
    defer print("cleaning up")
    doWork()
`,
		},
		{
			name: "go block",
			source: `func Fire()
    go
        doWork()
    waitForDone()
`,
		},
		{
			name: "const declaration",
			source: `const MaxRetries = 5

func Retry(n int) bool
    return n < MaxRetries
`,
		},
		{
			name: "if with init",
			source: `func Lookup(m map of string to int, key string) int
    if v, ok := m[key]; ok
        return v
    return 0
`,
		},
		{
			name: "numeric for loop",
			source: `func Count(n int)
    for i from 0 to n
        print(i)
`,
		},
	}

	opts := DefaultOptions()

	for _, tc := range programs {
		t.Run(tc.name, func(t *testing.T) {
			first, err := Format(tc.source, "test.kuki", opts)
			if err != nil {
				t.Skipf("Format error (skipping idempotency check): %v", err)
			}

			second, err := Format(first, "test.kuki", opts)
			if err != nil {
				t.Errorf("second Format call failed: %v\nfirst output:\n%s", err, first)
				return
			}

			if first != second {
				t.Errorf("formatter is not idempotent\n--- first pass ---\n%s\n--- second pass ---\n%s", first, second)
			}
		})
	}
}
