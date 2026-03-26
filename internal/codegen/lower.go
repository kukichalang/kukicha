package codegen

import (
	"fmt"
	"strings"

	"github.com/kukichalang/kukicha/internal/ast"
	"github.com/kukichalang/kukicha/internal/ir"
)

// Lowerer transforms high-level AST constructs (pipes, onerr) into IR nodes.
// It delegates sub-expression rendering to Generator.exprToString and uses
// Generator.inferReturnCount / isErrorOnlyReturn for semantic queries.
type Lowerer struct {
	gen *Generator
}

// newLowerer creates a Lowerer that shares the generator's semantic info.
func newLowerer(gen *Generator) *Lowerer {
	return &Lowerer{gen: gen}
}

// posOf converts an AST expression's position into an IR source position.
func posOf(expr ast.Expression) ir.SourcePos {
	p := expr.Pos()
	return ir.SourcePos{Line: p.Line, File: p.File}
}

// clausePos returns the IR source position for an onerr clause.
func clausePos(clause *ast.OnErrClause) ir.SourcePos {
	if clause == nil {
		return ir.SourcePos{}
	}
	return ir.SourcePos{Line: clause.Token.Line, File: clause.Token.File}
}

// recordVar records a mapping from a generated temp variable name to a
// human-readable source description for enhanced debugging output.
func (l *Lowerer) recordVar(name string, expr ast.Expression) {
	if l.gen.varMap == nil {
		return
	}
	p := expr.Pos()
	desc := summarizeExpr(expr)
	if p.Line > 0 {
		l.gen.varMap[name] = fmt.Sprintf("line %d: %s", p.Line, desc)
	} else if desc != "" {
		l.gen.varMap[name] = desc
	}
}

// summarizeExpr returns a short human-readable description of an expression.
func summarizeExpr(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.CallExpr:
		return summarizeExpr(e.Function) + "(...)"
	case *ast.MethodCallExpr:
		if e.Object != nil {
			return summarizeExpr(e.Object) + "." + e.Method.Value + "(...)"
		}
		return e.Method.Value + "(...)"
	case *ast.Identifier:
		return e.Value
	case *ast.FieldAccessExpr:
		if e.Object != nil {
			return summarizeExpr(e.Object) + "." + e.Field.Value
		}
		return e.Field.Value
	case *ast.PipeExpr:
		return summarizeExpr(e.Left) + " |> " + summarizeExpr(e.Right)
	}
	return ""
}

func (l *Lowerer) uniqueId(prefix string) string {
	// Delegate to the generator's counter so variable names are identical
	// to the old direct-emission code path.
	return l.gen.uniqueId(prefix)
}

// ---------- Phase 1: pipe chain lowering (no onerr) ----------

// lowerPipeChain lowers a pipe chain into a sequence of temp variable
// assignments. Returns the IR block and the name of the final temp variable.
func (l *Lowerer) lowerPipeChain(pipe *ast.PipeExpr) (*ir.Block, string) {
	base, steps, ok := flattenPipeChain(pipe)
	if !ok || base == nil || len(steps) == 0 {
		return nil, ""
	}

	block := &ir.Block{}
	current := l.uniqueId("pipe")
	baseExpr := l.gen.exprToString(base)

	if count, ok := l.gen.inferReturnCount(base); ok && count >= 2 {
		// Multi-return base: wrap in IIFE to take first value only
		blanks := make([]string, count-1)
		for i := range blanks {
			blanks[i] = "_"
		}
		retType := l.gen.inferExprReturnType(base)
		if retType == "" {
			retType = "any"
		}
		baseExpr = fmt.Sprintf("func() %s { val, %s := %s; return val }()", retType, strings.Join(blanks, ", "), baseExpr)
	}

	l.recordVar(current, base)
	block.Add(&ir.Assign{Names: []string{current}, Expr: baseExpr, Walrus: true, Pos: posOf(base)})

	for _, step := range steps {
		callExpr, ok := l.gen.generatePipedStepCall(step, current)
		if !ok {
			return nil, ""
		}

		next := l.uniqueId("pipe")
		if count, ok := l.gen.inferReturnCount(step); ok && count >= 2 {
			blanks := make([]string, count-1)
			for i := range blanks {
				blanks[i] = "_"
			}
			retType := l.gen.inferExprReturnType(step)
			if retType == "" {
				retType = "any"
			}
			callExpr = fmt.Sprintf("func() %s { val, %s := %s; return val }()", retType, strings.Join(blanks, ", "), callExpr)
		}

		l.recordVar(next, step)
		block.Add(&ir.Assign{Names: []string{next}, Expr: callExpr, Walrus: true, Pos: posOf(step)})
		current = next
	}

	return block, current
}

