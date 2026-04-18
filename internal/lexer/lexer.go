package lexer

import (
	"fmt"
	"strings"
	"unique"
)

// Lexer tokenizes Kukicha source code.
//
// ARCHITECTURE NOTE: Kukicha uses Python-style indentation-based blocks.
// The lexer converts 4-space indentation changes into INDENT and DEDENT tokens,
// which the parser then uses to determine block structure.
//
// The indentStack tracks nesting levels. When indentation increases by 4 spaces,
// an INDENT token is emitted and the level is pushed. When it decreases, DEDENT
// tokens are emitted (possibly multiple) until the stack matches the new level.
//
// Why 4 spaces only (no tabs)?
//   - Consistency: Eliminates "tabs vs spaces" debates
//   - Beginner-friendly: One clear rule, no configuration needed
//   - Error prevention: Mixed tabs/spaces is a common Python mistake
type Lexer struct {
	source             []rune
	start              int
	current            int
	line               int
	column             int
	file               string
	tokens             []Token
	indentStack        []int // Stack of indentation levels (in spaces). Always starts with [0].
	atLineStart        bool  // Whether we're at the start of a line
	indentationHandled bool  // Whether indentation has been handled for the current line
	errors             []error

	// Brace continuation support: inside [] or literal {}, NEWLINE tokens are
	// suppressed and indentation is consumed without emitting INDENT/DEDENT.
	// Pipe continuation (|>) is handled separately by mergeLineContinuations
	// as a post-tokenization pass, decoupling it from the indent stack.
	continuationLine  bool // true when the next line is inside a brace/paren continuation
	braceDepth        int  // current nesting level of [], literal {} (used for continuations)
	parenDepth        int  // current nesting level of ()
	inFunctionLiteral bool // true when we've just seen 'func' and are parsing its body

	// Paren continuation support: inside (), NEWLINE tokens are suppressed so
	// multi-line function calls work. The exception is block-bodied closures —
	// when => is followed by an indented block, normal INDENT/DEDENT processing
	// resumes for that closure's body. closureIndentStack tracks the indent
	// levels where those closure bodies started; closureParenDepthStack
	// tracks the parenDepth at the moment each body opened. We're in
	// paren-continuation mode when parenDepth exceeds the parenDepth recorded
	// when the topmost closure body opened (or, with no open body, simply
	// when parenDepth > 0).
	closureIndentStack    []int // indent levels of open block-bodied closures inside ()
	closureParenDepthStack []int // parenDepth at the moment each closure body opened
	afterFatArrow       bool  // true after emitting => inside paren-continuation mode
	pendingClosureBlock bool  // true when the next INDENT should open a closure block

	// Brace block support: { } as alternative to indentation for blocks.
	// Block braces (after if/for/func/switch/else/select/go/defer) preserve
	// newline emission and suppress INDENT/DEDENT from indentation.
	// Literal braces (struct/map/list literals) keep existing behavior.
	blockKeywordSeen bool   // true after a block keyword (if/for/func/etc.) until { or newline
	braceStack       []bool // stack tracking block (true) vs literal (false) for each {
	braceBlockDepth  int    // count of currently open block braces

	// String interpolation support: when scanning a string and encountering
	// {expr}, the lexer emits TOKEN_STRING_HEAD, returns to normal tokenization
	// for the expression, and resumes string scanning when the matching } is found.
	// Each entry in interpStack tracks the brace depth and the quote character
	// ('"' or '\'') so the continuation scanner knows which delimiter to look for.
	interpStack []interpState
}

// interpState tracks the context for a string interpolation level.
type interpState struct {
	braceDepth int  // nesting of {} within this interpolation expression
	quote      rune // '"' for double-quoted strings, '\'' for single-quoted strings
}

// NewLexer creates a new lexer for the given source code
func NewLexer(source string, filename string) *Lexer {
	return &Lexer{
		source:             []rune(source),
		file:               filename,
		line:               1,
		column:             1,
		indentStack:        []int{0},
		atLineStart:        true,
		indentationHandled: false,
		braceDepth:         0,
		parenDepth:         0,
		inFunctionLiteral:  false,
	}
}

// ScanTokens scans all tokens from the source
func (l *Lexer) ScanTokens() ([]Token, error) {
	for !l.isAtEnd() {
		l.start = l.current
		l.scanToken()
	}

	// Emit remaining dedents
	for len(l.indentStack) > 1 {
		l.addToken(TOKEN_DEDENT)
		l.indentStack = l.indentStack[:len(l.indentStack)-1]
	}

	l.addToken(TOKEN_EOF)

	if len(l.errors) > 0 {
		return nil, fmt.Errorf("lexer errors: %v", l.errors)
	}

	// Post-pass: merge pipe continuation lines. This removes NEWLINE/INDENT/
	// DEDENT tokens around |> chains so the parser sees them as single
	// expressions, without coupling to the indent stack logic above.
	l.tokens = mergeLineContinuations(l.tokens)

	return l.tokens, nil
}

