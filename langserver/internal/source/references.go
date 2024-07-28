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

	// drop duplicates
	exists := make(map[string]*ast.Location)
	for _, r := range result {
		key := fmt.Sprintf("%s-%d-%d", r.File, r.Row, r.Col)
		exists[key] = r
	}

	result = make([]*ast.Location, 0, len(exists))
	for _, l := range exists {
		result = append(result, l)
	}

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
	// Target term is defined in the definedRule
	definedRule := p.findRuleForTerm(term.Loc())
	isDefinedInRule := p.findDefinitionInRule(term, definedRule) != nil
	if isDefinedInRule {
		return p.findReferencesInRule(term, definedRule, isDefinedInRule)
	}

	result := make([]*ast.Location, 0)

	// get definition
	definitions := p.findDefinitionOutOfRule(term)
	result = append(result, definitions...)

	var definedPackage *ast.Package
	if len(definitions) > 0 {
		definition := definitions[0]
		policy := p.cache.Get(definition.File)
		definedPackage = policy.Module.Package
	}

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
			isDefinedPackage := definedPackage.Equal(module.Package)
			isSameWithCalledPackage := policy.Module.Package.Equal(module.Package)
			_, ok := findImportedPkg(definedPackage, module)
			if !ok && !isSameWithCalledPackage && !isDefinedPackage {
				continue
			}

			result = append(result, p.findImportsReferences(term, module.Imports)...)
			for _, rule := range module.Rules {
				t := getTermForPackage(term, policy.Module, module)
				// if definedRule = rule,
				isDefinedInRule := definedRule.Equal(rule)
				result = append(result, p.findReferencesInRule(t, rule, isDefinedInRule)...)
			}
		}
	}
	return result
}

func findImportedPkg(pkg *ast.Package, module *ast.Module) (*ast.Import, bool) {
	for _, imp := range module.Imports {
		path, ok := imp.Path.Value.(ast.Ref)
		if !ok {
			continue
		}
		if pkg.Path.Equal(path) {
			return imp, true
		}
	}
	return nil, false
}

func (p *Project) findImportsReferences(term *ast.Term, imports []*ast.Import) []*ast.Location {
	result := make([]*ast.Location, 0)
	for _, imp := range imports {
		path, ok := imp.Path.Value.(ast.Ref)
		if !ok {
			continue
		}
		last, ok := path[len(path)-1].Value.(ast.String)
		if !ok {
			continue
		}
		val := &ast.Term{
			Value:    ast.Var(last),
			Location: path[len(path)-1].Location,
		}
		result = append(result, p.findReferencesInTerm(term, val)...)
	}
	return result
}

func getTermForPackage(term *ast.Term, termModule, targetModule *ast.Module) *ast.Term {
	// Find defined package name.
	pkg, ok := findPackageName(term, termModule)
	if !ok {
		return term
	}

	pkgRefs, ok := pkg.Value.(ast.Ref)
	if !ok {
		return term
	}

	if !pkgRefs.Equal(targetModule.Package.Path) {
		for _, imp := range targetModule.Imports {
			if !imp.Path.Equal(pkg) {
				continue
			}

			impPath, ok := imp.Path.Value.(ast.Ref)
			if !ok {
				fmt.Fprintln(os.Stderr, "imp.Path is something wrong.")
				continue
			}
			str, ok := impPath[len(impPath)-1].Value.(ast.String)
			if !ok {
				fmt.Fprintln(os.Stderr, "imp.Path is something wrong.")
				continue
			}
			var prefix ast.Var = ast.Var(str)
			if imp.Alias != ast.Var("") {
				prefix = imp.Alias
			}

			switch v := term.Value.(type) {
			case ast.Ref:
				v[0] = &ast.Term{
					Value:    prefix,
					Location: v[0].Location,
				}
				return &ast.Term{Value: v, Location: term.Location}
			case ast.Var:
				refs := ast.Ref{
					&ast.Term{Value: prefix, Location: term.Location},
					&ast.Term{Value: ast.String(v), Location: term.Location},
				}
				return &ast.Term{Value: refs, Location: term.Location}
			}
			if !ok {
				return term
			}
		}
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

func findPackageName(term *ast.Term, termModule *ast.Module) (*ast.Term, bool) {
	termRef, ok := term.Value.(ast.Ref)
	if !ok {
		return &ast.Term{Value: termModule.Package.Path, Location: termModule.Package.Location}, true
	}

	if len(termRef) == 0 {
		return nil, false
	}

	val, ok := termRef[0].Value.(ast.Var)
	if !ok {
		return nil, false
	}

	for _, imp := range termModule.Imports {
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

func (p *Project) findReferencesInRule(term *ast.Term, rule *ast.Rule, isDefinedInRule bool) []*ast.Location {
	result := make([]*ast.Location, 0)

	if rule.Head.Name.Equal(term.Value) {
		if !isDefinedInRule {
			return result
		}
		loc := &ast.Location{
			Row:    rule.Head.Location.Row,
			Col:    rule.Head.Location.Col,
			Text:   []byte(rule.Head.Name),
			File:   rule.Head.Location.File,
			Offset: rule.Head.Location.Offset,
		}
		result = append(result, loc)
	}

	if rule.Head.Key != nil {
		keys := p.findReferencesInTerm(term, rule.Head.Key)
		if len(keys) > 0 && !isDefinedInRule {
			return result
		}
		result = append(result, keys...)
	}

	if rule.Head.Value != nil {
		values := p.findReferencesInTerm(term, rule.Head.Value)
		if len(values) > 0 && !isDefinedInRule {
			return result
		}
		result = append(result, values...)
	}

	args := p.findReferencesInTerms(term, rule.Head.Args)
	if len(args) > 0 && !isDefinedInRule {
		return result
	}
	result = append(result, args...)

	for _, b := range rule.Body {
		if isAssignExpr(b) && !isDefinedInRule {
			switch t := b.Terms.(type) {
			case []*ast.Term:
				terms := p.findReferencesInTerm(term, t[1])
				if len(terms) > 0 {
					return result
				}
			}
		}

		terms := make([]*ast.Location, 0)
		switch t := b.Terms.(type) {
		case *ast.Term:
			terms = p.findReferencesInTerm(term, t)
		case []*ast.Term:
			terms = p.findReferencesInTerms(term, t)
		default:
			fmt.Fprintf(os.Stderr, "type: %T", b.Terms)
		}
		result = append(result, terms...)
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
