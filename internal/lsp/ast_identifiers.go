package lsp

import "github.com/kukichalang/kukicha/internal/ast"

// walkProgramIdentifiers calls fn for every identifier occurrence in prog,
// including declaration names, expression identifiers, and method/field selectors.
func walkProgramIdentifiers(prog *ast.Program, fn func(name string, pos ast.Position)) {
	for _, decl := range prog.Declarations {
		walkDeclIdentifiers(decl, fn)
	}
}

func walkDeclIdentifiers(decl ast.Declaration, fn func(name string, pos ast.Position)) {
	switch d := decl.(type) {
	case *ast.FunctionDecl:
		fn(d.Name.Value, d.Name.Pos())
		if d.Receiver != nil {
			fn(d.Receiver.Name.Value, d.Receiver.Name.Pos())
		}
		for _, p := range d.Parameters {
			fn(p.Name.Value, p.Name.Pos())
			if p.DefaultValue != nil {
				walkExprIdentifiers(p.DefaultValue, fn)
			}
		}
		if d.Body != nil {
			walkBlockIdentifiers(d.Body, fn)
		}
	case *ast.TypeDecl:
		fn(d.Name.Value, d.Name.Pos())
		for _, f := range d.Fields {
			fn(f.Name.Value, f.Name.Pos())
		}
	case *ast.InterfaceDecl:
		fn(d.Name.Value, d.Name.Pos())
		for _, m := range d.Methods {
			fn(m.Name.Value, m.Name.Pos())
			for _, p := range m.Parameters {
				fn(p.Name.Value, p.Name.Pos())
			}
		}
	case *ast.EnumDecl:
		fn(d.Name.Value, d.Name.Pos())
		for _, c := range d.Cases {
			fn(c.Name.Value, c.Name.Pos())
			if c.Value != nil {
				walkExprIdentifiers(c.Value, fn)
			}
			for _, f := range c.Fields {
				fn(f.Name.Value, f.Name.Pos())
			}
		}
	case *ast.ConstDecl:
		for _, spec := range d.Specs {
			fn(spec.Name.Value, spec.Name.Pos())
			walkExprIdentifiers(spec.Value, fn)
		}
	case *ast.VarDeclStmt:
		for _, name := range d.Names {
			fn(name.Value, name.Pos())
		}
		for _, v := range d.Values {
			walkExprIdentifiers(v, fn)
		}
	}
}

// walkBlockIdentifiers walks all identifiers in a block statement.
func walkBlockIdentifiers(block *ast.BlockStmt, fn func(name string, pos ast.Position)) {
	if block == nil {
		return
	}
	for _, stmt := range block.Statements {
		walkStmtIdentifiers(stmt, fn)
	}
}

