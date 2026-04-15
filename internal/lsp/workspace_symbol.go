package lsp

import (
	"context"
	"encoding/json"
	"strings"

	lsp "github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
)

func (s *Server) handleWorkspaceSymbol(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	if req.Params == nil {
		return []lsp.SymbolInformation{}, nil
	}
	var params lsp.WorkspaceSymbolParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	query := strings.ToLower(params.Query)
	docs := s.documents.All()

	var results []lsp.SymbolInformation
	for _, doc := range docs {
		for _, sym := range s.getDocumentSymbols(doc) {
			if query == "" || strings.Contains(strings.ToLower(sym.Name), query) {
				results = append(results, sym)
			}
		}
	}

	if results == nil {
		return []lsp.SymbolInformation{}, nil
	}
	return results, nil
}
