package cache

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/ast/location"
)

func (p *Project) LookupDefinition(path string, location *location.Location) ([]LookUpResult, error) {
	module := p.GetModule(path)
	if module == nil {
		return nil, fmt.Errorf("cannot find module: %s", path)
	}

	rawText, err := p.GetRawText(path)
	if err != nil {
		return nil, err
	}

	targetTerm, rule, err := p.lookupRules(location, module.Rules, rawText)
	if err != nil {
		return nil, err
	}
	if targetTerm == nil {
		return nil, nil
	}

	target := p.findInRule(targetTerm, rule)
	if target != nil {
		result := []LookUpResult{
			{
				Location: target.Loc(),
				Path:     path,
			},
		}
		return result, nil
	}
	return p.findMethod(targetTerm, path), nil
}

func (p *Project) findInRule(term *ast.Term, rule *ast.Rule) *ast.Term {
	for _, b := range rule.Body {
		switch t := b.Terms.(type) {
		case *ast.Term:
			result := p.findInTerm(term, t)
			if result != nil {
				return result
			}
		case []*ast.Term:
			result := p.findInTerms(term, t)
			if result != nil {
				return result
			}
		default:
			fmt.Fprintf(os.Stderr, "type: %T", b.Terms)
		}
	}
	return nil
}

func (p *Project) findInTerms(target *ast.Term, terms []*ast.Term) *ast.Term {
	for _, term := range terms {
		t := p.findInTerm(target, term)
		if t != nil {
			return t
		}
	}
	return nil
}

func (p *Project) findInTerm(target *ast.Term, term *ast.Term) *ast.Term {
	switch v := term.Value.(type) {
	case ast.Call:
		return p.findInTerms(target, []*ast.Term(v))
	case ast.Ref:
		return p.findInTerms(target, []*ast.Term(v))
	case *ast.Array:
		for i := 0; i < v.Len(); i++ {
			t := p.findInTerm(target, v.Elem(i))
			if t == nil {
				continue
			}
			return t
		}
		return nil
	case ast.Var:
		if target.Equal(term) && !target.Loc().Equal(term.Loc()) {
			return term
		}
		return nil
	case ast.String, ast.Boolean, ast.Number:
		return nil
	default:
		return nil
	}
}

func (p *Project) lookupRules(location *location.Location, rules []*ast.Rule, rawText string) (*ast.Term, *ast.Rule, error) {
	for _, r := range rules {
		if !in(location, r.Loc()) {
			continue
		}
		term, err := p.lookupRule(location, r, rawText)
		r := r
		return term, r, err
	}
	return nil, nil, nil
}

func (p *Project) lookupRule(location *location.Location, rule *ast.Rule, rawText string) (*ast.Term, error) {
	for _, b := range rule.Body {
		if !in(location, b.Loc()) {
			continue
		}

		switch t := b.Terms.(type) {
		case *ast.Term:
			return p.lookupTerm(location, t, rawText)
		case []*ast.Term:
			return p.lookupTerms(location, t, rawText)
		}
	}
	return nil, nil
}

func (p *Project) lookupTerms(location *location.Location, terms []*ast.Term, rawText string) (*ast.Term, error) {
	for _, t := range terms {
		if in(location, t.Loc()) {
			return p.lookupTerm(location, t, rawText)
		}
	}
	return nil, nil
}

func (p *Project) lookupTerm(loc *location.Location, term *ast.Term, rawText string) (*ast.Term, error) {
	switch v := term.Value.(type) {
	case ast.Call:
		return p.lookupTerms(loc, []*ast.Term(v), rawText)
	case ast.Ref:
		if len(v) == 1 {
			return p.lookupTerm(loc, v[0], rawText)
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
		return p.lookupTerms(loc, []*ast.Term(v), rawText)
	case *ast.Array:
		for i := 0; i < v.Len(); i++ {
			t, err := p.lookupTerm(loc, v.Elem(i), rawText)
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

func (p *Project) findMethod(term *ast.Term, path string) []LookUpResult {
	word := term.String()
	module := p.GetModule(path)
	searchModules := make(map[string]*ast.Module)

	searchModuleName := ""
	if strings.Contains(word, ".") /* imported method */ {
		moduleName := word[:strings.Index(word, ".")]
		imp := findImportModule(moduleName, module.Imports)

		word = word[strings.Index(word, ".")+1:]
		searchModuleName = imp.Path.String()
	} else {
		searchModuleName = module.Package.Path.String()
		searchModules[path] = module
	}

	searchModules = p.findModuleFiles(searchModuleName)

	if len(searchModules) == 0 {
		return nil
	}

	result := make([]LookUpResult, 0)
	for path, mod := range searchModules {
		for _, rule := range mod.Rules {
			if rule.Head.Name.String() == word {
				r := rule
				result = append(result, LookUpResult{
					Location: r.Loc(),
					Path:     path,
				})
			}
		}
	}
	return result
}

func findImportModule(moduleName string, imports []*ast.Import) *ast.Import {
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

func (p *Project) findModuleFiles(moduleName string) map[string]*ast.Module {
	modules, err := p.getModules()
	if err != nil {
		return nil
	}
	result := make(map[string]*ast.Module)
	for path, module := range modules {
		if module.Package.Path.String() == moduleName {
			result[path] = module
		}
	}
	return result
}

func in(target, src *location.Location) bool {
	return target.Offset >= src.Offset && target.Offset <= (src.Offset+len(src.Text))
}

func (p *Project) GetRawText(path string) (string, error) {
	if f, ok := p.GetFile(path); ok {
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
