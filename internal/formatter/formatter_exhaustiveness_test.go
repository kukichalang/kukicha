package formatter

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExpressionExhaustiveness ensures that every AST type implementing
// exprNode() has a corresponding case in Printer.exprToString().
// When a new expression node is added to the AST, this test will fail
// until the formatter handles it.
func TestExpressionExhaustiveness(t *testing.T) {
	astTypes := collectMarkerTypes(t, "exprNode")
	printerCases := collectSwitchCases(t, "exprToString")

	for _, typ := range astTypes {
		if !printerCases[typ] {
			t.Errorf("ast.%s implements exprNode() but has no case in exprToString()", typ)
		}
	}
}

// TestStatementExhaustiveness ensures that every AST type implementing
// stmtNode() has a corresponding case in Printer.printStatement().
// BlockStmt is excluded because it is used internally as a container,
// not as a standalone statement to format.
func TestStatementExhaustiveness(t *testing.T) {
	skip := map[string]bool{
		"BlockStmt": true, // used internally as a container
		"ElseStmt":  true, // handled inline within printIfStmt
	}

	astTypes := collectMarkerTypes(t, "stmtNode")
	printerCases := collectSwitchCases(t, "printStatement")

	for _, typ := range astTypes {
		if skip[typ] {
			continue
		}
		if !printerCases[typ] {
			t.Errorf("ast.%s implements stmtNode() but has no case in printStatement()", typ)
		}
	}
}

// TestDeclarationExhaustiveness ensures that every AST type implementing
// declNode() has a corresponding case in Printer.printDeclaration().
// PetioleDecl, SkillDecl, ImportDecl, and VarDeclStmt are handled
// separately in Print() / printStatement(), not in printDeclaration().
func TestDeclarationExhaustiveness(t *testing.T) {
	skip := map[string]bool{
		"PetioleDecl": true, // handled in Print()
		"SkillDecl":   true, // handled in Print()
		"ImportDecl":  true, // handled in printImport()
		"VarDeclStmt": true, // handled in both printDeclaration and printStatement
	}

	astTypes := collectMarkerTypes(t, "declNode")
	printerCases := collectSwitchCases(t, "printDeclaration")

	for _, typ := range astTypes {
		if skip[typ] {
			continue
		}
		if !printerCases[typ] {
			t.Errorf("ast.%s implements declNode() but has no case in printDeclaration()", typ)
		}
	}
}

// collectMarkerTypes parses ast package source and returns all type names that
// have a method with the given name (e.g. "exprNode", "stmtNode", "declNode").
func collectMarkerTypes(t *testing.T, markerMethod string) []string {
	t.Helper()

	fset := token.NewFileSet()
	dir := "../ast"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read ast directory: %v", err)
	}

	var types []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		file, err := goparser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			t.Fatalf("failed to parse %s: %v", entry.Name(), err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name.Name != markerMethod {
				continue
			}
			recv := fn.Recv.List[0].Type
			if star, ok := recv.(*ast.StarExpr); ok {
				if ident, ok := star.X.(*ast.Ident); ok {
					types = append(types, ident.Name)
				}
			}
		}
	}
	if len(types) == 0 {
		t.Fatalf("found no types with marker method %s() in ast package", markerMethod)
	}
	return types
}

// collectSwitchCases parses formatter package source and returns all
// "case *ast.Foo:" types found in type switches within the named method.
func collectSwitchCases(t *testing.T, methodName string) map[string]bool {
	t.Helper()

	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("failed to read formatter directory: %v", err)
	}

	cases := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := goparser.ParseFile(fset, filepath.Join(".", entry.Name()), nil, 0)
		if err != nil {
			t.Fatalf("failed to parse %s: %v", entry.Name(), err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != methodName {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				cc, ok := n.(*ast.CaseClause)
				if !ok {
					return true
				}
				for _, expr := range cc.List {
					star, ok := expr.(*ast.StarExpr)
					if !ok {
						continue
					}
					sel, ok := star.X.(*ast.SelectorExpr)
					if !ok {
						continue
					}
					ident, ok := sel.X.(*ast.Ident)
					if !ok || ident.Name != "ast" {
						continue
					}
					cases[sel.Sel.Name] = true
				}
				return true
			})
		}
	}
	return cases
}

// TestExprToStringNoSilentEmpty verifies that exprToString does not silently
// return an empty string for any known expression type. This is a documentation
// test — the real enforcement is the exhaustiveness tests above.
func TestExprToStringNoSilentEmpty(t *testing.T) {
	// If this test exists, it means we've audited the default case.
	// The default case should produce a visible marker, not "".
	src := `func main()
    print("hello")
`
	opts := DefaultOptions()
	result, err := Format(src, "test.kuki", opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}
	if strings.Contains(result, "/* unhandled:") {
		t.Errorf("formatted output contains unhandled AST node marker:\n%s", result)
	}
}