// scanToken scans a single token
func (l *Lexer) scanToken() {
	// Brace continuation: the previous line was inside [] or {}, so this
	// line's leading whitespace is consumed without emitting INDENT/DEDENT.
	if l.atLineStart && l.continuationLine && !l.indentationHandled {
		l.continuationLine = false
		for !l.isAtEnd() && (l.peek() == ' ' || l.peek() == '\t') {
			l.advance()
		}
		l.start = l.current // move past consumed whitespace so it is not included in the next token
		l.indentationHandled = true
		// Fall through — the next token on this line is scanned normally.
	}

	// Brace block: inside { } blocks, consume indentation whitespace without
	// emitting INDENT/DEDENT. The braces define block structure, not indentation.
	if l.atLineStart && !l.indentationHandled && l.braceBlockDepth > 0 {
		for !l.isAtEnd() && (l.peek() == ' ' || l.peek() == '\t') {
			l.advance()
		}
		l.start = l.current
		l.indentationHandled = true
	}

	// Handle indentation at line start
	if l.atLineStart && !l.indentationHandled {
		c := l.peek()

		// If it's space or tab, we definitely need to handle indentation
		if c == ' ' || c == '\t' {
			l.indentationHandled = true
			l.handleIndentation()
			return
		}

		// Check for implicit dedent to 0 level (no indentation)
		// Don't process for newlines or comments which handle their own flow
		if c != '\n' && c != '\r' && c != '#' {
			if len(l.indentStack) > 1 {
				l.indentationHandled = true
				l.handleIndentation()
				return
			}
			// Mark indentation as handled even if we don't change indentation
			l.indentationHandled = true
		}
	}

	c := l.advance()

	l.atLineStart = false

	switch c {
	case ' ', '\t':
		// Skip whitespace (not at line start)
		for !l.isAtEnd() && (l.peek() == ' ' || l.peek() == '\t') {
			l.advance()
		}
	case '\n':
		// Implicit continuation inside [] or literal {} (braceDepth > 0), or
		// inside () when not currently inside a block-bodied closure body.
		// Pipe continuation (|>) is handled by mergeLineContinuations post-pass.
		if l.braceDepth > 0 {
			l.continuationLine = true
		} else if l.inParenContinuation() {
			// Inside a paren call. Two tokens can precede a block-bodied body
			// that needs real INDENT/DEDENT:
			//   • => (fat arrow closure)
			//   • end-of-func-signature (func literal, tracked by inFunctionLiteral)
			// In both cases, the block is signaled by the next line being more
			// indented than the current outer indent level.
			if (l.afterFatArrow || l.inFunctionLiteral) && l.peekNextLineIndent() > l.indentStack[len(l.indentStack)-1] {
				l.pendingClosureBlock = true
				l.addToken(TOKEN_NEWLINE)
			} else {
				l.continuationLine = true
			}
		} else {
			l.addToken(TOKEN_NEWLINE)
		}
		l.afterFatArrow = false
		l.inFunctionLiteral = false
		l.line++
		l.column = 0
		l.atLineStart = true
		l.indentationHandled = false
	case '\r':
		if l.peek() == '\n' {
			l.advance()
		}
		if l.braceDepth > 0 {
			l.continuationLine = true
		} else if l.inParenContinuation() {
			if (l.afterFatArrow || l.inFunctionLiteral) && l.peekNextLineIndent() > l.indentStack[len(l.indentStack)-1] {
				l.pendingClosureBlock = true
				l.addToken(TOKEN_NEWLINE)
			} else {
				l.continuationLine = true
			}
		} else {
			l.addToken(TOKEN_NEWLINE)
		}
		l.afterFatArrow = false
		l.inFunctionLiteral = false
		l.line++
		l.column = 0
		l.atLineStart = true
		l.indentationHandled = false
	case '#':
		l.scanComment()
	case ';':
		l.addToken(TOKEN_SEMICOLON)
	case '"':
		l.scanString()
	case '\'':
		l.scanSingleQuoteString()
	case '`':
		l.scanRawString()
	case '(':
		l.parenDepth++
		l.addToken(TOKEN_LPAREN)
	case ')':
		if l.parenDepth > 0 {
			l.parenDepth--
		}
		// When closing the parameter list of a function literal (parenDepth becomes 0),
		// we know the next tokens will be the return type annotations and then the body.
		// Keep inFunctionLiteral true; it will be reset when the body block is done.
		l.addToken(TOKEN_RPAREN)
	case '[':
		l.braceDepth++
		l.addToken(TOKEN_LBRACKET)
	case ']':
		if l.braceDepth > 0 {
			l.braceDepth--
		}
		l.addToken(TOKEN_RBRACKET)
	case '{':
		if len(l.interpStack) > 0 {
			l.interpStack[len(l.interpStack)-1].braceDepth++
		}
		if l.blockKeywordSeen {
			// Block brace: after if/for/func/switch/else/select/go/defer.
			// Emit INDENT instead of LBRACE so the parser treats it as a block.
			// Don't increment braceDepth so newlines are preserved inside.
			l.braceStack = append(l.braceStack, true)
			l.braceBlockDepth++
			l.blockKeywordSeen = false
			l.addToken(TOKEN_INDENT)
		} else {
			// Literal brace: struct/map/list composite literals.
			// Increment braceDepth to suppress newlines (existing behavior).
			l.braceStack = append(l.braceStack, false)
			l.braceDepth++
			l.addToken(TOKEN_LBRACE)
		}
	case '}':
		if len(l.interpStack) > 0 && l.interpStack[len(l.interpStack)-1].braceDepth == 0 {
			// End of string interpolation expression — resume string scanning
			quote := l.interpStack[len(l.interpStack)-1].quote
			l.interpStack = l.interpStack[:len(l.interpStack)-1]
			l.start = l.current
			l.scanStringContinuation(quote)
			return
		}
		if len(l.interpStack) > 0 {
			l.interpStack[len(l.interpStack)-1].braceDepth--
		}
		// Pop brace stack to determine if this was a block or literal brace.
		if len(l.braceStack) > 0 {
			wasBlock := l.braceStack[len(l.braceStack)-1]
			l.braceStack = l.braceStack[:len(l.braceStack)-1]
			if wasBlock {
				l.braceBlockDepth--
				l.addToken(TOKEN_DEDENT)
			} else {
				if l.braceDepth > 0 {
					l.braceDepth--
				}
				l.addToken(TOKEN_RBRACE)
			}
		} else {
			if l.braceDepth > 0 {
				l.braceDepth--
			}
			l.addToken(TOKEN_RBRACE)
		}
	case ',':
		l.addToken(TOKEN_COMMA)
	case '.':
		l.addToken(TOKEN_DOT)
	case '+':
		if l.match('+') {
			l.addToken(TOKEN_PLUS_PLUS)
		} else {
			l.addToken(TOKEN_PLUS)
		}
	case '-':
		if l.match('-') {
			l.addToken(TOKEN_MINUS_MINUS)
		} else {
			l.addToken(TOKEN_MINUS)
		}
	case '*':
		l.addToken(TOKEN_STAR)
	case '/':
		l.addToken(TOKEN_SLASH)
	case '%':
		l.addToken(TOKEN_PERCENT)
	case ':':
		if l.match('=') {
			l.addToken(TOKEN_WALRUS)
		} else {
			l.addToken(TOKEN_COLON)
		}
	case '=':
		if l.match('=') {
			l.addToken(TOKEN_DOUBLE_EQUALS)
		} else if l.match('>') {
			l.afterFatArrow = true
			l.addToken(TOKEN_FAT_ARROW)
		} else {
			l.addToken(TOKEN_ASSIGN)
		}
	case '!':
		if l.match('=') {
			l.addToken(TOKEN_NOT_EQUALS)
		} else {
			l.addToken(TOKEN_BANG)
		}
	case '<':
		if l.match('-') {
			l.addToken(TOKEN_ARROW_LEFT)
		} else if l.match('=') {
			l.addToken(TOKEN_LTE)
		} else {
			l.addToken(TOKEN_LT)
		}
	case '>':
		if l.match('=') {
			l.addToken(TOKEN_GTE)
		} else {
			l.addToken(TOKEN_GT)
		}
	case '|':
		if l.match('>') {
			l.addToken(TOKEN_PIPE)
		} else if l.match('|') {
			l.addToken(TOKEN_OR_OR)
		} else {
			l.addToken(TOKEN_BIT_OR)
		}
	case '&':
		if l.match('&') {
			l.addToken(TOKEN_AND_AND)
		} else if l.match('=') {
			l.addToken(TOKEN_BIT_AND_ASSIGN)
		} else {
			l.addToken(TOKEN_BIT_AND)
		}
	default:
		if isDigit(c) {
			l.scanNumber()
		} else if isAlpha(c) {
			l.scanIdentifier()
		} else {
			l.error(fmt.Sprintf("Unexpected character: %c", c))
		}
	}
}

