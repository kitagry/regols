package langserver

import (
	"context"
	"encoding/json"

	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *handler) handleTextDocumentHover(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.TextDocumentPositionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	return h.documentIdent(ctx, params.TextDocument.URI, params.Position)
}

func (h *handler) documentIdent(ctx context.Context, uri lsp.DocumentURI, position lsp.Position) (lsp.Hover, error) {
	loc := h.toOPALocation(position, uri)
	documentResults, err := h.project.TermDocument(loc)
	if err != nil {
		return lsp.Hover{}, err
	}

	result := make([]lsp.MarkedString, len(documentResults))
	for i, d := range documentResults {
		result[i] = lsp.MarkedString{Language: "rego", Value: d.Content}
	}
	return lsp.Hover{Contents: result}, nil
}
