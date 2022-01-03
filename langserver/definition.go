package langserver

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/open-policy-agent/opa/ast/location"
	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *handler) handleTextDocumentDefinition(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.TextDocumentPositionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	return h.lookupIdent(ctx, params.TextDocument.URI, params.Position)
}

func (h *handler) lookupIdent(ctx context.Context, uri lsp.DocumentURI, position lsp.Position) ([]lsp.Location, error) {
	path := documentURIToURI(uri)
	file, ok := h.project.GetFile(path)
	if !ok {
		return nil, nil
	}
	loc := toOPALocation(position, file.RowText)
	h.logger.Println(loc)
	lookupResults, err := h.project.LookupDefinition(path, loc)
	if err != nil {
		h.logger.Printf("failed to get definition: %v", err)
		return nil, nil
	}

	result := make([]lsp.Location, 0, len(lookupResults))
	for _, r := range lookupResults {
		rawFile, err := h.project.GetRawText(r.Path)
		if err != nil {
			continue
		}
		location := toLspLocation(r.Rule.Loc(), rawFile)
		location.URI = uriToDocumentURI(r.Path)
		result = append(result, location)
	}

	return result, nil
}

func toOPALocation(position lsp.Position, rawText string) *location.Location {
	startInd := 0
	for i := 0; i < position.Line; i++ {
		startInd += strings.Index(rawText[startInd:], "\n") + 1
	}
	startInd += position.Character

	return &location.Location{
		Row:    position.Line + 1,
		Col:    position.Character + 1,
		Offset: startInd,
	}
}

func toLspLocation(location *location.Location, rawText string) lsp.Location {
	if location == nil {
		return lsp.Location{Range: lsp.Range{Start: lsp.Position{}, End: lsp.Position{}}}
	}
	start := lsp.Position{
		Line:      location.Row - 1,
		Character: location.Col - 1,
	}

	endOffset := location.Offset + len(location.Text) - 1
	toEndText := rawText[:endOffset]
	line := strings.Count(toEndText, "\n")
	newLineInd := strings.LastIndex(toEndText, "\n")
	char := len(toEndText[newLineInd:]) - 1

	return lsp.Location{
		Range: lsp.Range{
			Start: start,
			End: lsp.Position{
				Line:      line,
				Character: char,
			},
		},
	}
}
