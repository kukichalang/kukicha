package semantic

import (
	"fmt"
	"strings"

	"github.com/kukichalang/kukicha/internal/ast"
	"github.com/kukichalang/kukicha/internal/lexer"
)

func (a *Analyzer) analyzeBlock(block *ast.BlockStmt) {
	for _, stmt := range block.Statements {
		a.analyzeStatement(stmt)
	}
}

func (a *Analyzer) analyzeStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.VarDeclStmt:
		a.analyzeVarDeclStmt(s)
		a.analyzeOnErrClause(s.OnErr)
		if s.OnErr == nil && len(s.Values) == 1 {
			a.warnPipeDiscardedErrors(s.Values[0])
		}
	case *ast.AssignStmt:
		a.analyzeAssignStmt(s)
		a.analyzeOnErrClause(s.OnErr)
		if s.OnErr == nil && len(s.Values) == 1 {
			a.warnPipeDiscardedErrors(s.Values[0])
		}
	case *ast.ReturnStmt:
		a.analyzeReturnStmt(s)
	case *ast.IfStmt:
		a.analyzeIfStmt(s)
	case *ast.SwitchStmt:
		a.analyzeSwitchStmt(s)
	case *ast.TypeSwitchStmt:
		a.analyzeTypeSwitchStmt(s)
	case *ast.ForRangeStmt:
		a.analyzeForRangeStmt(s)
	case *ast.ForNumericStmt:
		a.analyzeForNumericStmt(s)
	case *ast.ForConditionStmt:
		a.analyzeForConditionStmt(s)
	case *ast.DeferStmt:
		if s.Call != nil {
			a.analyzeExpression(s.Call)
		}
		if s.Block != nil {
			a.analyzeBlock(s.Block)
		}
	case *ast.GoStmt:
		if s.Call != nil {
			a.analyzeExpression(s.Call)
		}
		if s.Block != nil {
			a.analyzeBlock(s.Block)
		}
	case *ast.SendStmt:
		a.analyzeExpression(s.Value)
		a.analyzeExpression(s.Channel)
	case *ast.SelectStmt:
		a.switchDepth++ // reuse switchDepth so break works inside select
		defer func() { a.switchDepth-- }()
		for _, c := range s.Cases {
			a.symbolTable.EnterScope()
			if c.Recv != nil {
				a.analyzeExpression(c.Recv)
				for _, binding := range c.Bindings {
					sym := &Symbol{
						Name:    binding,
						Kind:    SymbolVariable,
						Type:    &TypeInfo{Kind: TypeKindUnknown},
						Defined: ast.Position{Line: c.Token.Line, Column: c.Token.Column, File: c.Token.File},
					}
					a.symbolTable.Define(sym)
				}
			}
			if c.Send != nil {
				a.analyzeExpression(c.Send.Value)
				a.analyzeExpression(c.Send.Channel)
			}
			a.analyzeBlock(c.Body)
			a.symbolTable.ExitScope()
		}
		if s.Otherwise != nil {
			a.analyzeBlock(s.Otherwise.Body)
		}
	case *ast.ExpressionStmt:
		a.analyzeExpression(s.Expression)
		a.analyzeOnErrClause(s.OnErr)
		if s.OnErr == nil {
			a.warnPipeDiscardedErrors(s.Expression)
		}
	case *ast.ContinueStmt:
		if a.loopDepth == 0 {
			a.error(s.Pos(), "continue statement outside of loop")
		}
	case *ast.BreakStmt:
		if a.loopDepth == 0 && a.switchDepth == 0 {
			a.error(s.Pos(), "break statement outside of loop")
		}
	case *ast.TypeDeclStmt:
		a.error(s.Pos(), "type declarations must be at the top level, not inside a function")
	}
}

