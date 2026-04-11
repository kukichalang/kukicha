package parser

import (
	"testing"

	"github.com/kukichalang/kukicha/internal/ast"
)

// --- Type position tests (parseTypeAnnotation) ---

func TestBracketListType(t *testing.T) {
	prog := mustParseProgram(t, "func f(items []string)\n    x := items\n")
	fn := prog.Declarations[0].(*ast.FunctionDecl)
	param := fn.Parameters[0]
	lt, ok := param.Type.(*ast.ListType)
	if !ok {
		t.Fatalf("expected ListType, got %T", param.Type)
	}
	pt, ok := lt.ElementType.(*ast.PrimitiveType)
	if !ok {
		t.Fatalf("expected PrimitiveType element, got %T", lt.ElementType)
	}
	if pt.Name != "string" {
		t.Errorf("expected element type 'string', got %q", pt.Name)
	}
}

func TestBracketMapType(t *testing.T) {
	prog := mustParseProgram(t, "func f(m map[string]int)\n    x := m\n")
	fn := prog.Declarations[0].(*ast.FunctionDecl)
	param := fn.Parameters[0]
	mt, ok := param.Type.(*ast.MapType)
	if !ok {
		t.Fatalf("expected MapType, got %T", param.Type)
	}
	keyPt, ok := mt.KeyType.(*ast.PrimitiveType)
	if !ok {
		t.Fatalf("expected PrimitiveType key, got %T", mt.KeyType)
	}
	if keyPt.Name != "string" {
		t.Errorf("expected key type 'string', got %q", keyPt.Name)
	}
	valPt, ok := mt.ValueType.(*ast.PrimitiveType)
	if !ok {
		t.Fatalf("expected PrimitiveType value, got %T", mt.ValueType)
	}
	if valPt.Name != "int" {
		t.Errorf("expected value type 'int', got %q", valPt.Name)
	}
}

func TestBracketNestedTypes(t *testing.T) {
	// []map[string][]int  ≡  list of map of string to list of int
	prog := mustParseProgram(t, "func f(x []map[string][]int)\n    y := x\n")
	fn := prog.Declarations[0].(*ast.FunctionDecl)
	param := fn.Parameters[0]

	outerList, ok := param.Type.(*ast.ListType)
	if !ok {
		t.Fatalf("expected outer ListType, got %T", param.Type)
	}
	innerMap, ok := outerList.ElementType.(*ast.MapType)
	if !ok {
		t.Fatalf("expected inner MapType, got %T", outerList.ElementType)
	}
	keyPt, ok := innerMap.KeyType.(*ast.PrimitiveType)
	if !ok || keyPt.Name != "string" {
		t.Fatalf("expected key type 'string', got %T %v", innerMap.KeyType, innerMap.KeyType)
	}
	innerList, ok := innerMap.ValueType.(*ast.ListType)
	if !ok {
		t.Fatalf("expected value ListType, got %T", innerMap.ValueType)
	}
	valPt, ok := innerList.ElementType.(*ast.PrimitiveType)
	if !ok || valPt.Name != "int" {
		t.Fatalf("expected innermost element type 'int', got %T %v", innerList.ElementType, innerList.ElementType)
	}
}

func TestBracketReturnType(t *testing.T) {
	prog := mustParseProgram(t, "func f() []string\n    return [\"a\"]\n")
	fn := prog.Declarations[0].(*ast.FunctionDecl)
	if len(fn.Returns) != 1 {
		t.Fatalf("expected 1 return type, got %d", len(fn.Returns))
	}
	lt, ok := fn.Returns[0].(*ast.ListType)
	if !ok {
		t.Fatalf("expected ListType return, got %T", fn.Returns[0])
	}
	pt := lt.ElementType.(*ast.PrimitiveType)
	if pt.Name != "string" {
		t.Errorf("expected element type 'string', got %q", pt.Name)
	}
}

func TestBracketMapReturnType(t *testing.T) {
	prog := mustParseProgram(t, "func f() map[string]int\n    return map[string]int{\"a\": 1}\n")
	fn := prog.Declarations[0].(*ast.FunctionDecl)
	if len(fn.Returns) != 1 {
		t.Fatalf("expected 1 return type, got %d", len(fn.Returns))
	}
	mt, ok := fn.Returns[0].(*ast.MapType)
	if !ok {
		t.Fatalf("expected MapType return, got %T", fn.Returns[0])
	}
	if mt.KeyType.(*ast.PrimitiveType).Name != "string" {
		t.Error("expected key type 'string'")
	}
	if mt.ValueType.(*ast.PrimitiveType).Name != "int" {
		t.Error("expected value type 'int'")
	}
}

