package parser

import (
	"fmt"

	"github.com/kukichalang/kukicha/internal/ast"
	"github.com/kukichalang/kukicha/internal/lexer"
)

// ============================================================================
// Declaration Parsing
// ============================================================================

func (p *Parser) parsePetioleDecl() *ast.PetioleDecl {
	token := p.advance() // consume 'petiole'
	p.skipNewlines()

	name := p.parseIdentifier()
	p.skipNewlines()

	return &ast.PetioleDecl{
		Token: token,
		Name:  name,
	}
}

func (p *Parser) parseSkillDecl() *ast.SkillDecl {
	token := p.advance() // consume 'skill'
	p.skipNewlines()

	name := p.parseIdentifier()

	decl := &ast.SkillDecl{
		Token: token,
		Name:  name,
	}

	p.skipNewlines()

	// Check for indented block with description/version fields
	if p.match(lexer.TOKEN_INDENT) {
		for !p.check(lexer.TOKEN_DEDENT) && !p.isAtEnd() {
			p.skipNewlines()
			if p.check(lexer.TOKEN_DEDENT) {
				break
			}

			// Parse field name (contextual identifier: "description" or "version")
			fieldToken := p.advance()
			if fieldToken.Type != lexer.TOKEN_IDENTIFIER {
				p.error(fieldToken, fmt.Sprintf("expected 'description' or 'version' in skill block, got %s", fieldToken.Type))
				p.skipNewlines()
				continue
			}

			p.consume(lexer.TOKEN_COLON, fmt.Sprintf("expected ':' after '%s'", fieldToken.Lexeme))

			// Parse string literal value
			valueToken := p.advance()
			if valueToken.Type != lexer.TOKEN_STRING {
				p.error(valueToken, fmt.Sprintf("expected string value for '%s'", fieldToken.Lexeme))
				p.skipNewlines()
				continue
			}

			switch fieldToken.Lexeme {
			case "description":
				decl.Description = valueToken.Lexeme
			case "version":
				decl.Version = valueToken.Lexeme
			default:
				p.error(fieldToken, fmt.Sprintf("unknown skill field '%s' (expected 'description' or 'version')", fieldToken.Lexeme))
			}

			p.skipNewlines()
		}

		p.consume(lexer.TOKEN_DEDENT, "expected dedent after skill block")
		p.skipNewlines()
	}

	return decl
}

func (p *Parser) parseImportDecl() *ast.ImportDecl {
	token := p.advance() // consume 'import'
	p.skipNewlines()
	return p.parseImportSpec(token)
}

// parseImportSpec parses a single import path + optional alias.
// Supports both Kukicha style ("path" as alias) and Go style (alias "path").
// importToken is the 'import' keyword token (used for error position only).
func (p *Parser) parseImportSpec(importToken lexer.Token) *ast.ImportDecl {
	// Go-style alias before path: import j "encoding/json"
	// Detect: next token is identifier (or _), token after that is string.
	if p.peekToken().Type == lexer.TOKEN_IDENTIFIER &&
		p.peekNextToken().Type == lexer.TOKEN_STRING {
		aliasIdent := p.parseIdentifier()
		pathToken := p.advance()
		decl := &ast.ImportDecl{
			Token: importToken,
			Path: &ast.StringLiteral{
				Token: pathToken,
				Value: pathToken.Lexeme,
			},
			Alias: aliasIdent,
		}
		return decl
	}

	pathToken := p.advance()
	if pathToken.Type != lexer.TOKEN_STRING {
		p.error(pathToken, "expected string literal for import path")
		return nil
	}

	decl := &ast.ImportDecl{
		Token: importToken,
		Path: &ast.StringLiteral{
			Token: pathToken,
			Value: pathToken.Lexeme,
		},
	}

	// Kukicha-style alias after path: "path" as alias
	if p.match(lexer.TOKEN_AS) {
		decl.Alias = p.parseIdentifier()
	}

	return decl
}