// ---------- Phase 2: onerr on simple (non-pipe) expressions ----------

// lowerOnErr produces IR for a single expression + onerr clause.
// names are the LHS variable names (empty for statement-level).
// returnCount is the number of values returned by the expression.
func (l *Lowerer) lowerOnErr(expr string, clause *ast.OnErrClause, names []string, walrus bool) *ir.Block {
	block := &ir.Block{}
	errVar := l.uniqueId("err")
	cp := clausePos(clause)

	if walrus {
		lhs := append(append([]string{}, names...), errVar)
		block.Add(&ir.Assign{Names: lhs, Expr: expr, Walrus: true})
	} else {
		block.Add(&ir.VarDecl{Name: errVar, Type: "error"})
		lhs := append(append([]string{}, names...), errVar)
		block.Add(&ir.Assign{Names: lhs, Expr: expr, Walrus: false})
	}

	handlerBlock := l.lowerOnErrHandler(clause, names, errVar)
	block.Add(&ir.IfErrCheck{ErrVar: errVar, Body: handlerBlock, Pos: cp})

	return block
}

// lowerOnErrHandler produces the IR body for an if-err-check block.
func (l *Lowerer) lowerOnErrHandler(clause *ast.OnErrClause, names []string, errVar string) *ir.Block {
	body := &ir.Block{}
	cp := clausePos(clause)

	if clause.ShorthandReturn {
		// onerr return (bare) — propagate error with zero values
		body.AddAll(l.buildReturnNode(errVar, cp))
		return body
	}

	if clause.ShorthandContinue {
		body.Add(&ir.ExprStmt{Expr: "continue", Pos: cp})
		return body
	}

	if clause.ShorthandBreak {
		body.Add(&ir.ExprStmt{Expr: "break", Pos: cp})
		return body
	}

	// explain wrapping
	if clause.Explain != "" {
		l.gen.addImport("fmt")
		body.Add(&ir.Assign{
			Names:  []string{errVar},
			Expr:   fmt.Sprintf(`fmt.Errorf("%s: %%w", %s)`, clause.Explain, errVar),
			Walrus: false,
			Pos:    cp,
		})
		if clause.Handler == nil {
			// Standalone explain: emit return
			body.AddAll(l.buildReturnNode(errVar, cp))
			return body
		}
	}

	if clause.Handler == nil {
		return body
	}

	// Render the handler using the existing codegen method via RawStmt capture.
	body.Add(&ir.RawStmt{Code: l.renderHandler(clause, names, errVar), Pos: cp})
	return body
}

// renderHandler captures the output of generateOnErrHandler into a string.
// It is the single point that sets currentOnErrVar and currentOnErrAlias,
// ensuring exprToString resolves "error" / alias identifiers to errVar
// during handler rendering.
func (l *Lowerer) renderHandler(clause *ast.OnErrClause, names []string, errVar string) string {
	// Save and restore generator state.
	savedOutput := l.gen.output
	l.gen.output = strings.Builder{}
	savedIndent := l.gen.indent
	l.gen.indent = 0

	prevOnErrVar := l.gen.currentOnErrVar
	l.gen.currentOnErrVar = errVar
	prevAlias := l.gen.currentOnErrAlias
	l.gen.currentOnErrAlias = clause.Alias

	idents := make([]*ast.Identifier, len(names))
	for i, n := range names {
		idents[i] = &ast.Identifier{Value: n}
	}
	l.gen.generateOnErrHandler(idents, clause.Handler, errVar)

	l.gen.currentOnErrVar = prevOnErrVar
	l.gen.currentOnErrAlias = prevAlias

	result := strings.TrimRight(l.gen.output.String(), "\n")
	l.gen.output = savedOutput
	l.gen.indent = savedIndent
	return result
}

