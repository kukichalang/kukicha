package semantic

import (
	"fmt"

	"github.com/kukichalang/kukicha/internal/ast"
)

func (a *Analyzer) analyzeExpression(expr ast.Expression) (result *TypeInfo) {
	if expr == nil {
		return &TypeInfo{Kind: TypeKindUnknown}
	}

	defer func() {
		a.recordType(expr, result)
	}()

	switch e := expr.(type) {
	case *ast.Identifier:
		return a.analyzeIdentifier(e)
	case *ast.IntegerLiteral:
		return &TypeInfo{Kind: TypeKindInt}
	case *ast.FloatLiteral:
		return &TypeInfo{Kind: TypeKindFloat}
	case *ast.StringLiteral:
		if e.Interpolated {
			a.analyzeStringInterpolation(e)
		}
		return &TypeInfo{Kind: TypeKindString}
	case *ast.BooleanLiteral:
		return &TypeInfo{Kind: TypeKindBool}
	case *ast.BinaryExpr:
		return a.analyzeBinaryExpr(e)
	case *ast.IsExpr:
		return a.analyzeIsExpr(e)
	case *ast.UnaryExpr:
		return a.analyzeUnaryExpr(e)
	case *ast.PipeExpr:
		return a.analyzePipeExpr(e)
	case *ast.PipedSwitchExpr:
		// Analyze the upstream pipe chain so call return counts and expression types
		// are populated for codegen. For the switch body, only analyze the return
		// expressions in each case — full statement analysis would misfire on the
		// bare switch / return-value checks that assume a function context.
		leftType := a.analyzeExpression(e.Left)
		return a.analyzePipedSwitchBody(e.Switch, leftType)
	case *ast.CallExpr:
		types := a.analyzeCallExpr(e, nil)
		if len(types) > 0 {
			return types[0]
		}
		return &TypeInfo{Kind: TypeKindUnknown}
	case *ast.MethodCallExpr:
		types := a.analyzeMethodCallExpr(e, nil)
		if len(types) > 0 {
			return types[0]
		}
		return &TypeInfo{Kind: TypeKindUnknown}
	case *ast.FieldAccessExpr:
		return a.analyzeFieldAccessExpr(e, nil)
	case *ast.IndexExpr:
		return a.analyzeIndexExpr(e)
	case *ast.SliceExpr:
		return a.analyzeSliceExpr(e)
	case *ast.ListLiteralExpr:
		return a.analyzeListLiteral(e)
	case *ast.EmptyExpr:
		if e.Type != nil {
			return a.typeAnnotationToTypeInfo(e.Type)
		}
		return &TypeInfo{Kind: TypeKindNil}
	case *ast.StructLiteralExpr:
		structType := a.typeAnnotationToTypeInfo(e.Type)

		// Resolve the struct's symbol to access its field definitions.
		var structFields map[string]*TypeInfo
		if structType.Kind == TypeKindNamed {
			if sym := a.symbolTable.Resolve(structType.Name); sym != nil && sym.Type != nil {
				structFields = sym.Type.Fields
			}
		}

		for _, field := range e.Fields {
			valueType := a.analyzeExpression(field.Value)

			if structFields != nil {
				fieldType, ok := structFields[field.Name.Value]
				if !ok {
					a.error(field.Name.Pos(), fmt.Sprintf("unknown field '%s' on struct '%s'", field.Name.Value, structType.Name))
				} else {
					// Record the field's resolved type and check value compatibility.
					a.recordType(field.Value, fieldType)
					if !a.typesCompatible(fieldType, valueType) {
						a.error(field.Name.Pos(), fmt.Sprintf("cannot use %s as %s in field '%s' of struct '%s'", valueType, fieldType, field.Name.Value, structType.Name))
					}
				}
			}
		}

		return structType
	case *ast.MakeExpr:
		return a.typeAnnotationToTypeInfo(e.Type)
	case *ast.ReceiveExpr:
		chanType := a.analyzeExpression(e.Channel)
		if chanType.Kind == TypeKindChannel && chanType.ElementType != nil {
			return chanType.ElementType
		}
		return &TypeInfo{Kind: TypeKindUnknown}
	case *ast.TypeCastExpr:
		// Analyze the expression being cast
		_ = a.analyzeExpression(e.Expression)
		// Return the target type
		return a.typeAnnotationToTypeInfo(e.TargetType)
	case *ast.FunctionLiteral:
		// Analyze function literal — parameters and body must be validated
		a.symbolTable.EnterScope()
		defer a.symbolTable.ExitScope()
		for _, param := range e.Parameters {
			if param.Type != nil {
				a.validateTypeAnnotation(param.Type)
			}
			paramSymbol := &Symbol{
				Name:    param.Name.Value,
				Kind:    SymbolParameter,
				Type:    a.typeAnnotationToTypeInfo(param.Type),
				Defined: param.Name.Pos(),
			}
			if err := a.symbolTable.Define(paramSymbol); err != nil {
				a.error(param.Name.Pos(), err.Error())
			}
		}
		for _, ret := range e.Returns {
			a.validateTypeAnnotation(ret)
		}
		if e.Body != nil {
			// Set currentFunc to a synthetic FunctionDecl so return
			// checking validates against the literal's return types.
			savedFunc := a.currentFunc
			a.currentFunc = &ast.FunctionDecl{
				Token:      e.Token,
				Parameters: e.Parameters,
				Returns:    e.Returns,
			}
			a.analyzeBlock(e.Body)
			a.currentFunc = savedFunc
		}
		return &TypeInfo{Kind: TypeKindUnknown}
	case *ast.ArrowLambda:
		// Analyze arrow lambda body — parameters must be in scope
		a.symbolTable.EnterScope()
		defer a.symbolTable.ExitScope()
		for _, param := range e.Parameters {
			if param.Type != nil {
				a.validateTypeAnnotation(param.Type)
			}
			paramType := a.typeAnnotationToTypeInfo(param.Type)
			// If param has no explicit type annotation, check if semantic inference
			// (inferLambdaParamTypes) already recorded an inferred type for it.
			if paramType == nil || paramType.Kind == TypeKindUnknown {
				if inferred, ok := a.exprTypes[param.Name]; ok && inferred != nil && inferred.Kind != TypeKindUnknown {
					paramType = inferred
				}
			}
			paramSymbol := &Symbol{
				Name:    param.Name.Value,
				Kind:    SymbolParameter,
				Type:    paramType,
				Defined: param.Name.Pos(),
			}
			if err := a.symbolTable.Define(paramSymbol); err != nil {
				a.error(param.Name.Pos(), err.Error())
			}
		}
		// If we have an expected signature (recorded by analyzeCallExpr/analyzeMethodCallExpr),
		// set a.currentFunc for the duration of the body/block analysis so that
		// return statements inside the lambda are validated against its returns.
		savedFunc := a.currentFunc
		if signature, ok := a.exprTypes[e]; ok && signature != nil && signature.Kind == TypeKindFunction {
			syntheticFunc := &ast.FunctionDecl{
				Returns: make([]ast.TypeAnnotation, len(signature.Returns)),
			}
			for i, ret := range signature.Returns {
				// Create appropriate type annotation based on the kind
				var dummy ast.TypeAnnotation
				switch ret.Kind {
				case TypeKindInt:
					dummy = &ast.PrimitiveType{Name: "int"}
				case TypeKindFloat:
					dummy = &ast.PrimitiveType{Name: "float64"}
				case TypeKindString:
					dummy = &ast.PrimitiveType{Name: "string"}
				case TypeKindBool:
					dummy = &ast.PrimitiveType{Name: "bool"}
				default:
					// For named types, interfaces, etc., use NamedType
					dummy = &ast.NamedType{Name: ret.Name}
				}
				syntheticFunc.Returns[i] = dummy
			}
			a.currentFunc = syntheticFunc
		}

		var bodyType *TypeInfo
		if e.Body != nil {
			bodyType = a.analyzeExpression(e.Body)
		}
		if e.Block != nil {
			a.analyzeBlock(e.Block)
		}

		// Restore original function context
		a.currentFunc = savedFunc

		// Return a function type with the body's return type so callers can
		// resolve generic placeholders (e.g., "result" in concurrent.MapWithLimit).
		if bodyType != nil && bodyType.Kind != TypeKindUnknown {
			return &TypeInfo{Kind: TypeKindFunction, Returns: []*TypeInfo{bodyType}}
		}
		// If it's a block lambda with an expected signature, use that.
		if signature, ok := a.exprTypes[e]; ok && signature != nil {
			return signature
		}
		return &TypeInfo{Kind: TypeKindUnknown}
	case *ast.ReturnExpr:
		if !a.inOnerr {
			a.error(e.Pos(), "'return' expression is only valid inside an onerr handler")
		}
		for _, v := range e.Values {
			a.analyzeExpression(v)
		}
		return &TypeInfo{Kind: TypeKindUnknown}
	case *ast.ErrorExpr:
		if e.Message != nil {
			a.analyzeExpression(e.Message)
		}
		return &TypeInfo{Kind: TypeKindNamed, Name: "error"}
	case *ast.BlockExpr:
		a.analyzeBlock(e.Body)
		return &TypeInfo{Kind: TypeKindUnknown}
	case *ast.IfExpression:
		a.analyzeExpression(e.Condition)
		thenType := a.analyzeExpression(e.Then)
		a.analyzeExpression(e.Else)
		if thenType != nil {
			return thenType
		}
		return &TypeInfo{Kind: TypeKindUnknown}
	default:
		return &TypeInfo{Kind: TypeKindUnknown}
	}
}

