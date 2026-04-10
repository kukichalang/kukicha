package formatter

import (
	"fmt"
	"strings"

	"github.com/kukichalang/kukicha/internal/ast"
)

// maxLineWidth is the target line width. Pipe chains that exceed this
// (accounting for current indentation) are broken across multiple lines.
const maxLineWidth = 100

// Printer prints an AST as formatted Kukicha source code
type Printer struct {
	output      strings.Builder
	indentLevel int
	indentStr   string // 4 spaces
}

// NewPrinter creates a new printer
func NewPrinter() *Printer {
	return &Printer{
		indentStr: "    ", // 4 spaces
	}
}

// Print prints the program and returns the formatted source code
func (p *Printer) Print(program *ast.Program) string {
	p.output.Reset()
	p.indentLevel = 0

	// Print petiole declaration if present
	if program.PetioleDecl != nil {
		p.writeLine(fmt.Sprintf("petiole %s", program.PetioleDecl.Name.Value))
		p.writeLine("")
	}

	// Print skill declaration if present
	if program.SkillDecl != nil {
		p.writeLine(fmt.Sprintf("skill %s", program.SkillDecl.Name.Value))
		p.writeLine("")
	}

	// Print imports
	for _, imp := range program.Imports {
		p.printImport(imp)
	}

	if len(program.Imports) > 0 {
		p.writeLine("")
	}

	// Print declarations with blank lines between them
	for i, decl := range program.Declarations {
		if i > 0 {
			p.writeLine("")
		}
		p.printDeclaration(decl)
	}

	return p.output.String()
}

func (p *Printer) printImport(imp *ast.ImportDecl) {
	if imp.Alias != nil {
		p.writeLine(fmt.Sprintf("import \"%s\" as %s", imp.Path.Value, imp.Alias.Value))
	} else {
		p.writeLine(fmt.Sprintf("import \"%s\"", imp.Path.Value))
	}
}

func (p *Printer) printDeclaration(decl ast.Declaration) {
	switch d := decl.(type) {
	case *ast.TypeDecl:
		p.printTypeDecl(d)
	case *ast.InterfaceDecl:
		p.printInterfaceDecl(d)
	case *ast.FunctionDecl:
		p.printFunctionDecl(d)
	case *ast.ConstDecl:
		p.printConstDecl(d)
	case *ast.EnumDecl:
		p.printEnumDecl(d)
	case *ast.VarDeclStmt:
		p.printTopLevelVarDecl(d)
	}
}

func (p *Printer) printEnumDecl(decl *ast.EnumDecl) {
	p.writeLine(fmt.Sprintf("enum %s", decl.Name.Value))
	p.indentLevel++
	for _, c := range decl.Cases {
		if c.Value != nil {
			// Value case: Name = literal
			p.writeLine(fmt.Sprintf("%s = %s", c.Name.Value, p.exprToString(c.Value)))
		} else if len(c.Fields) > 0 {
			// Variant case with fields
			p.writeLine(c.Name.Value)
			p.indentLevel++
			for _, f := range c.Fields {
				p.writeLine(fmt.Sprintf("%s %s", f.Name.Value, p.typeAnnotationToString(f.Type)))
			}
			p.indentLevel--
		} else {
			// Unit variant
			p.writeLine(c.Name.Value)
		}
	}
	p.indentLevel--
}

func (p *Printer) printConstDecl(decl *ast.ConstDecl) {
	if len(decl.Specs) == 1 {
		spec := decl.Specs[0]
		p.writeLine(fmt.Sprintf("const %s = %s", spec.Name.Value, p.exprToString(spec.Value)))
		return
	}
	p.writeLine("const")
	p.indentLevel++
	for _, spec := range decl.Specs {
		p.writeLine(fmt.Sprintf("%s = %s", spec.Name.Value, p.exprToString(spec.Value)))
	}
	p.indentLevel--
}

func (p *Printer) printTypeDecl(decl *ast.TypeDecl) {
	// Type alias (e.g., type Handler func(string))
	if decl.AliasType != nil {
		p.writeLine(fmt.Sprintf("type %s %s", decl.Name.Value, p.typeAnnotationToString(decl.AliasType)))
		return
	}

	p.writeLine(fmt.Sprintf("type %s", decl.Name.Value))
	p.indentLevel++

	for _, field := range decl.Fields {
		fieldType := p.typeAnnotationToString(field.Type)
		line := fmt.Sprintf("%s %s", field.Name.Value, fieldType)
		if field.Tag != "" {
			line += " " + formatStructTag(field.Tag)
		}
		p.writeLine(line)
	}

	p.indentLevel--
}

func (p *Printer) printInterfaceDecl(decl *ast.InterfaceDecl) {
	p.writeLine(fmt.Sprintf("interface %s", decl.Name.Value))
	p.indentLevel++

	for _, method := range decl.Methods {
		params := p.parametersToString(method.Parameters)
		returns := p.returnTypesToString(method.Returns)

		if returns != "" {
			p.writeLine(fmt.Sprintf("%s(%s) %s", method.Name.Value, params, returns))
		} else {
			p.writeLine(fmt.Sprintf("%s(%s)", method.Name.Value, params))
		}
	}

	p.indentLevel--
}

func (p *Printer) printDirectives(directives []ast.Directive) {
	for _, d := range directives {
		if len(d.Args) > 0 {
			// Re-quote args that contain spaces or were originally quoted
			quotedArgs := make([]string, len(d.Args))
			for i, arg := range d.Args {
				quotedArgs[i] = fmt.Sprintf("%q", arg)
			}
			p.writeLine(fmt.Sprintf("# kuki:%s %s", d.Name, strings.Join(quotedArgs, " ")))
		} else {
			p.writeLine(fmt.Sprintf("# kuki:%s", d.Name))
		}
	}
}

func (p *Printer) printFunctionDecl(decl *ast.FunctionDecl) {
	p.printDirectives(decl.Directives)
	params := p.parametersToString(decl.Parameters)
	returns := p.returnTypesToString(decl.Returns)

	var line string
	if decl.Receiver != nil {
		// Method declaration
		receiverType := p.typeAnnotationToString(decl.Receiver.Type)
		if params != "" {
			line = fmt.Sprintf("func %s on %s %s(%s)", decl.Name.Value, decl.Receiver.Name.Value, receiverType, params)
		} else {
			line = fmt.Sprintf("func %s on %s %s", decl.Name.Value, decl.Receiver.Name.Value, receiverType)
		}
	} else {
		// Regular function
		line = fmt.Sprintf("func %s(%s)", decl.Name.Value, params)
	}

	if returns != "" {
		line += " " + returns
	}
	p.writeLine(line)

	// Print body
	if decl.Body != nil {
		p.indentLevel++
		p.printBlock(decl.Body)
		p.indentLevel--
	}
}

