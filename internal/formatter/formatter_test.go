package formatter

import (
	"testing"
)

func assertFormatted(t *testing.T, source string, expected string) {
	t.Helper()

	opts := DefaultOptions()
	result, err := Format(source, "test.kuki", opts)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	if result != expected {
		t.Fatalf("unexpected formatted output:\n--- got ---\n%s--- want ---\n%s", result, expected)
	}

	roundTrip, err := Format(result, "test.kuki", opts)
	if err != nil {
		t.Fatalf("Round-trip format error: %v", err)
	}
	if roundTrip != expected {
		t.Fatalf("unexpected round-trip output:\n--- got ---\n%s--- want ---\n%s", roundTrip, expected)
	}
}

func TestFormatSimple(t *testing.T) {
	source := `import "fmt"

func main()
    fmt.Println("Hello")
`

	opts := DefaultOptions()
	result, err := Format(source, "test.kuki", opts)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	t.Logf("Result:\n%s", result)
}

func TestFormatArrowLambdaBlockAssignment(t *testing.T) {
	source := `func main()
    f := (r Repo) =>
        name := r.Name
        return name
`

	expected := `func main()
    f := (r Repo) =>
        name := r.Name
        return name
`

	assertFormatted(t, source, expected)
}

func TestFormatArrowLambdaBlockInMethodCall(t *testing.T) {
	source := `func main()
    result := repos |> slice.Filter((r Repo) =>
        name := r.Name
        return name
    )
`

	expected := `func main()
    result := repos |> slice.Filter((r Repo) =>
        name := r.Name
        return name
    )
`

	assertFormatted(t, source, expected)
}

func TestFormatWithComments(t *testing.T) {
	source := `# This is a comment
import "fmt"

# Main function
func main()
    # Print hello
    fmt.Println("Hello")
`

	opts := DefaultOptions()
	result, err := Format(source, "test.kuki", opts)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	t.Logf("Result:\n%s", result)
}

func TestFormatGoStyle(t *testing.T) {
	source := `import "fmt"

func main() {
    fmt.Println("Hello")
}
`

	opts := DefaultOptions()
	result, err := Format(source, "test.kuki", opts)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	t.Logf("Result:\n%s", result)
}

func TestFormatMakeExprPreservesParens(t *testing.T) {
	source := `func main()
    x := make(list of string, 0)
    y := make(map of string to int)
    z := make(channel of int, 10)
`

	expected := `func main()
    x := make(list of string, 0)
    y := make(map of string to int)
    z := make(channel of int, 10)
`

	assertFormatted(t, source, expected)
}

func TestFormatOnErrReturnWithValues(t *testing.T) {
	source := `func Validate(s string) (string, error)
    matched := regexp.MatchString(pattern, s) onerr return s, error "bad: {error}"
    return s, empty
`

	expected := `func Validate(s string) (string, error)
    matched := regexp.MatchString(pattern, s) onerr return s, error "bad: {error}"
    return s, empty
`

	assertFormatted(t, source, expected)
}

func TestFormatOnErrShorthandReturn(t *testing.T) {
	source := `func Read(path string) (string, error)
    data := files.ReadAll(path) onerr return
    return data, empty
`

	expected := `func Read(path string) (string, error)
    data := files.ReadAll(path) onerr return
    return data, empty
`

	assertFormatted(t, source, expected)
}

func TestFormatInlineOnErrReturnExpression(t *testing.T) {
	source := `func Read(path string) string
    return files.ReadAll(path) onerr ""
`

	expected := `func Read(path string) string
    return files.ReadAll(path) onerr ""
`

	assertFormatted(t, source, expected)
}

func TestFormatOnErrShorthandContinue(t *testing.T) {
	source := `func Process(items list of string) list of string
    results := make(list of string, 0)
    for item in items
        parsed := parse(item) onerr continue
        results = append(results, parsed)
    return results
`

	expected := `func Process(items list of string) list of string
    results := make(list of string, 0)
    for item in items
        parsed := parse(item) onerr continue
        results = append(results, parsed)
    return results
`

	assertFormatted(t, source, expected)
}