// analyzeExpressionMulti analyzes an expression and returns all its values
// This is used for multi-value assignments (e.g., x, y := f())
func (a *Analyzer) analyzeExpressionMulti(expr ast.Expression) []*TypeInfo {
	if expr == nil {
		return []*TypeInfo{{Kind: TypeKindUnknown}}
	}

	switch e := expr.(type) {
	case *ast.CallExpr:
		return a.analyzeCallExpr(e, nil)
	case *ast.MethodCallExpr:
		return a.analyzeMethodCallExpr(e, nil)
	case *ast.FieldAccessExpr:
		return []*TypeInfo{a.analyzeFieldAccessExpr(e, nil)}
	case *ast.PipeExpr:
		return a.analyzePipeExprMulti(e)
	case *ast.IndexExpr:
		// Map index supports two-value form: val, ok := m[key]
		leftType := a.analyzeExpression(e.Left)
		if leftType.Kind == TypeKindMap || leftType.Kind == TypeKindUnknown {
			elemType := a.analyzeIndexExpr(e)
			return []*TypeInfo{elemType, {Kind: TypeKindBool}}
		}
		return []*TypeInfo{a.analyzeIndexExpr(e)}
	default:
		return []*TypeInfo{a.analyzeExpression(expr)}
	}
}

func (a *Analyzer) analyzeIdentifier(ident *ast.Identifier) *TypeInfo {
	// Check for builtin functions first
	if ident.Value == "print" {
		// print is a variadic builtin that accepts any types
		return &TypeInfo{
			Kind:     TypeKindFunction,
			Params:   []*TypeInfo{{Kind: TypeKindUnknown}},
			Variadic: true,
			Returns:  nil, // print doesn't return anything
		}
	}

	if ident.Value == "len" {
		// len is a builtin that returns int
		return &TypeInfo{
			Kind:     TypeKindFunction,
			Params:   []*TypeInfo{{Kind: TypeKindUnknown}}, // accepts any collection type
			Variadic: false,
			Returns:  []*TypeInfo{{Kind: TypeKindInt}},
		}
	}

	if ident.Value == "append" {
		// append is a variadic builtin
		return &TypeInfo{
			Kind:     TypeKindFunction,
			Params:   []*TypeInfo{{Kind: TypeKindUnknown}}, // slice and variadic elements
			Variadic: true,
			Returns:  []*TypeInfo{{Kind: TypeKindUnknown}}, // returns same type as input slice
		}
	}

	if ident.Value == "delete" {
		// delete(map, key) removes the key from the map; returns nothing
		return &TypeInfo{
			Kind:     TypeKindFunction,
			Params:   []*TypeInfo{{Kind: TypeKindUnknown}, {Kind: TypeKindUnknown}},
			Variadic: false,
			Returns:  nil,
		}
	}

	// "_" is the pipe placeholder; treat as unknown in all contexts.
	if ident.Value == "_" {
		return &TypeInfo{Kind: TypeKindUnknown}
	}

	// Check symbol table first — local variables/params shadow builtins
	symbol := a.symbolTable.Resolve(ident.Value)
	if symbol != nil {
		return symbol.Type
	}

	// empty keyword parsed as identifier (when used as argument)
	if ident.Value == "empty" {
		return &TypeInfo{Kind: TypeKindNil}
	}

	// min/max are builtins added in Go 1.21; allow them when not shadowed
	if ident.Value == "min" || ident.Value == "max" {
		return &TypeInfo{
			Kind:     TypeKindFunction,
			Params:   []*TypeInfo{{Kind: TypeKindUnknown}, {Kind: TypeKindUnknown}},
			Variadic: true,
			Returns:  []*TypeInfo{{Kind: TypeKindUnknown}},
		}
	}

	a.error(ident.Pos(), fmt.Sprintf("undefined identifier '%s'", ident.Value))
	return &TypeInfo{Kind: TypeKindUnknown}
}