// --- Expression position tests ---

func TestBracketListLiteralExpr(t *testing.T) {
	prog := mustParseProgram(t, "func main()\n    x := []int{1, 2, 3}\n")
	fn := prog.Declarations[0].(*ast.FunctionDecl)
	decl := fn.Body.Statements[0].(*ast.VarDeclStmt)
	lit, ok := decl.Values[0].(*ast.ListLiteralExpr)
	if !ok {
		t.Fatalf("expected ListLiteralExpr, got %T", decl.Values[0])
	}
	if lit.Type == nil {
		t.Fatal("expected explicit element type")
	}
	pt := lit.Type.(*ast.PrimitiveType)
	if pt.Name != "int" {
		t.Errorf("expected element type 'int', got %q", pt.Name)
	}
	if len(lit.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(lit.Elements))
	}
}

func TestBracketMapLiteralExpr(t *testing.T) {
	prog := mustParseProgram(t, "func main()\n    x := map[string]int{\"a\": 1, \"b\": 2}\n")
	fn := prog.Declarations[0].(*ast.FunctionDecl)
	decl := fn.Body.Statements[0].(*ast.VarDeclStmt)
	lit, ok := decl.Values[0].(*ast.MapLiteralExpr)
	if !ok {
		t.Fatalf("expected MapLiteralExpr, got %T", decl.Values[0])
	}
	if lit.KeyType.(*ast.PrimitiveType).Name != "string" {
		t.Error("expected key type 'string'")
	}
	if lit.ValType.(*ast.PrimitiveType).Name != "int" {
		t.Error("expected value type 'int'")
	}
	if len(lit.Pairs) != 2 {
		t.Errorf("expected 2 pairs, got %d", len(lit.Pairs))
	}
}

func TestBracketListEmptyShorthand(t *testing.T) {
	prog := mustParseProgram(t, "func main()\n    x := []string\n")
	fn := prog.Declarations[0].(*ast.FunctionDecl)
	decl := fn.Body.Statements[0].(*ast.VarDeclStmt)
	empty, ok := decl.Values[0].(*ast.EmptyExpr)
	if !ok {
		t.Fatalf("expected EmptyExpr, got %T", decl.Values[0])
	}
	lt, ok := empty.Type.(*ast.ListType)
	if !ok {
		t.Fatalf("expected ListType in EmptyExpr, got %T", empty.Type)
	}
	if lt.ElementType.(*ast.PrimitiveType).Name != "string" {
		t.Error("expected element type 'string'")
	}
}

func TestBracketMapEmptyShorthand(t *testing.T) {
	prog := mustParseProgram(t, "func main()\n    x := map[string]int\n")
	fn := prog.Declarations[0].(*ast.FunctionDecl)
	decl := fn.Body.Statements[0].(*ast.VarDeclStmt)
	empty, ok := decl.Values[0].(*ast.EmptyExpr)
	if !ok {
		t.Fatalf("expected EmptyExpr, got %T", decl.Values[0])
	}
	mt, ok := empty.Type.(*ast.MapType)
	if !ok {
		t.Fatalf("expected MapType in EmptyExpr, got %T", empty.Type)
	}
	if mt.KeyType.(*ast.PrimitiveType).Name != "string" {
		t.Error("expected key type 'string'")
	}
	if mt.ValueType.(*ast.PrimitiveType).Name != "int" {
		t.Error("expected value type 'int'")
	}
}

// Ensure untyped list literals [1, 2, 3] still work
func TestUntypedListLiteralStillWorks(t *testing.T) {
	prog := mustParseProgram(t, "func main()\n    x := [1, 2, 3]\n")
	fn := prog.Declarations[0].(*ast.FunctionDecl)
	decl := fn.Body.Statements[0].(*ast.VarDeclStmt)
	lit, ok := decl.Values[0].(*ast.ListLiteralExpr)
	if !ok {
		t.Fatalf("expected ListLiteralExpr, got %T", decl.Values[0])
	}
	if lit.Type != nil {
		t.Error("expected nil type for untyped list literal")
	}
	if len(lit.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(lit.Elements))
	}
}