func (p *Printer) parametersToString(params []*ast.Parameter) string {
	if len(params) == 0 {
		return ""
	}

	parts := make([]string, len(params))
	for i, param := range params {
		paramType := p.typeAnnotationToString(param.Type)
		if param.Variadic {
			parts[i] = fmt.Sprintf("many %s %s", param.Name.Value, paramType)
		} else {
			parts[i] = fmt.Sprintf("%s %s", param.Name.Value, paramType)
		}
		if param.DefaultValue != nil {
			parts[i] += " = " + p.exprToString(param.DefaultValue)
		}
	}

	return strings.Join(parts, ", ")
}

func (p *Printer) returnTypesToString(returns []ast.TypeAnnotation) string {
	if len(returns) == 0 {
		return ""
	}

	if len(returns) == 1 {
		return p.typeAnnotationToString(returns[0])
	}

	parts := make([]string, len(returns))
	for i, ret := range returns {
		parts[i] = p.typeAnnotationToString(ret)
	}

	return "(" + strings.Join(parts, ", ") + ")"
}

func (p *Printer) typeAnnotationToString(typeAnn ast.TypeAnnotation) string {
	if typeAnn == nil {
		return ""
	}

	switch t := typeAnn.(type) {
	case *ast.PrimitiveType:
		return t.Name
	case *ast.NamedType:
		return t.Name
	case *ast.ReferenceType:
		return "reference " + p.typeAnnotationToString(t.ElementType)
	case *ast.ListType:
		return "list of " + p.typeAnnotationToString(t.ElementType)
	case *ast.MapType:
		keyType := p.typeAnnotationToString(t.KeyType)
		valueType := p.typeAnnotationToString(t.ValueType)
		return fmt.Sprintf("map of %s to %s", keyType, valueType)
	case *ast.ChannelType:
		return "channel of " + p.typeAnnotationToString(t.ElementType)
	case *ast.FunctionType:
		var paramTypes []string
		for _, param := range t.Parameters {
			paramTypes = append(paramTypes, p.typeAnnotationToString(param))
		}
		result := "func(" + strings.Join(paramTypes, ", ") + ")"
		if len(t.Returns) == 1 {
			result += " " + p.typeAnnotationToString(t.Returns[0])
		} else if len(t.Returns) > 1 {
			var returnTypes []string
			for _, ret := range t.Returns {
				returnTypes = append(returnTypes, p.typeAnnotationToString(ret))
			}
			result += " (" + strings.Join(returnTypes, ", ") + ")"
		}
		return result
	default:
		return "any"
	}
}

func (p *Printer) printBlock(block *ast.BlockStmt) {
	prevEndLine := 0
	for _, stmt := range block.Statements {
		stmtLine := stmtStartLine(stmt)
		// Preserve blank lines between statements when positions are available
		if prevEndLine > 0 && stmtLine > prevEndLine+1 {
			p.writeLine("")
		}
		p.printStatement(stmt)
		prevEndLine = p.estimateEndLine(stmt)
	}
}

// estimateEndLine returns the last line a statement occupies.
// For simple statements this is just Pos().Line; for compound statements
// we check the body to find the deepest line. For statements containing
// expressions that span multiple lines (e.g. multiline pipe chains),
// we walk the expression tree to find the deepest source line.
func (p *Printer) estimateEndLine(stmt ast.Statement) int {
	line := stmt.Pos().Line
	switch s := stmt.(type) {
	case *ast.IfStmt:
		if s.Alternative != nil {
			return p.estimateEndLine(s.Alternative)
		}
		if s.Consequence != nil && len(s.Consequence.Statements) > 0 {
			return p.estimateEndLine(s.Consequence.Statements[len(s.Consequence.Statements)-1])
		}
	case *ast.ElseStmt:
		if s.Body != nil && len(s.Body.Statements) > 0 {
			return p.estimateEndLine(s.Body.Statements[len(s.Body.Statements)-1])
		}
	case *ast.ForRangeStmt:
		if s.Body != nil && len(s.Body.Statements) > 0 {
			return p.estimateEndLine(s.Body.Statements[len(s.Body.Statements)-1])
		}
	case *ast.ForNumericStmt:
		if s.Body != nil && len(s.Body.Statements) > 0 {
			return p.estimateEndLine(s.Body.Statements[len(s.Body.Statements)-1])
		}
	case *ast.ForConditionStmt:
		if s.Body != nil && len(s.Body.Statements) > 0 {
			return p.estimateEndLine(s.Body.Statements[len(s.Body.Statements)-1])
		}
	case *ast.SwitchStmt:
		if s.Otherwise != nil && s.Otherwise.Body != nil && len(s.Otherwise.Body.Statements) > 0 {
			return p.estimateEndLine(s.Otherwise.Body.Statements[len(s.Otherwise.Body.Statements)-1])
		}
		if len(s.Cases) > 0 {
			lastCase := s.Cases[len(s.Cases)-1]
			if lastCase.Body != nil && len(lastCase.Body.Statements) > 0 {
				return p.estimateEndLine(lastCase.Body.Statements[len(lastCase.Body.Statements)-1])
			}
		}
	case *ast.VarDeclStmt:
		for _, v := range s.Values {
			if vl := maxExprLine(v); vl > line {
				line = vl
			}
		}
	case *ast.AssignStmt:
		for _, v := range s.Values {
			if vl := maxExprLine(v); vl > line {
				line = vl
			}
		}
	case *ast.ExpressionStmt:
		if el := maxExprLine(s.Expression); el > line {
			line = el
		}
	case *ast.ReturnStmt:
		for _, v := range s.Values {
			if vl := maxExprLine(v); vl > line {
				line = vl
			}
		}
	}
	return line
}

// stmtStartLine returns the line on which a statement visually starts.
// For most statements this is just stmt.Pos().Line, but CallExpr's token
// is the closing ')' which may be on a later line than the function name
// for multiline calls. This helper walks into the leftmost expression to
// find the earliest line so blank-line preservation works correctly after
// the formatter has introduced line wraps.
func stmtStartLine(stmt ast.Statement) int {
	line := stmt.Pos().Line
	var exprs []ast.Expression
	switch s := stmt.(type) {
	case *ast.ExpressionStmt:
		exprs = append(exprs, s.Expression)
	case *ast.VarDeclStmt:
		exprs = append(exprs, s.Values...)
	case *ast.AssignStmt:
		exprs = append(exprs, s.Targets...)
		exprs = append(exprs, s.Values...)
	case *ast.ReturnStmt:
		exprs = append(exprs, s.Values...)
	}
	for _, e := range exprs {
		if l := minExprLine(e); l > 0 && l < line {
			line = l
		}
	}
	return line
}

// minExprLine returns the minimum (earliest) source line touched by the
// expression tree. Used for CallExpr/MethodCallExpr where Token points to
// the closing ')' rather than the function name.
func minExprLine(expr ast.Expression) int {
	if expr == nil {
		return 0
	}
	line := expr.Pos().Line
	switch e := expr.(type) {
	case *ast.CallExpr:
		if e.Function != nil {
			if fl := minExprLine(e.Function); fl > 0 && fl < line {
				line = fl
			}
		}
	case *ast.MethodCallExpr:
		if e.Object != nil {
			if ol := minExprLine(e.Object); ol > 0 && ol < line {
				line = ol
			}
		}
	case *ast.FieldAccessExpr:
		if e.Object != nil {
			if ol := minExprLine(e.Object); ol > 0 && ol < line {
				line = ol
			}
		}
	case *ast.PipeExpr:
		if e.Left != nil {
			if ll := minExprLine(e.Left); ll > 0 && ll < line {
				line = ll
			}
		}
	}
	return line
}