// pipedSwitchConflict is a sentinel TypeInfo used to mark irreconcilable return
// types across piped switch cases.  It is distinct from TypeKindUnknown (which
// means "type not yet determined") so that a later case with a concrete type
// can still refine an Unknown, but cannot overwrite a confirmed conflict.
var pipedSwitchConflict = &TypeInfo{Kind: TypeKindNamed, Name: "any"}

func (a *Analyzer) mergePipedSwitchReturnType(inferred, candidate *TypeInfo) *TypeInfo {
	if candidate == nil {
		return inferred
	}
	if inferred == nil {
		return candidate
	}
	// Once a conflict is detected, keep it — no further refinement possible.
	if inferred == pipedSwitchConflict {
		return inferred
	}
	// Unknown inferred can be refined by a concrete candidate.
	if inferred.Kind == TypeKindUnknown {
		return candidate
	}
	if candidate.Kind == TypeKindUnknown {
		return inferred
	}
	if !a.typesCompatible(inferred, candidate) || !a.typesCompatible(candidate, inferred) {
		return pipedSwitchConflict
	}
	return inferred
}

// collectReturnTypes walks a block and collects the inferred type of the first
// return value in each ReturnStmt found at any nesting depth.
func (a *Analyzer) collectReturnTypes(body *ast.BlockStmt) *TypeInfo {
	if body == nil {
		return nil
	}
	var inferred *TypeInfo
	var walk func(stmts []ast.Statement)
	walk = func(stmts []ast.Statement) {
		for _, s := range stmts {
			if ret, ok := s.(*ast.ReturnStmt); ok && len(ret.Values) > 0 {
				retType := a.exprTypes[ret.Values[0]]
				inferred = a.mergePipedSwitchReturnType(inferred, retType)
			}
			if ifStmt, ok := s.(*ast.IfStmt); ok {
				if ifStmt.Consequence != nil {
					walk(ifStmt.Consequence.Statements)
				}
				switch alt := ifStmt.Alternative.(type) {
				case *ast.ElseStmt:
					if alt.Body != nil {
						walk(alt.Body.Statements)
					}
				case *ast.IfStmt:
					walk([]ast.Statement{alt})
				}
			}
		}
	}
	walk(body.Statements)
	return inferred
}

// analyzePipedSwitchBody fully analyzes each case/otherwise body of a piped
// switch (so that all expression types are recorded for codegen), then infers
// the value type from any return statements found in the bodies.
func (a *Analyzer) analyzePipedSwitchBody(stmt ast.PipedSwitchBody, leftType *TypeInfo) *TypeInfo {
	prev := a.inPipedSwitch
	a.inPipedSwitch = true
	defer func() { a.inPipedSwitch = prev }()

	var inferred *TypeInfo
	mergeBlock := func(body *ast.BlockStmt) {
		candidate := a.collectReturnTypes(body)
		inferred = a.mergePipedSwitchReturnType(inferred, candidate)
	}

	switch s := stmt.(type) {
	case *ast.SwitchStmt:
		for _, c := range s.Cases {
			a.analyzeBlock(c.Body)
			mergeBlock(c.Body)
		}
		if s.Otherwise != nil {
			a.analyzeBlock(s.Otherwise.Body)
			mergeBlock(s.Otherwise.Body)
		}
		// Exhaustiveness check for piped enum switches without otherwise
		if s.Otherwise == nil {
			a.checkEnumExhaustivenessFromType(leftType, s.Cases, s.Token.Line, s.Token.Column, s.Token.File)
		}
	case *ast.TypeSwitchStmt:
		for _, c := range s.Cases {
			a.symbolTable.EnterScope()
			bindingSymbol := &Symbol{
				Name:    s.Binding.Value,
				Kind:    SymbolVariable,
				Type:    a.typeAnnotationToTypeInfo(c.Type),
				Defined: s.Binding.Pos(),
			}
			a.symbolTable.Define(bindingSymbol)
			a.analyzeBlock(c.Body)
			mergeBlock(c.Body)
			a.symbolTable.ExitScope()
		}
		if s.Otherwise != nil {
			a.symbolTable.EnterScope()
			bindingSymbol := &Symbol{
				Name:    s.Binding.Value,
				Kind:    SymbolVariable,
				Type:    leftType,
				Defined: s.Binding.Pos(),
			}
			a.symbolTable.Define(bindingSymbol)
			a.analyzeBlock(s.Otherwise.Body)
			mergeBlock(s.Otherwise.Body)
			a.symbolTable.ExitScope()
		}
	}
	if inferred == nil {
		return &TypeInfo{Kind: TypeKindUnknown}
	}
	if inferred == pipedSwitchConflict {
		pos := stmt.(ast.Node).Pos()
		a.recordLint(LintTypeMismatch, pos, "piped switch cases return different types; result will be typed as 'any'")
	}
	return inferred
}