// parseGroupedImports parses: import ( "path1" \n "path2" ... )
// and returns a slice of ImportDecl, one per entry.
func (p *Parser) parseGroupedImports() []*ast.ImportDecl {
	importToken := p.advance() // consume 'import'
	p.skipGroupedImportWhitespace()
	p.consume(lexer.TOKEN_LPAREN, "expected '(' for grouped import")
	p.skipGroupedImportWhitespace()

	var decls []*ast.ImportDecl
	for !p.check(lexer.TOKEN_RPAREN) && !p.isAtEnd() {
		p.skipGroupedImportWhitespace()
		if p.check(lexer.TOKEN_RPAREN) {
			break
		}
		// Skip blank identifier imports (side-effect imports): _ "path"
		if decl := p.parseImportSpec(importToken); decl != nil {
			decls = append(decls, decl)
		}
		p.skipGroupedImportWhitespace()
	}

	p.consume(lexer.TOKEN_RPAREN, "expected ')' to close grouped import")
	p.skipNewlines()
	return decls
}

// skipGroupedImportWhitespace skips NEWLINE, INDENT, DEDENT, and SEMICOLON
// tokens that may appear between entries in a grouped import block.
func (p *Parser) skipGroupedImportWhitespace() {
	for {
		switch p.peekToken().Type {
		case lexer.TOKEN_NEWLINE, lexer.TOKEN_INDENT, lexer.TOKEN_DEDENT, lexer.TOKEN_SEMICOLON:
			p.advance()
		default:
			return
		}
	}
}

func (p *Parser) parseDeclaration() ast.Declaration {
	p.skipNewlines()

	// Drain any directives collected before this declaration.
	dirs := p.drainDirectives()

	var decl ast.Declaration
	switch p.peekToken().Type {
	case lexer.TOKEN_TYPE:
		decl = p.parseTypeDecl()
	case lexer.TOKEN_INTERFACE:
		decl = p.parseInterfaceDecl()
	case lexer.TOKEN_FUNC:
		decl = p.parseFunctionDecl()
	case lexer.TOKEN_VAR:
		decl = p.parseVarDeclaration()
	case lexer.TOKEN_CONST:
		decl = p.parseConstDecl()
	case lexer.TOKEN_ENUM:
		decl = p.parseEnumDecl()
	default:
		if !p.isAtEnd() {
			p.error(p.peekToken(), fmt.Sprintf("unexpected token %s, expected declaration", p.peekToken().Type))
			p.advance() // Skip the problematic token
		}
		return nil
	}

	// Attach directives to declarations that support them.
	if decl != nil && len(dirs) > 0 {
		switch d := decl.(type) {
		case *ast.FunctionDecl:
			d.Directives = dirs
		case *ast.TypeDecl:
			d.Directives = dirs
		case *ast.InterfaceDecl:
			d.Directives = dirs
		case *ast.EnumDecl:
			d.Directives = dirs
		}
	}

	return decl
}

func (p *Parser) parseTypeDecl() ast.Declaration {
	token := p.advance() // consume 'type'
	p.skipNewlines()

	name := p.parseIdentifier()
	p.skipNewlines()

	// Check for type alias: type Name func(...) ...
	if p.check(lexer.TOKEN_FUNC) {
		aliasType := p.parseTypeAnnotation()
		p.skipNewlines()
		return &ast.TypeDecl{
			Token:     token,
			Name:      name,
			AliasType: aliasType,
		}
	}

	fields := []*ast.FieldDecl{}

	// Expect INDENT for fields
	if !p.match(lexer.TOKEN_INDENT) {
		p.error(p.peekToken(), "expected indented block for type fields")
		return nil
	}

	// Parse fields
	for !p.check(lexer.TOKEN_DEDENT) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(lexer.TOKEN_DEDENT) {
			break
		}

		fieldName := p.parseIdentifier()
		fieldType := p.parseTypeAnnotation()
		alias := p.parseFieldAlias()

		// Parse optional struct tag (e.g., json:"name")
		tag := p.parseStructTag()
		if alias != "" && tag != "" {
			p.error(p.peekToken(), "cannot combine field alias and explicit struct tag on the same field")
		} else if alias != "" {
			tag = `json:"` + alias + `"`
		}

		fields = append(fields, &ast.FieldDecl{
			Name: fieldName,
			Type: fieldType,
			Tag:  tag,
		})
		p.skipNewlines()
	}

	p.consume(lexer.TOKEN_DEDENT, "expected dedent after type fields")
	p.skipNewlines()

	return &ast.TypeDecl{
		Token:  token,
		Name:   name,
		Fields: fields,
	}
}