func (a *Analyzer) analyzeBinaryExpr(expr *ast.BinaryExpr) *TypeInfo {
	leftType := a.analyzeExpression(expr.Left)
	rightType := a.analyzeExpression(expr.Right)

	switch expr.Operator {
	case "+":
		// String concatenation - allow Unknown on either side
		if (leftType.Kind == TypeKindString || leftType.Kind == TypeKindUnknown) &&
			(rightType.Kind == TypeKindString || rightType.Kind == TypeKindUnknown) &&
			(leftType.Kind == TypeKindString || rightType.Kind == TypeKindString) {
			return &TypeInfo{Kind: TypeKindString}
		}
		// Numeric addition
		if !isNumericType(leftType) || !isNumericType(rightType) {
			a.error(expr.Pos(), fmt.Sprintf("cannot apply %s to %s and %s", expr.Operator, leftType, rightType))
		}
		if leftType.Kind == TypeKindFloat || rightType.Kind == TypeKindFloat {
			return &TypeInfo{Kind: TypeKindFloat}
		}
		if leftType.Kind == TypeKindUnknown || rightType.Kind == TypeKindUnknown {
			return &TypeInfo{Kind: TypeKindUnknown}
		}
		return &TypeInfo{Kind: TypeKindInt}

	case "-", "*", "/", "%":
		// Arithmetic operators
		if !isNumericType(leftType) || !isNumericType(rightType) {
			a.error(expr.Pos(), fmt.Sprintf("cannot apply %s to %s and %s", expr.Operator, leftType, rightType))
		}
		// Special case: if one operand is a named type (like time.Duration), return that type for multiplication
		if expr.Operator == "*" {
			if leftType.Kind == TypeKindNamed && leftType.Name != "" {
				return leftType
			}
			if rightType.Kind == TypeKindNamed && rightType.Name != "" {
				return rightType
			}
		}
		// Result type is the wider of the two
		if leftType.Kind == TypeKindFloat || rightType.Kind == TypeKindFloat {
			return &TypeInfo{Kind: TypeKindFloat}
		}
		if leftType.Kind == TypeKindUnknown || rightType.Kind == TypeKindUnknown {
			return &TypeInfo{Kind: TypeKindUnknown}
		}
		return &TypeInfo{Kind: TypeKindInt}

	case "==", "!=", "<", ">", "<=", ">=", "equals", "not equals", "isnt":
		// Comparison operators
		if !a.typesCompatible(leftType, rightType) {
			a.error(expr.Pos(), fmt.Sprintf("cannot compare %s and %s", leftType, rightType))
		}
		return &TypeInfo{Kind: TypeKindBool}

	case "and", "or":
		// Logical operators - allow Unknown on either side (like 'not' operator does)
		leftOk := leftType.Kind == TypeKindBool || leftType.Kind == TypeKindUnknown
		rightOk := rightType.Kind == TypeKindBool || rightType.Kind == TypeKindUnknown
		if !leftOk || !rightOk {
			a.error(expr.Pos(), fmt.Sprintf("logical operator requires boolean operands, got %s and %s", leftType, rightType))
		}
		return &TypeInfo{Kind: TypeKindBool}

	case "&":
		if !isBitwiseType(leftType) || !isBitwiseType(rightType) {
			a.error(expr.Pos(), fmt.Sprintf("bitwise AND requires integer operands, got %s and %s", leftType, rightType))
		}
		return &TypeInfo{Kind: TypeKindInt}

	default:
		return &TypeInfo{Kind: TypeKindUnknown}
	}
}