func (a *Analyzer) analyzeSwitchStmt(stmt *ast.SwitchStmt) {
	if stmt.Expression != nil {
		a.analyzeExpression(stmt.Expression)
	}

	a.switchDepth++
	defer func() { a.switchDepth-- }()

	for _, c := range stmt.Cases {
		for _, val := range c.Values {
			valType := a.analyzeExpression(val)
			if stmt.Expression == nil && valType != nil && valType.Kind != TypeKindBool && valType.Kind != TypeKindUnknown {
				a.error(val.Pos(), "switch condition branch must be bool")
			}
		}
		a.symbolTable.EnterScope()
		a.analyzeBlock(c.Body)
		a.symbolTable.ExitScope()
	}

	if stmt.Otherwise != nil {
		a.symbolTable.EnterScope()
		a.analyzeBlock(stmt.Otherwise.Body)
		a.symbolTable.ExitScope()
	}

	// Exhaustiveness check for enum switches without otherwise
	if stmt.Expression != nil && stmt.Otherwise == nil {
		a.checkEnumExhaustiveness(stmt.Expression, stmt.Cases, stmt.Token.Line, stmt.Token.Column, stmt.Token.File)
	}
}

func (a *Analyzer) analyzeTypeSwitchStmt(stmt *ast.TypeSwitchStmt) {
	exprType := a.analyzeExpression(stmt.Expression)

	a.switchDepth++
	defer func() { a.switchDepth-- }()

	for _, c := range stmt.Cases {
		// Define the binding variable in a new scope for each case body
		a.symbolTable.EnterScope()
		bindingSymbol := &Symbol{
			Name:    stmt.Binding.Value,
			Kind:    SymbolVariable,
			Type:    &TypeInfo{Kind: TypeKindUnknown},
			Defined: stmt.Binding.Pos(),
		}
		a.symbolTable.Define(bindingSymbol)
		a.analyzeBlock(c.Body)
		a.symbolTable.ExitScope()
	}

	if stmt.Otherwise != nil {
		a.symbolTable.EnterScope()
		bindingSymbol := &Symbol{
			Name:    stmt.Binding.Value,
			Kind:    SymbolVariable,
			Type:    &TypeInfo{Kind: TypeKindUnknown},
			Defined: stmt.Binding.Pos(),
		}
		a.symbolTable.Define(bindingSymbol)
		a.analyzeBlock(stmt.Otherwise.Body)
		a.symbolTable.ExitScope()
	}

	// Exhaustiveness check for variant enums when there is no otherwise clause
	if stmt.Otherwise == nil {
		a.checkVariantExhaustiveness(exprType, stmt.Cases, stmt.Token.Line, stmt.Token.Column, stmt.Token.File)
	}
}