func walkStmtIdentifiers(stmt ast.Statement, fn func(name string, pos ast.Position)) {
	if stmt == nil {
		return
	}
	switch s := stmt.(type) {
	case *ast.ExpressionStmt:
		walkExprIdentifiers(s.Expression, fn)
	case *ast.VarDeclStmt:
		for _, name := range s.Names {
			fn(name.Value, name.Pos())
		}
		for _, v := range s.Values {
			walkExprIdentifiers(v, fn)
		}
		if s.OnErr != nil && s.OnErr.Handler != nil {
			walkExprIdentifiers(s.OnErr.Handler, fn)
		}
	case *ast.AssignStmt:
		for _, t := range s.Targets {
			walkExprIdentifiers(t, fn)
		}
		for _, v := range s.Values {
			walkExprIdentifiers(v, fn)
		}
		if s.OnErr != nil && s.OnErr.Handler != nil {
			walkExprIdentifiers(s.OnErr.Handler, fn)
		}
	case *ast.ReturnStmt:
		for _, v := range s.Values {
			walkExprIdentifiers(v, fn)
		}
	case *ast.IncDecStmt:
		walkExprIdentifiers(s.Variable, fn)
	case *ast.IfStmt:
		if s.Init != nil {
			walkStmtIdentifiers(s.Init, fn)
		}
		walkExprIdentifiers(s.Condition, fn)
		walkBlockIdentifiers(s.Consequence, fn)
		if s.Alternative != nil {
			walkStmtIdentifiers(s.Alternative, fn)
		}
	case *ast.ElseStmt:
		walkBlockIdentifiers(s.Body, fn)
	case *ast.SwitchStmt:
		if s.Expression != nil {
			walkExprIdentifiers(s.Expression, fn)
		}
		for _, c := range s.Cases {
			for _, v := range c.Values {
				walkExprIdentifiers(v, fn)
			}
			walkBlockIdentifiers(c.Body, fn)
		}
		if s.Otherwise != nil {
			walkBlockIdentifiers(s.Otherwise.Body, fn)
		}
	case *ast.SelectStmt:
		for _, c := range s.Cases {
			if c.Recv != nil {
				walkExprIdentifiers(c.Recv, fn)
			}
			if c.Send != nil {
				walkStmtIdentifiers(c.Send, fn)
			}
			walkBlockIdentifiers(c.Body, fn)
		}
		if s.Otherwise != nil {
			walkBlockIdentifiers(s.Otherwise.Body, fn)
		}
	case *ast.TypeSwitchStmt:
		walkExprIdentifiers(s.Expression, fn)
		if s.Binding != nil {
			fn(s.Binding.Value, s.Binding.Pos())
		}
		for _, c := range s.Cases {
			walkBlockIdentifiers(c.Body, fn)
		}
		if s.Otherwise != nil {
			walkBlockIdentifiers(s.Otherwise.Body, fn)
		}
	case *ast.ForRangeStmt:
		fn(s.Variable.Value, s.Variable.Pos())
		if s.Index != nil {
			fn(s.Index.Value, s.Index.Pos())
		}
		walkExprIdentifiers(s.Collection, fn)
		walkBlockIdentifiers(s.Body, fn)
	case *ast.ForNumericStmt:
		fn(s.Variable.Value, s.Variable.Pos())
		walkExprIdentifiers(s.Start, fn)
		walkExprIdentifiers(s.End, fn)
		walkBlockIdentifiers(s.Body, fn)
	case *ast.ForConditionStmt:
		walkExprIdentifiers(s.Condition, fn)
		walkBlockIdentifiers(s.Body, fn)
	case *ast.DeferStmt:
		if s.Call != nil {
			walkExprIdentifiers(s.Call, fn)
		}
		if s.Block != nil {
			walkBlockIdentifiers(s.Block, fn)
		}
	case *ast.GoStmt:
		if s.Call != nil {
			walkExprIdentifiers(s.Call, fn)
		}
		if s.Block != nil {
			walkBlockIdentifiers(s.Block, fn)
		}
	case *ast.SendStmt:
		walkExprIdentifiers(s.Value, fn)
		walkExprIdentifiers(s.Channel, fn)
	case *ast.BlockStmt:
		walkBlockIdentifiers(s, fn)
	case *ast.TypeDeclStmt:
		walkDeclIdentifiers(s.Decl, fn)
	// ContinueStmt, BreakStmt: nothing to walk
	}
}