func TestFormatOnErrShorthandBreak(t *testing.T) {
	source := `func Process(items list of string) string
    for item in items
        parsed := parse(item) onerr break
        return parsed
    return ""
`

	expected := `func Process(items list of string) string
    for item in items
        parsed := parse(item) onerr break
        return parsed
    return ""
`

	assertFormatted(t, source, expected)
}

func TestFormatPreservesBlankLineBeforeComment(t *testing.T) {
	source := `petiole mypkg

# Section header

# Doc comment for Foo
func Foo() string
    return "foo"

# Another section

# Doc comment for Bar
func Bar() string
    return "bar"
`

	expected := `petiole mypkg

# Section header

# Doc comment for Foo
func Foo() string
    return "foo"

# Another section

# Doc comment for Bar
func Bar() string
    return "bar"
`

	assertFormatted(t, source, expected)
}

func TestFormatStringEscapeSequences(t *testing.T) {
	// In Kukicha source, \\ is an escaped backslash, \t is a tab, \n is a newline
	// The formatter must re-escape these when emitting
	source := "func main()\n" +
		"    pattern := \"hello\\\\world\"\n" +
		"    tab := \"hello\\tworld\"\n" +
		"    newline := \"line1\\nline2\"\n" +
		"    quote := \"say \\\"hi\\\"\"\n"

	expected := "func main()\n" +
		"    pattern := \"hello\\\\world\"\n" +
		"    tab := \"hello\\tworld\"\n" +
		"    newline := \"line1\\nline2\"\n" +
		"    quote := \"say \\\"hi\\\"\"\n"

	assertFormatted(t, source, expected)
}

func TestFormatLongPipeChainWraps(t *testing.T) {
	// A long pipe chain should be broken across lines with each |> on its own line.
	source := `import "stdlib/cli"

func main()
    _ = cli.New("gke-deploy") |> cli.Description("Deploy Django to GKE with OpenTofu") |> cli.GlobalFlag("env", "Target: dev, test, prod, or all", "all") |> cli.Command("deploy", "Create GKE cluster") |> cli.RunApp()
`
	expected := `import "stdlib/cli"

func main()
    _ = cli.New("gke-deploy")
        |> cli.Description("Deploy Django to GKE with OpenTofu")
        |> cli.GlobalFlag("env", "Target: dev, test, prod, or all", "all")
        |> cli.Command("deploy", "Create GKE cluster")
        |> cli.RunApp()
`
	assertFormatted(t, source, expected)
}

func TestFormatShortPipeChainStaysOneLine(t *testing.T) {
	// A short pipe chain should remain on a single line.
	source := `func Double(x int) int
    return x |> Double() |> Double()
`
	assertFormatted(t, source, source)
}

func TestFormatMultilinePipePreserved(t *testing.T) {
	// A pipe chain that is already multiline should be normalized
	// to the same wrapped format and round-trip cleanly.
	source := `import "stdlib/cli"

func main()
    _ = cli.New("gke-deploy")
        |> cli.Description("Deploy Django to GKE with OpenTofu")
        |> cli.GlobalFlag("env", "Target: dev, test, prod, or all", "all")
        |> cli.Command("deploy", "Create GKE cluster")
        |> cli.RunApp()
`
	assertFormatted(t, source, source)
}

func TestFormatPipeChainWithOnerr(t *testing.T) {
	// Pipe chain with onerr suffix should wrap correctly.
	source := `import "stdlib/cli"

func main()
    _ = cli.New("gke-deploy") |> cli.Description("Deploy Django to GKE with OpenTofu") |> cli.GlobalFlag("env", "Target: dev, test, prod, or all", "all") |> cli.RunApp() onerr cli.Fatal("{error}")
`
	expected := `import "stdlib/cli"

func main()
    _ = cli.New("gke-deploy")
        |> cli.Description("Deploy Django to GKE with OpenTofu")
        |> cli.GlobalFlag("env", "Target: dev, test, prod, or all", "all")
        |> cli.RunApp() onerr cli.Fatal("{error}")
`
	assertFormatted(t, source, expected)
}

