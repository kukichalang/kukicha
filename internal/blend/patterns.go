package blend

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// --- Operator patterns: &&, ||, ! ---

func (c *collector) collectBinaryOperator(node *ast.BinaryExpr) {
	switch node.Op {
	case token.LAND:
		c.addAtPos("operators", "&& → and", "&&", "and", node.OpPos, node.OpPos+2)
	case token.LOR:
		c.addAtPos("operators", "|| → or", "||", "or", node.OpPos, node.OpPos+2)
	}
}

func (c *collector) collectUnaryNot(node *ast.UnaryExpr) {
	if node.Op != token.NOT {
		return
	}
	start := posToOffset(c.fset, node.Pos())
	// Check if the next char after ! is a space or paren — determines replacement
	endOff := start + 1
	replacement := "not "
	// If followed by '(', we need "not " before the paren
	if endOff < len(c.src) && c.src[endOff] == '(' {
		replacement = "not "
	}
	c.addAtPos("operators", "! → not", "!", replacement, node.Pos(), node.Pos()+1)
}

// --- Comparison patterns: ==, !=, nil ---

func (c *collector) collectComparison(node *ast.BinaryExpr) {
	switch node.Op {
	case token.EQL:
		c.addAtPos("comparisons", "== → equals", "==", "equals", node.OpPos, node.OpPos+2)
	case token.NEQ:
		c.addAtPos("comparisons", "!= → isnt", "!=", "isnt", node.OpPos, node.OpPos+2)
	}
}

func (c *collector) collectNil(node *ast.Ident) {
	if node.Name != "nil" {
		return
	}
	c.addAtPos("comparisons", "nil → empty", "nil", "empty", node.Pos(), node.End())
}

// --- Package pattern ---

func (c *collector) collectPackage(file *ast.File) {
	// Replace "package" keyword with "petiole"
	pkgPos := file.Package
	c.addAtPos("package", "package → petiole", "package", "petiole", pkgPos, pkgPos+7)
}

// --- Type patterns: []T, map[K]V, *T, &x ---

func (c *collector) collectTypes(file *ast.File) {
	// Walk type positions specifically — fields, params, returns, type specs
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		switch node := n.(type) {
		case *ast.Field:
			c.collectTypeExpr(node.Type)
			return false // Don't recurse into field values
		case *ast.TypeSpec:
			c.collectTypeExpr(node.Type)
			return false
		case *ast.CompositeLit:
			c.collectTypeExpr(node.Type)
			return true // Still recurse for nested composites
		case *ast.ValueSpec:
			if node.Type != nil {
				c.collectTypeExpr(node.Type)
			}
			return true // Recurse to find type assertions etc.
		case *ast.TypeAssertExpr:
			if node.Type != nil {
				c.collectTypeExpr(node.Type)
			}
			return true
		}
		return true
	})
}

func (c *collector) collectTypeExpr(expr ast.Expr) {
	if expr == nil {
		return
	}
	switch t := expr.(type) {
	case *ast.ArrayType:
		if t.Len == nil {
			// Slice: []T → list of T
			typeStr := c.sourceText(t.Elt.Pos(), t.Elt.End())
			original := c.sourceText(t.Pos(), t.Elt.End())
			replacement := "list of " + typeStr
			c.addAtPos("types", fmt.Sprintf("[]%s → list of %s", typeStr, typeStr),
				original, replacement, t.Pos(), t.Elt.End())
		}
		// Recurse into element type
		c.collectTypeExpr(t.Elt)

	case *ast.MapType:
		keyStr := c.sourceText(t.Key.Pos(), t.Key.End())
		valStr := c.sourceText(t.Value.Pos(), t.Value.End())
		original := c.sourceText(t.Pos(), t.Value.End())
		replacement := fmt.Sprintf("map of %s to %s", keyStr, valStr)
		c.addAtPos("types", fmt.Sprintf("map[%s]%s → map of %s to %s", keyStr, valStr, keyStr, valStr),
			original, replacement, t.Pos(), t.Value.End())
		// Recurse into key/value types
		c.collectTypeExpr(t.Key)
		c.collectTypeExpr(t.Value)

	case *ast.StarExpr:
		typeStr := c.sourceText(t.X.Pos(), t.X.End())
		original := c.sourceText(t.Pos(), t.X.End())
		replacement := "reference " + typeStr
		c.addAtPos("types", fmt.Sprintf("*%s → reference %s", typeStr, typeStr),
			original, replacement, t.Pos(), t.X.End())
		// Recurse into pointed-to type
		c.collectTypeExpr(t.X)

	case *ast.Ident, *ast.SelectorExpr:
		// Leaf type — no transformation needed

	case *ast.FuncType:
		// Recurse into func params and results
		if t.Params != nil {
			for _, f := range t.Params.List {
				c.collectTypeExpr(f.Type)
			}
		}
		if t.Results != nil {
			for _, f := range t.Results.List {
				c.collectTypeExpr(f.Type)
			}
		}

	case *ast.InterfaceType:
		// Recurse into interface methods
		if t.Methods != nil {
			for _, f := range t.Methods.List {
				c.collectTypeExpr(f.Type)
			}
		}

	case *ast.StructType:
		// Recurse into struct fields
		if t.Fields != nil {
			for _, f := range t.Fields.List {
				c.collectTypeExpr(f.Type)
			}
		}

	case *ast.Ellipsis:
		c.collectTypeExpr(t.Elt)

	case *ast.ChanType:
		c.collectTypeExpr(t.Value)

	case *ast.ParenExpr:
		c.collectTypeExpr(t.X)

	case *ast.IndexExpr:
		// Generic type like List[T] — recurse
		c.collectTypeExpr(t.X)
		c.collectTypeExpr(t.Index)

	case *ast.IndexListExpr:
		c.collectTypeExpr(t.X)
		for _, idx := range t.Indices {
			c.collectTypeExpr(idx)
		}
	}
}

