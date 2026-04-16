package semantic

import (
	"fmt"
	"strings"

	"github.com/kukichalang/kukicha/internal/ast"
)

func (a *Analyzer) checkPackageName() {
	if a.program.PetioleDecl == nil {
		return
	}

	name := a.program.PetioleDecl.Name.Value

	// Kukicha's own stdlib packages are allowed to use Go stdlib names.
	// Detect stdlib packages by the "stdlib/" prefix in the source file path.
	if strings.Contains(a.sourceFile, "stdlib/") {
		return
	}

	// List of reserved Go standard library packages
	reservedPackages := map[string]bool{
		"bufio": true, "bytes": true, "context": true, "crypto": true,
		"database": true, "encoding": true, "errors": true, "flag": true,
		"fmt": true, "html": true, "image": true, "io": true,
		"iter": true, "log": true, "math": true, "mime": true,
		"net": true, "os": true, "path": true, "plugin": true,
		"reflect": true, "regexp": true, "runtime": true, "slices": true,
		"sort": true, "strconv": true, "strings": true, "sync": true,
		"syscall": true, "testing": true, "text": true, "time": true,
		"unicode": true, "unsafe": true,
	}

	if reservedPackages[name] {
		a.error(a.program.PetioleDecl.Pos(), fmt.Sprintf("package name '%s' conflicts with Go standard library package", name))
	}
}

func (a *Analyzer) checkSkillDecl() {
	skill := a.program.SkillDecl
	if skill == nil {
		return
	}

	// Skill name must be exported (start with uppercase)
	if skill.Name != nil && !isExported(skill.Name.Value) {
		a.error(skill.Name.Pos(), fmt.Sprintf("skill name '%s' must be exported (start with uppercase letter)", skill.Name.Value))
	}

	// The agentskills.io spec caps the derived (kebab-case) name at 64 chars.
	if skill.Name != nil {
		if derived := skillKebabName(skill.Name.Value); len(derived) > 64 {
			a.error(skill.Name.Pos(), fmt.Sprintf("skill name '%s' is too long (derived name '%s' is %d chars; max 64 per agentskills.io)", skill.Name.Value, derived, len(derived)))
		}
	}

	// Skill requires petiole (skills are packages, not main programs)
	if a.program.PetioleDecl == nil {
		a.error(skill.Pos(), "skill declaration requires a petiole declaration (skills are packages)")
	}

	// Warn if description is empty
	if skill.Description == "" {
		a.error(skill.Pos(), "skill should have a description (skills should be self-documenting)")
	}

	// The agentskills.io spec caps description at 1024 chars.
	if n := len(skill.Description); n > 1024 {
		a.error(skill.Pos(), fmt.Sprintf("skill description is too long (%d chars; max 1024 per agentskills.io)", n))
	}

	// Basic semver validation if version is provided
	if skill.Version != "" {
		if !isBasicSemver(skill.Version) {
			a.error(skill.Pos(), fmt.Sprintf("skill version '%s' should follow semver format (e.g., '1.0.0')", skill.Version))
		}
	}
}

