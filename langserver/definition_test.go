package langserver

import (
	"testing"

	"github.com/sourcegraph/go-lsp"
)

func TestIn(t *testing.T) {
	tests := map[string]struct {
		position lsp.Position
		location lsp.Location

		expect bool
	}{
		"position equals location.range.start": {
			position: lsp.Position{
				Line:      10,
				Character: 1,
			},
			location: lsp.Location{
				Range: lsp.Range{
					Start: lsp.Position{
						Line:      10,
						Character: 1,
					},
					End: lsp.Position{
						Line:      10,
						Character: 5,
					},
				},
			},
			expect: true,
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			actual := in(tt.position, tt.location)
			if actual != tt.expect {
				t.Errorf("in expect %t, got %t", tt.expect, actual)
			}
		})
	}
}

func TestGetWord(t *testing.T) {
	tests := map[string]struct {
		input  string
		index  int
		expect string
	}{
		"getWord from first": {
			input:  "hoge(fuga)",
			index:  0,
			expect: "hoge",
		},
		"getWord from second": {
			input:  "hoge(fuga)",
			index:  5,
			expect: "fuga",
		},
		". is in word": {
			input:  "test.hoge(fuga)",
			index:  0,
			expect: "test.hoge",
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			got := getWord(tt.input, tt.index)
			if got != tt.expect {
				t.Errorf(`getWord expect="%s", got="%s"`, tt.expect, got)
			}
		})
	}
}
