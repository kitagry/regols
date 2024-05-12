package langserver

import (
	"context"
	"encoding/json"

	"github.com/kitagry/regols/langserver/internal/lsp"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *handler) handleTextDocumentReferences(ctx context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.TextDocumentPositionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	return h.lookupReferences(ctx, params.TextDocument.URI, params.Position)
}

func (h *handler) lookupReferences(_ context.Context, uri lsp.DocumentURI, position lsp.Position) ([]lsp.Location, error) {
	loc := h.toOPALocation(position, uri)
	locations, err := h.project.LookupReferences(loc)
	if err != nil {
		h.logger.Printf("failed to get references: %v", err)
		return nil, nil
	}

	result := make([]lsp.Location, 0, len(locations))
	for _, r := range locations {
		rawFile, err := h.project.GetRawText(r.File)
		if err != nil {
			continue
		}
		location := toLspLocation(r, rawFile)
		location.URI = uriToDocumentURI(r.File)
		result = append(result, location)
	}
	return result, nil
}
