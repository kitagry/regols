package langserver

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp/syntax"
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
	module := h.project.GetModule(documentURIToURI(uri))
	if module == nil {
		return nil, fmt.Errorf("failed to get file: %s", uri)
	}

	rule := h.lookupRules(position, module.Rules, module.String())
	if rule == nil {
		return nil, fmt.Errorf("failed to get rule")
	}
	word := h.lookupBody(position, rule.Body, module.String())
	if word == nil {
		return nil, nil
	}
	rules, path := h.project.LookupMethod(word.String(), documentURIToURI(uri))

	module = h.project.GetModule(path)
	if module == nil {
		return nil, fmt.Errorf("failed to get file: %s", path)
	}

	result := make([]lsp.Location, len(rules))
	for i, r := range rules {
		location := getLocation(r.Loc(), module.String())
		location.URI = uriToDocumentURI(path)
		result[i] = location
	}

	return result, nil
}

func (h *handler) lookupRules(position lsp.Position, rules []*ast.Rule, rawText string) *ast.Rule {
	for _, r := range rules {
		location := r.Loc()
		loc := getLocation(location, rawText)
		if in(position, loc) {
			r := r
			return r
		}
	}
	return nil
}

func (h *handler) lookupBody(position lsp.Position, body ast.Body, rawText string) *ast.Term {
	for _, b := range body {
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
	h.logger.Printf("%T: %v", term.Value, term)
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

	endOffset := location.Offset + len(location.Text)
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

func getWord(s string, ind int) string {
	startInd := 0
	for i, st := range s {
		if !(syntax.IsWordChar(st) || st == rune('.')) {
			if ind < i {
				return s[startInd:i]
			}
			startInd = i + 1
		}
	}
	return ""
}