// ---------- Phase 3: onerr pipe chains ----------

// lowerOnErrPipeChain lowers a pipe chain with onerr into IR.
// Each error-returning step gets its own temp variable and error check.
// Non-error steps are collapsed into nested expressions to reduce temp
// variable clutter — e.g., `a |> toLower() |> parse() onerr panic` becomes
// `result, err := parse(toLower(a))` instead of three separate assignments.
// When targetName is non-empty, the last value-producing step assigns directly
// to that variable instead of a temp, eliminating the redundant final copy.
// Returns the IR block and the final expression (variable name or nested call).
func (l *Lowerer) lowerOnErrPipeChain(pipe *ast.PipeExpr, clause *ast.OnErrClause, names []string, targetName string) (*ir.Block, string) {
	base, steps, ok := flattenPipeChain(pipe)
	if !ok || base == nil || len(steps) == 0 {
		return nil, ""
	}

	// Pre-scan to find the last error-returning step index for targetName.
	// Only error-returning steps materialize to temps, so that's where
	// we apply the target name optimization.
	lastErrStep := -1
	if targetName != "" {
		for i, step := range steps {
			if count, ok := l.gen.inferReturnCount(step); ok && count >= 2 {
				lastErrStep = i
			}
		}
	}

	block := &ir.Block{}
	cp := clausePos(clause)

	// Start with the base as an expression string, not a temp variable.
	// Only materialize to a temp if the base is multi-return (needs error check).
	current := l.gen.exprToString(base)

	stepNum := 0
	addStepComment := func(block *ir.Block, expr ast.Expression) {
		stepNum++
		block.Add(&ir.Comment{Text: fmt.Sprintf("pipe step %d: %s", stepNum, summarizeExpr(expr))})
	}

	if count, ok := l.gen.inferReturnCount(base); ok && count >= 2 {
		tempVar := l.uniqueId("pipe")
		// If all steps are error-only (no error-returning steps), base is the
		// last value producer — use targetName if available.
		if targetName != "" && lastErrStep == -1 {
			hasValueStep := false
			for _, step := range steps {
				if !l.gen.isErrorOnlyReturn(step) {
					hasValueStep = true
					break
				}
			}
			if !hasValueStep {
				tempVar = targetName
			}
		}
		errVar := l.uniqueId("err")
		l.recordVar(tempVar, base)
		addStepComment(block, base)
		block.Add(&ir.Assign{Names: []string{tempVar, errVar}, Expr: current, Walrus: true, Pos: posOf(base)})
		handlerBlock := l.lowerOnErrHandler(clause, names, errVar)
		block.Add(&ir.IfErrCheck{ErrVar: errVar, Body: handlerBlock, Pos: cp})
		current = tempVar
	}

	for i, step := range steps {
		callExpr, ok := l.gen.generatePipedStepCall(step, current)
		if !ok {
			return nil, ""
		}

		sp := posOf(step)
		if count, ok := l.gen.inferReturnCount(step); ok && count >= 2 {
			// Error-returning step: must materialize to temp + error check.
			next := l.uniqueId("pipe")
			if targetName != "" && i == lastErrStep {
				next = targetName
			}
			errVar := l.uniqueId("err")
			l.recordVar(next, step)
			addStepComment(block, step)
			block.Add(&ir.Assign{Names: []string{next, errVar}, Expr: callExpr, Walrus: true, Pos: sp})
			handlerBlock := l.lowerOnErrHandler(clause, names, errVar)
			block.Add(&ir.IfErrCheck{ErrVar: errVar, Body: handlerBlock, Pos: cp})
			current = next
		} else if l.gen.isErrorOnlyReturn(step) {
			// Error-only step: check error, don't advance pipe value.
			errVar := l.uniqueId("err")
			addStepComment(block, step)
			block.Add(&ir.Assign{Names: []string{errVar}, Expr: callExpr, Walrus: true, Pos: sp})
			handlerBlock := l.lowerOnErrHandler(clause, names, errVar)
			block.Add(&ir.IfErrCheck{ErrVar: errVar, Body: handlerBlock, Pos: cp})
		} else {
			// Non-error step: collapse into nested expression (no temp variable).
			if l.gen.isUnknownSingleReturn(step) {
				l.gen.warn(step.Pos(), fmt.Sprintf("return type of '%s' is unknown; if it returns error, that error is unchecked in this pipe chain", summarizeExpr(step)))
			}
			current = callExpr
		}
	}

	return block, current
}