// maxExprLine walks an expression tree and returns the maximum source
// line number found. This accounts for pipe chains and other expressions
// that may span multiple source lines after formatting.
func maxExprLine(expr ast.Expression) int {
	if expr == nil {
		return 0
	}
	line := expr.Pos().Line
	switch e := expr.(type) {
	case *ast.PipeExpr:
		pe := e
		for {
			if pe.Token.Line > line {
				line = pe.Token.Line
			}
			if rl := maxExprLine(pe.Right); rl > line {
				line = rl
			}
			left, ok := pe.Left.(*ast.PipeExpr)
			if !ok {
				if ll := maxExprLine(pe.Left); ll > line {
					line = ll
				}
				break
			}
			pe = left
		}
	case *ast.CallExpr:
		// Call's closing `)` is merged with the last arg's closing `}`/`]`/`)`
		// when that arg is multi-line (see callExprToString), so no +1 is
		// needed for the call itself.
		for _, arg := range e.Arguments {
			if al := maxExprLine(arg); al > line {
				line = al
			}
		}
	case *ast.MethodCallExpr:
		if ol := maxExprLine(e.Object); ol > line {
			line = ol
		}
		for _, arg := range e.Arguments {
			if al := maxExprLine(arg); al > line {
				line = al
			}
		}
	case *ast.StructLiteralExpr:
		openLine := line
		for _, field := range e.Fields {
			if field.Value != nil {
				if vl := maxExprLine(field.Value); vl > line {
					line = vl
				}
			}
		}
		if line > openLine {
			line++
		}
	case *ast.ListLiteralExpr:
		openLine := line
		for _, elem := range e.Elements {
			if el := maxExprLine(elem); el > line {
				line = el
			}
		}
		if line > openLine {
			line++
		}
	case *ast.MapLiteralExpr:
		openLine := line
		for _, pair := range e.Pairs {
			if pair.Value != nil {
				if vl := maxExprLine(pair.Value); vl > line {
					line = vl
				}
			}
			if pair.Key != nil {
				if kl := maxExprLine(pair.Key); kl > line {
					line = kl
				}
			}
		}
		if line > openLine {
			line++
		}
	}
	return line
}

func (p *Printer) printStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.VarDeclStmt:
		p.printVarDeclStmt(s)
	case *ast.AssignStmt:
		p.printAssignStmt(s)
	case *ast.ReturnStmt:
		p.printReturnStmt(s)
	case *ast.IfStmt:
		p.printIfStmt(s)
	case *ast.SwitchStmt:
		p.printSwitchStmt(s)
	case *ast.TypeSwitchStmt:
		p.printTypeSwitchStmt(s)
	case *ast.ForRangeStmt:
		p.printForRangeStmt(s)
	case *ast.ForNumericStmt:
		p.printForNumericStmt(s)
	case *ast.ForConditionStmt:
		p.printForConditionStmt(s)
	case *ast.DeferStmt:
		if s.Block != nil {
			p.writeLine("defer")
			p.indentLevel++
			p.printBlock(s.Block)
			p.indentLevel--
		} else {
			p.writeLine("defer " + p.exprToString(s.Call))
		}
	case *ast.GoStmt:
		if s.Block != nil {
			p.writeLine("go")
			p.indentLevel++
			for _, stmt := range s.Block.Statements {
				p.printStatement(stmt)
			}
			p.indentLevel--
		} else {
			p.writeLine("go " + p.exprToString(s.Call))
		}
	case *ast.SendStmt:
		channel := p.exprToString(s.Channel)
		value := p.exprToString(s.Value)
		p.writeLine(fmt.Sprintf("send %s to %s", value, channel))
	case *ast.BreakStmt:
		p.writeLine("break")
	case *ast.ContinueStmt:
		p.writeLine("continue")
	case *ast.IncDecStmt:
		p.writeLine(p.exprToString(s.Variable) + s.Operator)
	case *ast.ExpressionStmt:
		p.printExpressionStmt(s)
	case *ast.TypeDeclStmt:
		p.writeLine(fmt.Sprintf("type %s", s.Decl.Name.Value))
		p.indentLevel++
		for _, f := range s.Decl.Fields {
			p.writeLine(fmt.Sprintf("%s %s", f.Name.Value, p.typeAnnotationToString(f.Type)))
		}
		p.indentLevel--
	case *ast.SelectStmt:
		p.printSelectStmt(s)
	}
}

// stmtToString renders a statement as a single-line string (for if init statements).
func (p *Printer) stmtToString(stmt ast.Statement) string {
	switch s := stmt.(type) {
	case *ast.VarDeclStmt:
		names := make([]string, len(s.Names))
		for i, n := range s.Names {
			names[i] = n.Value
		}
		values := make([]string, len(s.Values))
		for i, v := range s.Values {
			values[i] = p.exprToString(v)
		}
		return fmt.Sprintf("%s := %s", strings.Join(names, ", "), strings.Join(values, ", "))
	case *ast.AssignStmt:
		targets := make([]string, len(s.Targets))
		for i, t := range s.Targets {
			targets[i] = p.exprToString(t)
		}
		values := make([]string, len(s.Values))
		for i, v := range s.Values {
			values[i] = p.exprToString(v)
		}
		op := s.Token.Lexeme
		if op == "" {
			op = "="
		}
		return fmt.Sprintf("%s %s %s", strings.Join(targets, ", "), op, strings.Join(values, ", "))
	case *ast.ExpressionStmt:
		return p.exprToString(s.Expression)
	default:
		return ""
	}
}

func (p *Printer) onErrSuffix(clause *ast.OnErrClause) string {
	if clause == nil {
		return ""
	}

	var suffix string
	if clause.Alias != "" {
		suffix = " onerr as " + clause.Alias + " "
	} else {
		suffix = " onerr "
	}

	switch {
	case clause.ShorthandReturn:
		return suffix + "return"
	case clause.ShorthandContinue:
		return suffix + "continue"
	case clause.ShorthandBreak:
		return suffix + "break"
	case clause.Explain != "" && clause.Handler == nil:
		return suffix + fmt.Sprintf("explain %q", clause.Explain)
	case clause.Handler != nil:
		handlerStr := p.exprToString(clause.Handler)
		if clause.Explain != "" {
			handlerStr += fmt.Sprintf(" explain %q", clause.Explain)
		}
		return suffix + handlerStr
	default:
		return suffix
	}
}

// printOnErrBlock prints an onerr clause that may contain a block handler.
// Returns true if a block was printed (caller should not print a newline).
func (p *Printer) printOnErrBlock(clause *ast.OnErrClause) {
	if clause == nil {
		return
	}
	if blockExpr, ok := clause.Handler.(*ast.BlockExpr); ok {
		var prefix string
		if clause.Alias != "" {
			prefix = " onerr as " + clause.Alias
		} else {
			prefix = " onerr"
		}
		// Append onerr to the current line (already written by caller)
		p.output.WriteString(prefix + "\n")
		p.indentLevel++
		p.printBlock(blockExpr.Body)
		p.indentLevel--
		return
	}
	p.output.WriteString(p.onErrSuffix(clause) + "\n")
}