func isBitwiseType(t *TypeInfo) bool {
	if t == nil {
		return false
	}
	return t.Kind == TypeKindInt || t.Kind == TypeKindUnknown
}

// analyzeIsExpr validates `EXPR is CaseName [as v]` — a variant enum case
// check. The value must be a variant enum, the case must belong to that
// enum. The optional binding variable is NOT defined here; analyzeIfStmt
// handles binding because `as v` is only valid in an `if` condition and the
// binding is scoped to the consequence block. An IsExpr with a binding in
// any other position is an error.
func (a *Analyzer) analyzeIsExpr(expr *ast.IsExpr) *TypeInfo {
	// A binding (`as v`) is only valid as the top-level `if` condition.
	// Disable the flag while analyzing sub-expressions so nested IsExprs can't
	// sneak in a binding (e.g. inside `and`/`or`).
	allowed := a.allowIsBinding
	a.allowIsBinding = false
	defer func() { a.allowIsBinding = allowed }()

	if expr.Binding != nil && !allowed {
		a.error(expr.Binding.Pos(), "'is' binding (`as v`) is only valid as the top-level condition of an `if` statement")
	}

	valueType := a.analyzeExpression(expr.Value)

	caseName := ""
	if expr.Case != nil {
		caseName = expr.Case.Value
	}

	// Resolve the variant type: either the expression's type is already a
	// variant, or it's a named type whose symbol is a variant.
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

	if variantType == nil {
		if valueType != nil && valueType.Kind != TypeKindUnknown {
			a.error(expr.Pos(), fmt.Sprintf("'is' requires a variant enum value, got %s", valueType))
		}
		return &TypeInfo{Kind: TypeKindBool}
	}

	if caseName == "" {
		return &TypeInfo{Kind: TypeKindBool}
	}

	if _, ok := variantType.VariantCases[caseName]; !ok {
		a.error(expr.Case.Pos(), fmt.Sprintf("'%s' is not a case of variant enum '%s'", caseName, variantType.Name))
	}

	return &TypeInfo{Kind: TypeKindBool}
}

