package langserver

import (
	"context"
	"encoding/json"

	"github.com/kitagry/regols/langserver/internal/lsp"
	"github.com/kitagry/regols/langserver/internal/source"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *handler) handleInitialize(_ context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	h.conn = conn

	var params lsp.InitializeParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}
	h.initializeParams = params

	p, err := source.NewProject(params.RootPath)
	if err != nil {
		return nil, err
	}
	h.project = p

	return lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync: &lsp.TextDocumentSyncOptionsOrKind{
				Kind: toPtr(lsp.TDSKFull),
			},
			DocumentFormattingProvider: true,
			DefinitionProvider:         true,
			HoverProvider:              true,
			ReferencesProvider:         true,
			CompletionProvider: &lsp.CompletionOptions{
				TriggerCharacters: []string{"*", "."},
				ResolveProvider:   true,
			},
		},
	}, nil
}

func toPtr[T any](t T) *T {
	return &t
}