func (c *collector) collectAddressOf(node *ast.UnaryExpr) {
	if node.Op != token.AND {
		return
	}
	exprStr := c.sourceText(node.X.Pos(), node.X.End())
	original := c.sourceText(node.Pos(), node.X.End())
	replacement := "reference of " + exprStr
	c.addAtPos("types", fmt.Sprintf("&%s → reference of %s", exprStr, exprStr),
		original, replacement, node.Pos(), node.X.End())
}

// --- Onerr pattern: if err != nil { return ... } ---

func (c *collector) collectOnerr(node *ast.IfStmt) {
	// Must have no init statement (we need the preceding assignment)
	// and the condition must be `err != nil`
	if node.Init != nil {
		return
	}
	if !isErrNotNil(node.Cond) {
		return
	}
	// Body must be a single return statement
	if len(node.Body.List) != 1 {
		return
	}
	ret, ok := node.Body.List[0].(*ast.ReturnStmt)
	if !ok {
		return
	}

	// Build the onerr replacement description
	var retParts []string
	for _, r := range ret.Results {
		// Skip the error value itself (last return value that's just "err")
		retParts = append(retParts, c.sourceText(r.Pos(), r.End()))
	}

	original := strings.TrimSpace(c.sourceText(node.Pos(), node.End()))

	var replacement string
	if len(retParts) == 0 {
		replacement = "onerr return"
	} else if len(retParts) == 1 && retParts[0] == "err" {
		replacement = "onerr return"
	} else {
		// Drop trailing "err" from return values since onerr implies it
		if len(retParts) > 0 && retParts[len(retParts)-1] == "err" {
			retParts = retParts[:len(retParts)-1]
		}
		if len(retParts) == 0 {
			replacement = "onerr return"
		} else {
			replacement = "onerr return " + strings.Join(retParts, ", ")
		}
	}

	c.addAtPos("onerr",
		fmt.Sprintf("if err != nil { return ... } → %s", replacement),
		original, replacement,
		node.Pos(), node.End())
}

// isErrNotNil returns true if expr is `err != nil`.
func isErrNotNil(expr ast.Expr) bool {
	bin, ok := expr.(*ast.BinaryExpr)
	if !ok || bin.Op != token.NEQ {
		return false
	}
	xIdent, xOk := bin.X.(*ast.Ident)
	yIdent, yOk := bin.Y.(*ast.Ident)
	if !xOk || !yOk {
		return false
	}
	return xIdent.Name == "err" && yIdent.Name == "nil"
}

// sourceText extracts source text between two positions.
func (c *collector) sourceText(start, end token.Pos) string {
	startOff := posToOffset(c.fset, start)
	endOff := posToOffset(c.fset, end)
	if startOff < 0 || endOff > len(c.src) || startOff >= endOff {
		return ""
	}
	return string(c.src[startOff:endOff])
}
