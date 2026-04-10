package formatter

import (
	"strings"
	"unicode"
)

// Preprocessor converts Go-style syntax to Kukicha-style indentation
type Preprocessor struct {
	source    []rune
	indentStr string
}

// NewPreprocessor creates a new preprocessor
func NewPreprocessor(source string) *Preprocessor {
	return &Preprocessor{
		source:    []rune(source),
		indentStr: "    ", // 4 spaces
	}
}

// Process converts Go-style braces to Kukicha-style indentation
// It handles:
// - Lines ending with { -> remove brace, increase indent for following lines
// - Lines that are just } -> remove brace, decrease indent
// - Trailing semicolons -> remove
// - Struct/map literals are preserved (braces in expressions)
func (p *Preprocessor) Process() string {
	source := string(p.source)

	// Check if the source uses Go-style braces for blocks
	// If not, return the source unchanged (preserving existing indentation)
	if !p.hasGoStyleBraces(source) {
		return source
	}

	lines := strings.Split(source, "\n")
	var result []string

	indentLevel := 0

	for i, line := range lines {
		processed := p.processLine(line, &indentLevel, i, lines)
		if processed != "" || (i < len(lines)-1) { // Keep empty lines except trailing
			result = append(result, processed)
		}
	}

	// Join and ensure single trailing newline
	output := strings.Join(result, "\n")
	output = strings.TrimRight(output, "\n") + "\n"
	return output
}

// hasGoStyleBraces checks if the source uses Go-style braces for blocks
// (not just for struct/map literals).
//
// Only block-opening braces (lines ending with { on control-flow keywords)
// are checked. Standalone "}" lines are NOT checked because they can appear
// in Kukicha code as closing braces of struct/slice/map literals. Every
// Go-style file will have at least one block-opening "{", so this is
// sufficient.
func (p *Preprocessor) hasGoStyleBraces(source string) bool {
	lines := strings.SplitSeq(source, "\n")

	inTripleQuote := false
	for line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track multi-line string literals (triple-quoted)
		count := strings.Count(trimmed, `"""`)
		if count > 0 {
			if count%2 == 1 {
				inTripleQuote = !inTripleQuote
			}
			continue
		}
		if inTripleQuote {
			continue
		}

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for lines ending with { that are control flow, not literals
		if strings.HasSuffix(trimmed, "{") && !p.isExpressionBrace(trimmed) {
			return true
		}

		// Check for } else { pattern (Go-style else). Only triggers when
		// followed by an opening brace which indicates Go-style blocks.
		if strings.HasPrefix(trimmed, "} else") && strings.HasSuffix(trimmed, "{") {
			return true
		}
	}

	return false
}

func (p *Preprocessor) processLine(line string, indentLevel *int, lineIdx int, allLines []string) string {
	trimmed := strings.TrimSpace(line)

	// Skip empty lines
	if trimmed == "" {
		return ""
	}

	// Handle closing brace only line
	if trimmed == "}" {
		*indentLevel--
		if *indentLevel < 0 {
			*indentLevel = 0
		}
		return "" // Remove the brace-only line
	}

	// Handle closing brace with else: } else {
	if strings.HasPrefix(trimmed, "} else") {
		*indentLevel--
		if *indentLevel < 0 {
			*indentLevel = 0
		}
		// Process the else part
		remaining := strings.TrimPrefix(trimmed, "} ")
		return p.processLine(remaining, indentLevel, lineIdx, allLines)
	}

	// Calculate current indentation
	currentIndent := strings.Repeat(p.indentStr, *indentLevel)

	// Remove trailing semicolon
	if before, ok := strings.CutSuffix(trimmed, ";"); ok {
		trimmed = before
		trimmed = strings.TrimSpace(trimmed)
	}

	// Check if line ends with opening brace (but not in a string or struct literal context)
	if strings.HasSuffix(trimmed, "{") && !p.isExpressionBrace(trimmed) {
		// Remove the trailing brace
		trimmed = strings.TrimSuffix(trimmed, "{")
		trimmed = strings.TrimSpace(trimmed)
		result := currentIndent + trimmed
		*indentLevel++
		return result
	}

	return currentIndent + trimmed
}

