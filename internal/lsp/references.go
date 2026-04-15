package lsp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kukichalang/kukicha/internal/ast"
	lsp "github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
)

func (s *Server) handleReferences(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	if req.Params == nil {
		return []lsp.Location{}, nil
	}
	var params lsp.ReferenceParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	doc := s.documents.Get(params.TextDocument.URI)
	if doc == nil || doc.Program == nil {
		return []lsp.Location{}, nil
	}

	word := doc.GetWordAtPosition(params.Position)
	if word == "" {
		return []lsp.Location{}, nil
	}

	results := s.findAllReferences(word, params.Context.IncludeDeclaration)
	return results, nil
}

func (s *Server) findAllReferences(word string, includeDeclaration bool) []lsp.Location {
	docs := s.documents.All()

	// Collect all identifier positions matching word across all open documents.
	var locs []lsp.Location
	for _, doc := range docs {
		if doc.Program == nil {
			continue
		}
		walkProgramIdentifiers(doc.Program, func(name string, pos ast.Position) {
			if name == word {
				locs = append(locs, lsp.Location{URI: doc.URI, Range: astPosToRange(pos)})
			}
		})
	}

	if includeDeclaration || len(locs) == 0 {
		if locs == nil {
			return []lsp.Location{}
		}
		return locs
	}

	// Build set of declaration locations to exclude.
	defKeys := make(map[string]bool)
	for _, doc := range docs {
		if def := s.findDefinition(doc, word); def != nil {
			defKeys[locKey(def.URI, def.Range.Start.Line, def.Range.Start.Character)] = true
		}
	}

	if len(defKeys) == 0 {
		return locs
	}

	filtered := locs[:0]
	for _, loc := range locs {
		if !defKeys[locKey(loc.URI, loc.Range.Start.Line, loc.Range.Start.Character)] {
			filtered = append(filtered, loc)
		}
	}
	return filtered
}

// astPosToRange converts a 1-indexed AST position to a 0-indexed LSP Range.
// Start == End (point range); the client uses the word boundaries for highlighting.
func astPosToRange(pos ast.Position) lsp.Range {
	line := pos.Line - 1
	col := pos.Column - 1
	if line < 0 {
		line = 0
	}
	if col < 0 {
		col = 0
	}
	p := lsp.Position{Line: line, Character: col}
	return lsp.Range{Start: p, End: p}
}

func locKey(uri lsp.DocumentURI, line, char int) string {
	return fmt.Sprintf("%s:%d:%d", uri, line, char)
}