// hasBlockOnErr returns true if the onerr clause has a block handler.
func hasBlockOnErr(clause *ast.OnErrClause) bool {
	if clause == nil {
		return false
	}
	_, ok := clause.Handler.(*ast.BlockExpr)
	return ok
}

// printTopLevelVarDecl prints a top-level var/variable declaration.
func (p *Printer) printTopLevelVarDecl(stmt *ast.VarDeclStmt) {
	keyword := "var"
	if stmt.Token.Lexeme == "variable" {
		keyword = "variable"
	}
	names := make([]string, len(stmt.Names))
	for i, n := range stmt.Names {
		names[i] = n.Value
	}
	line := keyword + " " + strings.Join(names, ", ")
	if stmt.Type != nil {
		line += " " + p.typeAnnotationToString(stmt.Type)
	}
	if len(stmt.Values) > 0 {
		values := make([]string, len(stmt.Values))
		for i, v := range stmt.Values {
			values[i] = p.exprToString(v)
		}
		line += " = " + strings.Join(values, ", ")
	}
	p.writeLine(line)
}

func (p *Printer) printVarDeclStmt(stmt *ast.VarDeclStmt) {
	names := make([]string, len(stmt.Names))
	for i, n := range stmt.Names {
		names[i] = n.Value
	}
	// Check for piped switch expression that needs multi-line printing
	if len(stmt.Values) == 1 {
		if ps, ok := stmt.Values[0].(*ast.PipedSwitchExpr); ok {
			p.printPipedSwitchWithPrefix(fmt.Sprintf("%s := ", strings.Join(names, ", ")), ps, stmt.OnErr)
			return
		}
	}
	values := make([]string, len(stmt.Values))
	for i, v := range stmt.Values {
		values[i] = p.exprToString(v)
	}
	line := fmt.Sprintf("%s := %s", strings.Join(names, ", "), strings.Join(values, ", "))
	if hasBlockOnErr(stmt.OnErr) {
		p.write(p.indent())
		p.output.WriteString(line)
		p.printOnErrBlock(stmt.OnErr)
	} else {
		p.writeLine(line + p.onErrSuffix(stmt.OnErr))
	}
}

func (p *Printer) printAssignStmt(stmt *ast.AssignStmt) {
	targets := make([]string, len(stmt.Targets))
	for i, t := range stmt.Targets {
		targets[i] = p.exprToString(t)
	}
	// Check for piped switch expression
	if len(stmt.Values) == 1 {
		if ps, ok := stmt.Values[0].(*ast.PipedSwitchExpr); ok {
			op := stmt.Token.Lexeme
			if op == "" {
				op = "="
			}
			p.printPipedSwitchWithPrefix(fmt.Sprintf("%s %s ", strings.Join(targets, ", "), op), ps, stmt.OnErr)
			return
		}
	}
	values := make([]string, len(stmt.Values))
	for i, v := range stmt.Values {
		values[i] = p.exprToString(v)
	}
	op := stmt.Token.Lexeme
	if op == "" {
		op = "="
	}
	line := fmt.Sprintf("%s %s %s", strings.Join(targets, ", "), op, strings.Join(values, ", "))
	if hasBlockOnErr(stmt.OnErr) {
		p.write(p.indent())
		p.output.WriteString(line)
		p.printOnErrBlock(stmt.OnErr)
	} else {
		p.writeLine(line + p.onErrSuffix(stmt.OnErr))
	}
}

// printExpressionStmt handles expression statements, including those with
// piped switch expressions or block onerr clauses that need multi-line output.
func (p *Printer) printExpressionStmt(s *ast.ExpressionStmt) {
	// Handle piped switch as a standalone expression statement
	if ps, ok := s.Expression.(*ast.PipedSwitchExpr); ok {
		p.printPipedSwitchWithPrefix("", ps, s.OnErr)
		return
	}
	line := p.exprToString(s.Expression)
	if hasBlockOnErr(s.OnErr) {
		p.write(p.indent())
		p.output.WriteString(line)
		p.printOnErrBlock(s.OnErr)
	} else {
		p.writeLine(line + p.onErrSuffix(s.OnErr))
	}
}

// printPipedSwitchWithPrefix prints a piped switch expression with an optional
// prefix (e.g. "x := " for assignment) and handles the multi-line switch body.
func (p *Printer) printPipedSwitchWithPrefix(prefix string, ps *ast.PipedSwitchExpr, onErr *ast.OnErrClause) {
	left := p.exprToString(ps.Left)
	switch sw := ps.Switch.(type) {
	case *ast.SwitchStmt:
		p.writeLine(prefix + left + " |> switch")
		p.indentLevel++
		for _, c := range sw.Cases {
			values := make([]string, len(c.Values))
			for i, v := range c.Values {
				values[i] = p.exprToString(v)
			}
			p.writeLine("when " + strings.Join(values, ", "))
			p.indentLevel++
			p.printBlock(c.Body)
			p.indentLevel--
		}
		if sw.Otherwise != nil {
			p.writeLine("otherwise")
			p.indentLevel++
			p.printBlock(sw.Otherwise.Body)
			p.indentLevel--
		}
		p.indentLevel--
	case *ast.TypeSwitchStmt:
		binding := ""
		if sw.Binding != nil {
			binding = " as " + sw.Binding.Value
		}
		p.writeLine(prefix + left + " |> switch" + binding)
		p.indentLevel++
		for _, c := range sw.Cases {
			p.writeLine("when " + p.typeAnnotationToString(c.Type))
			p.indentLevel++
			p.printBlock(c.Body)
			p.indentLevel--
		}
		if sw.Otherwise != nil {
			p.writeLine("otherwise")
			p.indentLevel++
			p.printBlock(sw.Otherwise.Body)
			p.indentLevel--
		}
		p.indentLevel--
	}
}

func (p *Printer) printReturnStmt(stmt *ast.ReturnStmt) {
	if len(stmt.Values) == 0 {
		p.writeLine("return")
		return
	}

	// Handle piped switch expression in return
	if len(stmt.Values) == 1 {
		if ps, ok := stmt.Values[0].(*ast.PipedSwitchExpr); ok {
			p.printPipedSwitchWithPrefix("return ", ps, nil)
			return
		}
	}

	values := make([]string, len(stmt.Values))
	for i, val := range stmt.Values {
		values[i] = p.exprToString(val)
	}

	p.writeLine(fmt.Sprintf("return %s", strings.Join(values, ", ")))
}