// handleIndentation handles indentation at the start of a line.
//
// ARCHITECTURE NOTE: The algorithm works as follows:
//  1. Count leading spaces (tabs are rejected with error)
//  2. Skip blank lines and comment-only lines (no tokens emitted)
//  3. Validate spacing is a multiple of 4
//  4. Compare to current indent level:
//     - If greater: push level, emit INDENT
//     - If smaller: pop levels, emit DEDENT for each
//     - If equal: no token (same level continues)
//
// The indentStack ensures dedents always return to a valid prior level.
// For example, going from 8 spaces directly to 0 emits two DEDENT tokens.
//
// Special case: paren continuation mode (parenDepth > len(closureIndentStack)).
// Inside a multi-line call, continuation-line indentation is decorative — we
// consume the whitespace without emitting INDENT/DEDENT. The exception is when
// a block-bodied closure follows =>, in which case we DO emit INDENT/DEDENT for
// the closure's body and track it in closureIndentStack.
func (l *Lexer) handleIndentation() {
	spaces := 0
	tabs := 0

	// Count spaces and tabs
	for !l.isAtEnd() && (l.peek() == ' ' || l.peek() == '\t') {
		if l.peek() == ' ' {
			spaces++
		} else {
			tabs++
		}
		l.advance()
	}

	// Check for tabs
	if tabs > 0 {
		l.error("indentation error: tabs are not allowed — use 4 spaces per indent level")
		return
	}

	// Skip blank lines and comment-only lines
	if l.isAtEnd() || l.peek() == '\n' || l.peek() == '\r' || l.peek() == '#' {
		return
	}

	// Paren continuation mode: we're inside a multi-line call and not currently
	// inside a block-bodied closure body. Just consume the decorative indentation
	// without emitting INDENT/DEDENT.
	// Exception: pendingClosureBlock means the previous line ended with => or a
	// func signature, and this line opens the closure/function body — handle
	// indentation normally so we emit the INDENT and register the closure.
	if l.inParenContinuation() && !l.pendingClosureBlock {
		return
	}

	// Must be multiple of 4
	if spaces%4 != 0 {
		nearest := ((spaces + 2) / 4) * 4
		if nearest == 0 {
			nearest = 4
		}
		l.error(fmt.Sprintf("indentation error: found %d spaces, but Kukicha requires multiples of 4 spaces (nearest valid: %d)", spaces, nearest))
		return
	}

	currentIndent := l.indentStack[len(l.indentStack)-1]

	if spaces > currentIndent {
		// Indent — normally must be exactly +4 at a time.
		// Exception: when opening a block-bodied closure inside a paren call
		// (pendingClosureBlock), the closure's declaration line used continuation
		// indentation (decorative, not tracked in indentStack), so the body can
		// appear at any multiple of 4 greater than currentIndent.
		if !l.pendingClosureBlock && spaces != currentIndent+4 {
			l.error(fmt.Sprintf("indentation error: indentation can only increase by 4 spaces at a time (jumped from %d to %d)", currentIndent, spaces))
			return
		}
		l.indentStack = append(l.indentStack, spaces)
		l.addToken(TOKEN_INDENT)
		// If this INDENT opens a block-bodied closure inside a paren call,
		// record it so we know when to resume paren-continuation suppression.
		if l.pendingClosureBlock {
			l.closureIndentStack = append(l.closureIndentStack, spaces)
			l.closureParenDepthStack = append(l.closureParenDepthStack, l.parenDepth)
			l.pendingClosureBlock = false
		}
	} else if spaces < currentIndent {
		// Capture valid levels before popping, for use in the error message.
		validLevels := make([]int, len(l.indentStack))
		copy(validLevels, l.indentStack)

		// Dedent (possibly multiple levels)
		for len(l.indentStack) > 1 && l.indentStack[len(l.indentStack)-1] > spaces {
			l.addToken(TOKEN_DEDENT)
			l.indentStack = l.indentStack[:len(l.indentStack)-1]
			// If we've dedented back to (or below) where a closure block started,
			// that closure is closed — resume paren-continuation suppression.
			if len(l.closureIndentStack) > 0 && l.indentStack[len(l.indentStack)-1] < l.closureIndentStack[len(l.closureIndentStack)-1] {
				l.closureIndentStack = l.closureIndentStack[:len(l.closureIndentStack)-1]
				l.closureParenDepthStack = l.closureParenDepthStack[:len(l.closureParenDepthStack)-1]
			}
		}

		// If we've exited all closure blocks and are back inside the paren call,
		// the current line's indentation is just continuation formatting — no
		// alignment check needed.
		if l.inParenContinuation() {
			return
		}

		// Verify we landed on a valid indentation level
		if l.indentStack[len(l.indentStack)-1] != spaces {
			parts := make([]string, len(validLevels))
			for i, v := range validLevels {
				parts[i] = fmt.Sprintf("%d", v)
			}
			l.error(fmt.Sprintf("indentation error: dedent does not match any outer indent level (found %d spaces, expected one of: %s)", spaces, strings.Join(parts, ", ")))
		}
	}
}

