package source

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/open-policy-agent/opa/ast"
)

func (p *Project) LookupDefinition(location *ast.Location) ([]*ast.Location, error) {
	targetTerm, err := p.SearchTargetTerm(location)
	if err != nil {
		return nil, err
	}
	if targetTerm == nil {
		return nil, nil
	}

	return p.findDefinition(targetTerm), nil
}

func (p *Project) findDefinition(term *ast.Term) []*ast.Location {
	rule := p.findRuleForTerm(term.Loc())
	if rule != nil {
		target := p.findDefinitionInRule(term, rule)
		if target != nil {
			return []*ast.Location{target.Loc()}
		}
	}
	if val, ok := term.Value.(ast.Var); ok {
		module := p.GetModule(term.Loc().File)
		for _, imp := range module.Imports {
			if imp.Alias != "" && val.Equal(imp.Alias) {
				return []*ast.Location{imp.Path.Location}
			}

			if imp.Alias == "" {
				ref, ok := imp.Path.Value.(ast.Ref)
				if ok && val.String() == string(ref[len(ref)-1].Value.(ast.String)) {
					return []*ast.Location{ref[len(ref)-1].Loc()}
				}
			}
		}
	}
	if isImportTerm(term) {
		return p.findImportDefinitions(term)
	}
	return p.findDefinitionInModule(term)
}

func (p *Project) findRuleForTerm(loc *ast.Location) *ast.Rule {
	module := p.GetModule(loc.File)
	if module == nil {
		return nil
	}

	for _, r := range module.Rules {
		if in(loc, r.Loc()) {
			return p.findElseRule(loc, r)
		}
	}
	return nil
}

func (p *Project) findElseRule(loc *ast.Location, rule *ast.Rule) *ast.Rule {
	for {
		if rule.Else == nil || !in(loc, rule.Else.Loc()) {
			return rule
		}
		rule = rule.Else
	}
}

func (p *Project) findDefinitionInRule(term *ast.Term, rule *ast.Rule) *ast.Term {
	// violation[msg]
	//           ^ this is key
	if rule.Head.Key != nil {
		result := p.findDefinitionInTerm(term, rule.Head.Key)
		if result != nil {
			return result
		}
	}

	// func() = test
	//          ^ this is value
	if rule.Head.Value != nil {
		result := p.findDefinitionInTerm(term, rule.Head.Value)
		if result != nil {
			return result
		}
	}

	// func(hello)
	//      ^ this is arg
	result := p.findDefinitionInTerms(term, rule.Head.Args)
	if result != nil {
		return result
	}

	for _, b := range rule.Body {
		switch t := b.Terms.(type) {
		case *ast.Term:
			result := p.findDefinitionInTerm(term, t)
			if result != nil {
				return result
			}
		case []*ast.Term:
			// equality -> [hoge, fuga] = split_hoge()
			// assign -> hoge := fuga()
			if ast.Equality.Ref().Equal(b.Operator()) || ast.Assign.Ref().Equal(b.Operator()) {
				result := p.findDefinitionInTerm(term, t[1])
				if result != nil {
					return result
				}
			}
		default:
			fmt.Fprintf(os.Stderr, "type: %T", b.Terms)
		}
	}
	return nil
}

func (p *Project) findDefinitionInTerms(target *ast.Term, terms []*ast.Term) *ast.Term {
	for _, term := range terms {
		t := p.findDefinitionInTerm(target, term)
		if t != nil {
			return t
		}
	}
	return nil
}

func (p *Project) findDefinitionInTerm(target *ast.Term, term *ast.Term) *ast.Term {
	switch v := term.Value.(type) {
	case ast.Call:
		return p.findDefinitionInTerms(target, []*ast.Term(v))
	case ast.Ref:
		// import data.a
		// a.b[c] -> a: ast.Var, b: ast.String, c: ast.Var
		// a.b.c  -> a: ast.Var, b: ast.String, c: ast.String
		return p.findDefinitionInTerms(target, []*ast.Term(v)[1:])
	case *ast.Array:
		for i := 0; i < v.Len(); i++ {
			t := p.findDefinitionInTerm(target, v.Elem(i))
			if t == nil {
				continue
			}
			return t
		}
		return nil
	case ast.Var:
		if target.Equal(term) && target.Loc().Offset > term.Loc().Offset {
			return term
		}
		return nil
	case ast.String, ast.Boolean, ast.Number:
		return nil
	default:
		return nil
	}
}

func (p *Project) findDefinitionInModule(term *ast.Term) []*ast.Location {
	searchPackageName := p.findPolicyRef(term)
	searchPolicies := p.cache.FindPolicies(searchPackageName)

	if len(searchPolicies) == 0 {
		return nil
	}

	word := term.String()
	if strings.Contains(word, ".") /* imported method */ {
		word = word[strings.Index(word, ".")+1:]
	}

	result := make([]*ast.Location, 0)
	for _, mod := range searchPolicies {
		for _, rule := range mod.Rules {
			if rule.Head.Name.String() == word {
				r := rule
				result = append(result, r.Loc())
			}
		}
	}
	return result
}

func (p *Project) findPolicyRef(term *ast.Term) ast.Ref {
	if term == nil {
		return nil
	}

	module := p.GetModule(term.Loc().File)
	if module == nil {
		return nil
	}

	if ref, ok := term.Value.(ast.Ref); ok && len(ref) > 1 {
		imp := findImportOutsidePolicy(ref[0].String(), module.Imports)
		if imp == nil {
			return nil
		}
		result, ok := imp.Path.Value.(ast.Ref)
		if !ok {
			return nil
		}
		return result
	}

	return module.Package.Path
}

func findImportOutsidePolicy(moduleName string, imports []*ast.Import) *ast.Import {
	for _, imp := range imports {
		alias := imp.Alias.String()
		if alias != "" && moduleName == alias {
			imp := imp
			return imp
		}
		m := imp.String()[strings.LastIndex(imp.String(), "."):]
		if strings.HasSuffix(m, moduleName) {
			imp := imp
			return imp
		}
	}
	return nil
}

func (p *Project) GetRawText(path string) (string, error) {
	if f, ok := p.GetFile(path); ok {
		return f, nil
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

// import data.xxx
//        ^ is import term
func isImportTerm(term *ast.Term) bool {
	val, ok := term.Value.(ast.Ref)
	if !ok {
		return false
	}

	if len(val) == 0 {
		return false
	}

	return ast.Var("data").Equal(val[0].Value)
}

func (p *Project) findImportDefinitions(term *ast.Term) []*ast.Location {
	val, ok := term.Value.(ast.Ref)
	if !ok {
		return nil
	}

	modules := p.cache.FindPolicies(val)
	result := make([]*ast.Location, len(modules))
	for i, m := range modules {
		result[i] = m.Package.Loc()
	}
	return result
}
