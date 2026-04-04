package semantic

import (
	"strings"
	"testing"
)

// =============================================================================
// Security check edge cases — piped arguments, additional variants, and
// boundary conditions NOT covered by semantic_test.go.
// =============================================================================

// --- XSS: piped variant ---

func TestHTMLNonLiteral_PipedResponseWriter(t *testing.T) {
	// When w is piped, content arg shifts to index 0.
	source := `import "net/http"
import "stdlib/http" as httphelper

func Handle(w http.ResponseWriter, r reference http.Request)
    body := "<h1>hello</h1>"
    w |> httphelper.HTML(body) onerr return
`
	assertSecurityError(t, source, "XSS risk")
}

func TestHTMLNonLiteral_PipedLiteralAllowed(t *testing.T) {
	source := `import "net/http"
import "stdlib/http" as httphelper

func Handle(w http.ResponseWriter, r reference http.Request)
    w |> httphelper.HTML("<h1>Safe</h1>") onerr return
`
	assertNoSecurityError(t, source, "XSS risk")
}

func TestHTMLNonLiteral_MissingContentArg(t *testing.T) {
	// Edge case: not enough args should not panic.
	source := `import "net/http"
import "stdlib/http" as httphelper

func Handle(w http.ResponseWriter, r reference http.Request)
    httphelper.HTML(w) onerr return
`
	analyzeIgnoringNonSecurity(t, source, "XSS risk")
}

// --- SSRF: fetch.New variant ---

func TestFetchInHandler_FetchNew(t *testing.T) {
	source := `import "net/http"
import "stdlib/fetch"

func Handle(w http.ResponseWriter, r reference http.Request)
    client := fetch.New()
    _ = client
`
	assertSecurityError(t, source, "SSRF risk")
}

func TestFetchInHandler_NotAHandler(t *testing.T) {
	// A function without http.ResponseWriter is NOT a handler.
	source := `import "stdlib/fetch"

func Background()
    resp, err := fetch.Get("https://example.com")
    _ = resp
    _ = err
`
	assertNoSecurityError(t, source, "SSRF risk")
}

// --- Path traversal: additional file operations ---

func TestFilesInHandler_Delete(t *testing.T) {
	source := `import "net/http"
import "stdlib/files"

func Handle(w http.ResponseWriter, r reference http.Request)
    files.Delete("/tmp/file") onerr return
`
	assertSecurityError(t, source, "path traversal risk")
}

func TestFilesInHandler_DeleteAll(t *testing.T) {
	source := `import "net/http"
import "stdlib/files"

func Handle(w http.ResponseWriter, r reference http.Request)
    files.DeleteAll("/tmp/dir") onerr return
`
	assertSecurityError(t, source, "path traversal risk")
}

func TestFilesInHandler_List(t *testing.T) {
	source := `import "net/http"
import "stdlib/files"

func Handle(w http.ResponseWriter, r reference http.Request)
    entries, err := files.List("/tmp")
    _ = entries
    _ = err
`
	assertSecurityError(t, source, "path traversal risk")
}

func TestFilesInHandler_ListRecursive(t *testing.T) {
	source := `import "net/http"
import "stdlib/files"

func Handle(w http.ResponseWriter, r reference http.Request)
    entries, err := files.ListRecursive("/tmp")
    _ = entries
    _ = err
`
	assertSecurityError(t, source, "path traversal risk")
}

func TestFilesInHandler_Append(t *testing.T) {
	source := `import "net/http"
import "stdlib/files"

func Handle(w http.ResponseWriter, r reference http.Request)
    files.Append("data", "/tmp/file") onerr return
`
	assertSecurityError(t, source, "path traversal risk")
}

func TestFilesInHandler_AppendString(t *testing.T) {
	source := `import "net/http"
import "stdlib/files"

func Handle(w http.ResponseWriter, r reference http.Request)
    files.AppendString("data", "/tmp/file") onerr return
`
	assertSecurityError(t, source, "path traversal risk")
}

