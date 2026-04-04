package codegen

import (
	"strings"
	"testing"
)

func TestExtractPkgName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"fmt", "fmt"},
		{"encoding/json", "json"},
		{"net/http", "http"},
		{"github.com/kukichalang/kukicha/stdlib/json", "json"},
		{"gopkg.in/yaml.v3", "yaml"},
		{"encoding/json/v2", "json"},
		{"github.com/jackc/pgx/v5", "pgx"},
		{"os", "os"},
		{"path/filepath", "filepath"},
		{"gopkg.in/check.v1", "check"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractPkgName(tt.input)
			if got != tt.expected {
				t.Errorf("extractPkgName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestRewriteStdlibImport(t *testing.T) {
	prog := mustParseProgram(t, "func main()\n    x := 1\n")
	gen := New(prog)

	tests := []struct {
		input    string
		expected string
	}{
		{"stdlib/json", "github.com/kukichalang/kukicha/stdlib/json"},
		{"stdlib/slice", "github.com/kukichalang/kukicha/stdlib/slice"},
		{"stdlib/fetch", "github.com/kukichalang/kukicha/stdlib/fetch"},
		{"stdlib/game", "github.com/kukichalang/game"},
		{"encoding/json", "encoding/json"},
		{"fmt", "fmt"},
		{`"stdlib/json"`, "github.com/kukichalang/kukicha/stdlib/json"},
		{`"stdlib/game"`, "github.com/kukichalang/game"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := gen.rewriteStdlibImport(tt.input)
			if got != tt.expected {
				t.Errorf("rewriteStdlibImport(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestImportCollisionAutoAlias(t *testing.T) {
	// When both stdlib/json and encoding/json are imported,
	// the stdlib import should get auto-aliased to "kukijson"
	input := `import "stdlib/json"
import "encoding/json"

type Data
    Value string

func main()
    d := Data{}
    json.Marshal(d)
`

	output := generateSource(t, input)

	if !strings.Contains(output, `kukijson "github.com/kukichalang/kukicha/stdlib/json"`) {
		t.Errorf("expected kukijson alias for stdlib/json collision, got:\n%s", output)
	}
	if !strings.Contains(output, `"encoding/json"`) {
		t.Errorf("expected encoding/json import, got:\n%s", output)
	}
}

func TestImportBuiltinTypeAlias(t *testing.T) {
	// When a package name collides with a Go built-in type (e.g., "string"),
	// it should get auto-aliased to "kukistring"
	input := `import "stdlib/string"

func main()
    x := string.ToUpper("hello")
`

	output := generateSource(t, input)

	if !strings.Contains(output, "kukistring") {
		t.Errorf("expected kukistring alias for string package, got:\n%s", output)
	}
}

func TestVersionSuffixImportAlias(t *testing.T) {
	input := `import "encoding/json/v2"

type Data
    Value string

func main()
    d := Data{}
    json.Marshal(d)
`

	output := generateSource(t, input)

	if !strings.Contains(output, `json "encoding/json/v2"`) {
		t.Errorf("expected json alias for v2 import, got:\n%s", output)
	}
}

func TestSingleImportFormat(t *testing.T) {
	// Single import should use `import "pkg"` not `import ( "pkg" )`
	input := `import "fmt"

func main()
    fmt.Println("hello")
`

	output := generateSource(t, input)

	if !strings.Contains(output, `import "fmt"`) {
		t.Errorf("expected single-line import format, got:\n%s", output)
	}
}

func TestMultipleImportsFormat(t *testing.T) {
	// Multiple imports should use grouped format
	input := `import "fmt"
import "os"

func main()
    fmt.Println(os.Args)
`

	output := generateSource(t, input)

	if !strings.Contains(output, "import (") {
		t.Errorf("expected grouped import format, got:\n%s", output)
	}
}

func TestAutoImportFmt(t *testing.T) {
	// String interpolation should trigger auto-import of fmt
	input := `func greet(name string) string
    return "Hello {name}!"
`

	output := generateSource(t, input)

	if !strings.Contains(output, `"fmt"`) {
		t.Errorf("expected auto-import of fmt for string interpolation, got:\n%s", output)
	}
}

func TestAutoImportErrors(t *testing.T) {
	// error expressions should trigger auto-import of errors
	input := `func fail() error
    return error "something went wrong"
`

	output := generateSource(t, input)

	if !strings.Contains(output, `"errors"`) {
		t.Errorf("expected auto-import of errors package, got:\n%s", output)
	}
}

func TestPackageAliasForPetiole(t *testing.T) {
	// "package" should be accepted as an alias for "petiole"
	input := `package mypkg

func main()
    x := 1
`
	output := generateSource(t, input)
	if !strings.Contains(output, "package mypkg") {
		t.Errorf("expected 'package mypkg' in output, got:\n%s", output)
	}
}

func TestGroupedImportSyntax(t *testing.T) {
	// import ( ... ) grouped syntax should produce the same result as separate imports
	input := `import (
    "fmt"
    "os"
)

func main()
    fmt.Println(os.Args)
`
	output := generateSource(t, input)
	if !strings.Contains(output, `"fmt"`) {
		t.Errorf("expected fmt import, got:\n%s", output)
	}
	if !strings.Contains(output, `"os"`) {
		t.Errorf("expected os import, got:\n%s", output)
	}
}

func TestGroupedImportWithAlias(t *testing.T) {
	// Grouped import with both Kukicha-style (as) and Go-style aliases
	input := `import (
    "encoding/json"
    j "encoding/json/v2"
)

func main()
    json.Marshal(nil)
    j.Marshal(nil)
`
	output := generateSource(t, input)
	if !strings.Contains(output, `"encoding/json"`) {
		t.Errorf("expected encoding/json import, got:\n%s", output)
	}
	if !strings.Contains(output, `j "encoding/json/v2"`) {
		t.Errorf("expected aliased v2 import, got:\n%s", output)
	}
}

func TestGoStyleImportAlias(t *testing.T) {
	// Go-style alias before path: import j "encoding/json"
	input := `import j "encoding/json"

func main()
    j.Marshal(nil)
`
	output := generateSource(t, input)
	if !strings.Contains(output, `j "encoding/json"`) {
		t.Errorf("expected 'j \"encoding/json\"' in import, got:\n%s", output)
	}
}

func TestRawGoStdlibImport(t *testing.T) {
	// Raw Go stdlib paths (without stdlib/ prefix) should pass through unchanged
	input := `import "strings"
import "strconv"

func main()
    x := strings.ToUpper("hello")
    n := strconv.Itoa(42)
    _ = x
    _ = n
`
	output := generateSource(t, input)
	if !strings.Contains(output, `"strings"`) {
		t.Errorf("expected 'strings' import, got:\n%s", output)
	}
	if !strings.Contains(output, `"strconv"`) {
		t.Errorf("expected 'strconv' import, got:\n%s", output)
	}
}