// lowerOnErrPipeChainWithLabels is like lowerOnErrPipeChain but uses goto
// for error handling instead of inline handlers (for piped switch support).
func (l *Lowerer) lowerOnErrPipeChainWithLabels(pipe *ast.PipeExpr, onErrLabel string, pipeErrVar string) (*ir.Block, string) {
	base, steps, ok := flattenPipeChain(pipe)
	if !ok {
		return nil, ""
	}

	block := &ir.Block{}
	gotoErrBlock := func(errVar string) *ir.Block {
		return &ir.Block{Nodes: []ir.Node{
			&ir.Assign{Names: []string{pipeErrVar}, Expr: errVar, Walrus: false},
			&ir.Goto{Label: onErrLabel},
		}}
	}

	stepNum := 0
	addStepComment := func(block *ir.Block, expr ast.Expression) {
		stepNum++
		block.Add(&ir.Comment{Text: fmt.Sprintf("pipe step %d: %s", stepNum, summarizeExpr(expr))})
	}

	// Start with the base as an expression, materialize only if multi-return.
	current := l.gen.exprToString(base)

	if count, ok := l.gen.inferReturnCount(base); ok && count >= 2 {
		tempVar := l.uniqueId("pipe")
		errVar := l.uniqueId("err")
		l.recordVar(tempVar, base)
		addStepComment(block, base)
		block.Add(&ir.Assign{Names: []string{tempVar, errVar}, Expr: current, Walrus: true, Pos: posOf(base)})
		block.Add(&ir.IfErrCheck{ErrVar: errVar, Body: gotoErrBlock(errVar)})
		current = tempVar
	}

	for _, step := range steps {
		callExpr, ok := l.gen.generatePipedStepCall(step, current)
		if !ok {
			return nil, ""
		}

		sp := posOf(step)
		if count, ok := l.gen.inferReturnCount(step); ok && count >= 2 {
			next := l.uniqueId("pipe")
			errVar := l.uniqueId("err")
			l.recordVar(next, step)
			addStepComment(block, step)
			block.Add(&ir.Assign{Names: []string{next, errVar}, Expr: callExpr, Walrus: true, Pos: sp})
			block.Add(&ir.IfErrCheck{ErrVar: errVar, Body: gotoErrBlock(errVar)})
			current = next
		} else if l.gen.isErrorOnlyReturn(step) {
			errVar := l.uniqueId("err")
			addStepComment(block, step)
			block.Add(&ir.Assign{Names: []string{errVar}, Expr: callExpr, Walrus: true, Pos: sp})
			block.Add(&ir.IfErrCheck{ErrVar: errVar, Body: gotoErrBlock(errVar)})
		} else {
			// Non-error step: collapse into nested expression.
			if l.gen.isUnknownSingleReturn(step) {
				l.gen.warn(step.Pos(), fmt.Sprintf("return type of '%s' is unknown; if it returns error, that error is unchecked in this pipe chain", summarizeExpr(step)))
			}
			current = callExpr
		}
	}

	return block, current
}

// ---------- Phase 4: piped switch with onerr ----------