func (p *Printer) printIfStmt(stmt *ast.IfStmt) {
	condition := p.exprToString(stmt.Condition)
	if stmt.Init != nil {
		initStr := p.stmtToString(stmt.Init)
		p.writeLine(fmt.Sprintf("if %s; %s", initStr, condition))
	} else {
		p.writeLine(fmt.Sprintf("if %s", condition))
	}

	p.indentLevel++
	p.printBlock(stmt.Consequence)
	p.indentLevel--

	if stmt.Alternative != nil {
		switch alt := stmt.Alternative.(type) {
		case *ast.ElseStmt:
			p.writeLine("else")
			p.indentLevel++
			p.printBlock(alt.Body)
			p.indentLevel--
		case *ast.IfStmt:
			// else if - print on same conceptual level
			p.write(p.indent())
			p.output.WriteString("else ")
			condition := p.exprToString(alt.Condition)
			if alt.Init != nil {
				initStr := p.stmtToString(alt.Init)
				p.output.WriteString(fmt.Sprintf("if %s; %s\n", initStr, condition))
			} else {
				p.output.WriteString(fmt.Sprintf("if %s\n", condition))
			}
			p.indentLevel++
			p.printBlock(alt.Consequence)
			p.indentLevel--
			if alt.Alternative != nil {
				p.printIfStmtAlternative(alt.Alternative)
			}
		}
	}
}

func (p *Printer) printIfStmtAlternative(alt ast.Statement) {
	switch a := alt.(type) {
	case *ast.ElseStmt:
		p.writeLine("else")
		p.indentLevel++
		p.printBlock(a.Body)
		p.indentLevel--
	case *ast.IfStmt:
		p.write(p.indent())
		p.output.WriteString("else ")
		condition := p.exprToString(a.Condition)
		if a.Init != nil {
			initStr := p.stmtToString(a.Init)
			p.output.WriteString(fmt.Sprintf("if %s; %s\n", initStr, condition))
		} else {
			p.output.WriteString(fmt.Sprintf("if %s\n", condition))
		}
		p.indentLevel++
		p.printBlock(a.Consequence)
		p.indentLevel--
		if a.Alternative != nil {
			p.printIfStmtAlternative(a.Alternative)
		}
	}
}

func (p *Printer) printForRangeStmt(stmt *ast.ForRangeStmt) {
	collection := p.exprToString(stmt.Collection)

	if stmt.Index != nil {
		p.writeLine(fmt.Sprintf("for %s, %s in %s", stmt.Index.Value, stmt.Variable.Value, collection))
	} else {
		p.writeLine(fmt.Sprintf("for %s in %s", stmt.Variable.Value, collection))
	}

	p.indentLevel++
	p.printBlock(stmt.Body)
	p.indentLevel--
}

func (p *Printer) printForNumericStmt(stmt *ast.ForNumericStmt) {
	varName := stmt.Variable.Value
	start := p.exprToString(stmt.Start)
	end := p.exprToString(stmt.End)

	keyword := "to"
	if stmt.Through {
		keyword = "through"
	}

	p.writeLine(fmt.Sprintf("for %s from %s %s %s", varName, start, keyword, end))

	p.indentLevel++
	p.printBlock(stmt.Body)
	p.indentLevel--
}

func (p *Printer) printForConditionStmt(stmt *ast.ForConditionStmt) {
	condition := p.exprToString(stmt.Condition)
	p.writeLine(fmt.Sprintf("for %s", condition))

	p.indentLevel++
	p.printBlock(stmt.Body)
	p.indentLevel--
}

func (p *Printer) printSwitchStmt(stmt *ast.SwitchStmt) {
	if stmt.Expression != nil {
		p.writeLine(fmt.Sprintf("switch %s", p.exprToString(stmt.Expression)))
	} else {
		p.writeLine("switch")
	}

	p.indentLevel++
	for _, c := range stmt.Cases {
		values := make([]string, len(c.Values))
		for i, v := range c.Values {
			values[i] = p.exprToString(v)
		}
		p.writeLine(fmt.Sprintf("when %s", strings.Join(values, ", ")))
		p.indentLevel++
		p.printBlock(c.Body)
		p.indentLevel--
	}

	if stmt.Otherwise != nil {
		p.writeLine("otherwise")
		p.indentLevel++
		p.printBlock(stmt.Otherwise.Body)
		p.indentLevel--
	}
	p.indentLevel--
}

func (p *Printer) printTypeSwitchStmt(stmt *ast.TypeSwitchStmt) {
	p.writeLine(fmt.Sprintf("switch %s as %s", p.exprToString(stmt.Expression), stmt.Binding.Value))

	p.indentLevel++
	for _, c := range stmt.Cases {
		p.writeLine(fmt.Sprintf("when %s", p.typeAnnotationToString(c.Type)))
		p.indentLevel++
		p.printBlock(c.Body)
		p.indentLevel--
	}

	if stmt.Otherwise != nil {
		p.writeLine("otherwise")
		p.indentLevel++
		p.printBlock(stmt.Otherwise.Body)
		p.indentLevel--
	}
	p.indentLevel--
}

