package formatter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kukichalang/kukicha/internal/ast"
	"github.com/kukichalang/kukicha/internal/lexer"
	"github.com/kukichalang/kukicha/internal/parser"
)

// FormatOptions contains options for formatting
type FormatOptions struct {
	// PreprocessGoStyle converts Go-style braces/semicolons to Kukicha style
	PreprocessGoStyle bool
}

// DefaultOptions returns the default formatting options
func DefaultOptions() FormatOptions {
	return FormatOptions{
		PreprocessGoStyle: true,
	}
}

// Format formats Kukicha source code and returns the formatted result
func Format(source string, filename string, opts FormatOptions) (string, error) {
	// Preprocess if needed (handle Go-style braces)
	processedSource := source
	if opts.PreprocessGoStyle {
		processedSource = ProcessSource(source)
	}

	// Lex to get tokens (including comments)
	l := lexer.NewLexer(processedSource, filename)
	tokens, err := l.ScanTokens()
	if err != nil {
		return "", fmt.Errorf("lexer error: %w", err)
	}

	// Extract comments from tokens
	comments := ExtractComments(tokens)

	// Parse to get AST
	p := parser.NewFromTokens(tokens)

	program, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		// Collect all errors into one message
		var errMsgs []string
		for _, e := range parseErrors {
			errMsgs = append(errMsgs, e.Error())
		}
		return "", fmt.Errorf("parse errors:\n  %s", strings.Join(errMsgs, "\n  "))
	}

	// Attach comments to AST nodes
	commentMap := AttachComments(comments, program)

	// Print formatted output
	printer := NewPrinterWithComments(commentMap)
	output := printer.Print(program)

	return output, nil
}

// FormatCheck checks if the source is already formatted
// Returns true if the source matches the formatted output
func FormatCheck(source string, filename string, opts FormatOptions) (bool, error) {
	formatted, err := Format(source, filename, opts)
	if err != nil {
		return false, err
	}

	// Normalize both for comparison (handle trailing newlines)
	normalizedSource := strings.TrimRight(source, "\n\r\t ") + "\n"
	normalizedFormatted := strings.TrimRight(formatted, "\n\r\t ") + "\n"

	return normalizedSource == normalizedFormatted, nil
}

// PrinterWithComments extends the basic printer with comment support
type PrinterWithComments struct {
	*Printer
	comments CommentMap
}

// NewPrinterWithComments creates a printer that includes comments
func NewPrinterWithComments(comments CommentMap) *PrinterWithComments {
	return &PrinterWithComments{
		Printer:  NewPrinter(),
		comments: comments,
	}
}

// Print prints the program with comments
func (p *PrinterWithComments) Print(program *ast.Program) string {
	p.output.Reset()
	p.indentLevel = 0

	// Print leading comments for the program (file-level comments)
	if program.PetioleDecl != nil {
		p.printLeadingComments(program.PetioleDecl)
		p.writeLine(fmt.Sprintf("petiole %s", program.PetioleDecl.Name.Value))
		p.printTrailingComment(program.PetioleDecl)
		p.writeLine("")
	}

	// Print skill declaration if present
	if program.SkillDecl != nil {
		p.printLeadingComments(program.SkillDecl)
		p.writeLine(fmt.Sprintf("skill %s", program.SkillDecl.Name.Value))
		p.printTrailingComment(program.SkillDecl)
		p.writeLine("")
	}

	// Print imports grouped into buckets (stdlib | third-party), sorted within each.
	p.printImportsWithComments(program.Imports)
	if len(program.Imports) > 0 {
		p.writeLine("")
	}

	// Print declarations with comments
	for i, decl := range program.Declarations {
		if i > 0 {
			p.writeLine("")
		}
		p.printLeadingComments(decl)
		p.printDeclarationWithComments(decl)
	}

	return p.output.String()
}

// printImportsWithComments emits imports bucketed by kind (stdlib vs third-party),
// sorted alphabetically within each bucket, with a blank line between non-empty
// buckets. Leading/trailing comments on each import travel with the import node.
func (p *PrinterWithComments) printImportsWithComments(imports []*ast.ImportDecl) {
	if len(imports) == 0 {
		return
	}
	const numBuckets = 2
	buckets := make([][]*ast.ImportDecl, numBuckets)
	for _, imp := range imports {
		b := importBucket(imp.Path.Value)
		buckets[b] = append(buckets[b], imp)
	}
	for _, bucket := range buckets {
		sort.SliceStable(bucket, func(i, j int) bool {
			return bucket[i].Path.Value < bucket[j].Path.Value
		})
	}
	first := true
	for _, bucket := range buckets {
		if len(bucket) == 0 {
			continue
		}
		if !first {
			p.writeLine("")
		}
		first = false
		for _, imp := range bucket {
			p.printLeadingComments(imp)
			p.printImport(imp)
			p.printTrailingComment(imp)
		}
	}
}