// inParenContinuation reports whether the lexer should currently treat
// newlines as continuations (suppressed) rather than statement
// terminators. True when we're inside an open `(` AND we are not at the
// natural scope of a block-bodied closure body.
func (l *Lexer) inParenContinuation() bool {
	if l.parenDepth == 0 {
		return false
	}
	if len(l.closureParenDepthStack) == 0 {
		return true
	}
	return l.parenDepth > l.closureParenDepthStack[len(l.closureParenDepthStack)-1]
}

// peekNextLineIndent returns the number of leading spaces on the line that
// starts at l.current (i.e. just after the newline character that was just
// consumed by advance()). Used to decide whether => is followed by a
// block-bodied closure.
func (l *Lexer) peekNextLineIndent() int {
	spaces := 0
	for i := l.current; i < len(l.source) && (l.source[i] == ' ' || l.source[i] == '\t'); i++ {
		if l.source[i] == ' ' {
			spaces++
		}
	}
	return spaces
}

// scanString scans a double-quoted string literal.
//
// If the next two characters are also '"', this is a triple-quoted multi-line
// string ("""...""") and is handled by scanTripleQuoteString.
//
// For non-interpolated strings, emits a single TOKEN_STRING.
// For interpolated strings (containing {expr}), emits TOKEN_STRING_HEAD for the
// leading literal, then returns so the main scanToken loop can tokenize the
// expression normally. When the matching } is found, scanStringContinuation
// resumes scanning, emitting TOKEN_STRING_MID or TOKEN_STRING_TAIL.
func (l *Lexer) scanString() {
	// Check for triple-quote: "" could be an empty string or the start of """
	if l.peek() == '"' {
		if l.current+1 < len(l.source) && l.source[l.current+1] == '"' {
			// Triple-quote: consume the two extra '"' and scan multi-line
			l.advance() // consume second "
			l.advance() // consume third "
			l.scanTripleQuoteString()
			return
		}
		// Empty string "" — consume closing quote, emit empty TOKEN_STRING
		l.advance()
		l.addTokenWithLexeme(TOKEN_STRING, "")
		return
	}
	l.scanStringBody(TOKEN_STRING_HEAD, TOKEN_STRING, '"')
}

