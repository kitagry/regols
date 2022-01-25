package source

import (
	"fmt"

	"github.com/open-policy-agent/opa/ast"
)

func (p *Project) SearchTargetTerm(location *ast.Location) (term *ast.Term, err error) {
	// When parse err like following, we should term as "lib.".
	// module doesn't have `lib.`. we should get `lib` var, and then we change it as ref `lib.`
	// ```
	// import lib
	// lib.
	// ```
	policy := p.cache.Get(location.File)
	if policy == nil {
		return nil, nil
	}

	var isParseErrLocation bool
	for _, err := range policy.Errs {
		if err.Code == ast.ParseErr && err.Location.Offset == location.Offset {
			isParseErrLocation = true
			location.Col--
			location.Offset--
		}
	}

	if policy.Module == nil {
		return nil, nil
	}

	for _, imp := range policy.Module.Imports {
		if imp.Path == nil || !in(location, imp.Path.Loc()) {
			continue
		}
		term, err = p.searchTargetTermInImport(location, imp)
		if err != nil {
			return nil, err
		}
		break
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

func (p *Project) searchTargetTermInImport(location *ast.Location, imp *ast.Import) (*ast.Term, error) {
	if in(location, imp.Path.Loc()) {
		return imp.Path, nil
	}
	return nil, nil
}

func (p *Project) searchTargetTermInRule(location *ast.Location, rule *ast.Rule) (*ast.Term, error) {
	for rule != nil {
		if rule.Head != nil {
			if rule.Head.Value != nil && in(location, rule.Head.Value.Loc()) {
				return p.searchTargetTermInTerm(location, rule.Head.Value)
			}
		}
		for _, b := range rule.Body {
			if !in(location, b.Loc()) {
				continue
			}

			switch t := b.Terms.(type) {
			case *ast.Term:
				if in(location, t.Loc()) {
					return p.searchTargetTermInTerm(location, t)
				}
			case []*ast.Term:
				return p.searchTargetTermInTerms(location, t)
			}
		}
		rule = rule.Else
	}
	return nil, nil
}

func (p *Project) searchTargetTermInTerms(location *ast.Location, terms []*ast.Term) (*ast.Term, error) {
	for _, t := range terms {
		if in(location, t.Loc()) {
			return p.searchTargetTermInTerm(location, t)
		}
	}
	return nil, nil
}

func (p *Project) searchTargetTermInTerm(loc *ast.Location, term *ast.Term) (*ast.Term, error) {
	switch v := term.Value.(type) {
	case ast.Call:
		return p.searchTargetTermInTerms(loc, []*ast.Term(v))
	case ast.Ref:
		if len(v) > 0 && in(loc, v[0].Loc()) {
			return v[0], nil
		}
		for i, t := range v {
			if in(loc, t.Loc()) {
				value := v[:i+1]
				return &ast.Term{Value: value, Location: &ast.Location{
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

func in(target, src *ast.Location) bool {
	return target.Offset >= src.Offset && target.Offset <= (src.Offset+len(src.Text))
}