func TestFormatShortMultilinePipePreserved(t *testing.T) {
	// A short pipe chain the user wrote across multiple lines in source
	// should be kept multiline even though it would fit on a single line.
	// This protects deliberately-laid-out demo/docstring code.
	source := `func Double(x int) int
    return x
        |> Double()
        |> Double()
`
	assertFormatted(t, source, source)
}

func TestFormatShortMultilineStructLiteralPreserved(t *testing.T) {
	// A struct literal the user wrote across multiple lines in source
	// should stay multiline even if it would fit on a single line.
	source := `type Point
    X int
    Y int

func main()
    p := Point{
        X: 1,
        Y: 2,
    }
    _ = p
`
	assertFormatted(t, source, source)
}

func TestFormatShortSingleLineStructLiteralStays(t *testing.T) {
	// A short struct literal written on one line should remain collapsed.
	source := `type Point
    X int
    Y int

func main()
    p := Point{X: 1, Y: 2}
    _ = p
`
	assertFormatted(t, source, source)
}

func TestFormatShortMultilineListLiteralPreserved(t *testing.T) {
	// A list literal the user wrote across multiple lines should stay
	// multiline even if it would fit on a single line.
	source := `func main()
    xs := list of int{
        1,
        2,
        3,
    }
    _ = xs
`
	assertFormatted(t, source, source)
}

func TestFormatShortSingleLineListLiteralStays(t *testing.T) {
	// A short list literal on one line should stay on one line.
	source := `func main()
    xs := list of int{1, 2, 3}
    _ = xs
`
	assertFormatted(t, source, source)
}

func TestFormatShortMultilineMapLiteralPreserved(t *testing.T) {
	// A map literal the user wrote across multiple lines should stay
	// multiline even if it would fit on a single line.
	source := `func main()
    m := map of string to int{
        "a": 1,
        "b": 2,
    }
    _ = m
`
	assertFormatted(t, source, source)
}

func TestFormatShortSingleLineMapLiteralStays(t *testing.T) {
	// A short map literal on one line should stay on one line.
	source := `func main()
    m := map of string to int{"a": 1, "b": 2}
    _ = m
`
	assertFormatted(t, source, source)
}

func TestFormatImportGroupingStdlibAndThirdParty(t *testing.T) {
	// Imports from Go stdlib, Kukicha stdlib, and third-party should be
	// grouped with a blank line between groups, sorted alphabetically within
	// each group. Go stdlib and Kukicha stdlib share a single group.
	source := `import "github.com/foo/bar"
import "fmt"
import "stdlib/slice"
import "os"
import "stdlib/json"

func main()
    _ = 1
`
	expected := `import "fmt"
import "os"
import "stdlib/json"
import "stdlib/slice"

import "github.com/foo/bar"

func main()
    _ = 1
`
	assertFormatted(t, source, expected)
}

func TestFormatImportGroupingStdlibOnly(t *testing.T) {
	// Single-group imports should be sorted but not separated.
	source := `import "stdlib/slice"
import "fmt"
import "os"

func main()
    _ = 1
`
	expected := `import "fmt"
import "os"
import "stdlib/slice"

func main()
    _ = 1
`
	assertFormatted(t, source, expected)
}

func TestFormatImportGroupingPreservesAliases(t *testing.T) {
	// Aliases must survive regrouping and sorting.
	source := `import "github.com/foo/bar" as baz
import "stdlib/string" as strpkg
import "context"

func main()
    _ = 1
`
	expected := `import "context"
import "stdlib/string" as strpkg

import "github.com/foo/bar" as baz

func main()
    _ = 1
`
	assertFormatted(t, source, expected)
}

