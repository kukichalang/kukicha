package semantic

import (
	"regexp"
	"strconv"
	"strings"
)

// Diagnostic is a structured compiler diagnostic (error or warning).
// It is the machine-readable form of the errors and warnings produced
// by the Analyzer. Use Diagnostics() to obtain them after Analyze().
//
// JSON field names match the format described in VERIFIER-TODO.md §4:
//
//	{
//	  "file": "app.kuki",
//	  "line": 12,
//	  "col": 5,
//	  "severity": "error",
//	  "category": "security/sql-injection",
//	  "message": "SQL injection risk: ...",
//	  "suggestion": "use parameter placeholders ($1, $2, ...) instead of string interpolation"
//	}
type Diagnostic struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	Col        int    `json:"col"`
	Severity   string `json:"severity"`   // "error" or "warning"
	Category   string `json:"category"`   // e.g. "security/sql-injection", or "" for non-categorised
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"` // concrete safe alternative, or ""
}

// posPattern matches the "file:line:col: " prefix produced by Analyzer.error / Analyzer.warn.
// The file portion is captured greedily up to the last colon-digit-colon-digit pattern.
var posPattern = regexp.MustCompile(`^(.+):(\d+):(\d+): (.+)$`)

// parseDiagnostic converts a formatted error string ("file:line:col: msg") into a
// Diagnostic. If the position prefix cannot be parsed the whole string is the message.
func parseDiagnostic(errMsg, severity string) Diagnostic {
	d := Diagnostic{Severity: severity, Message: errMsg}

	if m := posPattern.FindStringSubmatch(errMsg); m != nil {
		d.File = m[1]
		d.Line, _ = strconv.Atoi(m[2])
		d.Col, _ = strconv.Atoi(m[3])
		d.Message = m[4]
	}

	d.Category = inferCategory(d.Message)
	d.Suggestion = suggestionForCategory(d.Category)
	return d
}

// inferCategory returns a dot-separated category string for known message patterns.
func inferCategory(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "sql injection"):
		return "security/sql-injection"
	case strings.Contains(lower, "xss risk"),
		strings.Contains(lower, "inline <script"),
		strings.Contains(lower, "inline event handler"):
		return "security/xss"
	case strings.Contains(lower, "ssrf risk"):
		return "security/ssrf"
	case strings.Contains(lower, "path traversal"):
		return "security/path-traversal"
	case strings.Contains(lower, "command injection"):
		return "security/command-injection"
	case strings.Contains(lower, "open redirect"):
		return "security/open-redirect"
	case strings.Contains(lower, "deprecated"):
		return "deprecated"
	default:
		return ""
	}
}

// suggestionForCategory returns a concrete safe alternative for a security category.
func suggestionForCategory(category string) string {
	switch category {
	case "security/sql-injection":
		return "use parameter placeholders ($1, $2, ...) instead of string interpolation"
	case "security/xss":
		return "use http.SafeHTML to HTML-escape user-controlled content, or move scripts to static .js files"
	case "security/ssrf":
		return "use fetch.SafeGet or add fetch.Transport(netguard.HTTPTransport(...)) to restrict outbound requests"
	case "security/path-traversal":
		return "use sandbox.* with a restricted root for user-controlled paths"
	case "security/command-injection":
		return "use shell.Output() with separate arguments for variable input"
	case "security/open-redirect":
		return "use http.SafeRedirect(w, r, url, allowedHosts...) to validate the destination"
	default:
		return ""
	}
}

// Diagnostics returns all errors and warnings from the most recent Analyze() call
// as structured Diagnostic values. The errors come first, then warnings.
//
// Call after Analyze().
func (a *Analyzer) Diagnostics() []Diagnostic {
	errs := a.errors
	warns := a.warnings
	diags := make([]Diagnostic, 0, len(errs)+len(warns))
	for _, e := range errs {
		diags = append(diags, parseDiagnostic(e.Error(), "error"))
	}
	for _, w := range warns {
		diags = append(diags, parseDiagnostic(w.Error(), "warning"))
	}
	return diags
}

// ParseErrorToDiagnostic converts a raw error string from the parser (or any
// stage before semantic analysis) into a structured Diagnostic.
// This is used by the CLI when parsing fails before an Analyzer is available.
func ParseErrorToDiagnostic(errMsg string) Diagnostic {
	return parseDiagnostic(errMsg, "error")
}
