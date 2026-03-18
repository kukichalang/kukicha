package semantic

import (
	"testing"

	"github.com/duber000/kukicha/internal/ast"
)

// findLambdaParam walks the analyzed exprTypes map looking for a param identifier
// with the given name and returns its recorded TypeInfo.
func findLambdaParamType(a *Analyzer, paramName string) *TypeInfo {
	for expr, ti := range a.exprTypes {
		if id, ok := expr.(*ast.Identifier); ok && id.Value == paramName {
			if ti != nil && ti.Kind != TypeKindUnknown {
				return ti
			}
		}
	}
	return nil
}

func TestInferLambdaParam_SliceFilter_String(t *testing.T) {
	src := `petiole main
import "stdlib/slice"

func Foo()
    items := list of string{"a", "b"}
    result := items |> slice.Filter(r => r == "a")
    _ = result
`
	a, errs := analyzeSource(t, src)
	if len(errs) > 0 {
		t.Fatalf("unexpected semantic errors: %v", errs)
	}

	ti := findLambdaParamType(a, "r")
	if ti == nil {
		t.Fatal("expected type inferred for lambda param 'r', got nil")
	}
	if ti.Kind != TypeKindString {
		t.Errorf("expected TypeKindString for 'r', got %v", ti.Kind)
	}
}

func TestInferLambdaParam_SliceFilter_Named(t *testing.T) {
	src := `petiole main
import "stdlib/slice"

type Repo
    stars int

func Foo()
    repos := list of Repo{}
    result := repos |> slice.Filter(r => r.stars > 100)
    _ = result
`
	a, errs := analyzeSource(t, src)
	if len(errs) > 0 {
		t.Fatalf("unexpected semantic errors: %v", errs)
	}

	ti := findLambdaParamType(a, "r")
	if ti == nil {
		t.Fatal("expected type inferred for lambda param 'r', got nil")
	}
	if ti.Kind != TypeKindNamed || ti.Name != "Repo" {
		t.Errorf("expected TypeKindNamed 'Repo' for 'r', got %v %q", ti.Kind, ti.Name)
	}
}

func TestInferLambdaParam_SortByKey(t *testing.T) {
	src := `petiole main
import "stdlib/sort"

type Entry
    name string

func Foo()
    entries := list of Entry{}
    sorted := entries |> sort.ByKey(e => e.name)
    _ = sorted
`
	a, errs := analyzeSource(t, src)
	if len(errs) > 0 {
		t.Fatalf("unexpected semantic errors: %v", errs)
	}

	ti := findLambdaParamType(a, "e")
	if ti == nil {
		t.Fatal("expected type inferred for lambda param 'e', got nil")
	}
	if ti.Kind != TypeKindNamed || ti.Name != "Entry" {
		t.Errorf("expected TypeKindNamed 'Entry' for 'e', got %v %q", ti.Kind, ti.Name)
	}
}

func TestInferLambdaParam_SortBy_TwoParams(t *testing.T) {
	src := `petiole main
import "stdlib/sort"

type Item
    score int

func Foo()
    items := list of Item{}
    sorted := sort.By(items, (a, b) => a.score < b.score)
    _ = sorted
`
	a, errs := analyzeSource(t, src)
	if len(errs) > 0 {
		t.Fatalf("unexpected semantic errors: %v", errs)
	}

	tiA := findLambdaParamType(a, "a")
	tiB := findLambdaParamType(a, "b")
	if tiA == nil || tiB == nil {
		t.Fatal("expected types inferred for lambda params 'a' and 'b'")
	}
	if tiA.Kind != TypeKindNamed || tiA.Name != "Item" {
		t.Errorf("expected TypeKindNamed 'Item' for 'a', got %v %q", tiA.Kind, tiA.Name)
	}
	if tiB.Kind != TypeKindNamed || tiB.Name != "Item" {
		t.Errorf("expected TypeKindNamed 'Item' for 'b', got %v %q", tiB.Kind, tiB.Name)
	}
}

func TestInferLambdaParam_UserDefined(t *testing.T) {
	src := `petiole main

func doWork(action func(string))
    action("hello")

func Foo()
    doWork(s => print(s))
`
	a, errs := analyzeSource(t, src)
	if len(errs) > 0 {
		t.Fatalf("unexpected semantic errors: %v", errs)
	}

	ti := findLambdaParamType(a, "s")
	if ti == nil {
		t.Fatal("expected type inferred for lambda param 's', got nil")
	}
	if ti.Kind != TypeKindString {
		t.Errorf("expected TypeKindString for 's', got %v", ti.Kind)
	}
}

func TestInferLambdaParam_CliCommandAction(t *testing.T) {
	src := `petiole main
import "stdlib/cli"

func doList(a cli.Args)
    _ = a

func Foo()
    app := cli.New("test")
    _ = app |> cli.CommandAction("list", a => doList(a))
`
	a, errs := analyzeSource(t, src)
	if len(errs) > 0 {
		t.Fatalf("unexpected semantic errors: %v", errs)
	}

	ti := findLambdaParamType(a, "a")
	if ti == nil {
		t.Fatal("expected type inferred for lambda param 'a', got nil")
	}
	// Registry stores the fully qualified name "cli.Args" for cross-package types.
	if ti.Kind != TypeKindNamed || ti.Name != "cli.Args" {
		t.Errorf("expected TypeKindNamed 'cli.Args' for 'a', got %v %q", ti.Kind, ti.Name)
	}
}

func TestInferLambdaParam_TypedParamUnchanged(t *testing.T) {
	src := `petiole main
import "stdlib/slice"

func Foo()
    items := list of string{"a", "b"}
    result := items |> slice.Filter((r string) => r == "a")
    _ = result
`
	a, errs := analyzeSource(t, src)
	if len(errs) > 0 {
		t.Fatalf("unexpected semantic errors: %v", errs)
	}
	_ = a // typed param — inference not needed, test just verifies no errors
}

func TestInferLambdaParam_DirectCall_SliceFilter(t *testing.T) {
	src := `petiole main
import "stdlib/slice"

func Foo()
    items := list of string{"a", "b"}
    result := slice.Filter(items, r => r == "a")
    _ = result
`
	a, errs := analyzeSource(t, src)
	if len(errs) > 0 {
		t.Fatalf("unexpected semantic errors: %v", errs)
	}

	ti := findLambdaParamType(a, "r")
	if ti == nil {
		t.Fatal("expected type inferred for lambda param 'r', got nil")
	}
	if ti.Kind != TypeKindString {
		t.Errorf("expected TypeKindString for 'r', got %v", ti.Kind)
	}
}