func (a *Analyzer) analyzeUnaryExpr(expr *ast.UnaryExpr) *TypeInfo {
	rightType := a.analyzeExpression(expr.Right)

	switch expr.Operator {
	case "-":
		if !isNumericType(rightType) {
			a.error(expr.Pos(), "unary minus requires numeric type")
		}
		return rightType
	case "not":
		if rightType.Kind != TypeKindBool && rightType.Kind != TypeKindUnknown {
			a.error(expr.Pos(), "not operator requires boolean")
		}
		return &TypeInfo{Kind: TypeKindBool}
	default:
		return &TypeInfo{Kind: TypeKindUnknown}
	}
}

func (a *Analyzer) analyzePipeExpr(expr *ast.PipeExpr) *TypeInfo {
	types := a.analyzePipeExprMulti(expr)
	if len(types) > 0 {
		return types[0]
	}
	return &TypeInfo{Kind: TypeKindUnknown}
}

// analyzePipeExprMulti analyzes a pipe expression and returns all its values
// This handles cases like: return x |> f() where f() returns (T, error)
func (a *Analyzer) analyzePipeExprMulti(expr *ast.PipeExpr) []*TypeInfo {
	// Left side is piped as first argument to right side
	leftType := a.analyzeExpression(expr.Left)

	// Check for multiple placeholders
	checkPipeArgs := func(args []ast.Expression) {
		count := 0
		var pos ast.Position
		for _, arg := range args {
			if id, ok := arg.(*ast.Identifier); ok && id.Value == "_" {
				count++
				if count == 2 {
					pos = id.Pos()
				}
			} else if _, ok := arg.(*ast.DiscardExpr); ok {
				count++
				if count == 2 {
					pos = arg.Pos()
				}
			}
		}
		if count > 1 {
			a.error(pos, "pipe placeholder '_' may only appear once per step; use a temporary variable if you need the piped value in multiple positions")
		}
	}

	// Pass left type as piped argument to right side
	switch right := expr.Right.(type) {
	case *ast.CallExpr:
		checkPipeArgs(right.Arguments)
		types := a.analyzeCallExpr(right, leftType)
		a.recordReturnCount(expr, len(types))
		// Record type info on the step expression so codegen can detect
		// error-only returns in pipe chains (e.g., os.WriteFile returns only error).
		if len(types) > 0 {
			a.recordType(right, types[0])
		}
		return types
	case *ast.MethodCallExpr:
		checkPipeArgs(right.Arguments)
		types := a.analyzeMethodCallExpr(right, leftType)
		a.recordReturnCount(expr, len(types))
		if len(types) > 0 {
			a.recordType(right, types[0])
		}
		return types
	case *ast.FieldAccessExpr:
		fieldType := a.analyzeFieldAccessExpr(right, leftType)
		a.recordReturnCount(expr, 1)
		a.recordType(right, fieldType)
		return []*TypeInfo{fieldType}
	case *ast.PipeExpr:
		// Nested pipe: analyze recursively
		types := a.analyzePipeExprMulti(right)
		a.recordReturnCount(expr, len(types))
		return types
	default:
		// Fallback for other expressions
		types := []*TypeInfo{a.analyzeExpression(expr.Right)}
		a.recordReturnCount(expr, len(types))
		return types
	}
}

