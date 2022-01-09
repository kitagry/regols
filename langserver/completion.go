package langserver

import (
	"context"
	"encoding/json"

	"github.com/kitagry/regols/langserver/internal/source"
	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *handler) handleTextDocumentCompletion(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.TextDocumentPositionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	location := h.toOPALocation(params.Position, params.TextDocument.URI)

	items, err := h.project.ListCompletionItems(location)
	if err != nil {
		return nil, err
	}

	return completionItemToLspCompletionList(items), nil
}

func completionItemToLspCompletionList(items []source.CompletionItem) lsp.CompletionList {
	completoinItems := make([]lsp.CompletionItem, len(items))
	for i, c := range items {
		completoinItems[i] = lsp.CompletionItem{
			Label: c.Label,
			Kind:  kindToLspKind(c.Kind),
		}
	}

	return lsp.CompletionList{
		IsIncomplete: false,
		Items:        completoinItems,
	}
}

func kindToLspKind(kind source.CompletionKind) lsp.CompletionItemKind {
	switch kind {
	case source.VariableItem:
		return lsp.CIKVariable
	case source.PackageItem:
		return lsp.CIKModule
	case source.FunctionItem:
		return lsp.CIKFunction
	default:
		return lsp.CIKText
	}
}
