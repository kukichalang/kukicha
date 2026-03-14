package codegen

import (
	"github.com/duber000/kukicha/internal/parser"
	"github.com/duber000/kukicha/internal/semantic"
	"strings"
	"testing"
)

func TestNumericForLoopThrough(t *testing.T) {
	input := `func Test()
    for i from 0 through 10
        x := i
`

	output := generateSource(t, input)

	if !strings.Contains(output, "for i := _iStart; i != _iEnd+_iStep; i += _iStep {") {
		t.Errorf("expected numeric for loop with step variable, got: %s", output)
	}
}

func TestAddressOfExpr(t *testing.T) {
	input := `func GetUserPtr(user User) reference User
    return reference of user
`

	output := generateSource(t, input)

	if !strings.Contains(output, "return &user") {
		t.Errorf("expected '&user', got: %s", output)
	}
}

func TestDerefExpr(t *testing.T) {
	input := `func GetUserValue(userPtr reference User) User
    return dereference userPtr
`

	output := generateSource(t, input)

	if !strings.Contains(output, "return *userPtr") {
		t.Errorf("expected '*userPtr', got: %s", output)
	}
}

func TestDerefAssignment(t *testing.T) {
	input := `func SwapValues(a reference int, b reference int)
    temp := dereference a
    dereference a = dereference b
    dereference b = temp
`

	output := generateSource(t, input)

	if !strings.Contains(output, "*a = *b") {
		t.Errorf("expected '*a = *b', got: %s", output)
	}
}

func TestAddressOfWithFieldAccess(t *testing.T) {
	input := `func ScanField(row Row, field reference string)
    row.Scan(reference of field)
`

	output := generateSource(t, input)

	if !strings.Contains(output, "&field") {
		t.Errorf("expected '&field', got: %s", output)
	}
}

func TestAddressOfCallExprUsesNew(t *testing.T) {
	// Go 1.26 new(expr): function call returns are non-addressable,
	// so `reference of someFunc()` should generate `new(someFunc())`.
	input := `func GetName() string
    return "alice"

func main()
    ptr := reference of GetName()
`

	output := generateSource(t, input)

	if !strings.Contains(output, "new(GetName())") {
		t.Errorf("expected 'new(GetName())', got: %s", output)
	}
}

func TestAddressOfVariableStillUsesAmpersand(t *testing.T) {
	// Plain variables are addressable, so `reference of x` stays as `&x`.
	input := `func Ptr(x int) reference int
    return reference of x
`

	output := generateSource(t, input)

	if !strings.Contains(output, "&x") {
		t.Errorf("expected '&x', got: %s", output)
	}
	if strings.Contains(output, "new(x)") {
		t.Errorf("should not use new() for addressable variable, got: %s", output)
	}
}

func TestGoBlockSyntax(t *testing.T) {
	input := `func main()
    go
        doSomething()
        doSomethingElse()
`

	output := generateSource(t, input)

	if !strings.Contains(output, "go func() {") {
		t.Errorf("expected 'go func() {' in output, got: %s", output)
	}

	if !strings.Contains(output, "}()") {
		t.Errorf("expected '}()' in output, got: %s", output)
	}

	if !strings.Contains(output, "doSomething()") {
		t.Errorf("expected 'doSomething()' in output, got: %s", output)
	}
}

func TestGoCallSyntaxStillWorks(t *testing.T) {
	input := `func main()
    go processItem(item)
`

	output := generateSource(t, input)

	if !strings.Contains(output, "go processItem(item)") {
		t.Errorf("expected 'go processItem(item)' in output, got: %s", output)
	}
}

func TestArrowLambdaTypedExpression(t *testing.T) {
	input := `func main()
    f := (r Repo) => r.Stars > 100
`

	output := generateSource(t, input)

	if !strings.Contains(output, "func(r Repo) bool") {
		t.Errorf("expected 'func(r Repo) bool' in output, got: %s", output)
	}

	if !strings.Contains(output, "return (r.Stars > 100)") {
		t.Errorf("expected 'return (r.Stars > 100)' in output, got: %s", output)
	}
}