// checkVariantExhaustiveness warns when a typed switch on a variant enum is missing cases.
func (a *Analyzer) checkVariantExhaustiveness(exprType *TypeInfo, cases []*ast.TypeCase, line, col int, file string) {
	if exprType == nil {
		return
	}

	// Resolve to the variant enum type
	var variantSym *Symbol
	if exprType.Kind == TypeKindVariant {
		variantSym = a.symbolTable.Resolve(exprType.Name)
	} else if exprType.Name != "" {
		sym := a.symbolTable.Resolve(exprType.Name)
		if sym != nil && sym.Type != nil && sym.Type.Kind == TypeKindVariant {
			variantSym = sym
		}
	}

	if variantSym == nil || variantSym.Type == nil || variantSym.Type.VariantCases == nil {
		return
	}

	// Collect covered case names from when clauses
	covered := make(map[string]bool)
	for _, c := range cases {
		typeName := typeCaseName(c.Type)
		if typeName != "" {
			covered[typeName] = true
		}
	}

	// Find missing cases
	var missing []string
	for caseName := range variantSym.Type.VariantCases {
		if !covered[caseName] {
			missing = append(missing, variantSym.Name+"."+caseName)
		}
	}

	if len(missing) > 0 {
		sortStrings(missing)
		pos := ast.Position{Line: line, Column: col, File: file}
		a.recordLint(LintEnum, pos, fmt.Sprintf("switch on %s is missing cases: %s", variantSym.Name, strings.Join(missing, ", ")))
	}
}

// typeCaseName extracts the base type name from a TypeAnnotation for a when clause.
// Returns "Circle" for both `when Circle` and `when reference Circle`.
func typeCaseName(t ast.TypeAnnotation) string {
	switch tp := t.(type) {
	case *ast.NamedType:
		return tp.Name
	case *ast.ReferenceType:
		return typeCaseName(tp.ElementType)
	}
	return ""
}

func (a *Analyzer) analyzeVarDeclStmt(stmt *ast.VarDeclStmt) {
	// Resolve untyped composite literals before analyzing values.
	if stmt.Type != nil {
		for _, val := range stmt.Values {
			a.resolveUntypedLiteral(val, stmt.Type)
		}
	}

	// Analyze all value expressions
	valueTypes := make([]*TypeInfo, len(stmt.Values))
	for i, val := range stmt.Values {
		valueTypes[i] = a.analyzeExpression(val)
	}

	// Special handling for multi-value return from single function call or type assertion
	var multiValueTypes []*TypeInfo
	if len(stmt.Values) == 1 && len(stmt.Names) > 1 {
		// Check if this is a type assertion (e.g., value, ok := expr as Type)
		if len(stmt.Names) == 2 {
			if typeCast, ok := stmt.Values[0].(*ast.TypeCastExpr); ok {
				// Type assertion returns (value, bool)
				targetType := a.typeAnnotationToTypeInfo(typeCast.TargetType)
				multiValueTypes = []*TypeInfo{
					targetType,
					{Kind: TypeKindBool},
				}
			} else {
				// Regular multi-value return
				multiValueTypes = a.analyzeExpressionMulti(stmt.Values[0])
			}
		} else {
			// Regular multi-value return
			multiValueTypes = a.analyzeExpressionMulti(stmt.Values[0])
		}

		if len(multiValueTypes) != len(stmt.Names) {
			// If we can't match exact count, check if it's dynamic/unknown
			if len(multiValueTypes) == 1 && multiValueTypes[0].Kind == TypeKindUnknown {
				// Allow assignment of Unknown to multiple variables
			} else {
				a.error(stmt.Pos(), fmt.Sprintf("assignment mismatch: %d variables but %d values", len(stmt.Names), len(multiValueTypes)))
			}
		}
	} else {
		// Check that number of values matches number of names
		if len(stmt.Values) != len(stmt.Names) {
			a.error(stmt.Pos(), fmt.Sprintf("assignment mismatch: %d variables but %d values", len(stmt.Names), len(stmt.Values)))
		}
	}

	// Type inference and validation
	for i, name := range stmt.Names {
		if !isValidIdentifier(name.Value) {
			a.error(name.Pos(), fmt.Sprintf("invalid variable name '%s'", name.Value))
			continue
		}

		// Determine the type for this variable
		var varType *TypeInfo
		if stmt.Type != nil {
			// Explicit type annotation applies to all variables
			a.validateTypeAnnotation(stmt.Type)
			varType = a.typeAnnotationToTypeInfo(stmt.Type)
		} else if len(stmt.Values) == len(stmt.Names) {
			// One value per variable: use corresponding value type
			varType = valueTypes[i]
		} else if len(stmt.Values) == 1 {
			// Single expression (likely multi-value function call)
			if multiValueTypes != nil {
				if i < len(multiValueTypes) {
					varType = multiValueTypes[i]
				} else if len(multiValueTypes) == 1 && multiValueTypes[0].Kind == TypeKindUnknown {
					varType = multiValueTypes[0]
				} else {
					varType = &TypeInfo{Kind: TypeKindUnknown}
				}
			} else {
				// Fallback (shouldn't happen with correct logic above)
				varType = valueTypes[0]
			}
		} else {
			varType = &TypeInfo{Kind: TypeKindUnknown}
		}

		// Check type compatibility if explicit type is specified
		if stmt.Type != nil && len(stmt.Values) == len(stmt.Names) {
			if !a.typesCompatible(varType, valueTypes[i]) {
				a.error(stmt.Pos(), fmt.Sprintf("cannot assign %s to %s", valueTypes[i], varType))
			}
		}

		// Add variable to symbol table
		symbol := &Symbol{
			Name:    name.Value,
			Kind:    SymbolVariable,
			Type:    varType,
			Defined: name.Pos(),
			Mutable: true,
		}
		if err := a.symbolTable.Define(symbol); err != nil {
			a.error(name.Pos(), err.Error())
		}
	}
}