func (p *PrinterWithComments) printLeadingComments(node ast.Node) {
	if attachment, ok := p.comments[node]; ok && len(attachment.Leading) > 0 {
		for i, comment := range attachment.Leading {
			// Preserve blank line gaps between leading comments
			if i > 0 && comment.Line > attachment.Leading[i-1].Line+1 {
				p.writeLine("")
			}
			p.writeLine(comment.Text)
		}
		// Preserve blank line between last comment and the node (or its directives).
		lastComment := attachment.Leading[len(attachment.Leading)-1]
		nextLine := node.Pos().Line
		// If the node has directives, use the first directive's line instead
		if fd, ok := node.(*ast.FunctionDecl); ok && len(fd.Directives) > 0 {
			nextLine = fd.Directives[0].Token.Line
		}
		if nextLine > lastComment.Line+1 {
			p.writeLine("")
		}
	}
}

func (p *PrinterWithComments) printTrailingComment(node ast.Node) {
	if attachment, ok := p.comments[node]; ok && attachment.Trailing != nil {
		// Trailing comments go on the same line
		// We need to remove the last newline and add the comment
		output := p.output.String()
		if strings.HasSuffix(output, "\n") {
			p.output.Reset()
			p.output.WriteString(strings.TrimSuffix(output, "\n"))
			p.output.WriteString(" " + attachment.Trailing.Text + "\n")
		}
	}
}

func (p *PrinterWithComments) printDeclarationWithComments(decl ast.Declaration) {
	switch d := decl.(type) {
	case *ast.TypeDecl:
		p.printTypeDeclWithComments(d)
	case *ast.InterfaceDecl:
		p.printInterfaceDeclWithComments(d)
	case *ast.FunctionDecl:
		p.printFunctionDeclWithComments(d)
	case *ast.ConstDecl:
		p.printConstDeclWithComments(d)
	case *ast.EnumDecl:
		p.printEnumDecl(d)
	case *ast.VarDeclStmt:
		p.printTopLevelVarDecl(d)
	}
}

func (p *PrinterWithComments) printConstDeclWithComments(decl *ast.ConstDecl) {
	if len(decl.Specs) == 1 {
		spec := decl.Specs[0]
		p.writeLine(fmt.Sprintf("const %s = %s", spec.Name.Value, p.exprToString(spec.Value)))
		p.printTrailingComment(decl)
		return
	}
	p.writeLine("const")
	p.printTrailingComment(decl)
	p.indentLevel++
	for _, spec := range decl.Specs {
		p.writeLine(fmt.Sprintf("%s = %s", spec.Name.Value, p.exprToString(spec.Value)))
	}
	p.indentLevel--
}

func (p *PrinterWithComments) printTypeDeclWithComments(decl *ast.TypeDecl) {
	// Type alias (e.g., type Handler func(string))
	if decl.AliasType != nil {
		p.writeLine(fmt.Sprintf("type %s %s", decl.Name.Value, p.typeAnnotationToString(decl.AliasType)))
		p.printTrailingComment(decl)
		return
	}

	p.writeLine(fmt.Sprintf("type %s", decl.Name.Value))
	p.printTrailingComment(decl)
	p.indentLevel++

	for _, field := range decl.Fields {
		p.printLeadingComments(field.Name)
		fieldType := p.typeAnnotationToString(field.Type)
		line := fmt.Sprintf("%s %s", field.Name.Value, fieldType)
		if field.Tag != "" {
			line += " " + formatStructTag(field.Tag)
		}
		p.writeLine(line)
		p.printTrailingComment(field.Name)
	}

	p.indentLevel--
}

func (p *PrinterWithComments) printInterfaceDeclWithComments(decl *ast.InterfaceDecl) {
	p.writeLine(fmt.Sprintf("interface %s", decl.Name.Value))
	p.printTrailingComment(decl)
	p.indentLevel++

	for _, method := range decl.Methods {
		p.printLeadingComments(method.Name)
		params := p.parametersToString(method.Parameters)
		returns := p.returnTypesToString(method.Returns)

		if returns != "" {
			p.writeLine(fmt.Sprintf("%s(%s) %s", method.Name.Value, params, returns))
		} else {
			p.writeLine(fmt.Sprintf("%s(%s)", method.Name.Value, params))
		}
		p.printTrailingComment(method.Name)
	}

	p.indentLevel--
}