func (p *Parser) parseInterfaceDecl() *ast.InterfaceDecl {
	token := p.advance() // consume 'interface'
	p.skipNewlines()

	name := p.parseIdentifier()
	p.skipNewlines()

	methods := []*ast.MethodSignature{}

	// Expect INDENT for methods
	if !p.match(lexer.TOKEN_INDENT) {
		p.error(p.peekToken(), "expected indented block for interface methods")
		return nil
	}

	// Parse method signatures
	for !p.check(lexer.TOKEN_DEDENT) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(lexer.TOKEN_DEDENT) {
			break
		}

		methodName := p.parseIdentifier()

		// Parse parameters
		p.consume(lexer.TOKEN_LPAREN, "expected '(' for method parameters")
		params := p.parseParameters()
		p.consume(lexer.TOKEN_RPAREN, "expected ')' after method parameters")

		// Parse return types
		returns := []ast.TypeAnnotation{}
		if !p.check(lexer.TOKEN_NEWLINE) && !p.check(lexer.TOKEN_DEDENT) {
			returns = p.parseReturnTypes()
		}

		methods = append(methods, &ast.MethodSignature{
			Name:       methodName,
			Parameters: params,
			Returns:    returns,
		})
		p.skipNewlines()
	}

	p.consume(lexer.TOKEN_DEDENT, "expected dedent after interface methods")
	p.skipNewlines()

	return &ast.InterfaceDecl{
		Token:   token,
		Name:    name,
		Methods: methods,
	}
}

// isGoStyleReceiver checks whether a `(` after `func` is a Go-style receiver
// like `func (r Type) Name(...)` rather than a parameter list. At declaration
// level, `func (` always means a receiver in Go, but we verify by checking
// that the tokens inside parens look like `identifier type` (not empty parens
// or a comma-separated parameter list used as a grouped return).
func (p *Parser) isGoStyleReceiver() bool {
	// peekAt(0) is '(', peekAt(1) should be receiver name (identifier)
	tok1 := p.peekAt(1)
	return tok1.Type == lexer.TOKEN_IDENTIFIER
}

func (p *Parser) parseFunctionDecl() *ast.FunctionDecl {
	token := p.advance() // consume 'func'
	p.skipNewlines()

	decl := &ast.FunctionDecl{
		Token: token,
	}

	// Go-style receiver: func (r Type) Name(...)
	if p.check(lexer.TOKEN_LPAREN) && p.isGoStyleReceiver() {
		p.advance() // consume '('
		receiverName := p.parseIdentifier()
		receiverType := p.parseTypeAnnotation()
		p.consume(lexer.TOKEN_RPAREN, "expected ')' after receiver")
		decl.Receiver = &ast.Receiver{
			Name: receiverName,
			Type: receiverType,
		}
		p.skipNewlines()
		decl.Name = p.parseIdentifier()
	} else {
		// Parse function name
		decl.Name = p.parseIdentifier()

		// Kukicha-style receiver: func Name on receiverName Type
		if p.match(lexer.TOKEN_ON) {
			receiverName := p.parseIdentifier()
			receiverType := p.parseTypeAnnotation()
			decl.Receiver = &ast.Receiver{
				Name: receiverName,
				Type: receiverType,
			}
		}
	}

	// Parse parameters (optional for methods with no parameters)
	if p.check(lexer.TOKEN_LPAREN) {
		p.advance() // consume '('
		decl.Parameters = p.parseParameters()
		p.consume(lexer.TOKEN_RPAREN, "expected ')' after function parameters")
	} else {
		decl.Parameters = []*ast.Parameter{}
	}

	// Parse return types
	if !p.check(lexer.TOKEN_NEWLINE) && !p.check(lexer.TOKEN_INDENT) {
		decl.Returns = p.parseReturnTypes()
	}

	p.skipNewlines()

	// Parse function body
	decl.Body = p.parseBlock()

	return decl
}

