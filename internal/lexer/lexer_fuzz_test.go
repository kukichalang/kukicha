package lexer

import "testing"

// FuzzLexer feeds random input to the lexer and ensures it never panics.
// The lexer must always return errors rather than crashing, regardless of input.
//
// Run with: go test -fuzz=FuzzLexer ./internal/lexer/
// Target: 0 panics after 10M iterations.
func FuzzLexer(f *testing.F) {
	// Seed corpus — representative Kukicha constructs.
	seeds := []string{
		"func Add(a int, b int) int\n    return a + b\n",
		"type Person\n    Name string\n    Age int\n",
		`func Greet(name string) string
    return "Hello {name}!"
`,
		"import \"stdlib/slice\"\n",
		`petiole myapp

func main()
    x := 42
    if x > 0
        print("positive")
    else
        print("non-positive")
`,
		`func Divide(a float, b float) (float, error)
    if b equals 0.0
        return 0.0, error "division by zero"
    return a / b, empty
`,
		`func Pipeline(items list of string) list of string
    return items
`,
		`enum Status
    OK = 200
    NotFound = 404
`,
		`func Method on s MyStruct() string
    return s.Name
`,
		`func WithOnerr(x int) int
    result := riskyOp(x) onerr return 0
    return result
`,
		"# kuki:deprecated \"Use NewFunc instead\"\nfunc OldFunc()\n",
		`func Lambda() list of int
    items := [1, 2, 3]
    return items
`,
		"for i := 0; i < 10; i++\n    print(i)\n",
		"switch x\n    case 1\n        return \"one\"\n    otherwise\n        return \"other\"\n",
		`go
    doWork()
`,
		// Edge cases and tricky inputs
		"",
		"\n",
		"   \n",
		"func\n",
		"type\n",
		"{{{",
		`"unclosed string`,
		"# kuki:",
		"# kuki:security",
		"list of map of string to int",
		"reference reference int",
		`'single quoted
multiline'`,
		"func f()\n        deep := nested\n",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data string) {
		// The lexer must never panic. Errors are acceptable.
		l := NewLexer(data, "fuzz.kuki")
		//nolint:errcheck
		l.ScanTokens()
	})
}