// scanTripleQuoteString scans a triple-quoted string literal ("""...""").
//
// Behavior:
//   - Content spans multiple lines until the closing """
//   - Leading common indentation (dedent) is stripped: the minimum non-empty
//     leading-whitespace count across all content lines is removed from each line
//   - The first newline after the opening """ is not part of the content
//   - The last newline before the closing """ is not part of the content
//   - String interpolation ({expr}) works inside, same as regular strings
//
// The resulting content is passed through the same TOKEN_STRING / TOKEN_STRING_HEAD
// path as regular strings, so the rest of the pipeline (parser, codegen) is unchanged.
func (l *Lexer) scanTripleQuoteString() {
	startLine := l.line // Save start line for the token position
	raw := strings.Builder{}

	for !l.isAtEnd() {
		// Check for closing """
		if l.peek() == '"' && l.current+1 < len(l.source) && l.source[l.current+1] == '"' &&
			l.current+2 < len(l.source) && l.source[l.current+2] == '"' {
			l.advance() // consume first "
			l.advance() // consume second "
			l.advance() // consume third "
			break
		}
		ch := l.advance()
		if ch == '\n' {
			l.line++
			l.column = 0
		}
		raw.WriteRune(ch)
	}

	// Save the post-scan line (after closing """) so we can restore it after
	// the content injection. This ensures tokens following the triple-quote
	// string get the correct line number.
	endLine := l.line
	endColumn := l.column

	content := dedentTripleQuote(raw.String())

	// Set line to start so the emitted string token has the correct position.
	l.line = startLine

	// Now re-scan content string through the interpolation machinery by
	// injecting it as if it were scanned from a regular "..." string.
	l.scanStringFromContent(content)

	// Restore line/column to the position after the closing """ so subsequent
	// tokens get correct positions.
	l.line = endLine
	l.column = endColumn
}

// dedentTripleQuote strips the first newline (if any), the last newline (if any),
// and the common leading indentation from all non-empty lines.
func dedentTripleQuote(raw string) string {
	// Strip leading newline (the one right after opening """)
	if len(raw) > 0 && raw[0] == '\n' {
		raw = raw[1:]
	} else if len(raw) > 1 && raw[0] == '\r' && raw[1] == '\n' {
		raw = raw[2:]
	}

	// Strip trailing newline (the one right before closing """)
	if len(raw) > 0 && raw[len(raw)-1] == '\n' {
		raw = raw[:len(raw)-1]
		if len(raw) > 0 && raw[len(raw)-1] == '\r' {
			raw = raw[:len(raw)-1]
		}
	}

	// Find minimum indentation (spaces only) across non-empty lines
	lines := strings.Split(raw, "\n")
	minIndent := -1
	for _, line := range lines {
		if strings.TrimRight(line, " \t\r") == "" {
			continue // skip blank / whitespace-only lines
		}
		indent := 0
		for _, ch := range line {
			if ch == ' ' {
				indent++
			} else if ch == '\t' {
				indent += 4
			} else {
				break
			}
		}
		if minIndent < 0 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent <= 0 {
		return raw
	}

	// Strip minIndent spaces from the front of each line
	out := strings.Builder{}
	for i, line := range lines {
		stripped := 0
		j := 0
		for j < len(line) && stripped < minIndent {
			if line[j] == ' ' {
				stripped++
				j++
			} else if line[j] == '\t' {
				tabWidth := 4
				if stripped+tabWidth > minIndent {
					// Tab overshoots — only strip what's needed
					stripped = minIndent
				} else {
					stripped += tabWidth
				}
				j++
			} else {
				break
			}
		}
		out.WriteString(line[j:])
		if i < len(lines)-1 {
			out.WriteRune('\n')
		}
	}
	return out.String()
}

// scanStringFromContent injects pre-extracted string content back into the
// source stream so the existing scanStringBody machinery handles it naturally.
// Bare `"` characters (not preceded by `\`) are escaped to `\"` so they
// don't prematurely terminate the scan. All other escape sequences are kept
// as-is so scanStringEscape can process them normally.
func (l *Lexer) scanStringFromContent(content string) {
	runes := []rune(content)
	inject := make([]rune, 0, len(runes)+4)

	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		if ch == '\\' && i+1 < len(runes) {
			// Keep escape sequence intact (two characters)
			inject = append(inject, ch, runes[i+1])
			i++ // skip next char (already consumed)
		} else if ch == '"' {
			// Bare quote — escape it so scanStringBody doesn't stop here
			inject = append(inject, '\\', '"')
		} else if ch == '\n' {
			// Newline — inject as \n escape so scanStringBody doesn't error
			inject = append(inject, '\\', 'n')
		} else if ch == '\r' {
			// Skip bare CR (CR+LF was already normalized to LF during extraction)
		} else {
			inject = append(inject, ch)
		}
	}
	// Append synthetic closing quote
	inject = append(inject, '"')

	// Splice into l.source at l.current
	newSource := make([]rune, 0, len(l.source)+len(inject))
	newSource = append(newSource, l.source[:l.current]...)
	newSource = append(newSource, inject...)
	newSource = append(newSource, l.source[l.current:]...)
	l.source = newSource
	l.scanStringBody(TOKEN_STRING_HEAD, TOKEN_STRING, '"')
}

// scanStringContinuation resumes string scanning after a } closes an
// interpolation expression. The quote parameter indicates which delimiter
// to look for ('"' or '\''). Emits TOKEN_STRING_MID if another interpolation
// follows, or TOKEN_STRING_TAIL at the closing quote.
func (l *Lexer) scanStringContinuation(quote rune) {
	l.scanStringBody(TOKEN_STRING_MID, TOKEN_STRING_TAIL, quote)
}

