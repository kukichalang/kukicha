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