// lowerPipedSwitchVarDecl produces IR for: result := A |> B() |> switch ... onerr handler
func (l *Lowerer) lowerPipedSwitchVarDecl(varName string, ps *ast.PipedSwitchExpr, clause *ast.OnErrClause, names []*ast.Identifier) *ir.Block {
	block := &ir.Block{}
	sp := posOf(ps.Left)

	returnType := l.gen.pipedSwitchReturnType(ps)
	if returnType == "" {
		l.gen.warn(ps.Pos(), "piped switch return type could not be inferred; generated as 'any' — ensure all branches return the same type")
		returnType = "any"
	}

	onErrLabel := l.uniqueId("onerr")
	endLabel := l.uniqueId("end")
	pipeErrVar := l.uniqueId("pipeErr")

	// Declare pipeErr variable to capture errors from pipe steps before jumping to onErrLabel
	block.Add(&ir.VarDecl{Name: pipeErrVar, Type: "error"})

	// Declare result variable, pre-initialized to handler default if available
	handlerDefault := l.gen.extractDefaultValue(clause, returnType)
	block.Add(&ir.VarDecl{Name: varName, Type: returnType, Value: handlerDefault, Pos: sp})

	// Build scoped block for pipe chain + switch
	inner := &ir.Block{}

	var finalPipeVar string
	if pipe, ok := ps.Left.(*ast.PipeExpr); ok {
		pipeBlock, pipeVar := l.lowerOnErrPipeChainWithLabels(pipe, onErrLabel, pipeErrVar)
		if pipeBlock == nil {
			return nil
		}
		inner.AddAll(pipeBlock)
		finalPipeVar = pipeVar
	} else {
		finalPipeVar = l.gen.exprToString(ps.Left)
	}

	// Render switch IIFE — temporarily bump indent to match the ScopedBlock
	// context during emission (generatePipedSwitchExpr bakes absolute
	// indentation into the IIFE string).
	savedIndent := l.gen.indent
	l.gen.indent++
	savedLeft := ps.Left
	ps.Left = &ast.Identifier{Value: finalPipeVar}
	switchStr := l.gen.generatePipedSwitchExpr(ps)
	ps.Left = savedLeft
	l.gen.indent = savedIndent

	inner.Add(&ir.Assign{Names: []string{varName}, Expr: switchStr, Walrus: false})
	inner.Add(&ir.Goto{Label: endLabel})

	block.Add(&ir.ScopedBlock{Body: inner})

	// Error handler label
	block.Add(&ir.Label{Name: onErrLabel})
	if handlerDefault == "" {
		handlerBlock := l.lowerOnErrHandler(clause, identNames(names), pipeErrVar)
		block.AddAll(handlerBlock)
	}

	block.Add(&ir.Label{Name: endLabel})
	return block
}

// lowerPipedSwitchStmt produces IR for statement-level: A |> B() |> switch ... onerr handler
func (l *Lowerer) lowerPipedSwitchStmt(ps *ast.PipedSwitchExpr, clause *ast.OnErrClause) *ir.Block {
	block := &ir.Block{}

	onErrLabel := l.uniqueId("onerr")
	endLabel := l.uniqueId("end")
	pipeErrVar := l.uniqueId("pipeErr")

	// Declare pipeErr variable to capture errors from pipe steps before jumping to onErrLabel
	block.Add(&ir.VarDecl{Name: pipeErrVar, Type: "error"})

	inner := &ir.Block{}

	var finalPipeVar string
	if pipe, ok := ps.Left.(*ast.PipeExpr); ok {
		pipeBlock, pipeVar := l.lowerOnErrPipeChainWithLabels(pipe, onErrLabel, pipeErrVar)
		if pipeBlock == nil {
			return nil
		}
		inner.AddAll(pipeBlock)
		finalPipeVar = pipeVar
	} else {
		finalPipeVar = l.gen.exprToString(ps.Left)
	}

	// Render switch statement at indent=0, RawStmt multi-line handling
	// adds the emitter's indent to each line.
	switchStr := l.renderSwitchStmt(ps, finalPipeVar)
	inner.Add(&ir.RawStmt{Code: switchStr})
	inner.Add(&ir.Goto{Label: endLabel})

	block.Add(&ir.ScopedBlock{Body: inner})

	block.Add(&ir.Label{Name: onErrLabel})
	handlerBlock := l.lowerOnErrHandler(clause, []string{finalPipeVar}, pipeErrVar)
	block.AddAll(handlerBlock)

	block.Add(&ir.Label{Name: endLabel})
	return block
}

