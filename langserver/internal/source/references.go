package source

import (
	"fmt"
	"os"
	"sort"

	"github.com/open-policy-agent/opa/ast"
)

func (p *Project) LookupReferences(loc *ast.Location) ([]*ast.Location, error) {
	term, err := p.SearchTargetTerm(loc)
	if err != nil {
		return nil, err
	}
	if term == nil {
		return nil, nil
	}

	result := p.findReferences(term)

	// sort
	sort.Slice(result, func(i, j int) bool {
		if result[i].File != result[j].File {
			return result[i].File < result[j].File
		}

		return result[i].Row < result[j].Row
	})

	return result, nil
}

func (p *Project) findReferences(term *ast.Term) []*ast.Location {
	result := make([]*ast.Location, 0)

	ruleDefinitions := p.findDefinitionInModule(term)
	if len(ruleDefinitions) == 0 {
		// Target term is defined in the rule
		rule := p.findRuleForTerm(term.Loc())
		if rule != nil {
			result = append(result, p.findReferencesInRule(term, rule)...)
		}
		return result
	}

	// list definition
	result = append(result, p.findDefinitionInModule(term)...)

	policy := p.cache.Get(term.Loc().File)
	for _, rule := range policy.Module.Rules {
		result = append(result, p.findReferencesInRule(term, rule)...)
	}
	return result
}

func (p *Project) findReferencesInRule(term *ast.Term, rule *ast.Rule) []*ast.Location {
	result := make([]*ast.Location, 0)

	if rule.Head.Key != nil {
		result = append(result, p.findReferencesInTerm(term, rule.Head.Key)...)
	}

	if rule.Head.Value != nil {
		result = append(result, p.findReferencesInTerm(term, rule.Head.Value)...)
	}

	result = append(result, p.findReferencesInTerms(term, rule.Head.Args)...)

	for _, b := range rule.Body {
		switch t := b.Terms.(type) {
		case *ast.Term:
			result = append(result, p.findReferencesInTerm(term, t)...)
		case []*ast.Term:
			result = append(result, p.findReferencesInTerms(term, t)...)
		default:
			fmt.Fprintf(os.Stderr, "type: %T", b.Terms)
		}
	}

	return result
}

func (p *Project) findReferencesInTerms(target *ast.Term, terms []*ast.Term) []*ast.Location {
	result := make([]*ast.Location, 0)
	for _, term := range terms {
		result = append(result, p.findReferencesInTerm(target, term)...)
	}
	return result
}

func (p *Project) findReferencesInTerm(target *ast.Term, term *ast.Term) []*ast.Location {
	switch v := term.Value.(type) {
	case ast.Call:
		return p.findReferencesInTerms(target, []*ast.Term(v))
	case ast.Ref:
		return p.findReferencesInTerms(target, []*ast.Term(v))
	case ast.Var:
		if target.Equal(term) {
			return []*ast.Location{term.Location}
		}
	}
	return nil
}