// skillKebabName converts a Kukicha identifier to the kebab-case form used
// as the SKILL.md `name` field. Kept in sync with cmd/kukicha/pack.go's
// toKebabCase — acronym-aware ("HTTPClient" → "http-client").
func skillKebabName(s string) string {
	runes := []rune(s)
	var out []rune
	for i, r := range runes {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				prev := runes[i-1]
				prevLower := prev >= 'a' && prev <= 'z'
				prevUpper := prev >= 'A' && prev <= 'Z'
				nextLower := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'
				if prevLower || (prevUpper && nextLower) {
					out = append(out, '-')
				}
			}
			out = append(out, r+('a'-'A'))
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

// isBasicSemver performs a basic check that a version string looks semver-like
func isBasicSemver(v string) bool {
	// Allow optional leading 'v'
	s := strings.TrimPrefix(v, "v")
	parts := strings.Split(s, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return false
	}
	for _, part := range parts {
		if len(part) == 0 {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

// collectDeclarations collects all top-level declarations
func (a *Analyzer) collectDeclarations() {
	// Collect imports
	for _, imp := range a.program.Imports {
		path := imp.Path.Value
		if strings.ContainsAny(path, "\"\\\x00") {
			a.error(imp.Pos(), "import path must not contain '\"', '\\', or NUL")
			continue
		}
		name := a.extractPackageName(imp)
		err := a.symbolTable.Define(&Symbol{
			Name:    name,
			Kind:    SymbolVariable, // Treat as variable for now
			Type:    &TypeInfo{Kind: TypeKindUnknown},
			Defined: imp.Pos(),
		})
		if err != nil {
			a.error(imp.Pos(), err.Error())
		}
		// Track aliased imports so registry lookups can resolve aliases
		if imp.Alias != nil {
			baseName := extractBasePackageName(imp)
			if baseName != name {
				if a.importAliases == nil {
					a.importAliases = make(map[string]string)
				}
				a.importAliases[name] = baseName
			}
		}
	}

	for _, decl := range a.program.Declarations {
		switch d := decl.(type) {
		case *ast.TypeDecl:
			a.collectTypeDecl(d)
		case *ast.InterfaceDecl:
			a.collectInterfaceDecl(d)
		case *ast.FunctionDecl:
			a.collectFunctionDecl(d)
		case *ast.ConstDecl:
			a.collectConstDecl(d)
		case *ast.EnumDecl:
			a.collectEnumDecl(d)
		}
	}
}

func (a *Analyzer) collectConstDecl(decl *ast.ConstDecl) {
	for _, spec := range decl.Specs {
		if !isValidIdentifier(spec.Name.Value) {
			a.error(spec.Name.Pos(), fmt.Sprintf("invalid const name '%s'", spec.Name.Value))
			continue
		}
		err := a.symbolTable.Define(&Symbol{
			Name:    spec.Name.Value,
			Kind:    SymbolConst,
			Type:    &TypeInfo{Kind: TypeKindUnknown},
			Defined: spec.Name.Pos(),
		})
		if err != nil {
			a.error(spec.Name.Pos(), err.Error())
		}
	}
}

func (a *Analyzer) collectTypeDecl(decl *ast.TypeDecl) {
	// Check export rules: PascalCase = exported, camelCase = unexported
	if !isValidIdentifier(decl.Name.Value) {
		a.error(decl.Name.Pos(), fmt.Sprintf("invalid type name '%s'", decl.Name.Value))
		return
	}

	// Determine type kind based on alias vs struct
	typeKind := TypeKindStruct
	if decl.AliasType != nil {
		switch decl.AliasType.(type) {
		case *ast.FunctionType:
			typeKind = TypeKindFunction
		case *ast.ListType:
			typeKind = TypeKindList
		case *ast.MapType:
			typeKind = TypeKindMap
		case *ast.ChannelType:
			typeKind = TypeKindChannel
		case *ast.ReferenceType:
			typeKind = TypeKindReference
		case *ast.PrimitiveType:
			pt := decl.AliasType.(*ast.PrimitiveType)
			if info := primitiveTypeFromString(pt.Name); info != nil {
				typeKind = info.Kind
			} else {
				typeKind = TypeKindNamed
			}
		case *ast.NamedType:
			typeKind = TypeKindNamed
		default:
			typeKind = TypeKindUnknown
		}
	}

	// Build fields map for struct types so struct literals can validate field names.
	// typeAnnotationToTypeInfo is safe to call here (first pass) because it only
	// records names without resolving them — validation happens in analyzeTypeDecl.
	var fields map[string]*TypeInfo
	if decl.AliasType == nil && len(decl.Fields) > 0 {
		fields = make(map[string]*TypeInfo, len(decl.Fields))
		for _, f := range decl.Fields {
			fields[f.Name.Value] = a.typeAnnotationToTypeInfo(f.Type)
		}
	}

	// Add type to symbol table
	symbol := &Symbol{
		Name:     decl.Name.Value,
		Kind:     SymbolType,
		Type:     &TypeInfo{Kind: typeKind, Name: decl.Name.Value, Fields: fields},
		Defined:  decl.Name.Pos(),
		Exported: isExported(decl.Name.Value),
	}

	if err := a.symbolTable.Define(symbol); err != nil {
		a.error(decl.Name.Pos(), err.Error())
	}
}

func (a *Analyzer) collectInterfaceDecl(decl *ast.InterfaceDecl) {
	// Check export rules
	if !isValidIdentifier(decl.Name.Value) {
		a.error(decl.Name.Pos(), fmt.Sprintf("invalid interface name '%s'", decl.Name.Value))
		return
	}

	// Add interface to symbol table
	symbol := &Symbol{
		Name:     decl.Name.Value,
		Kind:     SymbolInterface,
		Type:     &TypeInfo{Kind: TypeKindInterface, Name: decl.Name.Value},
		Defined:  decl.Name.Pos(),
		Exported: isExported(decl.Name.Value),
	}

	if err := a.symbolTable.Define(symbol); err != nil {
		a.error(decl.Name.Pos(), err.Error())
	}
}

func (a *Analyzer) collectFunctionDecl(decl *ast.FunctionDecl) {
	// Check export rules
	if !isValidIdentifier(decl.Name.Value) {
		a.error(decl.Name.Pos(), fmt.Sprintf("invalid function name '%s'", decl.Name.Value))
		return
	}

	// Build function type
	params := make([]*TypeInfo, len(decl.Parameters))
	paramNames := make([]string, len(decl.Parameters))
	hasVariadic := false
	defaultCount := 0
	for i, param := range decl.Parameters {
		params[i] = a.typeAnnotationToTypeInfo(param.Type)
		paramNames[i] = param.Name.Value
		if param.Variadic {
			hasVariadic = true
		}
		if param.DefaultValue != nil {
			defaultCount++
		}
	}

	returns := make([]*TypeInfo, len(decl.Returns))
	for i, ret := range decl.Returns {
		returns[i] = a.typeAnnotationToTypeInfo(ret)
	}

	funcType := &TypeInfo{
		Kind:         TypeKindFunction,
		Params:       params,
		Returns:      returns,
		Variadic:     hasVariadic,
		ParamNames:   paramNames,
		DefaultCount: defaultCount,
	}

	// If this is a method (has receiver), register it on the receiver type
	if decl.Receiver != nil {
		a.registerMethod(decl, funcType)
		return
	}

	// Add function to symbol table
	symbol := &Symbol{
		Name:     decl.Name.Value,
		Kind:     SymbolFunction,
		Type:     funcType,
		Defined:  decl.Name.Pos(),
		Exported: isExported(decl.Name.Value),
	}

	if err := a.symbolTable.Define(symbol); err != nil {
		a.error(decl.Name.Pos(), err.Error())
	}
}

// registerMethod adds a method's type info to its receiver type's Methods map.
func (a *Analyzer) registerMethod(decl *ast.FunctionDecl, funcType *TypeInfo) {
	// Extract the receiver type name (strip "reference" if pointer receiver)
	typeName := ""
	switch rt := decl.Receiver.Type.(type) {
	case *ast.NamedType:
		typeName = rt.Name
	case *ast.PrimitiveType:
		typeName = rt.Name
	case *ast.ReferenceType:
		// reference Type → extract inner type name
		switch inner := rt.ElementType.(type) {
		case *ast.NamedType:
			typeName = inner.Name
		case *ast.PrimitiveType:
			typeName = inner.Name
		}
	}

	if typeName == "" {
		return
	}

	// Find the type symbol and attach the method
	sym := a.symbolTable.Resolve(typeName)
	if sym == nil || sym.Type == nil {
		return
	}

	if sym.Type.Methods == nil {
		sym.Type.Methods = make(map[string]*TypeInfo)
	}
	sym.Type.Methods[decl.Name.Value] = funcType
}

// analyzeDeclarations performs deep analysis of declarations
func (a *Analyzer) analyzeDeclarations() {
	for _, decl := range a.program.Declarations {
		switch d := decl.(type) {
		case *ast.TypeDecl:
			a.analyzeTypeDecl(d)
		case *ast.InterfaceDecl:
			a.analyzeInterfaceDecl(d)
		case *ast.FunctionDecl:
			a.analyzeFunctionDecl(d)
		case *ast.VarDeclStmt:
			a.analyzeGlobalVarDecl(d)
		case *ast.ConstDecl:
			a.analyzeConstDecl(d)
		case *ast.EnumDecl:
			a.analyzeEnumDecl(d)
		}
	}
}

func (a *Analyzer) analyzeConstDecl(decl *ast.ConstDecl) {
	for _, spec := range decl.Specs {
		a.analyzeExpression(spec.Value)
	}
}

func (a *Analyzer) analyzeTypeDecl(decl *ast.TypeDecl) {
	// Type alias: validate the alias type annotation
	if decl.AliasType != nil {
		a.validateTypeAnnotation(decl.AliasType)
		return
	}

	// Validate field types exist
	for _, field := range decl.Fields {
		if !isValidIdentifier(field.Name.Value) {
			a.error(field.Name.Pos(), fmt.Sprintf("invalid field name '%s'", field.Name.Value))
		}

		// Check that field type exists
		a.validateTypeAnnotation(field.Type)
	}
}

func (a *Analyzer) analyzeInterfaceDecl(decl *ast.InterfaceDecl) {
	// Validate method signatures
	for _, method := range decl.Methods {
		if !isValidIdentifier(method.Name.Value) {
			a.error(method.Name.Pos(), fmt.Sprintf("invalid method name '%s'", method.Name.Value))
		}

		// Validate parameter types
		for _, param := range method.Parameters {
			a.validateTypeAnnotation(param.Type)
		}

		// Validate return types
		for _, ret := range method.Returns {
			a.validateTypeAnnotation(ret)
		}
	}
}

func (a *Analyzer) analyzeGlobalVarDecl(stmt *ast.VarDeclStmt) {
	// Analyze values
	for _, val := range stmt.Values {
		a.analyzeExpression(val)
	}

	// Register each name in the global scope
	for i, name := range stmt.Names {
		if !isValidIdentifier(name.Value) {
			a.error(name.Pos(), fmt.Sprintf("invalid variable name '%s'", name.Value))
			continue
		}

		// Infer type from values or use explicit type
		var varType *TypeInfo
		if stmt.Type != nil {
			varType = a.typeAnnotationToTypeInfo(stmt.Type)
		} else if i < len(stmt.Values) {
			varType = a.analyzeExpression(stmt.Values[i])
		} else if len(stmt.Values) > 0 {
			varType = a.analyzeExpression(stmt.Values[0])
		} else {
			varType = &TypeInfo{Kind: TypeKindUnknown}
		}

		symbol := &Symbol{
			Name:     name.Value,
			Kind:     SymbolVariable,
			Type:     varType,
			Defined:  name.Pos(),
			Exported: isExported(name.Value),
		}

		if err := a.symbolTable.Define(symbol); err != nil {
			a.error(name.Pos(), err.Error())
		}
	}
}

func (a *Analyzer) analyzeFunctionDecl(decl *ast.FunctionDecl) {
	// Enter new scope for function
	a.symbolTable.EnterScope()
	defer a.symbolTable.ExitScope()

	// Track current function for return checking
	a.currentFunc = decl

	// Add receiver if present (for methods)
	if decl.Receiver != nil {
		a.validateTypeAnnotation(decl.Receiver.Type)

		receiverSymbol := &Symbol{
			Name:    decl.Receiver.Name.Value,
			Kind:    SymbolParameter,
			Type:    a.typeAnnotationToTypeInfo(decl.Receiver.Type),
			Defined: decl.Receiver.Name.Pos(),
		}
		if err := a.symbolTable.Define(receiverSymbol); err != nil {
			a.error(decl.Receiver.Name.Pos(), err.Error())
		}
	}

	// Validate variadic parameters (must be last, only one)
	variadicCount := 0
	for i, param := range decl.Parameters {
		if param.Variadic {
			variadicCount++
			if variadicCount > 1 {
				a.error(param.Name.Pos(), "only one variadic parameter allowed per function")
			}
			if i != len(decl.Parameters)-1 {
				a.error(param.Name.Pos(), "variadic parameter must be the last parameter")
			}
		}
	}

	// Add parameters to scope
	for _, param := range decl.Parameters {
		if !isValidIdentifier(param.Name.Value) {
			a.error(param.Name.Pos(), fmt.Sprintf("invalid parameter name '%s'", param.Name.Value))
		}

		a.validateTypeAnnotation(param.Type)

		paramType := a.typeAnnotationToTypeInfo(param.Type)
		// Variadic params are slices inside the function body (e.g., "many args string" → []string)
		if param.Variadic {
			paramType = &TypeInfo{Kind: TypeKindList, ElementType: paramType}
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

	// Validate return types exist
	for _, ret := range decl.Returns {
		a.validateTypeAnnotation(ret)
	}

	// Analyze function body
	if decl.Body != nil {
		a.analyzeBlock(decl.Body)
	}

	a.currentFunc = nil
}

func (a *Analyzer) collectEnumDecl(decl *ast.EnumDecl) {
	if !isValidIdentifier(decl.Name.Value) {
		a.error(decl.Name.Pos(), fmt.Sprintf("invalid enum name '%s'", decl.Name.Value))
		return
	}

	if len(decl.Cases) == 0 {
		a.error(decl.Pos(), fmt.Sprintf("enum '%s' must have at least one case", decl.Name.Value))
		return
	}

	if decl.IsVariant() {
		a.collectVariantEnumDecl(decl)
		return
	}

	// Value enum: validate first case value type
	switch decl.Cases[0].Value.(type) {
	case *ast.IntegerLiteral:
		// int-based enum
	case *ast.StringLiteral:
		// string-based enum
	default:
		a.error(decl.Cases[0].Value.Pos(), "enum case values must be integer or string literals")
		return
	}

	// Build enum cases map — each case maps to the enum type itself
	enumType := &TypeInfo{Kind: TypeKindEnum, Name: decl.Name.Value, EnumCases: make(map[string]*TypeInfo, len(decl.Cases))}
	for _, c := range decl.Cases {
		caseType := &TypeInfo{Kind: TypeKindEnum, Name: decl.Name.Value}
		enumType.EnumCases[c.Name.Value] = caseType
	}

	symbol := &Symbol{
		Name:     decl.Name.Value,
		Kind:     SymbolType,
		Type:     enumType,
		Defined:  decl.Name.Pos(),
		Exported: isExported(decl.Name.Value),
	}

	if err := a.symbolTable.Define(symbol); err != nil {
		a.error(decl.Name.Pos(), err.Error())
	}
}

func (a *Analyzer) collectVariantEnumDecl(decl *ast.EnumDecl) {
	// Validate: no case may have a value (= literal) — mixing is not allowed
	for _, c := range decl.Cases {
		if c.Value != nil {
			a.error(c.Value.Pos(), fmt.Sprintf("enum '%s': cannot mix value cases (= ...) and variant cases", decl.Name.Value))
			return
		}
	}

	// Build variant cases map — each case is a struct type
	variantType := &TypeInfo{
		Kind:         TypeKindVariant,
		Name:         decl.Name.Value,
		VariantCases: make(map[string]*TypeInfo, len(decl.Cases)),
	}

	for _, c := range decl.Cases {
		fields := make(map[string]*TypeInfo, len(c.Fields))
		for _, f := range c.Fields {
			fields[f.Name.Value] = a.typeAnnotationToTypeInfo(f.Type)
		}
		caseType := &TypeInfo{
			Kind:          TypeKindStruct,
			Name:          c.Name.Value,
			Fields:        fields,
			VariantParent: variantType,
		}
		variantType.VariantCases[c.Name.Value] = caseType

		// Register each case as a struct type so it resolves in expressions
		caseSym := &Symbol{
			Name:     c.Name.Value,
			Kind:     SymbolType,
			Type:     caseType,
			Defined:  c.Name.Pos(),
			Exported: isExported(c.Name.Value),
		}
		if err := a.symbolTable.Define(caseSym); err != nil {
			a.error(c.Name.Pos(), err.Error())
		}
	}

	symbol := &Symbol{
		Name:     decl.Name.Value,
		Kind:     SymbolType,
		Type:     variantType,
		Defined:  decl.Name.Pos(),
		Exported: isExported(decl.Name.Value),
	}

	if err := a.symbolTable.Define(symbol); err != nil {
		a.error(decl.Name.Pos(), err.Error())
	}
}

func (a *Analyzer) analyzeEnumDecl(decl *ast.EnumDecl) {
	if len(decl.Cases) == 0 {
		return
	}

	// Variant enums have no values to validate — nothing more to do here.
	if decl.IsVariant() {
		return
	}

	// Value enum: validate all cases have the same type
	var expectedKind string
	switch decl.Cases[0].Value.(type) {
	case *ast.IntegerLiteral:
		expectedKind = "integer"
	case *ast.StringLiteral:
		expectedKind = "string"
	}

	hasZero := false
	for _, c := range decl.Cases {
		if c.Value == nil {
			// Variant case mixed into a value enum
			a.error(c.Name.Pos(), fmt.Sprintf("enum '%s': cannot mix value cases (= ...) and variant cases", decl.Name.Value))
			continue
		}
		switch v := c.Value.(type) {
		case *ast.IntegerLiteral:
			if expectedKind != "integer" {
				a.error(c.Value.Pos(), fmt.Sprintf("enum '%s' mixes value types: expected %s, got integer", decl.Name.Value, expectedKind))
			}
			if v.Value == 0 {
				hasZero = true
			}
		case *ast.StringLiteral:
			if expectedKind != "string" {
				a.error(c.Value.Pos(), fmt.Sprintf("enum '%s' mixes value types: expected %s, got string", decl.Name.Value, expectedKind))
			}
		default:
			a.error(c.Value.Pos(), "enum case values must be integer or string literals")
		}
	}

	// Warn if integer enum has no case with value 0 (zero value of uninitialized variables)
	if expectedKind == "integer" && !hasZero {
		a.recordLint(LintEnum, decl.Pos(), fmt.Sprintf("enum %s has no case with value 0 — uninitialized variables will hold an invalid state", decl.Name.Value))
	}
}

// checkEnumExhaustiveness checks if a switch on an enum expression covers all cases.
// Called for non-piped switches where the expression is available.
func (a *Analyzer) checkEnumExhaustiveness(expr ast.Expression, cases []*ast.WhenCase, line, col int, file string) {
	exprType := a.exprTypes[expr]
	a.checkEnumExhaustivenessFromType(exprType, cases, line, col, file)
}

// checkEnumExhaustivenessFromType checks exhaustiveness given a resolved type.
// Used by both regular and piped switches.
func (a *Analyzer) checkEnumExhaustivenessFromType(exprType *TypeInfo, cases []*ast.WhenCase, line, col int, file string) {
	if exprType == nil {
		return
	}

	// Resolve to enum type — exprType might be the case type (TypeKindInt/TypeKindString with enum Name)
	var enumSym *Symbol
	if exprType.Kind == TypeKindEnum {
		enumSym = a.symbolTable.Resolve(exprType.Name)
	} else if exprType.Name != "" {
		sym := a.symbolTable.Resolve(exprType.Name)
		if sym != nil && sym.Type != nil && sym.Type.Kind == TypeKindEnum {
			enumSym = sym
		}
	}

	if enumSym == nil || enumSym.Type == nil || enumSym.Type.EnumCases == nil {
		return
	}

	// Collect all covered case names from when clauses
	covered := make(map[string]bool)
	for _, c := range cases {
		for _, val := range c.Values {
			if fa, ok := val.(*ast.FieldAccessExpr); ok {
				if ident, ok := fa.Object.(*ast.Identifier); ok && ident.Value == enumSym.Name {
					covered[fa.Field.Value] = true
				}
			}
		}
	}

	// Find missing cases
	var missing []string
	for caseName := range enumSym.Type.EnumCases {
		if !covered[caseName] {
			missing = append(missing, enumSym.Name+"."+caseName)
		}
	}

	if len(missing) > 0 {
		// Sort for deterministic output
		sortStrings(missing)
		pos := ast.Position{Line: line, Column: col, File: file}
		a.recordLint(LintEnum, pos, fmt.Sprintf("switch on %s is missing cases: %s", enumSym.Name, strings.Join(missing, ", ")))
	}
}

// sortStrings sorts a string slice in place (avoids importing sort).
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
