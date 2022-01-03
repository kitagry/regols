package langserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/open-policy-agent/opa/ast"
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
	module := h.project.GetModule(path)
	if module == nil {
		return nil, fmt.Errorf("failed to get file: %s", uri)
	}

	rawFile, err := h.getFile(path)
	if err != nil {
		return nil, nil
	}

	word := h.lookupRules(position, module.Rules, rawFile)
	if word == nil {
		return nil, nil
	}
	lookupResults := h.project.LookUp(word, documentURIToURI(uri))

	result := make([]lsp.Location, 0, len(lookupResults))
	for _, r := range lookupResults {
		rawFile, err := h.getFile(r.Path)
		if err != nil {
			continue
		}
		location := getLocation(r.Rule.Loc(), rawFile)
		location.URI = uriToDocumentURI(r.Path)
		result = append(result, location)
	}

	return result, nil
}

func (h *handler) lookupRules(position lsp.Position, rules []*ast.Rule, rawText string) *ast.Term {
	for _, r := range rules {
		location := r.Loc()
		loc := getLocation(location, rawText)
		if !in(position, loc) {
			continue
		}
		return h.lookupRule(position, r, rawText)
	}
	return nil
}

func (h *handler) lookupRule(position lsp.Position, rule *ast.Rule, rawText string) *ast.Term {
	for _, b := range rule.Body {
		loc := b.Loc()
		location := getLocation(loc, rawText)
		if !in(position, location) {
			continue
		}

		switch t := b.Terms.(type) {
		case *ast.Term:
			return h.lookupTerm(position, t, rawText)
		case []*ast.Term:
			return h.lookupTerms(position, t, rawText)
		}
	}
	return nil
}

func (h *handler) lookupTerm(position lsp.Position, term *ast.Term, rawText string) *ast.Term {
	switch v := term.Value.(type) {
	case ast.Call:
		return h.lookupTerms(position, []*ast.Term(v), rawText)
	case ast.Ref:
		if len(v) == 1 {
			return h.lookupTerm(position, v[0], rawText)
		}
		if len(v) >= 2 {
			// This is for imported method
			// If you use the following code.
			// ```
			// import data.lib.util
			// util.test[hoge]
			// ```
			// Then
			// util.test[hoge] <- ast.Ref
			// util <- ast.Var
			// test <- ast.String
			// I think this is a bit wrong...
			// https://www.openpolicyagent.org/docs/latest/policy-reference/#grammar
			_, ok1 := v[0].Value.(ast.Var)
			_, ok2 := v[1].Value.(ast.String)
			if ok1 && ok2 && (in(position, getLocation(v[0].Loc(), rawText)) || in(position, getLocation(v[1].Loc(), rawText))) {
				value := ast.Ref{v[0], v[1]}
				loc := v[0].Loc()
				return &ast.Term{Value: value, Location: &location.Location{
					Text:   []byte(value.String()),
					File:   loc.File,
					Row:    loc.Row,
					Col:    loc.Col,
					Offset: loc.Offset,
				}}
			}
		}
		return h.lookupTerms(position, []*ast.Term(v), rawText)
	case *ast.Array:
		for i := 0; i < v.Len(); i++ {
			t := h.lookupTerm(position, v.Elem(i), rawText)
			if t == nil {
				continue
			}
			loc := t.Loc()
			location := getLocation(loc, rawText)
			if in(position, location) {
				return t
			}
		}
		return nil
	case ast.Var:
		return term
	case ast.String, ast.Boolean, ast.Number:
		return nil
	default:
		h.logger.Printf("not certained type %T: %v\n", v, v)
		return nil
	}
}

func (h *handler) lookupTerms(position lsp.Position, terms []*ast.Term, rawText string) *ast.Term {
	for _, t := range terms {
		loc := t.Loc()
		location := getLocation(loc, rawText)
		if in(position, location) {
			return h.lookupTerm(position, t, rawText)
		}
	}
	return nil
}

func (h *handler) getFile(path string) (string, error) {
	if f, ok := h.project.GetFile(path); ok {
		return f.RowText, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var buf bytes.Buffer
	buf.ReadFrom(f)
	return buf.String(), nil
}

func in(position lsp.Position, location lsp.Location) bool {
	startCond := location.Range.Start.Line < position.Line || (location.Range.Start.Line == position.Line && location.Range.Start.Character <= position.Character)
	endCond := position.Line < location.Range.End.Line || (position.Line == location.Range.End.Line && position.Character <= location.Range.End.Character)
	return startCond && endCond
}

func getLocation(location *location.Location, rawText string) lsp.Location {
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