// scanStringBody is the shared string scanning logic.
// interpTokenType is emitted when a {expr} interpolation is found (HEAD or MID).
// endTokenType is emitted when the string ends with the closing quote (STRING or TAIL).
// quote is the terminator character ('"' for double-quoted, '\'' for single-quoted).
func (l *Lexer) scanStringBody(interpTokenType TokenType, endTokenType TokenType, quote rune) {
	value := strings.Builder{}

	for !l.isAtEnd() && l.peek() != quote {
		if l.peek() == '\n' {
			// Single-quote strings are multi-line; double-quote strings are not
			if quote == '"' {
				l.error("Unterminated string")
				return
			}
			// For single-quote strings, include the newline in content
			value.WriteRune(l.advance())
			l.line++
			l.column = 0
			continue
		}

		if l.peek() == '\\' {
			l.scanStringEscape(&value)
		} else if l.peek() == '{' && l.isInterpStart() {
			// String interpolation: emit the accumulated literal, push
			// interp state, and return so the expression gets tokenized.
			l.advance() // consume '{'
			l.addTokenWithLexeme(interpTokenType, value.String())
			l.interpStack = append(l.interpStack, interpState{braceDepth: 0, quote: quote})
			return
		} else {
			r := l.advance()
			if r == '\x00' {
				l.error("string literal contains invalid character (NUL)")
				return
			}
			value.WriteRune(r)
		}
	}

	if l.isAtEnd() {
		l.error("Unterminated string")
		return
	}

	l.advance() // consume closing quote
	l.addTokenWithLexeme(endTokenType, value.String())
}

// isInterpStart checks whether { at the current position starts a string
// interpolation. Requires the character after { to be an identifier-start
// character (letter or underscore). This avoids treating regex quantifiers
// like {2,} as interpolation.
func (l *Lexer) isInterpStart() bool {
	// peek() is '{', check the character after it
	nextIdx := l.current + 1
	if nextIdx >= len(l.source) {
		return false
	}
	c := l.source[nextIdx]
	return isAlpha(c)
}

// scanStringEscape handles a single escape sequence inside a string literal,
// writing the result into the provided builder.
func (l *Lexer) scanStringEscape(value *strings.Builder) {
	l.advance() // consume '\'
	if l.isAtEnd() {
		return
	}
	escaped := l.advance()
	switch escaped {
	case 'n':
		value.WriteRune('\n')
	case 't':
		value.WriteRune('\t')
	case 'r':
		value.WriteRune('\r')
	case '\\':
		value.WriteRune('\\')
	case '"':
		value.WriteRune('"')
	case '\'':
		value.WriteRune('\'')
	case '{':
		value.WriteRune('\uE000') // PUA sentinel for literal {
	case '}':
		value.WriteRune('\uE001') // PUA sentinel for literal }
	case 's':
		// Check for \sep (filepath separator)
		if l.peek() == 'e' && l.peekNext() == 'p' {
			l.advance()               // consume 'e'
			l.advance()               // consume 'p'
			value.WriteRune('\uE002') // PUA sentinel for filepath.Separator
		} else {
			value.WriteRune('s')
		}
	case 'x':
		// Hex escape: \xHH
		if l.isAtEnd() {
			l.error("Incomplete hex escape '\\x' at end of string")
			return
		}
		h1 := l.advance()
		if l.isAtEnd() {
			l.error("Incomplete hex escape '\\x' — expected 2 hex digits")
			return
		}
		h2 := l.advance()
		hi, ok1 := hexDigit(h1)
		lo, ok2 := hexDigit(h2)
		if ok1 && ok2 {
			value.WriteRune(rune(hi*16 + lo))
		} else {
			value.WriteString(`\x`)
			value.WriteRune(h1)
			value.WriteRune(h2)
		}
	default:
		// Octal escape: \0nn or \nnn (1-3 octal digits)
		if escaped >= '0' && escaped <= '7' {
			oct := int(escaped - '0')
			for range 2 {
				if l.isAtEnd() {
					break
				}
				next := l.peek()
				if next >= '0' && next <= '7' {
					oct = oct*8 + int(next-'0')
					l.advance()
				} else {
					break
				}
			}
			if oct > 255 {
				l.error("Octal escape value out of range (max \\377)")
				return
			}
			value.WriteByte(byte(oct))
		} else {
			value.WriteRune(escaped)
		}
	}
}

// scanSingleQuoteString scans a single-quoted multi-line string literal ('...').
//
// Behavior:
//   - Content spans multiple lines until the closing '
//   - Leading common indentation (dedent) is stripped (same as triple-quoted strings)
//   - The first newline after the opening ' is stripped
//   - The last newline before the closing ' is stripped
//   - String interpolation ({expr}) works inside, same as double-quoted strings
//   - Escape sequences work: \', \\, \{, \}, \n, \t, etc.
//
// This makes single-quoted strings ideal for HTML where double quotes are used
// for attributes: html.Render('<div class="hero">{html.Escape(title)}</div>')
func (l *Lexer) scanSingleQuoteString() {
	startLine := l.line
	raw := strings.Builder{}

	for !l.isAtEnd() {
		if l.peek() == '\'' {
			l.advance() // consume closing '
			break
		}
		if l.peek() == '\\' && l.current+1 < len(l.source) && l.source[l.current+1] == '\'' {
			// Escaped single quote — include literal '
			l.advance() // consume \
			l.advance() // consume '
			raw.WriteRune('\'')
			continue
		}
		ch := l.advance()
		if ch == '\n' {
			l.line++
			l.column = 0
		}
		raw.WriteRune(ch)
	}

	endLine := l.line
	endColumn := l.column

	content := dedentTripleQuote(raw.String())

	l.line = startLine
	l.scanStringFromContent(content)
	l.line = endLine
	l.column = endColumn
}



