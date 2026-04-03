package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func runMain(args []string) {
	runFlags := flag.NewFlagSet("run", flag.ContinueOnError)
	runFlags.SetOutput(os.Stderr)
	target := runFlags.String("target", "", "Run target")
	if err := runFlags.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "Usage: kukicha run [--target <target>] <file.kuki> [args...]")
		os.Exit(1)
	}
	runArgs := runFlags.Args()
	if len(runArgs) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: kukicha run [--target <target>] <file.kuki> [args...]")
		os.Exit(1)
	}
	runCommand(runArgs[0], *target, runArgs[1:])
}

func runCommand(filename string, targetFlag string, scriptArgs []string) {
	cr := compile(filename, targetFlag, "", false)

	// If stdlib is needed, extract it and ensure go.mod is configured.
	// Keep temp source in project context so local replace directives resolve.
	var tmp *os.File
	var err error
	if needsStdlib(cr.goCode, cr.projectDir) {
		ensureStdlibIfNeeded(cr.goCode, cr.projectDir)
		tmp, err = os.CreateTemp(filepath.Join(cr.projectDir, ".kukicha"), "run-*.go")
	} else {
		tmp, err = os.CreateTemp("", "kukicha-run-*.go")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temporary file: %v\n", err)
		os.Exit(1)
	}
	tmpFile := tmp.Name()
	defer os.Remove(tmpFile)

	if _, err := tmp.Write(cr.formatted); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing temporary file: %v\n", err)
		os.Exit(1)
	}
	if err := tmp.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Error closing temporary file: %v\n", err)
		os.Exit(1)
	}

	// Run with go run. Use -mod=mod so Go updates go.sum automatically when
	// stdlib transitive dependencies (e.g. gopkg.in/yaml.v3) are not yet listed.
	goArgs := append([]string{"run", "-mod=mod", tmpFile}, scriptArgs...)
	cmd := exec.Command("go", goArgs...)
	cmd.Dir = cr.projectDir
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if stderrBuf.Len() > 0 {
		rewritten := rewriteGoErrors(stderrBuf.Bytes(), tmpFile, cr.absFile)
		rewritten = rewriteVarNames(rewritten, cr.varMap)
		os.Stderr.Write(rewritten)
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