// walkExprIdentifiers recursively visits every identifier in an expression tree.
func walkExprIdentifiers(expr ast.Expression, fn func(name string, pos ast.Position)) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *ast.Identifier:
		fn(e.Value, e.Pos())
	case *ast.CallExpr:
		walkExprIdentifiers(e.Function, fn)
		for _, arg := range e.Arguments {
			walkExprIdentifiers(arg, fn)
		}
		for _, na := range e.NamedArguments {
			walkExprIdentifiers(na.Value, fn)
		}
	case *ast.MethodCallExpr:
		fn(e.Method.Value, e.Method.Pos())
		if e.Object != nil {
			walkExprIdentifiers(e.Object, fn)
		}
		for _, arg := range e.Arguments {
			walkExprIdentifiers(arg, fn)
		}
		for _, na := range e.NamedArguments {
			walkExprIdentifiers(na.Value, fn)
		}
	case *ast.FieldAccessExpr:
		fn(e.Field.Value, e.Field.Pos())
		if e.Object != nil {
			walkExprIdentifiers(e.Object, fn)
		}
	case *ast.BinaryExpr:
		walkExprIdentifiers(e.Left, fn)
		walkExprIdentifiers(e.Right, fn)
	case *ast.UnaryExpr:
		walkExprIdentifiers(e.Right, fn)
	case *ast.IsExpr:
		walkExprIdentifiers(e.Value, fn)
		fn(e.Case.Value, e.Case.Pos())
		if e.Binding != nil {
			fn(e.Binding.Value, e.Binding.Pos())
		}
	case *ast.PipeExpr:
		walkExprIdentifiers(e.Left, fn)
		walkExprIdentifiers(e.Right, fn)
	case *ast.PipedSwitchExpr:
		walkExprIdentifiers(e.Left, fn)
		switch sw := e.Switch.(type) {
		case *ast.SwitchStmt:
			walkStmtIdentifiers(sw, fn)
		case *ast.TypeSwitchStmt:
			walkStmtIdentifiers(sw, fn)
		}
	case *ast.NamedArgument:
		walkExprIdentifiers(e.Value, fn)
	case *ast.IndexExpr:
		walkExprIdentifiers(e.Left, fn)
		walkExprIdentifiers(e.Index, fn)
	case *ast.SliceExpr:
		walkExprIdentifiers(e.Left, fn)
		if e.Start != nil {
			walkExprIdentifiers(e.Start, fn)
		}
		if e.End != nil {
			walkExprIdentifiers(e.End, fn)
		}
	case *ast.StructLiteralExpr:
		for _, fv := range e.Fields {
			fn(fv.Name.Value, fv.Name.Pos())
			walkExprIdentifiers(fv.Value, fn)
		}
	case *ast.ListLiteralExpr:
		for _, elem := range e.Elements {
			walkExprIdentifiers(elem, fn)
		}
	case *ast.MapLiteralExpr:
		for _, pair := range e.Pairs {
			walkExprIdentifiers(pair.Key, fn)
			walkExprIdentifiers(pair.Value, fn)
		}
	case *ast.UntypedCompositeLiteral:
		for _, entry := range e.Entries {
			if entry.Key != nil {
				walkExprIdentifiers(entry.Key, fn)
			}
			walkExprIdentifiers(entry.Value, fn)
		}
	case *ast.FunctionLiteral:
		for _, p := range e.Parameters {
			fn(p.Name.Value, p.Name.Pos())
			if p.DefaultValue != nil {
				walkExprIdentifiers(p.DefaultValue, fn)
			}
		}
		walkBlockIdentifiers(e.Body, fn)
	case *ast.ArrowLambda:
		for _, p := range e.Parameters {
			fn(p.Name.Value, p.Name.Pos())
		}
		if e.Body != nil {
			walkExprIdentifiers(e.Body, fn)
		}
		if e.Block != nil {
			walkBlockIdentifiers(e.Block, fn)
		}
	case *ast.IfExpression:
		walkExprIdentifiers(e.Condition, fn)
		walkExprIdentifiers(e.Then, fn)
		walkExprIdentifiers(e.Else, fn)
	case *ast.BlockExpr:
		walkBlockIdentifiers(e.Body, fn)
	case *ast.AddressOfExpr:
		walkExprIdentifiers(e.Operand, fn)
	case *ast.DerefExpr:
		walkExprIdentifiers(e.Operand, fn)
	case *ast.ReceiveExpr:
		walkExprIdentifiers(e.Channel, fn)
	case *ast.TypeCastExpr:
		walkExprIdentifiers(e.Expression, fn)
	case *ast.TypeAssertionExpr:
		walkExprIdentifiers(e.Expression, fn)
	case *ast.ErrorExpr:
		walkExprIdentifiers(e.Message, fn)
	case *ast.PanicExpr:
		walkExprIdentifiers(e.Message, fn)
	case *ast.MakeExpr:
		for _, arg := range e.Args {
			walkExprIdentifiers(arg, fn)
		}
	case *ast.CloseExpr:
		walkExprIdentifiers(e.Channel, fn)
	case *ast.ReturnExpr:
		for _, v := range e.Values {
			walkExprIdentifiers(v, fn)
		}
	case *ast.OnErrExpr:
		walkExprIdentifiers(e.Expression, fn)
		if e.Default != nil {
			walkExprIdentifiers(e.Default, fn)
		}
	case *ast.StringLiteral:
		if e.Interpolated {
			for _, part := range e.Parts {
				if !part.IsLiteral && part.Expr != nil {
					walkExprIdentifiers(part.Expr, fn)
				}
			}
		}
	// Leaves: IntegerLiteral, FloatLiteral, BooleanLiteral, EmptyExpr,
	// DiscardExpr, RecoverExpr — nothing to walk.
	}
}
