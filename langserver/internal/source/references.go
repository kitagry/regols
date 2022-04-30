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

	ruleDefinitions := p.findDefinitionOutOfRule(term)
	if len(ruleDefinitions) == 0 {
		// Target term is defined in the rule
		rule := p.findRuleForTerm(term.Loc())
		if rule != nil {
			result = append(result, p.findReferencesInRule(term, rule)...)
		}
		return result
	}

	// get definition
	result = append(result, p.findDefinitionOutOfRule(term)...)

	// list package name
	for _, pkg := range p.cache.GetPackages() {
		if ast.Var(pkg[len(pkg)-1].Value.(ast.String)).Compare(term.Value) == 0 {
			modules := p.cache.FindPolicies(pkg)
			for _, module := range modules {
				result = append(result, module.Package.Path[len(module.Package.Path)-1].Location)
			}
		}
	}

	// list references in rules
	policy := p.cache.Get(term.Loc().File)
	for _, pkg := range p.cache.GetPackages() {
		modules := p.cache.FindPolicies(pkg)
		for _, module := range modules {
			for _, rule := range module.Rules {
				t := getTermForPackage(term, policy.Module, module)
				result = append(result, p.findReferencesInRule(t, rule)...)
			}
		}
	}
	return result
}

func getTermForPackage(term *ast.Term, termModule, targetModule *ast.Module) *ast.Term {
	pkg, ok := findPackageName(term, termModule.Imports)
	if !ok {
		return term
	}

	pkgRefs, ok := pkg.Value.(ast.Ref)
	if !ok {
		return term
	}

	if !pkgRefs.Equal(targetModule.Package.Path) {
		return term
	}

	refs, ok := term.Value.(ast.Ref)
	if !ok {
		return term
	}

	refs = refs[1:].Copy()
	refs[0].Value = ast.Var(refs[0].Value.(ast.String))
	return &ast.Term{Value: refs, Location: refs[0].Location}
}

func findPackageName(term *ast.Term, imports []*ast.Import) (*ast.Term, bool) {
	termRef, ok := term.Value.(ast.Ref)
	if !ok {
		return nil, false
	}

	if len(termRef) == 0 {
		return nil, false
	}

	val, ok := termRef[0].Value.(ast.Var)
	if !ok {
		return nil, false
	}

	for _, imp := range imports {
		if imp.Alias != "" && val.Equal(imp.Alias) {
			return imp.Path, true
		}

		if imp.Alias == "" {
			ref, ok := imp.Path.Value.(ast.Ref)
			if ok && val.String() == string(ref[len(ref)-1].Value.(ast.String)) {
				return imp.Path, true
			}
		}
	}
	return nil, false
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
		targetRef, ok := target.Value.(ast.Ref)
		if !ok {
			return p.findReferencesInTerms(target, []*ast.Term(v))
		}
		if len(targetRef) <= len(v) {
			if targetRef.Compare(v[:len(targetRef)]) == 0 {
				return []*ast.Location{v[len(targetRef)-1].Location}
			}
		}
	case *ast.Array:
		result := make([]*ast.Location, 0)
		for i := 0; i < v.Len(); i++ {
			result = append(result, p.findReferencesInTerm(target, v.Elem(i))...)
		}
		return result
	case ast.Var:
		if target.Equal(term) {
			return []*ast.Location{term.Location}
		}
	}
	return nil
}
