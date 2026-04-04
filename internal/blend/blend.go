package blend

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
)

// BlendFile parses a Go source file and returns suggestions for Kukicha
// transformations based on the active patterns.
func BlendFile(filename string, src []byte, patterns PatternSet) ([]Suggestion, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	c := &collector{
		fset:     fset,
		src:      src,
		filename: filename,
		patterns: patterns,
	}

	// Package → petiole
	if patterns.Package {
		c.collectPackage(file)
	}

	// Walk AST for patterns
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		switch node := n.(type) {
		case *ast.BinaryExpr:
			if patterns.Operators {
				c.collectBinaryOperator(node)
			}
			if patterns.Comparisons {
				c.collectComparison(node)
			}
		case *ast.UnaryExpr:
			if patterns.Operators {
				c.collectUnaryNot(node)
			}
			if patterns.Types {
				c.collectAddressOf(node)
			}
		case *ast.Ident:
			if patterns.Comparisons {
				c.collectNil(node)
			}
		case *ast.IfStmt:
			if patterns.Onerr {
				c.collectOnerr(node)
			}
		}
		return true
	})

	// Type patterns need context-aware walking
	if patterns.Types {
		c.collectTypes(file)
	}

	// Sort by offset ascending for display (descending applied in Apply)
	sort.Slice(c.suggestions, func(i, j int) bool {
		return c.suggestions[i].Start < c.suggestions[j].Start
	})

	return c.suggestions, nil
}

type collector struct {
	fset        *token.FileSet
	src         []byte
	filename    string
	patterns    PatternSet
	suggestions []Suggestion
}

func (c *collector) addAtPos(pattern, message, original, replacement string, start, end token.Pos) {
	startOff := posToOffset(c.fset, start)
	endOff := posToOffset(c.fset, end)
	c.suggestions = append(c.suggestions, Suggestion{
		Pattern:     pattern,
		Message:     message,
		File:        c.filename,
		Line:        c.fset.Position(start).Line,
		Col:         c.fset.Position(start).Column,
		EndLine:     c.fset.Position(end).Line,
		EndCol:      c.fset.Position(end).Column,
		Start:       startOff,
		End:         endOff,
		Original:    original,
		Replacement: replacement,
	})
}
