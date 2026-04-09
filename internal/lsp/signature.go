package lsp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/kukichalang/kukicha/internal/ast"
	golsp "github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
)

func (s *Server) handleSignatureHelp(ctx context.Context, req *jsonrpc2.Request) (*golsp.SignatureHelp, error) {
	if req.Params == nil {
		return nil, nil
	}
	var params golsp.TextDocumentPositionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	doc := s.documents.Get(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	// Find the function name by scanning backwards from cursor for an open paren
	line := doc.GetLineContent(int(params.Position.Line))
	col := utf16PosToByteOffset(line, int(params.Position.Character))
	funcName, activeParam := findCallContext(line, col)
	if funcName == "" {
		return nil, nil
	}

	// Look up function signature
	sig := findSignature(doc, funcName)
	if sig == nil {
		return nil, nil
	}

	result := &golsp.SignatureHelp{
		Signatures:      []golsp.SignatureInformation{*sig},
		ActiveSignature: 0,
		ActiveParameter: activeParam,
	}
	return result, nil
}

// findCallContext scans backwards from col to find the function name and which
// argument index the cursor is at. Returns ("", 0) if not in a call.
func findCallContext(line string, col int) (string, int) {
	if col > len(line) {
		col = len(line)
	}

	// Count commas at current paren depth to determine active parameter
	commas := 0
	depth := 0
	parenPos := -1

	for i := col - 1; i >= 0; i-- {
		switch line[i] {
		case ')':
			depth++
		case '(':
			if depth == 0 {
				parenPos = i
				goto found
			}
			depth--
		case ',':
			if depth == 0 {
				commas++
			}
		}
	}
	return "", 0

found:
	// Extract function name before the paren
	end := parenPos
	for end > 0 && line[end-1] == ' ' {
		end--
	}
	start := end
	for start > 0 && isIdentifierChar(line[start-1]) {
		start--
	}
	if start == end {
		return "", 0
	}

	// Handle method calls: skip past the dot and object
	name := line[start:end]

	return name, commas
}

// findSignature looks up a function by name in builtins, then AST declarations.
func findSignature(doc *Document, name string) *golsp.SignatureInformation {
	// Check builtins
	for _, b := range builtins {
		if b.Name == name {
			return builtinToSignature(b)
		}
	}

	if doc.Program == nil {
		return nil
	}

	// Check AST declarations
	for _, decl := range doc.Program.Declarations {
		fd, ok := decl.(*ast.FunctionDecl)
		if !ok || fd.Name.Value != name {
			continue
		}
		return functionDeclToSignature(fd)
	}

	return nil
}

func builtinToSignature(b BuiltinInfo) *golsp.SignatureInformation {
	// Parse parameter labels from the signature string
	// e.g. "func print(args ...any)" -> extract "args ...any"
	sig := &golsp.SignatureInformation{
		Label:         b.Signature,
		Documentation: b.Doc,
	}

	// Extract params between parens
	start := strings.Index(b.Signature, "(")
	end := strings.LastIndex(b.Signature, ")")
	if start >= 0 && end > start {
		paramStr := b.Signature[start+1 : end]
		if paramStr != "" {
			for _, p := range strings.Split(paramStr, ", ") {
				sig.Parameters = append(sig.Parameters, golsp.ParameterInformation{
					Label: strings.TrimSpace(p),
				})
			}
		}
	}

	return sig
}

func functionDeclToSignature(fd *ast.FunctionDecl) *golsp.SignatureInformation {
	label := formatFunctionDecl(fd)
	sig := &golsp.SignatureInformation{
		Label: label,
	}

	for _, param := range fd.Parameters {
		paramLabel := param.Name.Value + " " + formatTypeAnnotation(param.Type)
		if param.Variadic {
			paramLabel = "many " + paramLabel
		}
		sig.Parameters = append(sig.Parameters, golsp.ParameterInformation{
			Label: paramLabel,
		})
	}

	return sig
}
