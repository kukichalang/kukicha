package main

//go:generate sh -c "cd ../.. && go run ./cmd/genstdlibregistry"
//go:generate sh -c "cd ../.. && go run ./cmd/gengostdlib"

import (
	"fmt"
	"os"

	"github.com/kukichalang/kukicha/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "build":
		buildMain(args)
	case "run":
		runMain(args)
	case "check":
		checkMain(args)
	case "fmt":
		fmtMain(args)
	case "pack":
		packMain(args)
	case "audit":
		auditMain(args)
	case "brew":
		brewMain(args)
	case "init":
		initCommand(args)
	case "version":
		fmt.Printf("kukicha version %s\n", version.Version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Kukicha - A transpiler that compiles Kukicha to Go")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  kukicha build [--target t] [--vulncheck] [--wasm] <file.kuki|dir>  Compile Kukicha to Go")
	fmt.Fprintln(os.Stderr, "  kukicha run [--target t] <file.kuki|dir>   Transpile and execute Kukicha")
	fmt.Fprintln(os.Stderr, "  kukicha check [--json] [--strict-onerr] <file.kuki|dir> [...]   Type check Kukicha")
	fmt.Fprintln(os.Stderr, "  kukicha audit [--json] [--warn-only] [dir]  Check dependencies for vulnerabilities")
	fmt.Fprintln(os.Stderr, "  kukicha fmt [options] <files>  Fix indentation and normalize style")
	fmt.Fprintln(os.Stderr, "    -w          Write result to file instead of stdout")
	fmt.Fprintln(os.Stderr, "    --check     Check if files are formatted (exit 1 if not)")
	fmt.Fprintln(os.Stderr, "  kukicha brew [--stdout] [--remove-kuki] <file.kuki|dir>  Convert Kukicha to standalone Go")
	fmt.Fprintln(os.Stderr, "  kukicha pack [--output dir] <skill.kuki>  Package skill for distribution")
	fmt.Fprintln(os.Stderr, "  kukicha init [module-name]  Initialize project (go mod init + extract stdlib)")
	fmt.Fprintln(os.Stderr, "  kukicha version             Show version information")
	fmt.Fprintln(os.Stderr, "  kukicha help                Show this help message")
}