// isExpressionBrace determines if a line's trailing brace is part of an expression
// (struct literal, map literal, list literal) rather than a block opener
func (p *Preprocessor) isExpressionBrace(line string) bool {
	line = strings.TrimSpace(line)

	beforeBrace := strings.TrimSuffix(line, "{")
	beforeBrace = strings.TrimSpace(beforeBrace)

	// Kukicha-style collection types: "list of <type>{" or "map of <type> to <type>{"
	if strings.Contains(beforeBrace, "list of") || strings.Contains(beforeBrace, "map of") {
		return true
	}

	// If line contains := or = followed by something ending with {, it's likely a literal
	// e.g., "x := MyStruct{" or "y = map[string]int{"
	if strings.Contains(line, ":=") || (strings.Contains(line, "=") && !strings.Contains(line, "==")) {
		// Check if there's a type before the brace
		if strings.HasSuffix(beforeBrace, "]") { // slice/array type: []int{
			return true
		}
		if matched := isTypeName(beforeBrace); matched {
			return true
		}
	}

	// Check for struct/map/slice literal patterns
	// TypeName{ or ]Type{ or map[...]Type{

	// If it ends with a type indicator, it's a literal
	if strings.HasSuffix(beforeBrace, "]") {
		return true // []Type{ or [n]Type{
	}

	// Check for map type pattern
	if strings.Contains(beforeBrace, "map[") && strings.Contains(beforeBrace, "]") {
		return true
	}

	// Extract the identifier immediately before the {. This handles cases
	// where the type is nested inside a call or other expression, e.g.
	// `onStatus(StatusUpdate{` — the word-level scan below would pick up
	// `onStatus(StatusUpdate` and miss that `StatusUpdate` is the type.
	if ident := trailingIdentifier(beforeBrace); ident != "" {
		// package.Type or TypeName starting with uppercase is a struct literal
		name := ident
		if dot := strings.LastIndex(ident, "."); dot >= 0 {
			name = ident[dot+1:]
		}
		if len(name) > 0 && unicode.IsUpper(rune(name[0])) {
			// Guard against control-flow constructs like `if Foo{` at the
			// start of the line. Note: `return Foo{` is always a struct
			// literal (return cannot introduce a block), so it is not here.
			controlKeywords := []string{"if", "for", "func", "else", "type", "interface", "switch"}
			for _, kw := range controlKeywords {
				if strings.HasPrefix(line, kw+" ") || line == kw {
					return false
				}
			}
			return true
		}
	}

	// Check for slice/array literal with type: []Type{ or [n]Type{
	// The beforeBrace may have a word like "[]Shape" or "[5]int" which starts with [
	words := strings.Fields(beforeBrace)
	if len(words) > 0 {
		lastWord := words[len(words)-1]
		if strings.HasPrefix(lastWord, "[") {
			return true // []Type{ or [n]Type{
		}

		// Remove assignment operators
		lastWord = strings.TrimSuffix(lastWord, ":=")
		lastWord = strings.TrimSuffix(lastWord, "=")
		lastWord = strings.TrimSpace(lastWord)

		// If ends with package.Type or TypeName, it's likely a struct literal
		if strings.Contains(lastWord, ".") {
			parts := strings.Split(lastWord, ".")
			typeName := parts[len(parts)-1]
			if len(typeName) > 0 && unicode.IsUpper(rune(typeName[0])) {
				return true
			}
		} else if len(lastWord) > 0 && unicode.IsUpper(rune(lastWord[0])) {
			// Could be struct literal, but also could be: if Foo{} or similar
			// Check if preceded by control keyword
			controlKeywords := []string{"if", "for", "func", "else", "type", "interface"}
			for _, kw := range controlKeywords {
				if strings.HasPrefix(line, kw+" ") || line == kw {
					return false // It's a control structure, not a literal
				}
			}
			return true
		}
	}

	return false
}

// trailingIdentifier returns the identifier at the end of s (scanning back
// through [A-Za-z0-9_] and . characters). Returns "" if s ends with a
// non-identifier character. e.g. "onStatus(StatusUpdate" → "StatusUpdate",
// "foo.Bar" → "foo.Bar", "x := Foo" → "Foo".
func trailingIdentifier(s string) string {
	i := len(s)
	for i > 0 {
		r := rune(s[i-1])
		if r == '_' || r == '.' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			i--
			continue
		}
		break
	}
	ident := strings.Trim(s[i:], ".")
	return ident
}

// isTypeName checks if the end of a string looks like a type name for a struct literal
func isTypeName(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return false
	}

	// Get the last word
	words := strings.Fields(s)
	if len(words) == 0 {
		return false
	}

	lastWord := words[len(words)-1]

	// Check for package.Type pattern
	if strings.Contains(lastWord, ".") {
		parts := strings.Split(lastWord, ".")
		typeName := parts[len(parts)-1]
		return len(typeName) > 0 && unicode.IsUpper(rune(typeName[0]))
	}

	// Check for simple TypeName (starts with uppercase)
	return unicode.IsUpper(rune(lastWord[0]))
}

// ProcessSource is a convenience function to preprocess source code
func ProcessSource(source string) string {
	pp := NewPreprocessor(source)
	return pp.Process()
}
