package lsp

import (
	"github.com/kukichalang/kukicha/internal/ast"
	"github.com/sourcegraph/go-lsp"
)

// findUntypedLiteralAtPosition walks the AST to find an UntypedCompositeLiteral
// that contains the given cursor position. Returns nil if no such literal exists.
func findUntypedLiteralAtPosition(program *ast.Program, pos lsp.Position) *ast.UntypedCompositeLiteral {
	if program == nil {
		return nil
	}

	// AST positions are 1-indexed; LSP positions are 0-indexed.
	cursorLine := int(pos.Line) + 1
	cursorCol := int(pos.Character) + 1

	for _, decl := range program.Declarations {
		fn, ok := decl.(*ast.FunctionDecl)
		if !ok || fn.Body == nil {
			continue
		}
		if !blockContainsLine(fn.Body, cursorLine) {
			continue
		}
		if found := findUCLInBlock(fn.Body, cursorLine, cursorCol); found != nil {
			return found
		}
	}
	return nil
}

// findUCLInBlock recursively searches a block for an UntypedCompositeLiteral
// that contains the cursor.
func findUCLInBlock(block *ast.BlockStmt, line, col int) *ast.UntypedCompositeLiteral {
	if block == nil {
		return nil
	}
	for _, stmt := range block.Statements {
		if found := findUCLInStatement(stmt, line, col); found != nil {
			return found
		}
	}
	return nil
}

func findUCLInStatement(stmt ast.Statement, line, col int) *ast.UntypedCompositeLiteral {
	switch s := stmt.(type) {
	case *ast.ReturnStmt:
		for _, val := range s.Values {
			if found := findUCLInExpression(val, line, col); found != nil {
				return found
			}
		}
	case *ast.VarDeclStmt:
		for _, val := range s.Values {
			if found := findUCLInExpression(val, line, col); found != nil {
				return found
			}
		}
	case *ast.AssignStmt:
		for _, val := range s.Values {
			if found := findUCLInExpression(val, line, col); found != nil {
				return found
			}
		}
	case *ast.ExpressionStmt:
		if found := findUCLInExpression(s.Expression, line, col); found != nil {
			return found
		}
	case *ast.IfStmt:
		if s.Consequence != nil {
			if found := findUCLInBlock(s.Consequence, line, col); found != nil {
				return found
			}
		}
		if s.Alternative != nil {
			switch alt := s.Alternative.(type) {
			case *ast.ElseStmt:
				if found := findUCLInBlock(alt.Body, line, col); found != nil {
					return found
				}
			case *ast.IfStmt:
				if found := findUCLInStatement(alt, line, col); found != nil {
					return found
				}
			}
		}
	case *ast.ForRangeStmt:
		if found := findUCLInBlock(s.Body, line, col); found != nil {
			return found
		}
	case *ast.ForNumericStmt:
		if found := findUCLInBlock(s.Body, line, col); found != nil {
			return found
		}
	case *ast.ForConditionStmt:
		if found := findUCLInBlock(s.Body, line, col); found != nil {
			return found
		}
	case *ast.SwitchStmt:
		for _, c := range s.Cases {
			if found := findUCLInBlock(c.Body, line, col); found != nil {
				return found
			}
		}
		if s.Otherwise != nil {
			if found := findUCLInBlock(s.Otherwise.Body, line, col); found != nil {
				return found
			}
		}
	case *ast.DeferStmt:
		if found := findUCLInExpression(s.Call, line, col); found != nil {
			return found
		}
	}
	return nil
}

func findUCLInExpression(expr ast.Expression, line, col int) *ast.UntypedCompositeLiteral {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *ast.UntypedCompositeLiteral:
		if uclContainsCursor(e, line, col) {
			return e
		}
	case *ast.CallExpr:
		for _, arg := range e.Arguments {
			if found := findUCLInExpression(arg, line, col); found != nil {
				return found
			}
		}
	case *ast.MethodCallExpr:
		for _, arg := range e.Arguments {
			if found := findUCLInExpression(arg, line, col); found != nil {
				return found
			}
		}
	case *ast.BinaryExpr:
		if found := findUCLInExpression(e.Left, line, col); found != nil {
			return found
		}
		if found := findUCLInExpression(e.Right, line, col); found != nil {
			return found
		}
	case *ast.PipeExpr:
		if found := findUCLInExpression(e.Left, line, col); found != nil {
			return found
		}
		if found := findUCLInExpression(e.Right, line, col); found != nil {
			return found
		}
	}
	return nil
}

// uclContainsCursor checks whether the cursor position (1-indexed) falls inside
// the given UntypedCompositeLiteral. For single-line literals, both line and
// column must be within the braces. For multi-line literals, any line between
// the opening brace and the last entry is considered inside.
func uclContainsCursor(ucl *ast.UntypedCompositeLiteral, line, col int) bool {
	startLine := ucl.Token.Line
	startCol := ucl.Token.Column

	if len(ucl.Entries) == 0 {
		// Empty literal: `{}` — cursor must be on same line after `{`
		return line == startLine && col > startCol
	}

	// Find the last entry's line
	lastEntryLine := startLine
	for _, entry := range ucl.Entries {
		entryLine := entry.Value.Pos().Line
		if entryLine > lastEntryLine {
			lastEntryLine = entryLine
		}
		if entry.Key != nil {
			keyLine := entry.Key.Pos().Line
			if keyLine > lastEntryLine {
				lastEntryLine = keyLine
			}
		}
	}

	if ucl.WasMultiline {
		// Multi-line: cursor must be between start line and closing brace line
		// (closing brace is typically on lastEntryLine or lastEntryLine+1)
		return line >= startLine && line <= lastEntryLine+1
	}

	// Single-line: same line, after `{`
	return line == startLine && col > startCol
}

// structFieldsForLiteral looks up the struct fields for an UntypedCompositeLiteral
// with a resolved named type. Returns field name → formatted type string.
func structFieldsForLiteral(ucl *ast.UntypedCompositeLiteral, doc *Document) map[string]string {
	if ucl.ResolvedType == nil || doc.SymbolTable == nil {
		return nil
	}

	// Get the type name from ResolvedType
	var typeName string
	switch t := ucl.ResolvedType.(type) {
	case *ast.NamedType:
		typeName = t.Name
	case *ast.PrimitiveType:
		typeName = t.Name
	default:
		return nil
	}

	sym := doc.SymbolTable.Resolve(typeName)
	if sym == nil || sym.Type == nil || sym.Type.Fields == nil {
		return nil
	}

	fields := make(map[string]string, len(sym.Type.Fields))
	for name, typeInfo := range sym.Type.Fields {
		fields[name] = typeInfo.String()
	}
	return fields
}

// usedFieldNames returns the set of field names already used in the literal.
func usedFieldNames(ucl *ast.UntypedCompositeLiteral) map[string]bool {
	used := make(map[string]bool)
	for _, entry := range ucl.Entries {
		if ident, ok := entry.Key.(*ast.Identifier); ok {
			used[ident.Value] = true
		}
	}
	return used
}