func (a *Analyzer) analyzeAssignStmt(stmt *ast.AssignStmt) {
	// Check for reassignment to constants
	for _, target := range stmt.Targets {
		if ident, ok := target.(*ast.Identifier); ok && ident.Value != "_" {
			if sym := a.symbolTable.Resolve(ident.Value); sym != nil && sym.Kind == SymbolConst {
				a.error(ident.Pos(), fmt.Sprintf("cannot assign to constant '%s'", ident.Value))
			}
		}
	}

	// Analyze all target and value expressions
	targetTypes := make([]*TypeInfo, len(stmt.Targets))
	for i, target := range stmt.Targets {
		targetTypes[i] = a.analyzeExpression(target)
	}

	// Resolve untyped composite literals from target types.
	if len(stmt.Values) == len(stmt.Targets) {
		for i, val := range stmt.Values {
			if typeAnn := typeInfoToTypeAnnotation(targetTypes[i]); typeAnn != nil {
				a.resolveUntypedLiteral(val, typeAnn)
			}
		}
	}

	valueTypes := make([]*TypeInfo, len(stmt.Values))
	for i, val := range stmt.Values {
		valueTypes[i] = a.analyzeExpression(val)
	}

	if stmt.Token.Type == lexer.TOKEN_BIT_AND_ASSIGN {
		if len(stmt.Targets) != 1 || len(stmt.Values) != 1 {
			a.error(stmt.Pos(), "bitwise AND assignment requires a single target and a single value")
			return
		}
		if !isBitwiseType(targetTypes[0]) || !isBitwiseType(valueTypes[0]) {
			a.error(stmt.Pos(), fmt.Sprintf("bitwise AND assignment requires integer operands, got %s and %s", targetTypes[0], valueTypes[0]))
		}
	}

	// Special handling for multi-value return from single function call or type assertion
	var multiValueTypes []*TypeInfo
	if len(stmt.Values) == 1 && len(stmt.Targets) > 1 {
		// Check if this is a type assertion (e.g., value, ok := expr as Type)
		if len(stmt.Targets) == 2 {
			if typeCast, ok := stmt.Values[0].(*ast.TypeCastExpr); ok {
				// Type assertion returns (value, bool)
				targetType := a.typeAnnotationToTypeInfo(typeCast.TargetType)
				multiValueTypes = []*TypeInfo{
					targetType,
					{Kind: TypeKindBool},
				}
			} else {
				// Regular multi-value return
				multiValueTypes = a.analyzeExpressionMulti(stmt.Values[0])
			}
		} else {
			// Regular multi-value return
			multiValueTypes = a.analyzeExpressionMulti(stmt.Values[0])
		}

		if len(multiValueTypes) != len(stmt.Targets) {
			// If we can't match exact count, check if it's dynamic/unknown
			if len(multiValueTypes) == 1 && multiValueTypes[0].Kind == TypeKindUnknown {
				// Allow assignment of Unknown to multiple variables
			} else {
				a.error(stmt.Pos(), fmt.Sprintf("assignment mismatch: %d variables but %d values", len(stmt.Targets), len(multiValueTypes)))
				return
			}
		}

		// Check types for multi-value assignment
		for i := range stmt.Targets {
			var valType *TypeInfo
			if i < len(multiValueTypes) {
				valType = multiValueTypes[i]
			} else {
				valType = multiValueTypes[0] // Fallback for Unknown
			}

			if !a.typesCompatible(targetTypes[i], valType) {
				a.error(stmt.Pos(), fmt.Sprintf("cannot assign %s to %s", valType, targetTypes[i]))
			}
		}
		return
	}

	// Check that number of values matches number of targets
	if len(stmt.Values) != len(stmt.Targets) {
		a.error(stmt.Pos(), fmt.Sprintf("assignment mismatch: %d variables but %d values", len(stmt.Targets), len(stmt.Values)))
		return
	}

	// Type compatibility checking
	if len(stmt.Values) == len(stmt.Targets) {
		// One value per target: check each pair
		for i := range stmt.Targets {
			if !a.typesCompatible(targetTypes[i], valueTypes[i]) {
				a.error(stmt.Pos(), fmt.Sprintf("cannot assign %s to %s", valueTypes[i], targetTypes[i]))
			}
		}
	}
}