func (p *Parser) parseParameters() []*ast.Parameter {
	params := []*ast.Parameter{}
	hasDefaultValue := false // Track if we've seen a parameter with a default value

	if p.check(lexer.TOKEN_RPAREN) {
		return params
	}

	for {
		// Check for 'many' keyword (variadic parameter)
		variadic := false
		if p.check(lexer.TOKEN_MANY) {
			p.advance()
			variadic = true
		}

		paramName := p.parseIdentifier()

		// Type is optional for untyped variadic (many values)
		var paramType ast.TypeAnnotation
		if !p.check(lexer.TOKEN_COMMA) && !p.check(lexer.TOKEN_RPAREN) && !p.check(lexer.TOKEN_ASSIGN) {
			paramType = p.parseTypeAnnotation()
		}

		// Default untyped variadic to interface{}
		if variadic && paramType == nil {
			paramType = &ast.NamedType{
				Token: p.peekToken(),
				Name:  "interface{}",
			}
		}

		// Check for default value (e.g., count int = 10)
		var defaultValue ast.Expression
		if p.match(lexer.TOKEN_ASSIGN) {
			defaultValue = p.parseExpression()
			hasDefaultValue = true
		} else if hasDefaultValue {
			// Parameters with defaults must come after those without
			p.error(paramName.Token, fmt.Sprintf("parameter '%s' must have a default value (parameters with defaults must be contiguous at the end)", paramName.Value))
		}

		// Variadic parameters cannot have default values
		if variadic && defaultValue != nil {
			p.error(paramName.Token, fmt.Sprintf("variadic parameter '%s' cannot have a default value", paramName.Value))
		}

		params = append(params, &ast.Parameter{
			Name:         paramName,
			Type:         paramType,
			Variadic:     variadic,
			DefaultValue: defaultValue,
		})

		if !p.match(lexer.TOKEN_COMMA) {
			break
		}
	}

	return params
}

func (p *Parser) parseReturnTypes() []ast.TypeAnnotation {
	returns := []ast.TypeAnnotation{}

	// Single return type or multiple in parentheses
	if p.check(lexer.TOKEN_LPAREN) {
		p.advance() // consume '('
		for {
			returns = append(returns, p.parseTypeAnnotation())
			if !p.match(lexer.TOKEN_COMMA) {
				break
			}
		}
		p.consume(lexer.TOKEN_RPAREN, "expected ')' after return types")
	} else {
		returns = append(returns, p.parseTypeAnnotation())
	}

	return returns
}

// parseStructTag parses a struct tag like json:"name" or empty string if none present
// Format: identifier:stringLiteral
func (p *Parser) parseStructTag() string {
	// Check if next token is an identifier (tag name like "json", "xml", etc.)
	if !p.check(lexer.TOKEN_IDENTIFIER) {
		return ""
	}

	// Look ahead to see if there's a colon
	// Save current position
	savedPos := p.pos
	tagKeyToken := p.advance() // consume identifier

	if !p.check(lexer.TOKEN_COLON) {
		// Not a tag, restore position and return empty
		p.pos = savedPos
		return ""
	}

	// We have a tag - continue parsing
	tagKey := tagKeyToken.Lexeme
	p.consume(lexer.TOKEN_COLON, "expected ':' in struct tag")

	if !p.check(lexer.TOKEN_STRING) {
		p.error(p.peekToken(), "expected string value in struct tag")
		return ""
	}

	tagValueToken := p.advance() // consume string
	tagValue := tagValueToken.Lexeme

	// Return formatted tag: json:"name"
	return tagKey + ":" + `"` + tagValue + `"`
}

// parseFieldAlias parses optional field alias syntax: as "json_name"
// Returns empty string when no alias is present.
func (p *Parser) parseFieldAlias() string {
	if !p.match(lexer.TOKEN_AS) {
		return ""
	}

	if !p.check(lexer.TOKEN_STRING) {
		p.error(p.peekToken(), "expected string value after 'as' in field alias")
		return ""
	}

	return p.advance().Lexeme
}

