package langserver

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/regols/langserver/internal/lsp"
	"github.com/open-policy-agent/opa/ast"
)

func TestToLspLocation(t *testing.T) {
	tests := map[string]struct {
		location *ast.Location
		rawText  string
		expect   lsp.Location
	}{
		"location is row 1 col 1": {
			location: &ast.Location{
				Row:    1,
				Col:    1,
				Offset: 0,
				Text:   []byte("hello"),
				File:   "src.rego",
			},
			rawText: `hello`,
			expect: lsp.Location{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 0, Character: 4},
				},
			},
		},
		"location is row 2 col 2": {
			location: &ast.Location{
				Row:    2,
				Col:    1,
				Offset: len("hello\n"),
				Text:   []byte("world"),
				File:   "src.rego",
			},
			rawText: `hello
world`,
			expect: lsp.Location{
				Range: lsp.Range{
					Start: lsp.Position{Line: 1, Character: 0},
					End:   lsp.Position{Line: 1, Character: 4},
				},
			},
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			got := toLspLocation(tt.location, tt.rawText)
			if diff := cmp.Diff(tt.expect, got); diff != "" {
				t.Errorf("toLspLocation result diff (-expect, +got)\n%s", diff)
			}
		})
	}
}
