package codegen

import (
	"github.com/duber000/kukicha/internal/parser"
	"strings"
	"testing"
	"testing/synctest"
)

// TestConcurrentCodeGeneration tests that multiple code generators can run
// concurrently without data races or interference using synctest
func TestConcurrentCodeGeneration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Test that multiple code generators can run concurrently
		// without data races or interference

		programs := []string{
			`func main()
    x := 1`,
			`func add(a int, b int) int
    return a + b`,
			`type User
    name string`,
		}

		results := make(chan string, len(programs))

		for _, src := range programs {
			go func(source string) {
				p, err := parser.New(source, "test.kuki")
				if err != nil {
					t.Errorf("parser error: %v", err)
					results <- ""
					return
				}
				program, parseErrors := p.Parse()
				if len(parseErrors) > 0 {
					t.Errorf("parse errors: %v", parseErrors)
					results <- ""
					return
				}
				gen := New(program)
				code, err := gen.Generate()
				if err != nil {
					t.Errorf("codegen error: %v", err)
					results <- ""
					return
				}
				results <- code
			}(src)
		}

		synctest.Wait()

		// Verify all completed
		for range programs {
			select {
			case result := <-results:
				if result == "" {
					t.Error("Expected non-empty result")
				}
			default:
				t.Error("Expected result not received")
			}
		}
	})
}

func TestGroupByGenerics(t *testing.T) {
	input := `petiole slice

func GroupBy(items list of any, keyFunc func(any) any2) map of any2 to list of any
    result := make(map of any2 to list of any)
    for item in items
        key := keyFunc(item)
        result[key] = append(result[key], item)
    return result
`

	p, err := parser.New(input, "stdlib/slice/slice.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	gen := New(program)
	gen.SetSourceFile("stdlib/slice/slice.kuki")
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}

	// Verify generic type parameters are generated
	if !strings.Contains(output, "func GroupBy[T any, K comparable]") {
		t.Errorf("expected generic function signature with [T any, K comparable], got: %s", output)
	}

	// Verify the parameter signature
	if !strings.Contains(output, "(items []T, keyFunc func(T) K)") {
		t.Errorf("expected correct parameter types, got: %s", output)
	}

	// Verify return type
	if !strings.Contains(output, "map[K][]T") {
		t.Errorf("expected return type map[K][]T, got: %s", output)
	}
}

func TestGroupByFunction(t *testing.T) {
	input := `func GroupBy(items list of any, keyFunc func(any) any2) map of any2 to list of any
    result := make(map of any2 to list of any)
    for item in items
        key := keyFunc(item)
        result[key] = append(result[key], item)
    return result
`

	p, err := parser.New(input, "stdlib/slice/slice.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	gen := New(program)
	gen.SetSourceFile("stdlib/slice/slice.kuki")
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}

	// Verify function creates the result map properly
	if !strings.Contains(output, "result := make(map[K][]T)") {
		t.Errorf("expected make(map[K][]T), got: %s", output)
	}

	// Verify append is called correctly
	if !strings.Contains(output, "result[key] = append(result[key], item)") {
		t.Errorf("expected append to result[key], got: %s", output)
	}
}

func TestFetchJsonGenerics(t *testing.T) {
	input := `petiole fetch

func Json(resp reference http.Response, sample any) (any, error)
    data := sample
    return data, empty
`

	p, err := parser.New(input, "stdlib/fetch/fetch.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	gen := New(program)
	gen.SetSourceFile("stdlib/fetch/fetch.kuki")
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}

	if !strings.Contains(output, "func Json[T any](resp *http.Response, sample T) (T, error)") {
		t.Errorf("expected fetch.Json generic signature, got: %s", output)
	}
}

func TestJSONDecodeReadGenerics(t *testing.T) {
	input := `petiole json

func DecodeRead(reader io.Reader, sample any) (any, error)
    data := sample
    return data, empty
`

	p, err := parser.New(input, "stdlib/json/json.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	gen := New(program)
	gen.SetSourceFile("stdlib/json/json.kuki")
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}

	if !strings.Contains(output, "func DecodeRead[T any](reader io.Reader, sample T) (T, error)") {
		t.Errorf("expected json.DecodeRead generic signature, got: %s", output)
	}
}

func TestStdlibImportRewriting(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		expectedImport string
		shouldContain  string
	}{
		{
			name: "stdlib/json import",
			source: `import "stdlib/json"

type Config
    Name string

func main()
    cfg := Config{}
    cfg.Name = "test"
    data, _ := json.Marshal(cfg)
`,
			expectedImport: `"github.com/duber000/kukicha/stdlib/json"`,
			shouldContain:  "json.Marshal",
		},
		{
			name: "stdlib/fetch import",
			source: `import "stdlib/fetch"

func main()
    req := fetch.New("https://example.com")
`,
			expectedImport: `"github.com/duber000/kukicha/stdlib/fetch"`,
			shouldContain:  "fetch.New",
		},
		{
			name: "stdlib/json with alias",
			source: `import "stdlib/json" as j

type Data
    Value string

func main()
    d := Data{}
    j.Marshal(d)
`,
			expectedImport: `j "github.com/duber000/kukicha/stdlib/json"`,
			shouldContain:  "j.Marshal",
		},
		{
			name: "multiple imports with stdlib",
			source: `import "fmt"
import "stdlib/json"

type User
    Name string

func main()
    u := User{}
    data, _ := json.Marshal(u)
    fmt.Println(data)
`,
			expectedImport: `"github.com/duber000/kukicha/stdlib/json"`,
			shouldContain:  "json.Marshal",
		},
		{
			name: "non-stdlib import unchanged",
			source: `import "encoding/json"

type Data
    Value string

func main()
    d := Data{}
    json.Marshal(d)
`,
			expectedImport: `"encoding/json"`,
			shouldContain:  "json.Marshal",
		},
		{
			name: "version suffix import gets alias",
			source: `import "encoding/json/v2"

type Data
    Value string

func main()
    d := Data{}
    json.Marshal(d)
`,
			expectedImport: `json "encoding/json/v2"`,
			shouldContain:  "json.Marshal",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, err := parser.New(tt.source, "test.kuki")
			if err != nil {
				t.Fatalf("parser error: %v", err)
			}

			program, parseErrors := p.Parse()
			if len(parseErrors) > 0 {
				t.Fatalf("parse errors: %v", parseErrors)
			}

			gen := New(program)
			output, err := gen.Generate()
			if err != nil {
				t.Fatalf("codegen error: %v", err)
			}

			// Verify the import was rewritten correctly
			if !strings.Contains(output, tt.expectedImport) {
				t.Errorf("expected import %s in output, got: %s", tt.expectedImport, output)
			}

			// Verify the code using the import is present
			if !strings.Contains(output, tt.shouldContain) {
				t.Errorf("expected code %s in output, got: %s", tt.shouldContain, output)
			}
		})
	}
}

func TestExternalInterfaceTypeInFunctionSignature(t *testing.T) {
	input := `import "net/http"

func Wrap(handler http.Handler) http.Handler
    return handler
`

	output := generateSource(t, input)

	if !strings.Contains(output, "func Wrap(handler http.Handler) http.Handler") {
		t.Errorf("expected http.Handler in function signature, got: %s", output)
	}

	if !strings.Contains(output, "return handler") {
		t.Errorf("expected return handler, got: %s", output)
	}
}
