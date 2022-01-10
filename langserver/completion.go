package langserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

	return completionItemToLspCompletionList(items, h.clientSupportSnippets()), nil
}

func (h *handler) clientSupportSnippets() bool {
	return h.initializeParams.Capabilities.TextDocument.Completion.CompletionItem.SnippetSupport
}

func completionItemToLspCompletionList(items []source.CompletionItem, isSnippetSupport bool) lsp.CompletionList {
	insertTextFormat := lsp.ITFPlainText
	if isSnippetSupport {
		insertTextFormat = lsp.ITFSnippet
	}

	completoinItems := make([]lsp.CompletionItem, len(items))
	for i, c := range items {
		completoinItems[i] = createCompletionItem(c, insertTextFormat)
	}

	return lsp.CompletionList{
		IsIncomplete: false,
		Items:        completoinItems,
	}
}

func createCompletionItem(completionItem source.CompletionItem, insertTextFormat lsp.InsertTextFormat) lsp.CompletionItem {
	if insertTextFormat == lsp.ITFPlainText {
		return lsp.CompletionItem{
			Label:            completionItem.Label,
			Kind:             kindToLspKind(completionItem.Kind),
			Detail:           completionItem.Detail,
			InsertTextFormat: insertTextFormat,
		}
	}

	return lsp.CompletionItem{
		Label:            completionItem.Label,
		Kind:             kindToLspKind(completionItem.Kind),
		Detail:           completionItem.Detail,
		InsertText:       createSnippetText(completionItem.InsertText, completionItem.Kind),
		InsertTextFormat: insertTextFormat,
	}
}

func kindToLspKind(kind source.CompletionKind) lsp.CompletionItemKind {
	switch kind {
	case source.VariableItem:
		return lsp.CIKVariable
	case source.PackageItem:
		return lsp.CIKModule
	case source.FunctionItem, source.BuiltinFunctionItem:
		return lsp.CIKFunction
	default:
		return lsp.CIKText
	}
}

func createSnippetText(insertText string, kind source.CompletionKind) string {
	switch kind {
	case source.FunctionItem, source.BuiltinFunctionItem:
		if i := strings.Index(insertText, "("); i >= 0 {
			return addFunctionSnippet(insertText, "(", ")")
		}

		if i := strings.Index(insertText, "["); i >= 0 {
			return addFunctionSnippet(insertText, "[", "]")
		}
		return insertText
	default:
		return insertText
	}
}

func addFunctionSnippet(insertText string, lbracket string, rbracket string) string {
	if i := strings.Index(insertText, lbracket); i >= 0 {
		trimmed := insertText[:i]
		argStr := strings.Trim(insertText[i:], lbracket+rbracket)
		if len(argStr) == 0 {
			return trimmed + lbracket + rbracket
		}

		args := make([]string, 0)
		startInd := 0
		brackets := make([]rune, 0)
		branketsPair := map[rune]rune{'(': ')', '[': ']', '<': '>'}
		for i, b := range argStr {
			switch b {
			case ',':
				if len(brackets) == 0 {
					args = append(args, strings.TrimSpace(argStr[startInd:i]))
					startInd = i + 1
				}
			case '(', '[', '<':
				brackets = append(brackets, branketsPair[b])
			case ')', ']', '>':
				if len(brackets) > 0 && brackets[len(brackets)-1] == b {
					brackets = brackets[0 : len(brackets)-1]
				}
			}
		}
		args = append(args, strings.TrimSpace(argStr[startInd:]))

		snippetArgs := make([]string, len(args))
		for i, a := range args {
			snippetArgs[i] = fmt.Sprintf("${%d:%s}", i+1, a)
		}
		return trimmed + lbracket + strings.Join(snippetArgs, ", ") + rbracket
	}
	return insertText
}