// parseConstDecl parses a const declaration in one of two forms:
//
//	const MaxRetries = 5
//	const
//	    StatusOK  = 200
//	    StatusNotFound = 404
func (p *Parser) parseConstDecl() ast.Declaration {
	token := p.advance() // consume 'const'

	decl := &ast.ConstDecl{Token: token}

	// Grouped form: const followed by newline + INDENT
	if p.check(lexer.TOKEN_NEWLINE) || p.check(lexer.TOKEN_INDENT) {
		p.skipNewlines()
		if !p.match(lexer.TOKEN_INDENT) {
			p.error(p.peekToken(), "expected indented block or name after 'const'")
			return nil
		}
		for !p.check(lexer.TOKEN_DEDENT) && !p.isAtEnd() {
			p.skipNewlines()
			if p.check(lexer.TOKEN_DEDENT) {
				break
			}
			spec := p.parseConstSpec()
			if spec != nil {
				decl.Specs = append(decl.Specs, spec)
			}
			p.skipNewlines()
		}
		p.consume(lexer.TOKEN_DEDENT, "expected dedent after const block")
	} else {
		// Single-line form: const Name = value
		spec := p.parseConstSpec()
		if spec != nil {
			decl.Specs = append(decl.Specs, spec)
		}
	}

	p.skipNewlines()
	return decl
}

func (p *Parser) parseConstSpec() *ast.ConstSpec {
	name := p.parseIdentifier()
	if name == nil {
		return nil
	}
	p.consume(lexer.TOKEN_ASSIGN, fmt.Sprintf("expected '=' after const name '%s'", name.Value))
	value := p.parseExpression()
	return &ast.ConstSpec{Name: name, Value: value}
}

func (p *Parser) parseVarDeclaration() ast.Declaration {
	token := p.advance() // consume 'var'
	p.skipNewlines()

	// Parse identifiers
	var names []*ast.Identifier
	firstIdent := p.parseIdentifier()
	if firstIdent == nil {
		return nil
	}
	names = append(names, firstIdent)

	for p.match(lexer.TOKEN_COMMA) {
		ident := p.parseIdentifier()
		if ident == nil {
			return nil
		}
		names = append(names, ident)
	}

	// Parse type (optional)
	var typeAnnot ast.TypeAnnotation
	// Check if next is assignment or implicit newline/EOF (if allowed?)
	// If not assignment, try to parse type.
	if !p.check(lexer.TOKEN_ASSIGN) {
		typeAnnot = p.parseTypeAnnotation()
	}

	// Parse values
	var values []ast.Expression
	if p.match(lexer.TOKEN_ASSIGN) {
		values = p.parseExpressionList()
	}

	p.skipNewlines()

	return &ast.VarDeclStmt{
		Token:  token,
		Names:  names,
		Type:   typeAnnot,
		Values: values,
	}
}

// parseEnumDecl parses an enum declaration:
//
//	enum Status
//	    OK = 200
//	    NotFound = 404
func (p *Parser) parseEnumDecl() ast.Declaration {
	token := p.advance() // consume 'enum'
	p.skipNewlines()

	name := p.parseIdentifier()
	if name == nil {
		return nil
	}

	p.skipNewlines()

	if !p.match(lexer.TOKEN_INDENT) {
		p.error(p.peekToken(), "expected indented block for enum cases")
		return nil
	}

	var cases []*ast.EnumCase
	for !p.check(lexer.TOKEN_DEDENT) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(lexer.TOKEN_DEDENT) {
			break
		}

		caseName := p.parseIdentifier()
		if caseName == nil {
			p.skipNewlines()
			continue
		}
		p.consume(lexer.TOKEN_ASSIGN, fmt.Sprintf("expected '=' after enum case '%s'", caseName.Value))
		value := p.parseExpression()

		cases = append(cases, &ast.EnumCase{
			Name:  caseName,
			Value: value,
		})
		p.skipNewlines()
	}

	p.consume(lexer.TOKEN_DEDENT, "expected dedent after enum cases")
	p.skipNewlines()

	return &ast.EnumDecl{
		Token: token,
		Name:  name,
		Cases: cases,
	}
}