func (a *Analyzer) analyzeReturnStmt(stmt *ast.ReturnStmt) {
	if a.currentFunc == nil {
		a.error(stmt.Pos(), "return statement outside of function")
		return
	}

	// Inside piped switch bodies, return statements are IIFE returns (not function returns).
	// Analyze expressions for type recording but skip return-count/type validation.
	if a.inPipedSwitch {
		for _, v := range stmt.Values {
			a.analyzeExpression(v)
		}
		return
	}

	// Resolve untyped composite literals from return types.
	if len(stmt.Values) == len(a.currentFunc.Returns) {
		for i, val := range stmt.Values {
			a.resolveUntypedLiteral(val, a.currentFunc.Returns[i])
		}
	}

	// Special handling for multi-value return from single expression (e.g., pipe expression)
	var valueTypes []*TypeInfo
	if len(stmt.Values) == 1 && len(a.currentFunc.Returns) > 1 {
		valueTypes = a.analyzeExpressionMulti(stmt.Values[0])

		if len(valueTypes) != len(a.currentFunc.Returns) {
			// If we can't match exact count, check if it's dynamic/unknown
			if len(valueTypes) == 1 && valueTypes[0].Kind == TypeKindUnknown {
				// Allow return of Unknown to multiple return positions
			} else {
				a.error(stmt.Pos(), fmt.Sprintf("expected %d return values, got %d", len(a.currentFunc.Returns), len(valueTypes)))
				return
			}
		}

		// Check types for multi-value return
		for i := range a.currentFunc.Returns {
			var valType *TypeInfo
			if i < len(valueTypes) {
				valType = valueTypes[i]
			} else {
				valType = valueTypes[0] // Fallback for Unknown
			}
			expectedType := a.typeAnnotationToTypeInfo(a.currentFunc.Returns[i])

			if !a.typesCompatible(expectedType, valType) {
				a.error(stmt.Pos(), fmt.Sprintf("cannot return %s as %s", valType, expectedType))
			}
		}
		return
	}

	// Check return value count
	if len(stmt.Values) != len(a.currentFunc.Returns) {
		a.error(stmt.Pos(), fmt.Sprintf("expected %d return values, got %d", len(a.currentFunc.Returns), len(stmt.Values)))
		return
	}

	// Check return value types
	for i, value := range stmt.Values {
		valueType := a.analyzeExpression(value)
		expectedType := a.typeAnnotationToTypeInfo(a.currentFunc.Returns[i])

		if !a.typesCompatible(expectedType, valueType) {
			a.error(stmt.Pos(), fmt.Sprintf("cannot return %s as %s", valueType, expectedType))
		}
	}
}