func (p *PrinterWithComments) printFunctionDeclWithComments(decl *ast.FunctionDecl) {
	p.printDirectives(decl.Directives)
	// Build signature
	var signature string
	if decl.Receiver != nil {
		receiverType := p.typeAnnotationToString(decl.Receiver.Type)
		params := p.parametersToString(decl.Parameters)
		returns := p.returnTypesToString(decl.Returns)
		if params != "" {
			signature = fmt.Sprintf("func %s on %s %s(%s)", decl.Name.Value, decl.Receiver.Name.Value, receiverType, params)
		} else {
			signature = fmt.Sprintf("func %s on %s %s", decl.Name.Value, decl.Receiver.Name.Value, receiverType)
		}
		if returns != "" {
			signature += " " + returns
		}
	} else {
		params := p.parametersToString(decl.Parameters)
		returns := p.returnTypesToString(decl.Returns)
		signature = fmt.Sprintf("func %s(%s)", decl.Name.Value, params)
		if returns != "" {
			signature += " " + returns
		}
	}

	p.writeLine(signature)
	p.printTrailingComment(decl)

	// Print body with comments
	if decl.Body != nil {
		p.indentLevel++
		p.printBlockWithComments(decl.Body)
		p.indentLevel--
	}
}

func (p *PrinterWithComments) printBlockWithComments(block *ast.BlockStmt) {
	prevEndLine := 0
	for _, stmt := range block.Statements {
		stmtLine := stmtStartLine(stmt)
		// Check if there's a leading comment — if so, use its line for blank line detection
		if attachment, ok := p.comments[stmt]; ok && len(attachment.Leading) > 0 {
			stmtLine = attachment.Leading[0].Line
		}
		// Preserve blank lines between statements
		if prevEndLine > 0 && stmtLine > prevEndLine+1 {
			p.writeLine("")
		}
		p.printLeadingComments(stmt)
		p.printStatementWithComments(stmt)
		p.printTrailingComment(stmt)
		prevEndLine = p.estimateEndLine(stmt)
	}
}

func (p *PrinterWithComments) printStatementWithComments(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.VarDeclStmt:
		p.printVarDeclStmt(s)
	case *ast.AssignStmt:
		p.printAssignStmt(s)
	case *ast.ReturnStmt:
		p.printReturnStmt(s)
	case *ast.IfStmt:
		p.printIfStmtWithComments(s)
	case *ast.SwitchStmt:
		p.printSwitchStmtWithComments(s)
	case *ast.SelectStmt:
		p.printSelectStmtWithComments(s)
	case *ast.TypeSwitchStmt:
		p.printTypeSwitchStmtWithComments(s)
	case *ast.ForRangeStmt:
		p.printForRangeStmtWithComments(s)
	case *ast.ForNumericStmt:
		p.printForNumericStmtWithComments(s)
	case *ast.ForConditionStmt:
		p.printForConditionStmtWithComments(s)
	case *ast.DeferStmt:
		if s.Block != nil {
			p.writeLine("defer")
			p.indentLevel++
			p.printBlockWithComments(s.Block)
			p.indentLevel--
		} else {
			p.writeLine("defer " + p.exprToString(s.Call))
		}
	case *ast.GoStmt:
		if s.Block != nil {
			p.writeLine("go")
			p.indentLevel++
			for _, stmt := range s.Block.Statements {
				p.printStatementWithComments(stmt)
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
		// Type declarations inside functions are rejected by semantic analysis,
		// but the formatter should preserve them so the user sees the error.
		p.writeLine(fmt.Sprintf("type %s", s.Decl.Name.Value))
		p.indentLevel++
		for _, f := range s.Decl.Fields {
			p.writeLine(fmt.Sprintf("%s %s", f.Name.Value, p.typeAnnotationToString(f.Type)))
		}
		p.indentLevel--
	}
}

func (p *PrinterWithComments) printIfStmtWithComments(stmt *ast.IfStmt) {
	condition := p.exprToString(stmt.Condition)
	if stmt.Init != nil {
		initStr := p.stmtToString(stmt.Init)
		p.writeLine(fmt.Sprintf("if %s; %s", initStr, condition))
	} else {
		p.writeLine(fmt.Sprintf("if %s", condition))
	}

	p.indentLevel++
	p.printBlockWithComments(stmt.Consequence)
	p.indentLevel--

	if stmt.Alternative != nil {
		switch alt := stmt.Alternative.(type) {
		case *ast.ElseStmt:
			p.printLeadingComments(alt)
			p.writeLine("else")
			p.indentLevel++
			p.printBlockWithComments(alt.Body)
			p.indentLevel--
		case *ast.IfStmt:
			p.printLeadingComments(alt)
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
			p.printBlockWithComments(alt.Consequence)
			p.indentLevel--
			if alt.Alternative != nil {
				p.printIfAlternativeWithComments(alt.Alternative)
			}
		}
	}
}

func (p *PrinterWithComments) printIfAlternativeWithComments(alt ast.Statement) {
	switch a := alt.(type) {
	case *ast.ElseStmt:
		p.printLeadingComments(a)
		p.writeLine("else")
		p.indentLevel++
		p.printBlockWithComments(a.Body)
		p.indentLevel--
	case *ast.IfStmt:
		p.printLeadingComments(a)
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
		p.printBlockWithComments(a.Consequence)
		p.indentLevel--
		if a.Alternative != nil {
			p.printIfAlternativeWithComments(a.Alternative)
		}
	}
}

func (p *PrinterWithComments) printForRangeStmtWithComments(stmt *ast.ForRangeStmt) {
	collection := p.exprToString(stmt.Collection)

	if stmt.Index != nil {
		p.writeLine(fmt.Sprintf("for %s, %s in %s", stmt.Index.Value, stmt.Variable.Value, collection))
	} else {
		p.writeLine(fmt.Sprintf("for %s in %s", stmt.Variable.Value, collection))
	}

	p.indentLevel++
	p.printBlockWithComments(stmt.Body)
	p.indentLevel--
}

func (p *PrinterWithComments) printForNumericStmtWithComments(stmt *ast.ForNumericStmt) {
	varName := stmt.Variable.Value
	start := p.exprToString(stmt.Start)
	end := p.exprToString(stmt.End)

	keyword := "to"
	if stmt.Through {
		keyword = "through"
	}

	p.writeLine(fmt.Sprintf("for %s from %s %s %s", varName, start, keyword, end))

	p.indentLevel++
	p.printBlockWithComments(stmt.Body)
	p.indentLevel--
}

func (p *PrinterWithComments) printSwitchStmtWithComments(stmt *ast.SwitchStmt) {
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
		p.printBlockWithComments(c.Body)
		p.indentLevel--
	}

	if stmt.Otherwise != nil {
		p.writeLine("otherwise")
		p.indentLevel++
		p.printBlockWithComments(stmt.Otherwise.Body)
		p.indentLevel--
	}
	p.indentLevel--
}