func (p *Printer) exprToString(expr ast.Expression) string {
	if expr == nil {
		return ""
	}

	switch e := expr.(type) {
	case *ast.Identifier:
		return e.Value
	case *ast.IntegerLiteral:
		// Preserve original lexeme for prefixed/underscored literals
		if lexeme := e.Token.Lexeme; len(lexeme) > 1 && (lexeme[0] == '0' || strings.Contains(lexeme, "_")) {
			return lexeme
		}
		return fmt.Sprintf("%d", e.Value)
	case *ast.FloatLiteral:
		// Preserve original lexeme for underscored float literals
		if strings.Contains(e.Token.Lexeme, "_") {
			return e.Token.Lexeme
		}
		// Use the original lexeme to preserve decimal points (e.g., 1.0 stays 1.0)
		return e.Token.Lexeme
	case *ast.StringLiteral:
		return p.stringLiteralToString(e)
	case *ast.BooleanLiteral:
		if e.Value {
			return "true"
		}
		return "false"
	case *ast.BinaryExpr:
		return p.binaryExprToString(e)
	case *ast.IsExpr:
		return p.isExprToString(e)
	case *ast.UnaryExpr:
		return p.unaryExprToString(e)
	case *ast.PipeExpr:
		return p.pipeExprToString(e)
	// Note: OnErrExpr removed — onerr is now a clause on VarDeclStmt, AssignStmt, ExpressionStmt
	case *ast.CallExpr:
		return p.callExprToString(e)
	case *ast.MethodCallExpr:
		return p.methodCallExprToString(e)
	case *ast.FieldAccessExpr:
		return p.fieldAccessExprToString(e)
	case *ast.IndexExpr:
		left := p.exprToString(e.Left)
		index := p.exprToString(e.Index)
		return fmt.Sprintf("%s[%s]", left, index)
	case *ast.SliceExpr:
		return p.sliceExprToString(e)
	case *ast.StructLiteralExpr:
		return p.structLiteralToString(e)
	case *ast.ListLiteralExpr:
		return p.listLiteralToString(e)
	case *ast.MapLiteralExpr:
		return p.mapLiteralToString(e)
	case *ast.ReceiveExpr:
		channel := p.exprToString(e.Channel)
		return fmt.Sprintf("receive from %s", channel)
	case *ast.TypeCastExpr:
		targetType := p.typeAnnotationToString(e.TargetType)
		expr := p.exprToString(e.Expression)
		return fmt.Sprintf("%s as %s", expr, targetType)
	case *ast.EmptyExpr:
		if e.Type != nil {
			targetType := p.typeAnnotationToString(e.Type)
			return fmt.Sprintf("empty %s", targetType)
		}
		return "empty"
	case *ast.DiscardExpr:
		return "discard"
	case *ast.ErrorExpr:
		message := p.exprToString(e.Message)
		return fmt.Sprintf("error %s", message)
	case *ast.MakeExpr:
		return p.makeExprToString(e)
	case *ast.CloseExpr:
		channel := p.exprToString(e.Channel)
		return fmt.Sprintf("close %s", channel)
	case *ast.PanicExpr:
		message := p.exprToString(e.Message)
		return fmt.Sprintf("panic %s", message)
	case *ast.ReturnExpr:
		if len(e.Values) == 0 {
			return "return"
		}
		vals := make([]string, len(e.Values))
		for i, v := range e.Values {
			vals[i] = p.exprToString(v)
		}
		return "return " + strings.Join(vals, ", ")
	case *ast.RecoverExpr:
		return "recover"
	case *ast.ArrowLambda:
		return p.arrowLambdaToString(e)
	case *ast.AddressOfExpr:
		return "reference of " + p.exprToString(e.Operand)
	case *ast.DerefExpr:
		return "dereference " + p.exprToString(e.Operand)
	case *ast.FunctionLiteral:
		return p.functionLiteralToString(e)
	case *ast.PipedSwitchExpr:
		left := p.exprToString(e.Left)
		return fmt.Sprintf("%s |> switch", left)
	case *ast.TypeAssertionExpr:
		expr := p.exprToString(e.Expression)
		targetType := p.typeAnnotationToString(e.TargetType)
		return fmt.Sprintf("%s.(%s)", expr, targetType)
	case *ast.BlockExpr:
		// BlockExpr wraps an indented block used as an expression (e.g. onerr handler).
		// When used inline (not at statement level), we can't represent multi-line
		// blocks. Return a best-effort single-line representation for simple cases.
		if e.Body != nil && len(e.Body.Statements) == 1 {
			if exprStmt, ok := e.Body.Statements[0].(*ast.ExpressionStmt); ok {
				return p.exprToString(exprStmt.Expression)
			}
		}
		return ""
	case *ast.IfExpression:
		cond := p.exprToString(e.Condition)
		then := p.exprToString(e.Then)
		els := p.exprToString(e.Else)
		return fmt.Sprintf("if %s then %s else %s", cond, then, els)
	case *ast.NamedArgument:
		return fmt.Sprintf("%s: %s", e.Name.Value, p.exprToString(e.Value))
	default:
		return ""
	}
}

func (p *Printer) stringLiteralToString(lit *ast.StringLiteral) string {
	if lit.Raw {
		return "`" + lit.Value + "`"
	}
	// The lexer resolves escape sequences in Value (e.g., \n → newline,
	// \\ → backslash). We must re-escape when emitting source.
	if len(lit.Parts) > 0 {
		// Interpolated string: reconstruct from Parts so that expression
		// sub-strings are printed via exprToString, not re-escaped.
		var b strings.Builder
		b.WriteByte('"')
		for _, part := range lit.Parts {
			if part.IsLiteral {
				b.WriteString(escapeStringValue(part.Literal))
			} else {
				b.WriteByte('{')
				b.WriteString(p.exprToString(part.Expr))
				b.WriteByte('}')
			}
		}
		b.WriteByte('"')
		return b.String()
	}
	return "\"" + escapeStringValue(lit.Value) + "\""
}

