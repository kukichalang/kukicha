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
		fmt.Fprintln(os.Stderr, "Usage: kukicha run [--target <target>] <file.kuki|dir|module@version> [args...]")
		os.Exit(1)
	}
	runArgs := runFlags.Args()
	if len(runArgs) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: kukicha run [--target <target>] <file.kuki|dir|module@version> [args...]")
		os.Exit(1)
	}
	var exitCode int
	if isModulePath(runArgs[0]) {
		exitCode = runModuleCommand(runArgs[0], *target, runArgs[1:])
	} else {
		exitCode = runCommand(runArgs[0], *target, runArgs[1:])
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func runCommand(filename string, targetFlag string, scriptArgs []string) int {
	cr, err := compile(filename, targetFlag, "", false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	// If stdlib is needed, extract it and ensure go.mod is configured.
	// Keep temp source in project context so local replace directives resolve.
	var tmp *os.File
	if needsStdlib(cr.goCode, cr.projectDir) {
		if err := ensureStdlibIfNeeded(cr.goCode, cr.projectDir); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		tmp, err = os.CreateTemp(filepath.Join(cr.projectDir, ".kukicha"), "run-*.go")
	} else {
		tmp, err = os.CreateTemp("", "kukicha-run-*.go")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temporary file: %v\n", err)
		return 1
	}
	tmpFile := tmp.Name()
	defer os.Remove(tmpFile)

	if _, err := tmp.Write(cr.formatted); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing temporary file: %v\n", err)
		return 1
	}
	if err := tmp.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Error closing temporary file: %v\n", err)
		return 1
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
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		return 1
	}
	return 0
}
