package codegen

import (
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

	singleOut := generateSource(t, singleLine)
	multiOut := generateSource(t, multiLine)

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

	output := generateAnalyzedSource(t, input)

	mustNotContainPattern(t, output, `val, _ :=`,
		"expected no intermediate error discards in pipe onerr lowering")

	// Each error-returning step gets its own temp var + error var.
	// Patterns use \d+ to decouple from exact counter values.
	mustContainPattern(t, output,
		`pipe_\d+, err_\d+ := fetch\.Get\(url\)`,
		"expected explicit fetch.Get error capture")
	mustContainPattern(t, output,
		`pipe_\d+, err_\d+ := fetch\.CheckStatus\(pipe_\d+\)`,
		"expected explicit fetch.CheckStatus error capture")
	// Last step assigns directly to the target variable instead of a temp
	mustContainPattern(t, output,
		`repos, err_\d+ := fetch\.Json\(pipe_\d+, \[\]Repo\{\}\)`,
		"expected last pipe step to assign directly to 'repos'")
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

	output := generateAnalyzedSource(t, input)

	// GetInput() is a non-error step — collapsed into the Parse call.
	// Parse() is the last (and only error-returning) step, assigns directly to 'result'.
	mustContainPattern(t, output,
		`result, err_\d+ := Parse\(GetInput\(\)\)`,
		"expected collapsed pipe chain 'result, err_N := Parse(GetInput())'")

}

func TestOnErrPipeChainKnownExternalMultiReturn(t *testing.T) {
	input := `import "os"

func Run(path string) (list of os.DirEntry, error)
    entries := path |> os.ReadDir() onerr return
    return entries, empty
`

	output := generateAnalyzedSource(t, input)

	// path is a non-error base — collapsed directly into os.ReadDir call.
	// os.ReadDir is the last step, assigns directly to 'entries'.
	mustContainPattern(t, output,
		`entries, err_\d+ := os\.ReadDir\(path\)`,
		"expected collapsed pipe 'entries, err_N := os.ReadDir(path)'")
	mustContainPattern(t, output,
		`return \[\]os\.DirEntry\{\}, err_\d+`,
		"expected onerr return to propagate os.ReadDir error")
}

func TestOnErrPipeChainErrorOnlyReturn(t *testing.T) {
	input := `import "os"

func Write(data list of byte, path string) error
    data |> os.WriteFile(path, _, 0644) onerr return
    return empty
`

	output := generateAnalyzedSource(t, input)

	// data is a non-error base — collapsed directly into os.WriteFile call.
	// os.WriteFile returns only error — should generate error check, not value assignment.
	mustContainPattern(t, output,
		`err_\d+ := os\.WriteFile\(path, data, 0644\)`,
		"expected collapsed os.WriteFile with data directly")
	mustContainPattern(t, output,
		`if err_\d+ != nil \{`,
		"expected error check for os.WriteFile")
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

	output := generateAnalyzedSource(t, input)

	// data base is non-error — collapsed into marshalPretty call.
	// marshalPretty returns 2 values — split into value + error.
	mustContainPattern(t, output,
		`pipe_\d+, err_\d+ := marshalPretty\(data\)`,
		"expected collapsed marshalPretty(data)")
	// os.WriteFile returns only error — should check error directly.
	// Verify the pipe var from marshalPretty is passed to WriteFile.
	mustContainPattern(t, output,
		`err_\d+ := os\.WriteFile\(path, pipe_\d+, 0644\)`,
		"expected os.WriteFile error assigned to err var")
	mustContainPattern(t, output,
		`if err_\d+ != nil \{`,
		"expected error check for os.WriteFile")
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

	output := generateAnalyzedSource(t, input)

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

	output := generateAnalyzedSource(t, input)

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

	output := generateAnalyzedSource(t, input)

	if !strings.Contains(output, "func() int {") {
		t.Errorf("expected typed IIFE 'func() int {', got: %s", output)
	}
	if !strings.Contains(output, "return exitErr.ExitCode()") {
		t.Errorf("expected computed return to be preserved, got: %s", output)
	}
}