func TestFormatImportGroupingIdempotent(t *testing.T) {
	// Already-grouped input must format to itself (round-trip handled by assertFormatted).
	source := `import "fmt"
import "stdlib/slice"

import "github.com/foo/bar"

func main()
    _ = 1
`
	assertFormatted(t, source, source)
}

// TestFormatBinaryPrecedenceParens verifies that the formatter preserves
// parentheses that are semantically necessary to override left-associativity
// when the right-hand sub-expression has equal precedence to its parent.
func TestFormatBinaryPrecedenceParens(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name: "div by product (equal prec on right)",
			source: `func f() float64
    return a / (b * c)
`,
		},
		{
			name: "sub minus sub (equal prec on right, different associativity)",
			source: `func f() float64
    return a - (b - c)
`,
		},
		{
			name: "sub minus add (equal prec on right)",
			source: `func f() float64
    return a - (b + c)
`,
		},
		{
			name: "div by div (equal prec on right)",
			source: `func f() float64
    return a / (b / c)
`,
		},
		{
			name: "left-assoc mul+div left paren not needed",
			source: `func f() float64
    return a * b / c
`,
		},
		{
			name: "lower-prec child on right — parens needed",
			source: `func f() float64
    return a * (b + c)
`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertFormatted(t, tc.source, tc.source)
		})
	}
}

// TestFormatTypeCastOperandParens verifies that parentheses around the
// operand of an `as` cast are preserved whenever the inner expression has
// lower precedence than `as` (binary ops, unary prefixes, `is` checks).
// Without parens, `(h % dim) as int` re-parses as `h % (dim as int)` —
// `as` binds tighter than any binary operator.
func TestFormatTypeCastOperandParens(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name: "binary mod cast to int",
			source: `func f() int
    return (h % dim) as int
`,
		},
		{
			name: "binary add cast",
			source: `func f() uint32
    return (a + b) as uint32
`,
		},
		{
			name: "unary minus cast",
			source: `func f() int
    return (-x) as int
`,
		},
		{
			name: "left-assoc cast chain — no parens",
			source: `func f() int
    return x as uint32 as int
`,
		},
		{
			name: "primary operand — no parens",
			source: `func f() int
    return x as int
`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertFormatted(t, tc.source, tc.source)
		})
	}
}

// TestFormatBacktickRawStringWithBrace guards against a preprocessor bug
// where `{` inside a backtick raw string was misread as a Go-style block
// opener, corrupting indentation for the rest of the file.
func TestFormatLongCallWraps(t *testing.T) {
	source := `func f()
    db.Exec(pool, "INSERT INTO longer_table (col_a, col_b, col_c) VALUES (?, ?, ?)", "v1", "v2", "v3")
`
	expected := `func f()
    db.Exec(
        pool,
        "INSERT INTO longer_table (col_a, col_b, col_c) VALUES (?, ?, ?)",
        "v1",
        "v2",
        "v3",
    )
`
	assertFormatted(t, source, expected)
}

func TestFormatShortCallStaysSingleLine(t *testing.T) {
	source := `func f()
    foo(a, b, c)
`
	assertFormatted(t, source, source)
}

func TestFormatLongMethodCallWraps(t *testing.T) {
	source := `func f()
    obj.MethodWithLongerName("argument_one_is_long", "argument_two_is_long", "argument_three_is_long")
`
	expected := `func f()
    obj.MethodWithLongerName(
        "argument_one_is_long",
        "argument_two_is_long",
        "argument_three_is_long",
    )
`
	assertFormatted(t, source, expected)
}

func TestFormatBacktickRawStringWithBrace(t *testing.T) {
	source := "func f() string\n" +
		"    q := `query($x: String) {\n" +
		"    user(login: $x) {\n" +
		"        name\n" +
		"    }\n" +
		"}\n" +
		"`\n" +
		"    return q\n"
	assertFormatted(t, source, source)
}