func (p *PrinterWithComments) printSelectStmtWithComments(stmt *ast.SelectStmt) {
	p.writeLine("select")
	p.indentLevel++
	for _, c := range stmt.Cases {
		var whenLine string
		if c.Recv != nil {
			ch := p.exprToString(c.Recv.Channel)
			switch len(c.Bindings) {
			case 0:
				whenLine = fmt.Sprintf("when receive from %s", ch)
			case 1:
				whenLine = fmt.Sprintf("when %s := receive from %s", c.Bindings[0], ch)
			case 2:
				whenLine = fmt.Sprintf("when %s, %s := receive from %s", c.Bindings[0], c.Bindings[1], ch)
			}
		} else if c.Send != nil {
			val := p.exprToString(c.Send.Value)
			ch := p.exprToString(c.Send.Channel)
			whenLine = fmt.Sprintf("when send %s to %s", val, ch)
		}
		p.writeLine(whenLine)
		p.indentLevel++
		p.printBlockWithComments(c.Body)
		p.indentLevel--
	}
	if stmt.Otherwise != nil {
		p.writeLine("otherwise")
		p.indentLevel++
		p.printBlockWithComments(stmt.Otherwise.Body)
		p.indentLevel--
	}
	p.indentLevel--
}

func (p *PrinterWithComments) printTypeSwitchStmtWithComments(stmt *ast.TypeSwitchStmt) {
	p.writeLine(fmt.Sprintf("switch %s as %s", p.exprToString(stmt.Expression), stmt.Binding.Value))

	p.indentLevel++
	for _, c := range stmt.Cases {
		p.writeLine(fmt.Sprintf("when %s", p.typeAnnotationToString(c.Type)))
		p.indentLevel++
		p.printBlockWithComments(c.Body)
		p.indentLevel--
	}

	if stmt.Otherwise != nil {
		p.writeLine("otherwise")
		p.indentLevel++
		p.printBlockWithComments(stmt.Otherwise.Body)
		p.indentLevel--
	}
	p.indentLevel--
}

func (p *PrinterWithComments) printForConditionStmtWithComments(stmt *ast.ForConditionStmt) {
	condition := p.exprToString(stmt.Condition)
	p.writeLine(fmt.Sprintf("for %s", condition))

	p.indentLevel++
	p.printBlockWithComments(stmt.Body)
	p.indentLevel--
}
