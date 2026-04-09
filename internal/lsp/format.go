package lsp

import (
	"context"
	"encoding/json"
	"log"

	"github.com/kukichalang/kukicha/internal/formatter"
	golsp "github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
)

func (s *Server) handleFormatting(ctx context.Context, req *jsonrpc2.Request) ([]golsp.TextEdit, error) {
	if req.Params == nil {
		return nil, nil
	}
	var params golsp.DocumentFormattingParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	doc := s.documents.Get(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	opts := formatter.DefaultOptions()
	formatted, err := formatter.Format(doc.Content, uriToFilename(doc.URI), opts)
	if err != nil {
		log.Printf("Format error: %v", err)
		return nil, nil // return no edits on format failure
	}

	if formatted == doc.Content {
		return nil, nil // no changes needed
	}

	// Replace entire document content
	lastLine := len(doc.Lines) - 1
	if lastLine < 0 {
		lastLine = 0
	}
	lastChar := 0
	if lastLine < len(doc.Lines) {
		lastChar = len(doc.Lines[lastLine])
	}

	return []golsp.TextEdit{
		{
			Range: golsp.Range{
				Start: golsp.Position{Line: 0, Character: 0},
				End:   golsp.Position{Line: lastLine, Character: lastChar},
			},
			NewText: formatted,
		},
	}, nil
}
