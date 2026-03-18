package codegen

import (
	"strings"
	"testing"

	kukiparser "github.com/duber000/kukicha/internal/parser"
	"github.com/duber000/kukicha/internal/semantic"
)

// pipelineLambda runs lex→parse→semantic→codegen for lambda inference tests.
func pipelineLambda(t *testing.T, source string) string {
	t.Helper()
	p, err := kukiparser.New(source, "test.kuki")
	if err != nil {
		t.Fatalf("parser init error: %v", err)
	}
	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("parse errors: %v", parseErrors)
	}
	analyzer := semantic.NewWithFile(program, "test.kuki")
	semanticErrors := analyzer.Analyze()
	if len(semanticErrors) > 0 {
		t.Fatalf("semantic errors: %v", semanticErrors)
	}
	gen := New(program)
	gen.SetSourceFile("test.kuki")
	gen.SetExprReturnCounts(analyzer.ReturnCounts())
	gen.SetExprTypes(analyzer.ExprTypes())
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}
	return output
}

func TestUntypedLambda_SliceFilter_String(t *testing.T) {
	src := `petiole main
import "stdlib/slice"

func Foo()
    items := list of string{"a", "b"}
    result := items |> slice.Filter(r => r == "a")
    _ = result
`
	out := pipelineLambda(t, src)
	if !strings.Contains(out, "func(r string)") {
		t.Errorf("expected 'func(r string)' in output, got:\n%s", out)
	}
}

func TestUntypedLambda_SliceFilter_Named(t *testing.T) {
	src := `petiole main
import "stdlib/slice"

type Repo
    stars int

func Foo()
    repos := list of Repo{}
    result := repos |> slice.Filter(r => r.stars > 100)
    _ = result
`
	out := pipelineLambda(t, src)
	if !strings.Contains(out, "func(r Repo)") {
		t.Errorf("expected 'func(r Repo)' in output, got:\n%s", out)
	}
}

func TestUntypedLambda_SortByKey(t *testing.T) {
	src := `petiole main
import "stdlib/sort"

type Entry
    name string

func Foo()
    entries := list of Entry{}
    sorted := entries |> sort.ByKey(e => e.name)
    _ = sorted
`
	out := pipelineLambda(t, src)
	if !strings.Contains(out, "func(e Entry)") {
		t.Errorf("expected 'func(e Entry)' in output, got:\n%s", out)
	}
}

func TestUntypedLambda_SortBy_TwoParams(t *testing.T) {
	src := `petiole main
import "stdlib/sort"

type Item
    score int

func Foo()
    items := list of Item{}
    sorted := sort.By(items, (a, b) => a.score < b.score)
    _ = sorted
`
	out := pipelineLambda(t, src)
	if !strings.Contains(out, "func(a Item, b Item)") {
		t.Errorf("expected 'func(a Item, b Item)' in output, got:\n%s", out)
	}
}

func TestUntypedLambda_UserDefined(t *testing.T) {
	src := `petiole main

func doWork(action func(string))
    action("hello")

func Foo()
    doWork(s => print(s))
`
	out := pipelineLambda(t, src)
	if !strings.Contains(out, "func(s string)") {
		t.Errorf("expected 'func(s string)' in output, got:\n%s", out)
	}
}

func TestUntypedLambda_CliCommandAction(t *testing.T) {
	src := `petiole main
import "stdlib/cli"

func doList(a cli.Args)
    _ = a

func Foo()
    app := cli.New("test")
    _ = app |> cli.CommandAction("list", a => doList(a))
`
	out := pipelineLambda(t, src)
	if !strings.Contains(out, "func(a cli.Args)") {
		t.Errorf("expected 'func(a cli.Args)' in output, got:\n%s", out)
	}
}

func TestUntypedLambda_TypedParam_Unchanged(t *testing.T) {
	src := `petiole main
import "stdlib/slice"

func Foo()
    items := list of string{"a", "b"}
    result := items |> slice.Filter((r string) => r == "a")
    _ = result
`
	out := pipelineLambda(t, src)
	// explicit type annotation must still be emitted correctly
	if !strings.Contains(out, "func(r string)") {
		t.Errorf("expected 'func(r string)' in output, got:\n%s", out)
	}
}

func TestUntypedLambda_DirectCall_SliceFilter(t *testing.T) {
	src := `petiole main
import "stdlib/slice"

func Foo()
    items := list of string{"a", "b"}
    result := slice.Filter(items, r => r == "a")
    _ = result
`
	out := pipelineLambda(t, src)
	if !strings.Contains(out, "func(r string)") {
		t.Errorf("expected 'func(r string)' in output, got:\n%s", out)
	}
}
