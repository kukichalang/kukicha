package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// AuditOptions controls vulnerability scanning behavior.
type AuditOptions struct {
	Dir      string
	JSON     bool
	WarnOnly bool
}

// findProjectRoot walks up from dir looking for go.mod and returns the
// directory containing it. Unlike findProjectDir it returns an error when
// no go.mod is found instead of silently falling back.
func findProjectRoot(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	for d := absDir; d != filepath.Dir(d); d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d, nil
		}
	}

	return "", fmt.Errorf("no go.mod found in %s or any parent directory", absDir)
}

// runAudit executes govulncheck and returns an exit code:
//
//	0 = no vulnerabilities
//	1 = error (govulncheck missing, no go.mod, etc.)
//	3 = vulnerabilities found (govulncheck convention)
//
// When WarnOnly is set, exit code 3 is converted to 0.
func runAudit(opts AuditOptions) int {
	if _, err := exec.LookPath("govulncheck"); err != nil {
		fmt.Fprintln(os.Stderr, "Error: govulncheck is not installed.")
		fmt.Fprintln(os.Stderr, "Install it with: go install golang.org/x/vuln/cmd/govulncheck@latest")
		return 1
	}

	dir := opts.Dir
	if dir == "" {
		dir = "."
	}

	root, err := findProjectRoot(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	args := []string{"./..."}
	if opts.JSON {
		args = append([]string{"-json"}, args...)
	}

	cmd := exec.Command("govulncheck", args...)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			if code == 3 && opts.WarnOnly {
				return 0
			}
			return code
		}
		fmt.Fprintf(os.Stderr, "Error running govulncheck: %v\n", err)
		return 1
	}

	return 0
}

// auditCommand is the entry point called from the main command switch.
func auditCommand(args []string, jsonFlag bool, warnOnly bool) {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	code := runAudit(AuditOptions{
		Dir:      dir,
		JSON:     jsonFlag,
		WarnOnly: warnOnly,
	})
	if code != 0 {
		os.Exit(code)
	}
}