// renderSwitchStmt captures the output of a switch statement into a string.
func (l *Lowerer) renderSwitchStmt(ps *ast.PipedSwitchExpr, finalPipeVar string) string {
	savedOutput := l.gen.output
	l.gen.output = strings.Builder{}
	savedIndent := l.gen.indent
	l.gen.indent = 0

	switch stmt := ps.Switch.(type) {
	case *ast.SwitchStmt:
		originalExpr := stmt.Expression
		stmt.Expression = &ast.Identifier{Value: finalPipeVar}
		l.gen.generateSwitchStmt(stmt)
		stmt.Expression = originalExpr
	case *ast.TypeSwitchStmt:
		originalExpr := stmt.Expression
		stmt.Expression = &ast.Identifier{Value: finalPipeVar}
		l.gen.generateTypeSwitchStmt(stmt)
		stmt.Expression = originalExpr
	}

	result := strings.TrimRight(l.gen.output.String(), "\n")
	l.gen.output = savedOutput
	l.gen.indent = savedIndent
	return result
}

// lowerOnErrStmt produces IR for a statement-level onerr (no named targets).
// It generates blank _ assignments for non-error return values.
func (l *Lowerer) lowerOnErrStmt(exprStr string, expr ast.Expression, clause *ast.OnErrClause) *ir.Block {
	block := &ir.Block{}
	errVar := l.uniqueId("err")
	cp := clausePos(clause)

	var blanks []string
	if count, ok := l.gen.inferReturnCount(expr); ok && count > 1 {
		for i := 0; i < count-1; i++ {
			blanks = append(blanks, "_")
		}
	} else if !ok {
		// Return count inference failed — emit a comment so the user knows.
		block.Add(&ir.Comment{Text: "kukicha: could not infer return count; use explicit capture if incorrect"})
	}

	lhs := append(blanks, errVar)
	block.Add(&ir.Assign{Names: lhs, Expr: exprStr, Walrus: true, Pos: posOf(expr)})

	handlerBlock := l.lowerOnErrHandler(clause, []string{}, errVar)
	block.Add(&ir.IfErrCheck{ErrVar: errVar, Body: handlerBlock, Pos: cp})

	return block
}

// lowerOnErrWithExplicitErr produces IR for onerr where the user provides
// the error variable as the last name (multi-return case).
// When the last name is "_", a unique error variable is generated instead,
// since Go does not allow reading the blank identifier as a value.
func (l *Lowerer) lowerOnErrWithExplicitErr(nameStrs []string, expr string, clause *ast.OnErrClause, walrus bool) *ir.Block {
	block := &ir.Block{}
	cp := clausePos(clause)
	errVar := nameStrs[len(nameStrs)-1]
	if errVar == "_" {
		errVar = l.uniqueId("err")
		nameStrs = append(nameStrs[:len(nameStrs)-1:len(nameStrs)-1], errVar)
	}
	block.Add(&ir.Assign{Names: nameStrs, Expr: expr, Walrus: walrus})
	handlerNames := nameStrs[:len(nameStrs)-1]
	handlerBlock := l.lowerOnErrHandler(clause, handlerNames, errVar)
	block.Add(&ir.IfErrCheck{ErrVar: errVar, Body: handlerBlock, Pos: cp})
	return block
}

// buildReturnNode creates an ir.Block with zero-value var declarations (if needed)
// followed by an ir.ReturnStmt with those values and errVar in the last position.
// Returns a block so that var _zeroN T declarations can precede the return statement.
func (l *Lowerer) buildReturnNode(errVar string, pos ir.SourcePos) *ir.Block {
	block := &ir.Block{}
	if len(l.gen.currentReturnTypes) == 0 {
		block.Add(&ir.ReturnStmt{Values: []string{errVar}, Pos: pos})
		return block
	}

	var parts []string
	for i, ret := range l.gen.currentReturnTypes {
		if i == len(l.gen.currentReturnTypes)-1 {
			parts = append(parts, errVar)
		} else {
			parts = append(parts, l.gen.zeroValueForType(ret))
		}
	}
	preDecls, parts := replaceGenericZeroExprs(parts)
	for _, typeDecl := range preDecls {
		// typeDecl is "var <name> <type>" — split into components for the IR node
		fields := strings.SplitN(typeDecl, " ", 3)
		block.Add(&ir.VarDecl{Name: fields[1], Type: fields[2]})
	}
	block.Add(&ir.ReturnStmt{Values: parts, Pos: pos})
	return block
}
