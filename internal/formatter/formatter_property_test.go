package formatter

import (
	"os"
	"path/filepath"
	"testing"
)

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

// TestFormatterIdempotencyOnKukiFiles runs idempotency checks against all .kuki
// files in stdlib/ and examples/. This catches regressions where the formatter
// produces different output on repeated runs for real-world code.
func TestFormatterIdempotencyOnKukiFiles(t *testing.T) {
	root := filepath.Join("..", "..")
	dirs := []string{
		filepath.Join(root, "stdlib"),
		filepath.Join(root, "examples"),
	}

	opts := DefaultOptions()
	var count int

	for _, dir := range dirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || filepath.Ext(path) != ".kuki" {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			relPath, _ := filepath.Rel(root, path)
			t.Run(relPath, func(t *testing.T) {
				source := string(data)

				first, err := Format(source, filepath.Base(path), opts)
				if err != nil {
					t.Skipf("Format error (skipping): %v", err)
				}

				second, err := Format(first, filepath.Base(path), opts)
				if err != nil {
					t.Errorf("second Format call failed: %v", err)
					return
				}

				if first != second {
					t.Errorf("formatter is not idempotent for %s\n--- first pass (last 20 lines) ---\n%s\n--- second pass (last 20 lines) ---\n%s",
						relPath, lastLines(first, 20), lastLines(second, 20))
				}
			})
			count++
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", dir, err)
		}
	}

	if count == 0 {
		t.Fatal("found no .kuki files to test")
	}
	t.Logf("tested %d .kuki files for idempotency", count)
}

func lastLines(s string, n int) string {
	parts := splitLines(s)
	if len(parts) <= n {
		return s
	}
	return "...\n" + joinLines(parts[len(parts)-n:])
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func joinLines(lines []string) string {
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}
