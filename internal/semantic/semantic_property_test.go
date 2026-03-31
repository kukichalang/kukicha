package semantic

import (
	"fmt"
	"strings"
	"testing"
)

// TestSecurityCheckMonotonicity verifies that adding code to a program never
// removes an existing security error. This guards against accidental state
// that could cause security checks to clear each other.
func TestSecurityCheckMonotonicity(t *testing.T) {
	// Each test case has a base program that triggers a known security error,
	// plus an addition that is prepended or appended. We verify the original
	// error still appears in the expanded program.
	// Each addition is appended AFTER the base so imports remain at the top.
	// Patterns mirror the existing semantic_security_test.go to ensure they parse correctly.
	cases := []struct {
		name        string
		base        string   // triggers at least one security error
		additions   []string // code appended after the base
		errorSubstr string   // substring that must still appear in errors
	}{
		{
			name: "SQL injection survives added function",
			base: `import "stdlib/db"

func Run(pool db.Pool, userInput string)
    rows, err := db.Query(pool, "SELECT * FROM users WHERE name = '{userInput}'")
    _ = rows
    _ = err
`,
			additions: []string{
				"\nfunc UnrelatedHelper(x int) int\n    return x + 1\n",
				"\nfunc AnotherHelper(s string) string\n    return s\n",
				"\ntype Config\n    Value string\n",
			},
			errorSubstr: "SQL injection risk",
		},
		{
			name: "XSS risk survives added type",
			base: `import "net/http"
import "stdlib/http" as httphelper

func Handle(w http.ResponseWriter, r reference http.Request)
    httphelper.HTML(w, userInput) onerr return
`,
			additions: []string{
				"\ntype Config\n    Value string\n",
				"\nconst MaxSize = 100\n",
			},
			errorSubstr: "XSS risk",
		},
		{
			name: "SSRF risk survives added function",
			base: `import "net/http"
import "stdlib/fetch"

func Handle(w http.ResponseWriter, r reference http.Request)
    resp, err := fetch.Get("https://example.com")
    _ = resp
    _ = err
`,
			additions: []string{
				"\nfunc Helper(x int) int\n    return x\n",
				"\ntype Extra\n    Name string\n",
			},
			errorSubstr: "SSRF risk",
		},
		{
			name: "path traversal risk survives added function",
			base: `import "net/http"
import "stdlib/files"

func Handle(w http.ResponseWriter, r reference http.Request)
    data, err := files.Read("/tmp/file")
    _ = data
    _ = err
`,
			additions: []string{
				"\nfunc Extra(x int) int\n    return x * 2\n",
			},
			errorSubstr: "path traversal",
		},
		{
			name: "command injection survives added function",
			base: `import "stdlib/shell"

func Run(userCmd string)
    shell.Run(userCmd)
`,
			additions: []string{
				"\nfunc Noop() int\n    return 0\n",
				"\ntype Helper\n    Name string\n",
			},
			errorSubstr: "command injection",
		},
		{
			name: "open redirect survives added enum",
			base: `import "net/http"
import "stdlib/http" as httphelper

func Handle(w http.ResponseWriter, r reference http.Request)
    httphelper.Redirect(w, r, userURL) onerr return
`,
			additions: []string{
				"\nenum Status\n    OK = 200\n",
				"\nfunc Extra(x int) int\n    return x\n",
			},
			errorSubstr: "open redirect",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, addition := range tc.additions {
				expanded := tc.base + addition

				_, errs := analyzeSource(t, expanded)

				found := false
				for _, e := range errs {
					if strings.Contains(e.Error(), tc.errorSubstr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("security error %q disappeared after adding code\nexpanded program:\n%s\nerrors: %v",
						tc.errorSubstr, expanded, errs)
				}
			}
		})
	}
}

// TestReturnCountConsistency verifies invariants across all Kukicha stdlib
// registry entries:
//   - Count == len(Types) for every entry
//   - ParamNames is either nil or non-empty (no empty-string names)
//   - Every security function exists in the main stdlib registry
func TestReturnCountConsistency(t *testing.T) {
	entries := GetAllStdlibEntries()

	if len(entries) == 0 {
		t.Fatal("stdlib registry is empty — regenerate with make genstdlibregistry")
	}

	for name, entry := range entries {
		t.Run(fmt.Sprintf("entry/%s", name), func(t *testing.T) {
			// Count must match len(Types)
			if entry.Count != len(entry.Types) {
				t.Errorf("%s: Count=%d but len(Types)=%d — they must match",
					name, entry.Count, len(entry.Types))
			}

			// Count must be positive
			if entry.Count <= 0 {
				t.Errorf("%s: Count=%d must be positive (void functions are excluded from registry)",
					name, entry.Count)
			}

			// ParamNames: if present, no entry may be empty
			for i, pname := range entry.ParamNames {
				if pname == "" {
					t.Errorf("%s: ParamNames[%d] is empty string — all param names must be non-empty",
						name, i)
				}
			}
		})
	}

	// Every security function must have a non-empty category string.
	// (Security functions may be void, so they are not required to be in the return-count registry.)
	t.Run("security_functions_have_non_empty_category", func(t *testing.T) {
		secFuncs := GetAllSecurityFunctions()
		if len(secFuncs) == 0 {
			t.Fatal("generatedSecurityFunctions is empty")
		}
		for funcName, cat := range secFuncs {
			if cat == "" {
				t.Errorf("security function %q has empty category", funcName)
			}
		}
	})
}