func (a *Analyzer) analyzeIfStmt(stmt *ast.IfStmt) {
	a.symbolTable.EnterScope()
	defer a.symbolTable.ExitScope()

	if stmt.Init != nil {
		a.analyzeStatement(stmt.Init)
	}

	// Detect `if VALUE is CaseName as v` — only the top-level IsExpr in the
	// condition gets a binding scoped to the consequence block. Nested forms
	// (e.g. inside `and`/`or`) are rejected by analyzeIsExpr.
	isBinding, _ := stmt.Condition.(*ast.IsExpr)

	// Permit the `as v` binding only for the direct top-level IsExpr.
	savedAllow := a.allowIsBinding
	if isBinding != nil {
		a.allowIsBinding = true
	}
	condType := a.analyzeExpression(stmt.Condition)
	a.allowIsBinding = savedAllow
	if condType.Kind != TypeKindBool && condType.Kind != TypeKindUnknown {
		a.error(stmt.Condition.Pos(), "if condition must be boolean")
	}

	// Analyze consequence
	a.symbolTable.EnterScope()
	if isBinding != nil && isBinding.Binding != nil {
		a.defineIsExprBinding(isBinding)
	}
	a.analyzeBlock(stmt.Consequence)
	a.symbolTable.ExitScope()

	// Analyze alternative
	if stmt.Alternative != nil {
		a.symbolTable.EnterScope()
		switch alt := stmt.Alternative.(type) {
		case *ast.ElseStmt:
			a.analyzeBlock(alt.Body)
		case *ast.IfStmt:
			a.analyzeIfStmt(alt)
		}
		a.symbolTable.ExitScope()
	}
}

// defineIsExprBinding defines the `as v` binding for an `if EXPR is Case as v`
// condition in the current scope. The binding's type is the variant case
// struct so field access works naturally in the consequence block.
func (a *Analyzer) defineIsExprBinding(expr *ast.IsExpr) {
	if expr.Binding == nil || expr.Case == nil {
		return
	}

	valueType := a.exprTypes[expr.Value]
	var variantType *TypeInfo
	if valueType != nil {
		switch valueType.Kind {
		case TypeKindVariant:
			variantType = valueType
		case TypeKindNamed:
			if sym := a.symbolTable.Resolve(valueType.Name); sym != nil && sym.Type != nil && sym.Type.Kind == TypeKindVariant {
				variantType = sym.Type
			}
		}
	}

	// Default to Unknown so downstream uses don't cascade errors when the
	// type couldn't be resolved (analyzeIsExpr already reported the root cause).
	bindingType := &TypeInfo{Kind: TypeKindUnknown}
	if variantType != nil {
		if caseType, ok := variantType.VariantCases[expr.Case.Value]; ok && caseType != nil {
			bindingType = caseType
		}
	}

	sym := &Symbol{
		Name:    expr.Binding.Value,
		Kind:    SymbolVariable,
		Type:    bindingType,
		Defined: expr.Binding.Pos(),
	}
	if err := a.symbolTable.Define(sym); err != nil {
		a.error(expr.Binding.Pos(), err.Error())
	}
	// Record the binding identifier's type so codegen can look it up if needed.
	a.recordType(expr.Binding, bindingType)
}

