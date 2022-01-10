package langserver

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/regols/langserver/internal/source"
	"github.com/sourcegraph/go-lsp"
)

func TestCompletionItemToLspCompletionList(t *testing.T) {
	tests := map[string]struct {
		items            []source.CompletionItem
		isSnippetSupport bool

		expectCompletionList lsp.CompletionList
	}{
		"client snippet support": {
			items: []source.CompletionItem{
				{
					Label:      "method",
					Kind:       source.FunctionItem,
					Detail:     "detail",
					InsertText: "method(a, b)",
				},
				{
					Label:      "method",
					Kind:       source.FunctionItem,
					Detail:     "detail",
					InsertText: "method()",
				},
				{
					Label:      "mes",
					Kind:       source.FunctionItem,
					Detail:     "detail",
					InsertText: "mes[a]",
				},
			},
			isSnippetSupport: true,
			expectCompletionList: lsp.CompletionList{
				IsIncomplete: false,
				Items: []lsp.CompletionItem{
					{
						Label:            "method",
						Kind:             lsp.CIKFunction,
						Detail:           "detail",
						InsertTextFormat: lsp.ITFSnippet,
						InsertText:       "method(${1:a}, ${2:b})",
					},
					{
						Label:            "method",
						Kind:             lsp.CIKFunction,
						Detail:           "detail",
						InsertTextFormat: lsp.ITFSnippet,
						InsertText:       "method()",
					},
					{
						Label:            "mes",
						Kind:             lsp.CIKFunction,
						Detail:           "detail",
						InsertTextFormat: lsp.ITFSnippet,
						InsertText:       "mes[${1:a}]",
					},
				},
			},
		},
		"client doesn't support snippet": {
			items: []source.CompletionItem{
				{
					Label:      "method",
					Kind:       source.FunctionItem,
					Detail:     "detail",
					InsertText: "method(a, b)",
				},
			},
			isSnippetSupport: false,
			expectCompletionList: lsp.CompletionList{
				IsIncomplete: false,
				Items: []lsp.CompletionItem{
					{
						Label:            "method",
						Kind:             lsp.CIKFunction,
						Detail:           "detail",
						InsertTextFormat: lsp.ITFPlainText,
					},
				},
			},
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			got := completionItemToLspCompletionList(tt.items, tt.isSnippetSupport)
			if diff := cmp.Diff(tt.expectCompletionList, got); diff != "" {
				t.Errorf("completionItemToLspCompletionList result diff (-expect, +got)\n%s", diff)
			}
		})
	}
}