// Ensure English-style types still work alongside bracket types
func TestEnglishAndBracketTypesCoexist(t *testing.T) {
	input := `func f(a list of string, b []int, c map of string to int, d map[string]bool)
    x := a
`
	prog := mustParseProgram(t, input)
	fn := prog.Declarations[0].(*ast.FunctionDecl)
	if len(fn.Parameters) != 4 {
		t.Fatalf("expected 4 params, got %d", len(fn.Parameters))
	}

	// All should be the correct types regardless of syntax used
	if _, ok := fn.Parameters[0].Type.(*ast.ListType); !ok {
		t.Error("param 'a' should be ListType")
	}
	if _, ok := fn.Parameters[1].Type.(*ast.ListType); !ok {
		t.Error("param 'b' should be ListType")
	}
	if _, ok := fn.Parameters[2].Type.(*ast.MapType); !ok {
		t.Error("param 'c' should be MapType")
	}
	if _, ok := fn.Parameters[3].Type.(*ast.MapType); !ok {
		t.Error("param 'd' should be MapType")
	}
}

// --- Feature 2: Untyped map literal tests ---

func TestUntypedMapLiteral(t *testing.T) {
	prog := mustParseProgram(t, "func main()\n    x := {\"a\": 1, \"b\": 2}\n")
	fn := prog.Declarations[0].(*ast.FunctionDecl)
	decl := fn.Body.Statements[0].(*ast.VarDeclStmt)
	lit, ok := decl.Values[0].(*ast.UntypedCompositeLiteral)
	if !ok {
		t.Fatalf("expected UntypedCompositeLiteral, got %T", decl.Values[0])
	}
	if !lit.IsKeyed {
		t.Error("expected IsKeyed=true for {key: val} literal")
	}
	if len(lit.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(lit.Entries))
	}
}

func TestUntypedMapLiteralEmpty(t *testing.T) {
	prog := mustParseProgram(t, "func main()\n    x := {}\n")
	fn := prog.Declarations[0].(*ast.FunctionDecl)
	decl := fn.Body.Statements[0].(*ast.VarDeclStmt)
	lit, ok := decl.Values[0].(*ast.UntypedCompositeLiteral)
	if !ok {
		t.Fatalf("expected UntypedCompositeLiteral, got %T", decl.Values[0])
	}
	if len(lit.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(lit.Entries))
	}
}

func TestUntypedPositionalLiteral(t *testing.T) {
	prog := mustParseProgram(t, "func main()\n    x := {1, 2, 3}\n")
	fn := prog.Declarations[0].(*ast.FunctionDecl)
	decl := fn.Body.Statements[0].(*ast.VarDeclStmt)
	lit, ok := decl.Values[0].(*ast.UntypedCompositeLiteral)
	if !ok {
		t.Fatalf("expected UntypedCompositeLiteral, got %T", decl.Values[0])
	}
	if lit.IsKeyed {
		t.Error("expected IsKeyed=false for positional literal")
	}
	if len(lit.Entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(lit.Entries))
	}
	for _, entry := range lit.Entries {
		if entry.Key != nil {
			t.Error("positional entries should have nil Key")
		}
	}
}

func TestUntypedNestedLiteral(t *testing.T) {
	prog := mustParseProgram(t, "func main()\n    x := {\"a\": {1, 2}}\n")
	fn := prog.Declarations[0].(*ast.FunctionDecl)
	decl := fn.Body.Statements[0].(*ast.VarDeclStmt)
	lit, ok := decl.Values[0].(*ast.UntypedCompositeLiteral)
	if !ok {
		t.Fatalf("expected outer UntypedCompositeLiteral, got %T", decl.Values[0])
	}
	if len(lit.Entries) != 1 {
		t.Fatalf("expected 1 outer entry, got %d", len(lit.Entries))
	}
	inner, ok := lit.Entries[0].Value.(*ast.UntypedCompositeLiteral)
	if !ok {
		t.Fatalf("expected inner UntypedCompositeLiteral, got %T", lit.Entries[0].Value)
	}
	if inner.IsKeyed {
		t.Error("inner literal should be positional")
	}
	if len(inner.Entries) != 2 {
		t.Errorf("expected 2 inner entries, got %d", len(inner.Entries))
	}
}