func (a *Analyzer) analyzeForRangeStmt(stmt *ast.ForRangeStmt) {
	a.loopDepth++
	defer func() { a.loopDepth-- }()

	// Analyze collection
	collType := a.analyzeExpression(stmt.Collection)

	a.symbolTable.EnterScope()
	defer a.symbolTable.ExitScope()

	// Determine loop variable types from collection type
	var indexType, elemType *TypeInfo
	if collType.Kind == TypeKindMap {
		// for key, value in map: key is KeyType, value is ValueType
		if collType.KeyType != nil {
			indexType = collType.KeyType
		} else {
			indexType = &TypeInfo{Kind: TypeKindUnknown}
		}
		if collType.ValueType != nil {
			elemType = collType.ValueType
		} else {
			elemType = &TypeInfo{Kind: TypeKindUnknown}
		}
	} else {
		// for index, elem in list/string/channel: index is int
		indexType = &TypeInfo{Kind: TypeKindInt}
		if collType.Kind == TypeKindList && collType.ElementType != nil {
			elemType = collType.ElementType
		} else if collType.Kind == TypeKindString {
			elemType = &TypeInfo{Kind: TypeKindInt} // rune
		} else {
			elemType = &TypeInfo{Kind: TypeKindUnknown}
		}
	}

	// Add loop variables to scope
	if stmt.Index != nil {
		indexSymbol := &Symbol{
			Name:    stmt.Index.Value,
			Kind:    SymbolVariable,
			Type:    indexType,
			Defined: stmt.Index.Pos(),
			Mutable: true,
		}
		a.symbolTable.Define(indexSymbol)
	}

	varSymbol := &Symbol{
		Name:    stmt.Variable.Value,
		Kind:    SymbolVariable,
		Type:    elemType,
		Defined: stmt.Variable.Pos(),
		Mutable: true,
	}
	a.symbolTable.Define(varSymbol)

	// Analyze body
	a.analyzeBlock(stmt.Body)
}

func (a *Analyzer) analyzeForNumericStmt(stmt *ast.ForNumericStmt) {
	a.loopDepth++
	defer func() { a.loopDepth-- }()

	// Analyze start and end expressions
	startType := a.analyzeExpression(stmt.Start)
	endType := a.analyzeExpression(stmt.End)

	if startType.Kind != TypeKindInt && startType.Kind != TypeKindUnknown {
		a.error(stmt.Pos(), "for loop start must be int")
	}
	if endType.Kind != TypeKindInt && endType.Kind != TypeKindUnknown {
		a.error(stmt.Pos(), "for loop end must be int")
	}

	a.symbolTable.EnterScope()
	defer a.symbolTable.ExitScope()

	// Add loop variable to scope
	varSymbol := &Symbol{
		Name:    stmt.Variable.Value,
		Kind:    SymbolVariable,
		Type:    &TypeInfo{Kind: TypeKindInt},
		Defined: stmt.Variable.Pos(),
		Mutable: true,
	}
	a.symbolTable.Define(varSymbol)

	// Analyze body
	a.analyzeBlock(stmt.Body)
}

func (a *Analyzer) analyzeForConditionStmt(stmt *ast.ForConditionStmt) {
	a.loopDepth++
	defer func() { a.loopDepth-- }()

	// Analyze condition
	condType := a.analyzeExpression(stmt.Condition)
	if condType.Kind != TypeKindBool && condType.Kind != TypeKindUnknown {
		a.error(stmt.Pos(), "for condition must be boolean")
	}

	a.symbolTable.EnterScope()
	defer a.symbolTable.ExitScope()

	// Analyze body
	a.analyzeBlock(stmt.Body)
}
