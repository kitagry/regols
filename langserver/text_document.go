package langserver

import (
	"context"
	"encoding/json"

	"github.com/kitagry/regols/langserver/internal/lsp"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *handler) handleTextDocumentDidOpen(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.DidOpenTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	h.updateDocument(params.TextDocument.URI, params.TextDocument.Text, params.TextDocument.Version)

	return nil, nil
}

func (h *handler) handleTextDocumentDidChange(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.DidChangeTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	h.updateDocument(params.TextDocument.URI, params.ContentChanges[0].Text, params.TextDocument.Version)

	return nil, nil
}

func (h *handler) handleTextDocumentDidClose(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.DidCloseTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	h.project.DeleteFile(documentURIToURI(params.TextDocument.URI))

	return nil, nil
}

func (h *handler) handleTextDocumentDidSave(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (result any, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.DidSaveTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	h.diagnosticRequest <- params.TextDocument.URI

	return nil, nil
}

func (h *handler) updateDocument(uri lsp.DocumentURI, text string, version int) {
	h.project.UpdateFile(documentURIToURI(uri), text, version)
	h.diagnosticRequest <- uri
}
