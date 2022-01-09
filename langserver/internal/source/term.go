package source

import (
	"fmt"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/ast/location"
)

func (p *Project) SearchTargetTerm(location *location.Location) (term *ast.Term, err error) {
	// When parse err like following, we should term as "lib.".
	// module doesn't have `lib.`. we should get `lib` var, and then we change it as ref `lib.`
	// ```
	// import lib
	// lib.
	// ```
	policy := p.cache.Get(location.File)
	var isParseErrLocation bool
	for _, err := range policy.Errs {
		if err.Code == ast.ParseErr && err.Location.Offset == location.Offset {
			isParseErrLocation = true
			location.Col--
			location.Offset--
		}
	}

	for _, r := range policy.Module.Rules {
		if !in(location, r.Loc()) {
			continue
		}
		term, err = p.searchTargetTermInRule(location, r)
		if err != nil {
			return nil, err
		}
		break
	}

	if isParseErrLocation && term != nil {
		_, ok := term.Value.(ast.Var)
		if ok {
			term = &ast.Term{
				Location: &ast.Location{
					Row:    term.Location.Row,
					Col:    term.Location.Col,
					Offset: term.Location.Offset,
					Text:   []byte(term.Value.String() + "."),
					File:   term.Location.File,
				},
				Value: ast.Ref{
					term,
					{
						Location: location,
						Value:    ast.String(""),
					},
				},
			}
		}
	}
	return term, err
}

func (p *Project) searchTargetTermInRule(location *location.Location, rule *ast.Rule) (*ast.Term, error) {
	for _, b := range rule.Body {
		if !in(location, b.Loc()) {
			continue
		}

		switch t := b.Terms.(type) {
		case *ast.Term:
			return p.searchTargetTermInTerm(location, t)
		case []*ast.Term:
			return p.searchTargetTermInTerms(location, t)
		}
	}
	return nil, nil
}

func (p *Project) searchTargetTermInTerms(location *location.Location, terms []*ast.Term) (*ast.Term, error) {
	for _, t := range terms {
		if in(location, t.Loc()) {
			return p.searchTargetTermInTerm(location, t)
		}
	}
	return nil, nil
}

func (p *Project) searchTargetTermInTerm(loc *location.Location, term *ast.Term) (*ast.Term, error) {
	switch v := term.Value.(type) {
	case ast.Call:
		return p.searchTargetTermInTerms(loc, []*ast.Term(v))
	case ast.Ref:
		if len(v) == 1 {
			return p.searchTargetTermInTerm(loc, v[0])
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
			if ok1 && ok2 && (in(loc, v[0].Loc()) || in(loc, v[1].Loc())) {
				value := ast.Ref{v[0], v[1]}
				loc := v[0].Loc()
				return &ast.Term{Value: value, Location: &location.Location{
					Text:   []byte(value.String()),
					File:   loc.File,
					Row:    loc.Row,
					Col:    loc.Col,
					Offset: loc.Offset,
				}}, nil
			}
		}
		return p.searchTargetTermInTerms(loc, []*ast.Term(v))
	case *ast.Array:
		for i := 0; i < v.Len(); i++ {
			t, err := p.searchTargetTermInTerm(loc, v.Elem(i))
			if err != nil {
				return nil, err
			}
			if t == nil {
				continue
			}
			if in(loc, t.Loc()) {
				return t, nil
			}
		}
		return nil, nil
	case ast.Var:
		return term, nil
	case ast.String, ast.Boolean, ast.Number:
		return nil, nil
	default:
		return nil, fmt.Errorf("not supported type %T: %v\n", v, v)
	}
}