func TestOnErrPipeChainFull(t *testing.T) {
	input := `import "stdlib/fetch"
func Run(url string)
    url |> fetch.Get() |> fetch.CheckStatus() onerr panic "failed"
`
	output := generateAnalyzedSource(t, input)

	// url base is non-error — collapsed directly into fetch.Get call.
	mustContainPattern(t, output,
		`pipe_\d+, err_\d+ := fetch\.Get\(url\)`,
		"expected collapsed fetch.Get(url)")
	mustContainPattern(t, output,
		`pipe_\d+, err_\d+ := fetch\.CheckStatus\(pipe_\d+\)`,
		"expected fetch.CheckStatus to capture error")

	// Step context comments should identify each error-returning step.
	mustContainPattern(t, output,
		`// pipe step 1: fetch\.Get\(\.\.\.\)`,
		"expected step 1 comment for fetch.Get")
	mustContainPattern(t, output,
		`// pipe step 2: fetch\.CheckStatus\(\.\.\.\)`,
		"expected step 2 comment for fetch.CheckStatus")
}

func TestOnErrPipeValueFallbackAssignsTarget(t *testing.T) {
	// Regression: `b := "data" |> Risky() onerr "default"` used to emit an
	// empty handler body because the target name wasn't threaded into the
	// generateOnErrHandler names slice when the final step wrote directly
	// to the target variable.
	input := `func Risky(input string) (string, error)
    return "", error "oops"

func Run() string
    b := "data" |> Risky() onerr "default"
    return b
`
	output := generateAnalyzedSource(t, input)

	mustContainPattern(t, output,
		`b, err_\d+ := Risky\("data"\)`,
		"expected target-name optimization: b declared inline")
	if !strings.Contains(output, `b = "default"`) {
		t.Errorf("expected onerr value fallback to assign to target b, got:\n%s", output)
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
	output := generateAnalyzedSource(t, input)

	// First onerr: data, err_N := readData()
	mustContainPattern(t, output,
		`data, err_\d+ := readData\(\)`,
		"expected first onerr assignment")
	// Second onerr: err_N := writeData(data)
	mustContainPattern(t, output,
		`err_\d+ := writeData\(data\)`,
		"expected second onerr assignment")
	// Both should have error checks
	mustContainPattern(t, output,
		`if err_\d+ != nil`,
		"expected error checks for both onerr handlers")
}

func TestPipeTempVarSkipsUserDefinedNames(t *testing.T) {
	// A user variable named pipe_1 should not collide with generated temps
	input := `import "os"

func Run() (list of byte, error)
    pipe_1 := "hello"
    result := pipe_1 |> os.ReadFile() onerr return
    return result, empty
`
	output := generateAnalyzedSource(t, input)

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
	output := generateSource(t, input)

	if !strings.Contains(output, "for _, u := range slice.Filter(users, func(u string) bool { return (u != \"\") }) {") {
		t.Errorf("expected proper iterator pipeline codegen, got: \n%s", output)
	}
}

func TestPipedStdlibDefaultParamFill(t *testing.T) {
	// Piped calls to stdlib functions with default parameters must fill in
	// the defaults just like direct calls do.
	input := `import "stdlib/string"

func Run(s string) string
    return s |> string.PadRight(10)
`
	output := generateSource(t, input)

	if !strings.Contains(output, `PadRight(s, 10, " ")`) {
		t.Errorf("expected piped stdlib call to fill default param, got:\n%s", output)
	}
}

func TestPipedStdlibDefaultParamFillAliased(t *testing.T) {
	// Same as above, but with a user-defined package alias. The codegen
	// must resolve the alias back to the canonical stdlib name before
	// looking up default values.
	input := `import "stdlib/string" as strpkg

func Run(s string) string
    return s |> strpkg.PadRight(10)
`
	output := generateSource(t, input)

	if !strings.Contains(output, `PadRight(s, 10, " ")`) {
		t.Errorf("expected piped aliased stdlib call to fill default param, got:\n%s", output)
	}
}

func TestBarePkgFuncPipeTarget(t *testing.T) {
	// A bare pkg.Func (no parens) as the pipe terminator should call the
	// function with the piped value as its single argument — just like a
	// bare identifier pipe target.
	input := `import "strings"

func Run(s string) string
    return s |> strings.ToUpper
`
	output := generateSource(t, input)

	if !strings.Contains(output, "strings.ToUpper(s)") {
		t.Errorf("expected bare pkg.Func pipe target to be called with piped value, got:\n%s", output)
	}
}

func TestPipeAwareIteratorsTypedReducerLambda(t *testing.T) {
	input := `import "stdlib/iterator"

func Run() int
    items := list of int{1, 2, 3, 4}
    return items |> iterator.Values() |> iterator.Reduce(0, (acc int, n int) => acc + n)
`

	output := generateAnalyzedSource(t, input)

	if !strings.Contains(output, "func(acc int, n int) int { return (acc + n) }") {
		t.Fatalf("expected typed reducer lambda to emit an int return type, got: %s", output)
	}
}
