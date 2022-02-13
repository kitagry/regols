package langserver

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/regols/langserver/internal/lsp"
	"github.com/kitagry/regols/langserver/internal/source"
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
					Label:  "method",
					Kind:   source.FunctionItem,
					Detail: "detail",
					TextEdit: &source.TextEdit{
						Row:  1,
						Col:  1,
						Text: "method(a, b)",
					},
				},
				{
					Label:  "method",
					Kind:   source.FunctionItem,
					Detail: "detail",
					TextEdit: &source.TextEdit{
						Row:  1,
						Col:  1,
						Text: "method()",
					},
				},
				{
					Label:  "mes",
					Kind:   source.FunctionItem,
					Detail: "detail",
					TextEdit: &source.TextEdit{
						Row:  1,
						Col:  1,
						Text: "mes[a]",
					},
				},
				{
					Label:  "json.patch",
					Kind:   source.BuiltinFunctionItem,
					Detail: "(any, array[object<op: string, path: any>[any: any]]) => any",
					TextEdit: &source.TextEdit{
						Row:  1,
						Col:  1,
						Text: "json.patch(any, array[object<op: string, path: any>[any: any]] => any)",
					},
				},
				{
					Label: "lib",
					Kind:  source.PackageItem,
					AdditionalTextEdits: []source.TextEdit{
						{
							Row:  3,
							Col:  1,
							Text: "import data.lib\n",
						},
					},
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
						TextEdit: &lsp.TextEdit{
							Range: lsp.Range{
								Start: lsp.Position{
									Line:      0,
									Character: 0,
								},
								End: lsp.Position{
									Line:      0,
									Character: len("method(a, b)"),
								},
							},
							NewText: "method(${1:a}, ${2:b})",
						},
						AdditionalTextEdits: []lsp.TextEdit{},
					},
					{
						Label:            "method",
						Kind:             lsp.CIKFunction,
						Detail:           "detail",
						InsertTextFormat: lsp.ITFSnippet,
						TextEdit: &lsp.TextEdit{
							Range: lsp.Range{
								Start: lsp.Position{
									Line:      0,
									Character: 0,
								},
								End: lsp.Position{
									Line:      0,
									Character: len("method()"),
								},
							},
							NewText: "method()",
						},
						AdditionalTextEdits: []lsp.TextEdit{},
					},
					{
						Label:            "mes",
						Kind:             lsp.CIKFunction,
						Detail:           "detail",
						InsertTextFormat: lsp.ITFSnippet,
						TextEdit: &lsp.TextEdit{
							Range: lsp.Range{
								Start: lsp.Position{
									Line:      0,
									Character: 0,
								},
								End: lsp.Position{
									Line:      0,
									Character: len("mes[a]"),
								},
							},
							NewText: "mes[${1:a}]",
						},
						AdditionalTextEdits: []lsp.TextEdit{},
					},
					{
						Label:            "json.patch",
						Kind:             lsp.CIKFunction,
						Detail:           "(any, array[object<op: string, path: any>[any: any]]) => any",
						InsertTextFormat: lsp.ITFSnippet,
						TextEdit: &lsp.TextEdit{
							Range: lsp.Range{
								Start: lsp.Position{
									Line:      0,
									Character: 0,
								},
								End: lsp.Position{
									Line:      0,
									Character: len("json.patch(any, array[object<op: string, path: any>[any: any]] => any)"),
								},
							},
							NewText: "json.patch(${1:any}, ${2:array[object<op: string, path: any>[any: any]] => any})",
						},
						AdditionalTextEdits: []lsp.TextEdit{},
					},
					{
						Label:            "lib",
						Kind:             lsp.CIKModule,
						InsertTextFormat: lsp.ITFSnippet,
						AdditionalTextEdits: []lsp.TextEdit{
							{
								Range: lsp.Range{
									Start: lsp.Position{
										Line:      2,
										Character: 0,
									},
									End: lsp.Position{
										Line:      2,
										Character: 0,
									},
								},
								NewText: "import data.lib\n",
							},
						},
					},
				},
			},
		},
		"client doesn't support snippet": {
			items: []source.CompletionItem{
				{
					Label:  "method",
					Kind:   source.FunctionItem,
					Detail: "detail",
					TextEdit: &source.TextEdit{
						Row:  1,
						Col:  1,
						Text: "method(a, b)",
					},
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
