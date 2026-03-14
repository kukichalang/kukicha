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