func TestFilesInHandler_ReadBytes(t *testing.T) {
	source := `import "net/http"
import "stdlib/files"

func Handle(w http.ResponseWriter, r reference http.Request)
    data, err := files.ReadBytes("/tmp/file")
    _ = data
    _ = err
`
	assertSecurityError(t, source, "path traversal risk")
}

func TestFilesInHandler_WriteString(t *testing.T) {
	source := `import "net/http"
import "stdlib/files"

func Handle(w http.ResponseWriter, r reference http.Request)
    files.WriteString("data", "/tmp/file") onerr return
`
	assertSecurityError(t, source, "path traversal risk")
}

func TestFilesOutsideHandler_AllAllowed(t *testing.T) {
	source := `import "stdlib/files"

func Cleanup()
    files.Delete("/tmp/old") onerr return
    files.DeleteAll("/tmp/dir") onerr return
`
	assertNoSecurityError(t, source, "path traversal risk")
}

// --- Command injection: piped arg edge case ---

func TestShellRun_PipedArgWarning(t *testing.T) {
	// When argument is piped, should NOT error but SHOULD warn.
	source := `import "stdlib/shell"

func Run(cmd string) (string, error)
    return cmd |> shell.Run()
`
	assertNoSecurityError(t, source, "command injection risk")
	assertSecurityWarning(t, source, "command injection risk")
}

func TestShellRun_NoArgs(t *testing.T) {
	// Edge case: shell.Run with no arguments should not panic.
	source := `import "stdlib/shell"

func Run() (string, error)
    return shell.Run()
`
	analyzeIgnoringNonSecurity(t, source, "command injection")
}

// --- Open redirect: piped variant ---

func TestRedirectNonLiteral_PipedResponseWriter(t *testing.T) {
	// When w is piped, URL index shifts from 2 to 1.
	source := `import "net/http"
import "stdlib/http" as httphelper

func Handle(w http.ResponseWriter, r reference http.Request)
    target := r.URL.Query().Get("to")
    w |> httphelper.Redirect(r, target)
`
	assertSecurityError(t, source, "open redirect risk")
}

func TestRedirectNonLiteral_PipedLiteralAllowed(t *testing.T) {
	source := `import "net/http"
import "stdlib/http" as httphelper

func Handle(w http.ResponseWriter, r reference http.Request)
    w |> httphelper.Redirect(r, "/safe")
`
	assertNoSecurityError(t, source, "open redirect risk")
}

func TestRedirectNonLiteral_StdlibExempt(t *testing.T) {
	// Stdlib source files are exempt from redirect checks.
	source := `import "net/http"
import "stdlib/http" as httphelper

func Handle(w http.ResponseWriter, r reference http.Request)
    target := "/dashboard"
    httphelper.Redirect(w, r, target)
`
	_, errors := analyzeSourceWithFile(t, source, "stdlib/http/redirect.kuki")

	for _, e := range errors {
		if strings.Contains(e.Error(), "open redirect risk") {
			t.Fatalf("stdlib files should be exempt from redirect check, got: %v", e)
		}
	}
}

func TestRedirectNonLiteral_TooFewArgs(t *testing.T) {
	// Edge case: not enough arguments should not panic.
	source := `import "net/http"
import "stdlib/http" as httphelper

func Handle(w http.ResponseWriter, r reference http.Request)
    httphelper.Redirect(w, r)
`
	analyzeIgnoringNonSecurity(t, source, "open redirect risk")
}

// --- isInHTTPHandler edge cases ---

func TestIsInHTTPHandler_NoFunc(t *testing.T) {
	// A top-level type decl should not trigger handler detection.
	source := `type Config
    Port int
`
	assertNoSecurityError(t, source, "risk")
}

// --- SQL injection: db.* functions ---

func TestSQLInterpolation_DbQuery(t *testing.T) {
	source := `import "stdlib/db"

func Run(pool db.Pool, userInput string)
    rows, err := db.Query(pool, "SELECT * FROM users WHERE name = '{userInput}'")
    _ = rows
    _ = err
`
	assertSecurityError(t, source, "SQL injection risk")
}