func TestArrowLambdaZeroParams(t *testing.T) {
	input := `func main()
    f := () => print("hello")
`

	output := generateSource(t, input)

	if !strings.Contains(output, "func()") {
		t.Errorf("expected 'func()' in output, got: %s", output)
	}
}

func TestArrowLambdaBlockForm(t *testing.T) {
	input := `func main()
    f := (r Repo) =>
        name := r.Name
        return name
`

	output := generateSource(t, input)

	if !strings.Contains(output, "func(r Repo)") {
		t.Errorf("expected 'func(r Repo)' in output, got: %s", output)
	}

	if !strings.Contains(output, "name := r.Name") {
		t.Errorf("expected 'name := r.Name' in output, got: %s", output)
	}

	if !strings.Contains(output, "return name") {
		t.Errorf("expected 'return name' in output, got: %s", output)
	}
}

func TestBitwiseAndEndToEnd(t *testing.T) {
	input := `func main()
    mask := 6 & 3
    flags := 7
    flags &= 3
    _ = mask
    _ = flags
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
	semanticErrors := analyzer.Analyze()
	if len(semanticErrors) > 0 {
		t.Fatalf("semantic errors: %v", semanticErrors)
	}

	gen := New(program)
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}

	if !strings.Contains(output, "mask := (6 & 3)") {
		t.Fatalf("expected bitwise AND expression in output, got:\n%s", output)
	}
	if !strings.Contains(output, "flags &= 3") {
		t.Fatalf("expected bitwise AND assignment in output, got:\n%s", output)
	}
}

func TestSkillComment(t *testing.T) {
	input := `petiole weather

skill WeatherService
    description: "Provides real-time weather data."
    version: "2.1.0"

func GetForecast(city string) string
    return city
`

	output := generateSource(t, input)

	if !strings.Contains(output, "// Skill: WeatherService") {
		t.Errorf("expected '// Skill: WeatherService' in output, got: %s", output)
	}

	if !strings.Contains(output, "// Description: Provides real-time weather data.") {
		t.Errorf("expected '// Description: Provides real-time weather data.' in output, got: %s", output)
	}

	if !strings.Contains(output, "// Version: 2.1.0") {
		t.Errorf("expected '// Version: 2.1.0' in output, got: %s", output)
	}
}

func TestSkillCommentNoFields(t *testing.T) {
	input := `petiole myskill

skill MySkill

func Hello() string
    return "hello"
`

	output := generateSource(t, input)

	if !strings.Contains(output, "// Skill: MySkill") {
		t.Errorf("expected '// Skill: MySkill' in output, got: %s", output)
	}

	// Should NOT contain Description or Version lines since they're empty
	if strings.Contains(output, "// Description:") {
		t.Errorf("did not expect '// Description:' in output when description is empty")
	}
	if strings.Contains(output, "// Version:") {
		t.Errorf("did not expect '// Version:' in output when version is empty")
	}
}

func TestTypeSwitchStatement(t *testing.T) {
	input := `func Handle(event any)
    switch event as e
        when reference a2a.TaskStatusUpdateEvent
            print(e.Status.State)
        when string
            print(e)
        otherwise
            print("unknown")
`

	output := generateSource(t, input)

	if !strings.Contains(output, "switch e := event.(type) {") {
		t.Errorf("expected type switch, got: %s", output)
	}
	if !strings.Contains(output, "case *a2a.TaskStatusUpdateEvent:") {
		t.Errorf("expected pointer type case, got: %s", output)
	}
	if !strings.Contains(output, "case string:") {
		t.Errorf("expected string type case, got: %s", output)
	}
	if !strings.Contains(output, "default:") {
		t.Errorf("expected default branch, got: %s", output)
	}
}

func TestTypeSwitchNoOtherwise(t *testing.T) {
	input := `func Handle(event any)
    switch event as e
        when int
            print(e)
        when string
            print(e)
`

	output := generateSource(t, input)

	if !strings.Contains(output, "switch e := event.(type) {") {
		t.Errorf("expected type switch, got: %s", output)
	}
	if !strings.Contains(output, "case int:") {
		t.Errorf("expected int type case, got: %s", output)
	}
	if !strings.Contains(output, "case string:") {
		t.Errorf("expected string type case, got: %s", output)
	}
	if strings.Contains(output, "default:") {
		t.Errorf("should not have default branch, got: %s", output)
	}
}

func TestSelectStatementCodegen(t *testing.T) {
	input := `func Run(ch channel of string, done channel of string, out channel of string)
    select
        when receive from done
            return
        when msg := receive from ch
            return
        when msg, ok := receive from ch
            return
        when send "ping" to out
            return
        otherwise
            return
`

	output := generateSource(t, input)

	if !strings.Contains(output, "select {") {
		t.Errorf("expected 'select {', got: %s", output)
	}
	if !strings.Contains(output, "case <-done:") {
		t.Errorf("expected bare receive case 'case <-done:', got: %s", output)
	}
	if !strings.Contains(output, "case msg := <-ch:") {
		t.Errorf("expected 1-var receive case 'case msg := <-ch:', got: %s", output)
	}
	if !strings.Contains(output, "case msg, ok := <-ch:") {
		t.Errorf("expected 2-var receive case 'case msg, ok := <-ch:', got: %s", output)
	}
	if !strings.Contains(output, "case out <- \"ping\":") {
		t.Errorf("expected send case 'case out <- \"ping\":', got: %s", output)
	}
	if !strings.Contains(output, "default:") {
		t.Errorf("expected default branch, got: %s", output)
	}
}

func TestNegativeIndexRewriting(t *testing.T) {
	input := `func Main()
    items := list of string{"a", "b", "c"}
    last := items[-1]
`

	output := generateSource(t, input)

	if !strings.Contains(output, "items[len(items)-1]") {
		t.Errorf("expected items[len(items)-1], got: %s", output)
	}
}

func TestNegativeSliceStartRewriting(t *testing.T) {
	input := `func Main()
    items := list of string{"a", "b", "c"}
    tail := items[-3:]
`

	output := generateSource(t, input)

	if !strings.Contains(output, "items[len(items)-3:]") {
		t.Errorf("expected items[len(items)-3:], got: %s", output)
	}
}

func TestNegativeSliceEndRewriting(t *testing.T) {
	input := `func Main()
    items := list of string{"a", "b", "c"}
    init := items[:-1]
`

	output := generateSource(t, input)

	if !strings.Contains(output, "items[:len(items)-1]") {
		t.Errorf("expected items[:len(items)-1], got: %s", output)
	}
}

func TestNegativeSliceBothRewriting(t *testing.T) {
	input := `func Main()
    items := list of string{"a", "b", "c"}
    middle := items[1:-1]
`

	output := generateSource(t, input)

	if !strings.Contains(output, "items[1:len(items)-1]") {
		t.Errorf("expected items[1:len(items)-1], got: %s", output)
	}
}

func TestArrowLambdaMethodCallReturnType(t *testing.T) {
	// Regression test: arrow lambda whose body is a method call on a *regexp.Regexp variable
	// must emit the return type in the Go closure signature so it type-checks as func(T) bool.
	input := `import "regexp"
import "stdlib/slice"

func FilterSemver(tags list of string) list of string
    re := regexp.MustCompile("^v[0-9]+")
    return tags |> slice.Filter((tag string) => re.MatchString(tag))
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
	output, genErr := gen.Generate()
	if genErr != nil {
		t.Fatalf("codegen error: %v", genErr)
	}

	if strings.Contains(output, "func(tag string) {") {
		t.Errorf("arrow lambda emitted without return type; got:\n%s", output)
	}
	if !strings.Contains(output, "func(tag string) bool {") {
		t.Errorf("expected arrow lambda with 'bool' return type; got:\n%s", output)
	}
}
