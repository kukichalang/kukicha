package codegen

import (
	"github.com/kukichalang/kukicha/internal/parser"
	"github.com/kukichalang/kukicha/internal/semantic"
	"strings"
	"testing"
)

func TestMultiLinePipeCodegen(t *testing.T) {
	// Multi-line pipe must produce the same nested call as single-line.
	singleLine := `func Test() string
    return "hello" |> ToUpper() |> TrimSpace()
`
	multiLine := `func Test() string
    return "hello" |>
        ToUpper() |>
        TrimSpace()
`

	generate := func(src string) string {
		t.Helper()
		p, err := parser.New(src, "test.kuki")
		if err != nil {
			t.Fatalf("lexer error: %v", err)
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
		return output
	}

	singleOut := generate(singleLine)
	multiOut := generate(multiLine)

	// Both must contain the fully-nested call.
	want := "TrimSpace(ToUpper(\"hello\"))"
	if !strings.Contains(singleOut, want) {
		t.Errorf("single-line output missing %q:\n%s", want, singleOut)
	}
	if !strings.Contains(multiOut, want) {
		t.Errorf("multi-line output missing %q:\n%s", want, multiOut)
	}
}

func TestOnErrPipeChainPreservesIntermediateErrors(t *testing.T) {
	input := `import "stdlib/fetch"

type Repo
    name string as "name"

func Load(url string) list of Repo
    repos := fetch.Get(url)
        |> fetch.CheckStatus()
        |> fetch.Json(empty list of Repo) onerr return empty list of Repo
    return repos
`

	p, err := parser.New(input, "test.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	analyzer := semantic.New(program)
	analyzer.Analyze()

	gen := New(program)
	gen.SetExprReturnCounts(analyzer.ReturnCounts())
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}

	if strings.Contains(output, "val, _ :=") {
		t.Fatalf("expected no intermediate error discards in pipe onerr lowering, got: %s", output)
	}
	if !strings.Contains(output, "pipe_1, err_2 := fetch.Get(url)") {
		t.Fatalf("expected explicit fetch.Get error capture, got: %s", output)
	}
	if !strings.Contains(output, "pipe_3, err_4 := fetch.CheckStatus(pipe_1)") {
		t.Fatalf("expected explicit fetch.CheckStatus error capture, got: %s", output)
	}
	// Last step assigns directly to the target variable instead of a temp
	if !strings.Contains(output, "repos, err_6 := fetch.Json(pipe_3, []Repo{})") {
		t.Fatalf("expected last pipe step to assign directly to 'repos', got: %s", output)
	}
}

func TestOnErrPipeChainUserDefinedMultiReturn(t *testing.T) {
	input := `import "strconv"

func GetInput() string
    return "42"

func Parse(data string) (int, error)
    return strconv.Atoi(data)

func Run() int
    result := GetInput() |> Parse() onerr return 0
    return result
`

	p, err := parser.New(input, "test.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	analyzer := semantic.New(program)
	analyzer.Analyze()

	gen := New(program)
	gen.SetExprReturnCounts(analyzer.ReturnCounts())
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}

	// GetInput() is a non-error step — collapsed into the Parse call.
	// Parse() is the last (and only error-returning) step, assigns directly to 'result'.
	if !strings.Contains(output, "result, err_2 := Parse(GetInput())") {
		t.Fatalf("expected collapsed pipe chain 'result, err_2 := Parse(GetInput())', got: %s", output)
	}
}

func TestOnErrPipeChainKnownExternalMultiReturn(t *testing.T) {
	input := `import "os"

func Run(path string) (list of os.DirEntry, error)
    entries := path |> os.ReadDir() onerr return
    return entries, empty
`

	p, err := parser.New(input, "test.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	analyzer := semantic.New(program)
	semErrors := analyzer.Analyze()
	if len(semErrors) > 0 {
		t.Fatalf("semantic errors: %v", semErrors)
	}

	gen := New(program)
	gen.SetExprReturnCounts(analyzer.ReturnCounts())
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}

	// path is a non-error base — collapsed directly into os.ReadDir call.
	// os.ReadDir is the last step, assigns directly to 'entries'.
	if !strings.Contains(output, "entries, err_2 := os.ReadDir(path)") {
		t.Fatalf("expected collapsed pipe 'entries, err_2 := os.ReadDir(path)', got: %s", output)
	}
	if !strings.Contains(output, "return []os.DirEntry{}, err_2") {
		t.Fatalf("expected onerr return to propagate os.ReadDir error, got: %s", output)
	}
}

func TestOnErrPipeChainErrorOnlyReturn(t *testing.T) {
	input := `import "os"

func Write(data list of byte, path string) error
    data |> os.WriteFile(path, _, 0644) onerr return
    return empty
`

	p, err := parser.New(input, "test.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	analyzer := semantic.New(program)
	semErrors := analyzer.Analyze()
	if len(semErrors) > 0 {
		t.Fatalf("semantic errors: %v", semErrors)
	}

	gen := New(program)
	gen.SetExprReturnCounts(analyzer.ReturnCounts())
	gen.SetExprTypes(analyzer.ExprTypes())
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}

	// data is a non-error base — collapsed directly into os.WriteFile call.
	// os.WriteFile returns only error — should generate error check, not value assignment.
	if !strings.Contains(output, "err_1 := os.WriteFile(path, data, 0644)") {
		t.Errorf("expected collapsed os.WriteFile with data directly, got:\n%s", output)
	}
	if !strings.Contains(output, "if err_1 != nil {") {
		t.Errorf("expected error check for os.WriteFile, got:\n%s", output)
	}
}

func TestOnErrPipeChainErrorOnlyAfterMultiReturn(t *testing.T) {
	// Simulates the files.Write pattern: data |> marshalFunc() |> os.WriteFile(path, _, 0644) onerr return
	// where marshalFunc returns ([]byte, error) and os.WriteFile returns only error.
	input := `import "os"

func marshalPretty(data any) (list of byte, error)
    return data as list of byte, empty

func Write(data any, path string) error
    data |> marshalPretty() |> os.WriteFile(path, _, 0644) onerr return
    return empty
`

	p, err := parser.New(input, "test.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	analyzer := semantic.New(program)
	semErrors := analyzer.Analyze()
	if len(semErrors) > 0 {
		t.Fatalf("semantic errors: %v", semErrors)
	}

	gen := New(program)
	gen.SetExprReturnCounts(analyzer.ReturnCounts())
	gen.SetExprTypes(analyzer.ExprTypes())
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}

	// data base is non-error — collapsed into marshalPretty call.
	// marshalPretty returns 2 values — split into value + error.
	if !strings.Contains(output, "pipe_1, err_2 := marshalPretty(data)") {
		t.Errorf("expected collapsed marshalPretty(data), got:\n%s", output)
	}
	// os.WriteFile returns only error — should check error directly
	if !strings.Contains(output, "err_3 := os.WriteFile(path, pipe_1, 0644)") {
		t.Errorf("expected os.WriteFile error assigned to err var, got:\n%s", output)
	}
	if !strings.Contains(output, "if err_3 != nil {") {
		t.Errorf("expected error check for os.WriteFile, got:\n%s", output)
	}
}

func TestPipedSwitchCodegen(t *testing.T) {
	input := `func HandleEvent(event string)
    event |> switch
        when "click"
            print("clicked")
        when "hover"
            print("hovered")
        otherwise
            print("unknown")

func ConvertEvent(event string) string
    result := event |> string.ToUpper() |> switch
        when "CLICK"
            return "C"
        otherwise
            return "U"
    return result
`

	output := generateSource(t, input)

	if !strings.Contains(output, "switch event {") {
		t.Errorf("expected flattened 'switch event {', got: %s", output)
	}

	if !strings.Contains(output, "func() string {") {
		t.Errorf("expected typed IIFE 'func() string {', got: %s", output)
	}
}

func TestTypedPipedSwitchCodegen(t *testing.T) {
	input := `func Convert(value any) string
    result := value |> switch as v
        when string
            return v
        when int
            return "number"
        otherwise
            return "other"
    return result
`

	p, err := parser.New(input, "test.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	analyzer := semantic.New(program)
	analyzer.Analyze()

	gen := New(program)
	gen.SetExprTypes(analyzer.ExprTypes())
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}

	if !strings.Contains(output, "func() string {") {
		t.Errorf("expected typed IIFE 'func() string {', got: %s", output)
	}
	if !strings.Contains(output, "switch v := value.(type) {") {
		t.Errorf("expected typed piped switch codegen, got: %s", output)
	}
	if !strings.Contains(output, "case string:") {
		t.Errorf("expected string case, got: %s", output)
	}
}

func TestOnErrTypedPipedSwitchCodegen(t *testing.T) {
	input := `func Convert(value string) string
    result := value |> Risky() |> switch as v
        when string
            return v
        otherwise
            return "other"
    onerr "fallback"
    return result

func Risky(value string) (any, error)
    return value, empty
`

	p, err := parser.New(input, "test.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}
	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	analyzer := semantic.New(program)
	analyzer.Analyze()

	gen := New(program)
	gen.SetExprTypes(analyzer.ExprTypes())
	gen.SetExprReturnCounts(analyzer.ReturnCounts())
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}

	if !strings.Contains(output, "switch v := pipe_") {
		t.Errorf("expected typed switch over piped temp var, got: %s", output)
	}
	if !strings.Contains(output, "var result string = \"fallback\"") {
		t.Errorf("expected onerr default initialization, got: %s", output)
	}
}

func TestTypedPipedSwitchComputedReturnCodegen(t *testing.T) {
	input := `import "os/exec"

func ExitCodeOrOne(err error) int
    code := err |> switch as exitErr
        when reference exec.ExitError
            return exitErr.ExitCode()
        otherwise
            return 1
    return code
`

	p, err := parser.New(input, "test.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	analyzer := semantic.New(program)
	if semanticErrors := analyzer.Analyze(); len(semanticErrors) > 0 {
		t.Fatalf("semantic errors: %v", semanticErrors)
	}

	gen := New(program)
	gen.SetExprTypes(analyzer.ExprTypes())
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}

	if !strings.Contains(output, "func() int {") {
		t.Errorf("expected typed IIFE 'func() int {', got: %s", output)
	}
	if !strings.Contains(output, "return exitErr.ExitCode()") {
		t.Errorf("expected computed return to be preserved, got: %s", output)
	}
}

func TestOnErrPipeChainFull(t *testing.T) {
	input := `import "stdlib/fetch"
func Run()
    url |> fetch.Get() |> fetch.CheckStatus() onerr panic "failed"
`
	p, err := parser.New(input, "test.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}
	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	analyzer := semantic.New(program)
	analyzer.Analyze()

	gen := New(program)
	gen.SetExprReturnCounts(analyzer.ReturnCounts())
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}

	// url base is non-error — collapsed directly into fetch.Get call.
	if !strings.Contains(output, "pipe_1, err_2 := fetch.Get(url)") {
		t.Errorf("expected collapsed fetch.Get(url), got: \n%s", output)
	}
	if !strings.Contains(output, "pipe_3, err_4 := fetch.CheckStatus(pipe_1)") {
		t.Errorf("expected fetch.CheckStatus to capture err_4, got: \n%s", output)
	}
}

func TestNestedOnErrCodegen(t *testing.T) {
	// An onerr block body that contains another statement with onerr.
	// Both onerr handlers should resolve {error} to their own error variable.
	input := `func readData() (string, error)
    return "data", empty

func writeData(data string) error
    return empty

func Process() error
    data := readData() onerr return
    writeData(data) onerr return
    return empty
`
	p, err := parser.New(input, "test.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}
	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	analyzer := semantic.New(program)
	analyzer.Analyze()

	gen := New(program)
	gen.SetExprReturnCounts(analyzer.ReturnCounts())
	gen.SetExprTypes(analyzer.ExprTypes())
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}

	// First onerr: data, err_1 := readData()
	if !strings.Contains(output, "data, err_1 := readData()") {
		t.Errorf("expected first onerr assignment, got:\n%s", output)
	}
	// Second onerr: err_2 := writeData(data)
	if !strings.Contains(output, "err_2 := writeData(data)") {
		t.Errorf("expected second onerr assignment, got:\n%s", output)
	}
	// Both should have error checks
	if !strings.Contains(output, "if err_1 != nil") {
		t.Errorf("expected first error check, got:\n%s", output)
	}
	if !strings.Contains(output, "if err_2 != nil") {
		t.Errorf("expected second error check, got:\n%s", output)
	}
}

func TestPipeTempVarSkipsUserDefinedNames(t *testing.T) {
	// A user variable named pipe_1 should not collide with generated temps
	input := `import "os"

func Run() (string, error)
    pipe_1 := "hello"
    result := pipe_1 |> os.ReadFile() onerr return
    return result, empty
`
	p, err := parser.New(input, "test.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}
	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	analyzer := semantic.New(program)
	analyzer.Analyze()

	gen := New(program)
	gen.SetExprReturnCounts(analyzer.ReturnCounts())
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}

	// The user declared pipe_1, so the generated temp should skip to pipe_2
	if !strings.Contains(output, "pipe_1 := \"hello\"") {
		t.Errorf("expected user variable pipe_1, got:\n%s", output)
	}
	// The pipe chain should use pipe_2 (not pipe_1) to avoid collision
	if strings.Contains(output, "pipe_1 := pipe_1") {
		t.Errorf("generated temp collided with user variable pipe_1, got:\n%s", output)
	}
}

func TestPipeAwareIterators(t *testing.T) {
	input := `import "stdlib/slice"
func PrintActiveUsers(users list of string)
    for u in users |> slice.Filter((u string) => u != "")
        print(u)
`
	p, err := parser.New(input, "test.kuki")
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

	if !strings.Contains(output, "for _, u := range slice.Filter(users, func(u string) bool { return (u != \"\") }) {") {
		t.Errorf("expected proper iterator pipeline codegen, got: \n%s", output)
	}
}

func TestPipeAwareIteratorsTypedReducerLambda(t *testing.T) {
	input := `import "stdlib/iterator"

func Run() int
    items := list of int{1, 2, 3, 4}
    return items |> iterator.Values() |> iterator.Reduce(0, (acc int, n int) => acc + n)
`

	p, err := parser.New(input, "test.kuki")
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}

	analyzer := semantic.New(program)
	semErrors := analyzer.Analyze()
	if len(semErrors) > 0 {
		t.Fatalf("semantic errors: %v", semErrors)
	}

	gen := New(program)
	gen.SetExprTypes(analyzer.ExprTypes())
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}

	if !strings.Contains(output, "func(acc int, n int) int { return (acc + n) }") {
		t.Fatalf("expected typed reducer lambda to emit an int return type, got: %s", output)
	}
}
