package lsp

import (
	"testing"

	"github.com/kukichalang/kukicha/internal/ast"
	"github.com/kukichalang/kukicha/internal/parser"
)

func TestWalkProgramIdentifiers_BasicFunction(t *testing.T) {
	src := `func add(x int, y int) int
    return x + y

func main()
    result := add(1, 2)
`
	p, _ := parser.New(src, "test.kuki")
	prog, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	counts := make(map[string]int)
	walkProgramIdentifiers(prog, func(name string, _ ast.Position) {
		counts[name]++
	})

	// "add" appears at: declaration (line 1), call site (line 5)
	if counts["add"] < 2 {
		t.Errorf("expected at least 2 occurrences of 'add', got %d", counts["add"])
	}
	// "x" appears at: param decl, return expr
	if counts["x"] < 2 {
		t.Errorf("expected at least 2 occurrences of 'x', got %d", counts["x"])
	}
	// "result" appears at var decl
	if counts["result"] < 1 {
		t.Errorf("expected at least 1 occurrence of 'result', got %d", counts["result"])
	}
}

func TestWalkProgramIdentifiers_FieldAccess(t *testing.T) {
	src := `type User
    name string

func getName(u User) string
    return u.name
`
	p, _ := parser.New(src, "test.kuki")
	prog, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	counts := make(map[string]int)
	walkProgramIdentifiers(prog, func(name string, _ ast.Position) {
		counts[name]++
	})

	// "name" appears: field decl in type, field access in return expression
	if counts["name"] < 2 {
		t.Errorf("expected at least 2 occurrences of 'name', got %d", counts["name"])
	}
	// "u" appears: param decl + use in u.name
	if counts["u"] < 2 {
		t.Errorf("expected at least 2 occurrences of 'u', got %d", counts["u"])
	}
}

func TestWalkProgramIdentifiers_ForRange(t *testing.T) {
	src := `func sumSlice(nums list of int) int
    total := 0
    for n in nums
        total = total + n
    return total
`
	p, _ := parser.New(src, "test.kuki")
	prog, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	counts := make(map[string]int)
	walkProgramIdentifiers(prog, func(name string, _ ast.Position) {
		counts[name]++
	})

	if counts["n"] < 1 {
		t.Errorf("expected loop variable 'n' to be visited, got %d", counts["n"])
	}
	if counts["total"] < 2 {
		t.Errorf("expected 'total' to appear multiple times, got %d", counts["total"])
	}
}

func TestWalkProgramIdentifiers_EnumAndConst(t *testing.T) {
	src := `enum Status
    Active = 1
    Inactive = 2

const MaxRetries = 5
`
	p, _ := parser.New(src, "test.kuki")
	prog, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	counts := make(map[string]int)
	walkProgramIdentifiers(prog, func(name string, _ ast.Position) {
		counts[name]++
	})

	if counts["Status"] < 1 {
		t.Errorf("expected 'Status' to be visited")
	}
	if counts["Active"] < 1 {
		t.Errorf("expected 'Active' to be visited")
	}
	if counts["MaxRetries"] < 1 {
		t.Errorf("expected 'MaxRetries' to be visited")
	}
}

func TestWalkProgramIdentifiers_IfStmt(t *testing.T) {
	src := `func check(x int) bool
    if x > 0
        return true
    return false
`
	p, _ := parser.New(src, "test.kuki")
	prog, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	counts := make(map[string]int)
	walkProgramIdentifiers(prog, func(name string, _ ast.Position) {
		counts[name]++
	})

	// "x" appears: param decl + condition
	if counts["x"] < 2 {
		t.Errorf("expected at least 2 occurrences of 'x', got %d", counts["x"])
	}
}