// scanRawString scans a backtick-delimited raw string literal.
//
// Follows Go semantics: no escape processing, no interpolation, may span
// multiple lines. Backtick characters cannot appear in the content (Go's
// own restriction — there is no escape mechanism inside raw strings).
// NUL bytes are rejected as they are not valid in source files.
func (l *Lexer) scanRawString() {
	startLine := l.line
	var value strings.Builder
	for {
		if l.isAtEnd() {
			l.line = startLine
			l.error("unterminated raw string literal")
			return
		}
		ch := l.advance()
		if ch == '`' {
			break
		}
		if ch == '\x00' {
			l.error("raw string literal contains NUL byte")
			return
		}
		if ch == '\n' {
			l.line++
			l.column = 0
		}
		value.WriteRune(ch)
	}
	l.addTokenWithLexeme(TOKEN_STRING_RAW, value.String())
}

// scanNumber scans a number (integer or float)
func (l *Lexer) scanNumber() {
	// Check for 0x (hex) or 0o (octal) or 0b (binary) prefixes
	if l.source[l.start] == '0' {
		next := l.peek()
		if next == 'x' || next == 'X' {
			l.advance() // consume 'x'
			if !isHexDigit(l.peek()) {
				l.error("hexadecimal literal has no digits")
			}
			for isHexDigit(l.peek()) || l.peek() == '_' {
				l.advance()
			}
			l.addToken(TOKEN_INTEGER)
			return
		}
		if next == 'o' || next == 'O' {
			l.advance() // consume 'o'
			if !isOctalDigit(l.peek()) {
				l.error("octal literal has no digits")
			}
			for isOctalDigit(l.peek()) || l.peek() == '_' {
				l.advance()
			}
			l.addToken(TOKEN_INTEGER)
			return
		}
		if next == 'b' || next == 'B' {
			l.advance() // consume 'b'
			if l.peek() != '0' && l.peek() != '1' {
				l.error("binary literal has no digits")
			}
			for l.peek() == '0' || l.peek() == '1' || l.peek() == '_' {
				l.advance()
			}
			l.addToken(TOKEN_INTEGER)
			return
		}
	}

	for isDigit(l.peek()) || l.peek() == '_' {
		l.advance()
	}

	// Look for decimal point
	if l.peek() == '.' && isDigit(l.peekNext()) {
		l.advance() // consume .

		for isDigit(l.peek()) || l.peek() == '_' {
			l.advance()
		}

		l.addToken(TOKEN_FLOAT)
	} else {
		l.addToken(TOKEN_INTEGER)
	}
}

// scanIdentifier scans an identifier or keyword
func (l *Lexer) scanIdentifier() {
	for isAlphaNumeric(l.peek()) {
		l.advance()
	}

	text := unique.Make(string(l.source[l.start:l.current])).Value()
	tokenType := LookupKeyword(text)
	l.addTokenWithLexeme(tokenType, text)

	// Track when we enter a function literal context
	if tokenType == TOKEN_FUNC {
		l.inFunctionLiteral = true
	}
}

// scanComment scans a comment. If the comment starts with "# kuki:", it is
// emitted as TOKEN_DIRECTIVE so the parser can attach it to a declaration.
// Otherwise it is emitted as a regular TOKEN_COMMENT.
func (l *Lexer) scanComment() {
	// Consume the rest of the comment line
	for !l.isAtEnd() && l.peek() != '\n' {
		l.advance()
	}
	// Check if this is a directive comment (# kuki:...)
	lexeme := string(l.source[l.start:l.current])
	if strings.HasPrefix(lexeme, "# kuki:") {
		l.addToken(TOKEN_DIRECTIVE)
	} else {
		l.addToken(TOKEN_COMMENT)
	}
}

// Helper methods

func (l *Lexer) isAtEnd() bool {
	return l.current >= len(l.source)
}

func (l *Lexer) advance() rune {
	if l.isAtEnd() {
		return 0
	}
	c := l.source[l.current]
	l.current++
	l.column++
	return c
}

func (l *Lexer) peek() rune {
	if l.isAtEnd() {
		return 0
	}
	return l.source[l.current]
}

func (l *Lexer) peekNext() rune {
	if l.current+1 >= len(l.source) {
		return 0
	}
	return l.source[l.current+1]
}

func (l *Lexer) match(expected rune) bool {
	if l.isAtEnd() {
		return false
	}
	if l.source[l.current] != expected {
		return false
	}
	l.current++
	l.column++
	return true
}

func (l *Lexer) addToken(tokenType TokenType) {
	l.addTokenWithLexeme(tokenType, string(l.source[l.start:l.current]))
}