func TestSQLInterpolation_DbQuerySafe(t *testing.T) {
	source := `import "stdlib/db"

func Run(pool db.Pool)
    rows, err := db.Query(pool, "SELECT * FROM users WHERE active = $1", true)
    _ = rows
    _ = err
`
	assertNoSecurityError(t, source, "SQL injection risk")
}

func TestSQLInterpolation_DbExec(t *testing.T) {
	source := `import "stdlib/db"

func Run(pool db.Pool, name string)
    n, err := db.Exec(pool, "DELETE FROM users WHERE name = '{name}'")
    _ = n
    _ = err
`
	assertSecurityError(t, source, "SQL injection risk")
}

func TestSQLInterpolation_DbTxExec(t *testing.T) {
	source := `import "stdlib/db"

func Run(tx db.Tx, name string)
    n, err := db.TxExec(tx, "UPDATE users SET active = true WHERE name = '{name}'")
    _ = n
    _ = err
`
	assertSecurityError(t, source, "SQL injection risk")
}

func TestSQLInterpolation_DbCount(t *testing.T) {
	source := `import "stdlib/db"

func Run(pool db.Pool, table string)
    n, err := db.Count(pool, "SELECT COUNT(*) FROM {table}")
    _ = n
    _ = err
`
	assertSecurityError(t, source, "SQL injection risk")
}

func TestSQLInterpolation_DbpkgAlias(t *testing.T) {
	source := `import "stdlib/db" as dbpkg

func Run(pool dbpkg.Pool, name string)
    rows, err := dbpkg.Query(pool, "SELECT * FROM users WHERE name = '{name}'")
    _ = rows
    _ = err
`
	assertSecurityError(t, source, "SQL injection risk")
}

func TestSQLInterpolation_PipedCallShiftsArgIndex(t *testing.T) {
	// When the pool is piped in, the SQL string is at index 0 (not 1).
	source := `import "stdlib/db"

func Bad(pool db.Pool, id int)
    rows := pool |> db.Query("SELECT * FROM users WHERE id = {id}") onerr return
    _ = rows
`
	assertSecurityError(t, source, "SQL injection risk")
}

func TestSQLInterpolation_TxQueryRejected(t *testing.T) {
	source := `import "stdlib/db"

func Bad(tx db.Tx, id int)
    rows, err := db.TxQuery(tx, "SELECT * FROM t WHERE id = {id}")
    _ = rows
    _ = err
`
	assertSecurityError(t, source, "SQL injection risk")
}

func TestSQLInterpolation_TxQueryRowRejected(t *testing.T) {
	source := `import "stdlib/db"

func Bad(tx db.Tx, id int)
    row := db.TxQueryRow(tx, "SELECT name FROM t WHERE id = {id}")
    _ = row
`
	assertSecurityError(t, source, "SQL injection risk")
}

func TestSQLInterpolation_NonSQLFunctionIgnored(t *testing.T) {
	// Calling a non-db function with interpolation must NOT flag SQL injection.
	source := `func Format(name string) string
    return "Hello {name}"
`
	assertNoSecurityError(t, source, "SQL injection")
}

func TestSQLInterpolation_NoArgs(t *testing.T) {
	// Edge case: db.Query with too few arguments should not panic.
	source := `import "stdlib/db"

func Bad(pool db.Pool)
    rows, err := db.Query(pool)
    _ = rows
    _ = err
`
	analyzeIgnoringNonSecurity(t, source, "SQL injection")
}

func TestSQLInterpolation_PlainLiteralAllowed(t *testing.T) {
	source := `import "stdlib/db"

func Good(pool db.Pool)
    n, err := db.Exec(pool, "DELETE FROM sessions WHERE expired = true")
    _ = n
    _ = err
`
	assertNoSecurityError(t, source, "SQL injection risk")
}

// --- Inline JS in html.Render ---

func TestHTMLRenderInlineScript_Warns(t *testing.T) {
	source := `import "stdlib/html"

func Page() html.Fragment
    return html.Render("<div><script>alert('xss')</script></div>")
`
	assertSecurityWarning(t, source, "inline <script>")
}