// warnPipeDiscardedErrors walks a pipe chain and warns when an intermediate
// step returns multiple values (typically (T, error)) but there is no onerr
// clause to handle the error.  Call this from analyzeStatement only when the
// enclosing statement has OnErr == nil.
func (a *Analyzer) warnPipeDiscardedErrors(expr ast.Expression) {
	pipe, ok := expr.(*ast.PipeExpr)
	if !ok {
		return
	}

	// Walk the left-spine of the pipe chain.
	// For  a |> b() |> c()  the AST is  Pipe(Pipe(a, b()), c()).
	// We check if any Left node has a recorded return count >= 2.
	for cur := pipe; cur != nil; {
		if count, recorded := a.exprReturnCounts[cur.Left]; recorded && count >= 2 {
			name := pipeStepName(cur.Left)
			a.recordLint(LintPipe, cur.Left.Pos(), fmt.Sprintf("pipe discards error from %s (add onerr to handle it)", name))
		}
		inner, ok := cur.Left.(*ast.PipeExpr)
		if !ok {
			break
		}
		cur = inner
	}
}

// pipeStepName returns a short human-readable label for a pipe step expression.
func pipeStepName(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.CallExpr:
		return callName(e.Function)
	case *ast.MethodCallExpr:
		if e.Object != nil {
			return callName(e.Object) + "." + e.Method.Value + "()"
		}
		return e.Method.Value + "()"
	case *ast.PipeExpr:
		// The pipe as a whole — name the last step in the sub-chain.
		return pipeStepName(e.Right)
	default:
		return expr.TokenLiteral()
	}
}