// escapeStringValue re-escapes a processed string value back to Kukicha source form.
// PUA sentinels are converted back to their escape sequences (\{, \}, \sep).
func escapeStringValue(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 10)
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		case '\uE000': // PUA sentinel for \{
			b.WriteString(`\{`)
		case '\uE001': // PUA sentinel for \}
			b.WriteString(`\}`)
		case '\uE002': // PUA sentinel for \sep
			b.WriteString(`\sep`)
		default:
			if r < 0x20 || r == 0x7F {
				// Non-printable characters as hex escapes
				b.WriteString(fmt.Sprintf(`\x%02x`, r))
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// operatorPrecedence returns a numeric precedence for binary operators.
// Higher values bind tighter.
func operatorPrecedence(op string) int {
	switch op {
	case "or", "||":
		return 1
	case "and", "&&":
		return 2
	case "|":
		return 3
	case "^":
		return 4
	case "&":
		return 5
	case "equals", "not equals", "==", "!=", "isnt":
		return 6
	case "<", ">", "<=", ">=", "in":
		return 7
	case "<<", ">>":
		return 8
	case "+", "-":
		return 9
	case "*", "/", "%":
		return 10
	default:
		return 0
	}
}

// binaryOperandToString wraps a child expression in parens only when needed
// to preserve the correct precedence relative to the parent operator.
func (p *Printer) binaryOperandToString(child ast.Expression, parentPrec int) string {
	s := p.exprToString(child)
	if bin, ok := child.(*ast.BinaryExpr); ok {
		childPrec := operatorPrecedence(bin.Operator)
		if childPrec < parentPrec {
			return "(" + s + ")"
		}
	}
	return s
}

func (p *Printer) binaryExprToString(expr *ast.BinaryExpr) string {
	prec := operatorPrecedence(expr.Operator)
	left := p.binaryOperandToString(expr.Left, prec)
	right := p.binaryOperandToString(expr.Right, prec)

	return fmt.Sprintf("%s %s %s", left, expr.Operator, right)
}

func (p *Printer) isExprToString(expr *ast.IsExpr) string {
	value := p.exprToString(expr.Value)
	caseName := ""
	if expr.Case != nil {
		caseName = expr.Case.Value
	}
	if expr.Binding != nil {
		return fmt.Sprintf("%s is %s as %s", value, caseName, expr.Binding.Value)
	}
	return fmt.Sprintf("%s is %s", value, caseName)
}

func (p *Printer) unaryExprToString(expr *ast.UnaryExpr) string {
	right := p.exprToString(expr.Right)

	// Word operators (not) need a space; symbol operators (-) don't
	if expr.Operator == "not" || expr.Operator == "!" {
		// If the operand is a binary/logical/pipe expression, it has lower
		// precedence than not, so we must parenthesize to preserve semantics.
		if needsParensAfterNot(expr.Right) {
			return fmt.Sprintf("%s (%s)", expr.Operator, right)
		}
		return fmt.Sprintf("%s %s", expr.Operator, right)
	}
	return fmt.Sprintf("%s%s", expr.Operator, right)
}

// needsParensAfterNot reports whether expr has lower precedence than "not" and
// therefore needs parentheses when used as the operand of a not/! expression.
func needsParensAfterNot(expr ast.Expression) bool {
	switch expr.(type) {
	case *ast.BinaryExpr, *ast.PipeExpr, *ast.IsExpr:
		return true
	}
	return false
}

// pipeExprToString formats a pipe chain expression.
// Short chains stay on one line; long chains are broken across lines
// with each |> stage on its own indented line.
func (p *Printer) pipeExprToString(expr *ast.PipeExpr) string {
	stages := flattenPipeChain(expr)

	formatted := make([]string, len(stages))
	hasMultilineStage := false
	for i, stage := range stages {
		formatted[i] = p.exprToString(stage)
		if strings.Contains(formatted[i], "\n") {
			hasMultilineStage = true
		}
	}

	singleLine := strings.Join(formatted, " |> ")

	// If any stage is already multi-line (e.g., contains a function literal
	// body), keep the pipe join on one line so the existing multi-line
	// formatting within the stage is preserved.
	if hasMultilineStage {
		return singleLine
	}

	// Keep on one line if it fits within the target width.
	lineWidth := len(p.indent()) + len(singleLine)
	if lineWidth <= maxLineWidth {
		return singleLine
	}

	// Multi-line: first stage on current line, subsequent stages each on
	// their own line indented one level deeper than the current context.
	contIndent := p.indent() + p.indentStr
	var b strings.Builder
	b.WriteString(formatted[0])
	for _, stage := range formatted[1:] {
		b.WriteByte('\n')
		b.WriteString(contIndent)
		b.WriteString("|> ")
		b.WriteString(stage)
	}
	return b.String()
}

// flattenPipeChain collects all stages in a pipe chain.
// For a |> b |> c (parsed as (a |> b) |> c), returns [a, b, c].
func flattenPipeChain(expr *ast.PipeExpr) []ast.Expression {
	var stages []ast.Expression
	current := ast.Expression(expr)
	for {
		pe, ok := current.(*ast.PipeExpr)
		if !ok {
			stages = append(stages, current)
			break
		}
		stages = append(stages, pe.Right)
		current = pe.Left
	}
	// Reverse: we collected right-to-left.
	for i, j := 0, len(stages)-1; i < j; i, j = i+1, j-1 {
		stages[i], stages[j] = stages[j], stages[i]
	}
	return stages
}

func (p *Printer) callExprToString(expr *ast.CallExpr) string {
	funcName := p.exprToString(expr.Function)
	args := make([]string, len(expr.Arguments))
	for i, arg := range expr.Arguments {
		s := p.exprToString(arg)
		// Add "many" prefix for the last argument if variadic
		if expr.Variadic && i == len(expr.Arguments)-1 {
			s = "many " + s
		}
		args[i] = s
	}

	joined := strings.Join(args, ", ")
	if !strings.Contains(joined, "\n") {
		return fmt.Sprintf("%s(%s)", funcName, joined)
	}

	// Multi-line argument: closing ) must be on its own dedented line so the
	// parser sees a proper DEDENT before the paren. However, if the last
	// line is already just closing delimiters (from nested calls or
	// multiline literals), merge our ) onto that line.
	lastNL := strings.LastIndex(joined, "\n")
	lastLine := strings.TrimSpace(joined[lastNL+1:])
	if lastLine != "" && strings.Trim(lastLine, ")}]") == "" {
		return fmt.Sprintf("%s(%s)", funcName, joined)
	}
	return fmt.Sprintf("%s(%s\n%s)", funcName, joined, p.indent())
}

func (p *Printer) methodCallExprToString(expr *ast.MethodCallExpr) string {
	object := p.exprToString(expr.Object)
	method := expr.Method.Value

	if len(expr.Arguments) == 0 {
		return fmt.Sprintf("%s.%s()", object, method)
	}

	args := make([]string, len(expr.Arguments))
	for i, arg := range expr.Arguments {
		s := p.exprToString(arg)
		if expr.Variadic && i == len(expr.Arguments)-1 {
			s = "many " + s
		}
		args[i] = s
	}

	joined := strings.Join(args, ", ")
	if !strings.Contains(joined, "\n") {
		return fmt.Sprintf("%s.%s(%s)", object, method, joined)
	}

	// Merge closing delimiters if the last line is already just )}] chars.
	lastNL := strings.LastIndex(joined, "\n")
	lastLine := strings.TrimSpace(joined[lastNL+1:])
	if lastLine != "" && strings.Trim(lastLine, ")}]") == "" {
		return fmt.Sprintf("%s.%s(%s)", object, method, joined)
	}
	return fmt.Sprintf("%s.%s(%s\n%s)", object, method, joined, p.indent())
}

func (p *Printer) fieldAccessExprToString(expr *ast.FieldAccessExpr) string {
	object := p.exprToString(expr.Object)
	// Type cast expressions need parens when used as a field access receiver,
	// e.g. (val as Item).Id, otherwise it parses as val as Item.Id.
	if _, ok := expr.Object.(*ast.TypeCastExpr); ok {
		object = "(" + object + ")"
	}
	return fmt.Sprintf("%s.%s", object, expr.Field.Value)
}

func (p *Printer) sliceExprToString(expr *ast.SliceExpr) string {
	left := p.exprToString(expr.Left)

	var start, end string
	if expr.Start != nil {
		start = p.exprToString(expr.Start)
	}
	if expr.End != nil {
		end = p.exprToString(expr.End)
	}

	return fmt.Sprintf("%s[%s:%s]", left, start, end)
}

func (p *Printer) structLiteralToString(expr *ast.StructLiteralExpr) string {
	typeName := p.typeAnnotationToString(expr.Type)

	if len(expr.Fields) == 0 {
		return fmt.Sprintf("%s{}", typeName)
	}

	fields := make([]string, len(expr.Fields))
	for i, field := range expr.Fields {
		value := p.exprToString(field.Value)
		fields[i] = fmt.Sprintf("%s: %s", field.Name.Value, value)
	}

	singleLine := fmt.Sprintf("%s{%s}", typeName, strings.Join(fields, ", "))
	if len(p.indent())+len(singleLine) <= maxLineWidth {
		return singleLine
	}

	return p.multilineBraced(typeName, fields)
}

func (p *Printer) listLiteralToString(expr *ast.ListLiteralExpr) string {
	if len(expr.Elements) == 0 {
		if expr.Type != nil {
			elemType := p.typeAnnotationToString(expr.Type)
			return fmt.Sprintf("empty list of %s", elemType)
		}
		return "empty list"
	}

	elements := make([]string, len(expr.Elements))
	for i, elem := range expr.Elements {
		elements[i] = p.exprToString(elem)
	}

	if expr.Type != nil {
		elemType := p.typeAnnotationToString(expr.Type)
		singleLine := fmt.Sprintf("list of %s{%s}", elemType, strings.Join(elements, ", "))
		if len(p.indent())+len(singleLine) <= maxLineWidth {
			return singleLine
		}
		return p.multilineBraced("list of "+elemType, elements)
	}

	singleLine := fmt.Sprintf("[%s]", strings.Join(elements, ", "))
	if len(p.indent())+len(singleLine) <= maxLineWidth {
		return singleLine
	}
	return p.multilineBracketed(elements)
}

func (p *Printer) mapLiteralToString(expr *ast.MapLiteralExpr) string {
	keyType := p.typeAnnotationToString(expr.KeyType)
	valType := p.typeAnnotationToString(expr.ValType)

	if len(expr.Pairs) == 0 {
		return fmt.Sprintf("empty map of %s to %s", keyType, valType)
	}

	pairs := make([]string, len(expr.Pairs))
	for i, pair := range expr.Pairs {
		key := p.exprToString(pair.Key)
		value := p.exprToString(pair.Value)
		pairs[i] = fmt.Sprintf("%s: %s", key, value)
	}

	prefix := fmt.Sprintf("map of %s to %s", keyType, valType)
	singleLine := fmt.Sprintf("%s{%s}", prefix, strings.Join(pairs, ", "))
	if len(p.indent())+len(singleLine) <= maxLineWidth {
		return singleLine
	}

	return p.multilineBraced(prefix, pairs)
}

// multilineBraced formats entries as a multiline brace-delimited literal:
//
//	prefix{
//	    entry1,
//	    entry2,
//	}
func (p *Printer) multilineBraced(prefix string, entries []string) string {
	innerIndent := p.indent() + p.indentStr
	var b strings.Builder
	b.WriteString(prefix)
	b.WriteString("{\n")
	for _, entry := range entries {
		b.WriteString(innerIndent)
		b.WriteString(entry)
		b.WriteString(",\n")
	}
	b.WriteString(p.indent())
	b.WriteByte('}')
	return b.String()
}

// multilineBracketed formats entries as a multiline bracket-delimited list:
//
//	[
//	    entry1,
//	    entry2,
//	]
func (p *Printer) multilineBracketed(entries []string) string {
	innerIndent := p.indent() + p.indentStr
	var b strings.Builder
	b.WriteString("[\n")
	for _, entry := range entries {
		b.WriteString(innerIndent)
		b.WriteString(entry)
		b.WriteString(",\n")
	}
	b.WriteString(p.indent())
	b.WriteByte(']')
	return b.String()
}

func (p *Printer) makeExprToString(expr *ast.MakeExpr) string {
	targetType := p.typeAnnotationToString(expr.Type)

	if len(expr.Args) == 0 {
		return fmt.Sprintf("make(%s)", targetType)
	}

	args := make([]string, len(expr.Args))
	for i, arg := range expr.Args {
		args[i] = p.exprToString(arg)
	}

	return fmt.Sprintf("make(%s, %s)", targetType, strings.Join(args, ", "))
}

func (p *Printer) arrowLambdaToString(lambda *ast.ArrowLambda) string {
	// Build parameter string
	var paramParts []string
	for _, param := range lambda.Parameters {
		if param.Type != nil {
			paramParts = append(paramParts, param.Name.Value+" "+p.typeAnnotationToString(param.Type))
		} else {
			paramParts = append(paramParts, param.Name.Value)
		}
	}

	var paramsStr string
	if len(lambda.Parameters) == 1 && lambda.Parameters[0].Type == nil {
		// Single untyped param: no parens
		paramsStr = lambda.Parameters[0].Name.Value
	} else {
		paramsStr = "(" + strings.Join(paramParts, ", ") + ")"
	}

	if lambda.Body != nil {
		return fmt.Sprintf("%s => %s", paramsStr, p.exprToString(lambda.Body))
	}

	blockPrinter := NewPrinter()
	blockPrinter.indentStr = p.indentStr
	blockPrinter.indentLevel = p.indentLevel + 1
	blockPrinter.printBlock(lambda.Block)

	return fmt.Sprintf("%s =>\n%s", paramsStr, strings.TrimRight(blockPrinter.output.String(), "\n"))
}

// Helper methods

func (p *Printer) indent() string {
	return strings.Repeat(p.indentStr, p.indentLevel)
}

func (p *Printer) write(s string) {
	p.output.WriteString(s)
}

func (p *Printer) writeLine(s string) {
	if s == "" {
		// Blank line — no indentation
		p.output.WriteString("\n")
	} else if strings.Contains(s, "\n") {
		// Multi-line content (e.g., function literals with bodies):
		// indent the first line, write subsequent lines as-is since they
		// already carry their own indentation from the printer.
		lines := strings.SplitAfter(s, "\n")
		p.output.WriteString(p.indent())
		for _, line := range lines {
			p.output.WriteString(line)
		}
		if !strings.HasSuffix(s, "\n") {
			p.output.WriteString("\n")
		}
	} else {
		p.output.WriteString(p.indent())
		p.output.WriteString(s)
		p.output.WriteString("\n")
	}
}

// formatStructTag converts a Go-style struct tag back to Kukicha syntax.
// Simple json-only tags like `json:"name"` become `as "name"`.
// Complex tags are preserved as-is.
func formatStructTag(tag string) string {
	// Check for simple json:"value" pattern (the most common case from 'as' syntax)
	if strings.HasPrefix(tag, `json:"`) && strings.HasSuffix(tag, `"`) {
		// Extract the json value
		inner := tag[6 : len(tag)-1] // strip json:" and trailing "
		// Only convert to 'as' syntax if it's a simple name (no options like omitempty)
		if !strings.Contains(inner, ",") {
			return fmt.Sprintf(`as "%s"`, inner)
		}
	}
	return tag
}

func (p *Printer) functionLiteralToString(expr *ast.FunctionLiteral) string {
	params := p.parametersToString(expr.Parameters)
	returns := p.returnTypesToString(expr.Returns)

	var sig string
	if returns != "" {
		sig = fmt.Sprintf("func(%s) %s", params, returns)
	} else {
		sig = fmt.Sprintf("func(%s)", params)
	}

	if expr.Body == nil {
		return sig
	}

	// Render the body using the full statement printer to handle nested blocks.
	// Capture output into a separate buffer.
	savedOutput := p.output
	p.output = strings.Builder{}
	p.indentLevel++
	p.printBlock(expr.Body)
	p.indentLevel--
	body := p.output.String()
	p.output = savedOutput

	return sig + "\n" + strings.TrimRight(body, "\n")
}

func (p *Printer) printSelectStmt(stmt *ast.SelectStmt) {
	p.writeLine("select")
	p.indentLevel++
	for _, c := range stmt.Cases {
		if c.Send != nil {
			channel := p.exprToString(c.Send.Channel)
			value := p.exprToString(c.Send.Value)
			p.writeLine(fmt.Sprintf("when send %s to %s", value, channel))
		} else if c.Recv != nil {
			channel := p.exprToString(c.Recv.Channel)
			if len(c.Bindings) > 0 {
				p.writeLine(fmt.Sprintf("when %s := receive from %s", strings.Join(c.Bindings, ", "), channel))
			} else {
				p.writeLine(fmt.Sprintf("when receive from %s", channel))
			}
		}
		p.indentLevel++
		if c.Body != nil {
			p.printBlock(c.Body)
		}
		p.indentLevel--
	}
	if stmt.Otherwise != nil {
		p.writeLine("otherwise")
		p.indentLevel++
		p.printBlock(stmt.Otherwise.Body)
		p.indentLevel--
	}
	p.indentLevel--
}