func TestHTMLRenderInlineScript_CaseInsensitive(t *testing.T) {
	source := `import "stdlib/html"

func Page() html.Fragment
    return html.Render("<div><SCRIPT>alert('xss')</SCRIPT></div>")
`
	assertSecurityWarning(t, source, "inline <script>")
}

func TestHTMLRenderScriptSrc_Warns(t *testing.T) {
	// Even <script src="..."> inside html.Render should warn —
	// use a <script src> in the static HTML layout instead.
	source := `import "stdlib/html"

func Page() html.Fragment
    return html.Render("<script src='/static/app.js'></script>")
`
	assertSecurityWarning(t, source, "inline <script>")
}

func TestHTMLRenderNoScript_NoWarning(t *testing.T) {
	source := `import "stdlib/html"

func Page() html.Fragment
    return html.Render("<div>{html.Escape(name)}</div>")
`
	assertNoSecurityWarning(t, source, "inline <script>")
}

func TestHTMLRenderOnclick_Warns(t *testing.T) {
	source := `import "stdlib/html"

func Page() html.Fragment
    return html.Render("<button onclick='alert(1)'>Click</button>")
`
	assertSecurityWarning(t, source, "inline event handler")
}

func TestHTMLRenderOnload_Warns(t *testing.T) {
	source := `import "stdlib/html"

func Page() html.Fragment
    return html.Render("<img onerror='alert(1)' src='x'>")
`
	assertSecurityWarning(t, source, "inline event handler")
}

func TestHTMLRenderHxGet_NoWarning(t *testing.T) {
	// HTMX attributes like hx-get are NOT event handlers.
	source := `import "stdlib/html"

func Page() html.Fragment
    return html.Render("<div hx-get='/data'>Load</div>")
`
	assertNoSecurityWarning(t, source, "inline event handler")
}

func TestHTMLRenderOnErr_NoFalsePositive(t *testing.T) {
	// "onerr" in Kukicha is NOT an event handler — make sure it doesn't trigger.
	source := `import "stdlib/html"

func Page() html.Fragment
    return html.Render("<p>Use onerr to handle errors</p>")
`
	assertNoSecurityWarning(t, source, "inline event handler")
}

// =============================================================================
// Test helpers
// =============================================================================

// assertSecurityError parses source and asserts an error containing substr.
func assertSecurityError(t *testing.T, source string, substr string) {
	t.Helper()
	_, errors := analyzeSource(t, source)

	for _, e := range errors {
		if strings.Contains(e.Error(), substr) {
			return
		}
	}
	t.Fatalf("expected error containing %q, got: %v", substr, errors)
}

// assertNoSecurityError parses source and asserts NO error containing substr.
func assertNoSecurityError(t *testing.T, source string, substr string) {
	t.Helper()
	_, errors := analyzeSource(t, source)

	for _, e := range errors {
		if strings.Contains(e.Error(), substr) {
			t.Fatalf("unexpected error containing %q: %v", substr, e)
		}
	}
}

// assertSecurityWarning parses source and asserts a warning containing substr.
func assertSecurityWarning(t *testing.T, source string, substr string) {
	t.Helper()
	result := analyzeSourceResult(t, source)

	for _, w := range result.Warnings {
		if strings.Contains(w.Error(), substr) {
			return
		}
	}
	t.Fatalf("expected warning containing %q, got warnings: %v", substr, result.Warnings)
}

// assertNoSecurityWarning parses source and asserts NO warning containing substr.
func assertNoSecurityWarning(t *testing.T, source string, substr string) {
	t.Helper()
	result := analyzeSourceResult(t, source)

	for _, w := range result.Warnings {
		if strings.Contains(w.Error(), substr) {
			t.Fatalf("unexpected warning containing %q: %v", substr, w)
		}
	}
}

// analyzeIgnoringNonSecurity parses source and ensures it doesn't panic,
// ignoring all errors except those containing the security substr.
func analyzeIgnoringNonSecurity(t *testing.T, source string, securitySubstr string) {
	t.Helper()
	_, errors := analyzeSource(t, source)

	for _, e := range errors {
		if strings.Contains(e.Error(), securitySubstr) {
			t.Fatalf("unexpected security error: %v", e)
		}
	}
}