// callName extracts a readable name from a function expression.
func callName(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.Identifier:
		return e.Value + "()"
	case *ast.FieldAccessExpr:
		if e.Object != nil {
			return callName(e.Object) + "." + e.Field.Value
		}
		return e.Field.Value
	default:
		return expr.TokenLiteral()
	}
}

func (a *Analyzer) analyzeIndexExpr(expr *ast.IndexExpr) *TypeInfo {
	leftType := a.analyzeExpression(expr.Left)
	indexType := a.analyzeExpression(expr.Index)

	// Index must be int for lists
	if leftType.Kind == TypeKindList {
		if indexType.Kind != TypeKindInt && indexType.Kind != TypeKindUnknown {
			a.error(expr.Pos(), "list index must be int")
		}
		if leftType.ElementType != nil {
			return leftType.ElementType
		}
	}

	// For maps, validate key type
	if leftType.Kind == TypeKindMap {
		if leftType.KeyType != nil && !a.typesCompatible(leftType.KeyType, indexType) {
			a.error(expr.Pos(), fmt.Sprintf("cannot use %s as map key type %s", indexType, leftType.KeyType))
		}
		if leftType.ValueType != nil {
			return leftType.ValueType
		}
	}

	return &TypeInfo{Kind: TypeKindUnknown}
}

func (a *Analyzer) analyzeSliceExpr(expr *ast.SliceExpr) *TypeInfo {
	leftType := a.analyzeExpression(expr.Left)

	if expr.Start != nil {
		startType := a.analyzeExpression(expr.Start)
		if startType.Kind != TypeKindInt && startType.Kind != TypeKindUnknown {
			a.error(expr.Pos(), "slice start must be int")
		}
	}

	if expr.End != nil {
		endType := a.analyzeExpression(expr.End)
		if endType.Kind != TypeKindInt && endType.Kind != TypeKindUnknown {
			a.error(expr.Pos(), "slice end must be int")
		}
	}

	// Slicing a list returns the same list type
	return leftType
}

func (a *Analyzer) analyzeListLiteral(expr *ast.ListLiteralExpr) *TypeInfo {
	var elemType *TypeInfo

	// Use explicitly declared element type when present (e.g., list of Shape{...}).
	// This allows heterogeneous interface lists where elements have different concrete types.
	if expr.Type != nil {
		elemType = a.typeAnnotationToTypeInfo(expr.Type)
		for _, elem := range expr.Elements {
			a.analyzeExpression(elem)
		}
	} else if len(expr.Elements) > 0 {
		// Infer element type from first element
		elemType = a.analyzeExpression(expr.Elements[0])

		// Check all elements have compatible types
		for i, elem := range expr.Elements[1:] {
			et := a.analyzeExpression(elem)
			if !a.typesCompatible(elemType, et) {
				a.error(expr.Pos(), fmt.Sprintf("list element %d: incompatible type %s, expected %s", i+2, et, elemType))
			}
		}
	} else {
		elemType = &TypeInfo{Kind: TypeKindUnknown}
	}

	return &TypeInfo{
		Kind:        TypeKindList,
		ElementType: elemType,
	}
}
