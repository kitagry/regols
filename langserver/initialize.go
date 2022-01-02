package langserver

import (
	"context"
	"encoding/json"

	"github.com/kitagry/rego-langserver/langserver/internal/cache"
	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *handler) handleInitialize(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	h.conn = conn

	var params lsp.InitializeParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	p, err := cache.NewProject(params.RootPath)
	if err != nil {
		return nil, err
	}
	h.project = p

	return lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync: &lsp.TextDocumentSyncOptionsOrKind{
				Kind: tdskToPTr(lsp.TDSKFull),
			},
			DocumentFormattingProvider: true,
		},
	}, nil
}

func tdskToPTr(s lsp.TextDocumentSyncKind) *lsp.TextDocumentSyncKind {
	return &s
}
