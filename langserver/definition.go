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
	loc := h.toOPALocation(position, uri)
	lookupResults, err := h.project.LookupDefinition(loc)
	if err != nil {
		h.logger.Printf("failed to get definition: %v", err)
		return nil, nil
	}

	result := make([]lsp.Location, 0, len(lookupResults))
	for _, r := range lookupResults {
		rawFile, err := h.project.GetRawText(r.File)
		if err != nil {
			continue
		}
		location := toLspLocation(r, rawFile)
		location.URI = uriToDocumentURI(r.File)
		result = append(result, location)
	}

	return result, nil
}

func (h *handler) toOPALocation(position lsp.Position, uri lsp.DocumentURI) *location.Location {
	path := documentURIToURI(uri)
	file, ok := h.project.GetFile(path)
	if !ok {
		return nil
	}

	startInd := 0
	for i := 0; i < position.Line; i++ {
		startInd += strings.Index(file.RowText[startInd:], "\n") + 1
	}
	startInd += position.Character

	return &location.Location{
		Row:    position.Line + 1,
		Col:    position.Character + 1,
		Offset: startInd,
		File:   path,
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