func (l *Lexer) addTokenWithLexeme(tokenType TokenType, lexeme string) {
	token := Token{
		Type:   tokenType,
		Lexeme: lexeme,
		Line:   l.line,
		Column: max(0, l.column-len([]rune(lexeme))),
		File:   l.file,
	}
	l.tokens = append(l.tokens, token)

	// afterFatArrow is only meaningful when => is the last significant token
	// before a newline. Clear it as soon as any other real token is emitted.
	if l.afterFatArrow && tokenType != TOKEN_FAT_ARROW && tokenType != TOKEN_COMMENT && tokenType != TOKEN_DIRECTIVE {
		l.afterFatArrow = false
	}

	// Track block keywords for brace block detection.
	// Set blockKeywordSeen when a keyword that introduces a block is emitted;
	// clear it when the block actually starts (INDENT, LBRACE) or when the
	// line ends (NEWLINE) — Go-style requires { on the same line.
	switch tokenType {
	case TOKEN_IF, TOKEN_FOR, TOKEN_FUNC, TOKEN_SWITCH, TOKEN_ELSE, TOKEN_SELECT, TOKEN_GO, TOKEN_DEFER:
		l.blockKeywordSeen = true
	case TOKEN_NEWLINE, TOKEN_INDENT, TOKEN_OF:
		// Clear on newline/indent (block uses indentation, not braces),
		// and on 'of' (list of T{...} / map of K to V{...} is a type literal).
		l.blockKeywordSeen = false
	}
}

func (l *Lexer) error(message string) {
	err := fmt.Errorf("%s:%d:%d: %s", l.file, l.line, l.column, message)
	l.errors = append(l.errors, err)
}

// Errors returns the individual lexer errors collected during scanning.
// Each error is formatted as "file:line:col: message".
func (l *Lexer) Errors() []error {
	return l.errors
}

// Character classification helpers

func isDigit(c rune) bool {
	return c >= '0' && c <= '9'
}

func hexDigit(c rune) (int, bool) {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0'), true
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10, true
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10, true
	default:
		return 0, false
	}
}

func isHexDigit(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func isOctalDigit(c rune) bool {
	return c >= '0' && c <= '7'
}

func isAlpha(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		c == '_'
}

func isAlphaNumeric(c rune) bool {
	return isAlpha(c) || isDigit(c)
}

// IsKeyword checks if a string is a keyword
func IsKeyword(s string) bool {
	_, ok := keywords[s]
	return ok
}

// mergeLineContinuations removes NEWLINE/INDENT/DEDENT tokens around pipe
// chains so the parser sees pipe expressions as single continuous lines.
// This is a post-tokenization pass that decouples pipe continuation handling
// from the indent stack logic in scanToken.
//
// Three patterns are handled:
//  1. Trailing pipe: PIPE [COMMENT*] NEWLINE [INDENT*] → keep PIPE/COMMENTs, remove NEWLINE/INDENTs
//  2. Leading pipe: NEWLINE [INDENT*] PIPE → remove NEWLINE/INDENTs (no DEDENTs allowed between)
//  3. Leading onerr: NEWLINE [INDENT*] ONERR → same as (2), only when in a pipe chain context
//
// For each INDENT absorbed, a corresponding DEDENT is also absorbed later.
func mergeLineContinuations(tokens []Token) []Token {
	result := make([]Token, 0, len(tokens))
	absorbedIndents := 0
	inPipeChain := false

	i := 0
	for i < len(tokens) {
		tok := tokens[i]

		// Absorb DEDENTs that correspond to removed INDENTs.
		if tok.Type == TOKEN_DEDENT && absorbedIndents > 0 {
			absorbedIndents--
			i++
			continue
		}

		// Rule 1: Trailing pipe — PIPE [COMMENT*] NEWLINE [INDENT*]
		if tok.Type == TOKEN_PIPE {
			result = append(result, tok)
			inPipeChain = true
			i++
			// Preserve comments after the pipe.
			for i < len(tokens) && tokens[i].Type == TOKEN_COMMENT {
				result = append(result, tokens[i])
				i++
			}
			if i < len(tokens) && tokens[i].Type == TOKEN_NEWLINE {
				// Look ahead past the NEWLINE for INDENTs.
				j := i + 1
				indents := 0
				for j < len(tokens) && tokens[j].Type == TOKEN_INDENT {
					indents++
					j++
				}
				// Continuation only if no DEDENT follows (the next line
				// is at the same or deeper indentation).
				if j < len(tokens) && tokens[j].Type != TOKEN_DEDENT && tokens[j].Type != TOKEN_EOF {
					absorbedIndents += indents
					i = j // skip NEWLINE and INDENTs
				}
			}
			continue
		}

		// Rule 2 & 3: Leading pipe/onerr — NEWLINE [INDENT*] PIPE|ONERR
		if tok.Type == TOKEN_NEWLINE {
			j := i + 1
			indents := 0
			for j < len(tokens) && tokens[j].Type == TOKEN_INDENT {
				indents++
				j++
			}
			if j < len(tokens) {
				isPipe := tokens[j].Type == TOKEN_PIPE
				isOnerr := tokens[j].Type == TOKEN_ONERR && inPipeChain
				if isPipe || isOnerr {
					absorbedIndents += indents
					i = j // skip NEWLINE and INDENTs, next iteration picks up PIPE/ONERR
					continue
				}
			}
			// Not a continuation — emit NEWLINE and end the pipe chain.
			result = append(result, tok)
			inPipeChain = false
			i++
			continue
		}

		result = append(result, tok)
		i++
	}

	return result
}
