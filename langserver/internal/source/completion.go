package source

import (
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/ast"
)

type CompletionItem struct {
	Label        string
	Kind         CompletionKind
	Detail       string
	InsertText   string
	FunctionHead *FunctionHead
}

type FunctionHead struct {
	Args  []string
	Value *string
}

type CompletionKind int

const (
	Unknown CompletionKind = iota
	VariableItem
	PackageItem
	FunctionItem
	BuiltinFunctionItem
)

func (p *Project) ListCompletionItems(location *ast.Location) ([]CompletionItem, error) {
	term, err := p.SearchTargetTerm(location)
	if err != nil {
		return nil, err
	}

	// list candidates
	list := p.listCompletionCandidates(location, term)

	// filter items
	list = filterCompletionItems(term, list)

	return list, nil
}

func (p *Project) listCompletionCandidates(location *ast.Location, target *ast.Term) []CompletionItem {
	module := p.GetModule(location.File)
	if module == nil {
		return nil
	}

	for _, r := range module.Rules {
		if in(location, r.Loc()) {
			return p.listCompletionItemsForTerms(location, target)
		}
	}
	return nil
}

func (p *Project) listCompletionItemsForTerms(location *ast.Location, target *ast.Term) []CompletionItem {
	result := make([]CompletionItem, 0)

	module := p.GetModule(location.File)
	if module == nil {
		return nil
	}

	if !p.isLibraryTerm(target) {
		for _, i := range module.Imports {
			result = append(result, CompletionItem{
				Label: importToLabel(i),
				Kind:  PackageItem,
			})
		}

		rule := p.findRuleForTerm(location)
		if rule != nil {
			list := p.listCompletionItemsInRule(location, rule)
			result = append(result, list...)
		}

		for _, r := range module.Rules {
			result = append(result, createRuleCompletionItem(r))
		}
	}

	if p.isLibraryTerm(target) {
		if _, ok := target.Value.(ast.Ref); ok {
			importRef := p.findPolicyRef(target)
			policies := p.cache.FindPolicies(importRef)
			for _, p := range policies {
				for _, r := range p.Rules {
					result = append(result, createRuleCompletionItem(r))
				}
			}
		}
	}

	result = append(result, p.listBuiltinFunction(target)...)

	return result
}

// When target is "lib." return true, else return false
func (p *Project) isLibraryTerm(target *ast.Term) bool {
	if target == nil {
		return false
	}

	_, ok := target.Value.(ast.Ref)
	return ok
}

func (p *Project) listCompletionItemsInRule(loc *ast.Location, rule *ast.Rule) []CompletionItem {
	result := make([]CompletionItem, 0)
	if !in(loc, rule.Head.Loc()) {
		if rule.Head.Key != nil {
			result = append(result, CompletionItem{
				Label: rule.Head.Key.String(),
				Kind:  VariableItem,
			})
		}

		for _, arg := range rule.Head.Args {
			result = append(result, CompletionItem{
				Label: arg.String(),
				Kind:  VariableItem,
			})
		}
	}

	for _, b := range rule.Body {
		if b.Loc().Row >= loc.Row {
			break
		}

		switch t := b.Terms.(type) {
		case *ast.Term:
			list := p.listCompletionItemsInTerm(loc, t)
			result = append(result, list...)
		case []*ast.Term:
			if ast.Equality.Ref().Equal(b.Operator()) || ast.Assign.Ref().Equal(b.Operator()) {
				list := p.listCompletionItemsInTerm(loc, t[1])
				result = append(result, list...)
			}
		}
	}

	return result
}

func (p *Project) listCompletionItemsInTerm(loc *ast.Location, term *ast.Term) []CompletionItem {
	result := make([]CompletionItem, 0)
	switch v := term.Value.(type) {
	case *ast.Array:
		for i := 0; i < v.Len(); i++ {
			result = append(result, p.listCompletionItemsInTerm(loc, v.Elem(i))...)
		}
	case ast.Ref:
		// skip library name
		// ```
		// import lib
		// lib.hoge[fuga]
		// ```
		for i := 1; i < len(v); i++ {
			result = append(result, p.listCompletionItemsInTerm(loc, v[i])...)
		}
	case ast.Var:
		result = append(result, CompletionItem{
			Label: v.String(),
			Kind:  VariableItem,
		})
	}
	return result
}

func (p *Project) listBuiltinFunction(term *ast.Term) []CompletionItem {
	if term == nil {
		return nil
	}

	result := make([]CompletionItem, 0)
	ref, ok := term.Value.(ast.Ref)
	if !ok {
		for _, b := range ast.DefaultBuiltins {
			if b.Infix != "" {
				continue
			}
			result = append(result, CompletionItem{
				Label:      b.Name,
				Kind:       BuiltinFunctionItem,
				Detail:     b.Decl.String(),
				InsertText: fmt.Sprintf("%s%s", b.Name, b.Decl.FuncArgs().String()),
			})
		}
		return result
	}

	val := ref[0]
	for _, b := range ast.DefaultBuiltins {
		if b.Infix != "" {
			continue
		}
		if strings.HasPrefix(b.Name, fmt.Sprintf("%s.", val.Value.String())) {
			name := strings.TrimLeft(b.Name, fmt.Sprintf("%s.", val.Value.String()))
			result = append(result, CompletionItem{
				Label:      name,
				Kind:       BuiltinFunctionItem,
				Detail:     b.Decl.String(),
				InsertText: fmt.Sprintf("%s%s", name, b.Decl.FuncArgs().String()),
			})
		}
	}
	return result
}

func importToLabel(imp *ast.Import) string {
	alias := imp.Alias.String()
	if alias != "" {
		return alias
	}

	m := imp.String()[strings.LastIndex(imp.String(), ".")+1:]
	return m
}

func filterCompletionItems(target *ast.Term, list []CompletionItem) []CompletionItem {
	termPrefix := getTermPrefix(target)

	result := make([]CompletionItem, 0)
	exist := make(map[CompletionItem]struct{})
	for _, item := range list {
		if strings.HasPrefix(item.Label, termPrefix) {
			if _, ok := exist[item]; !ok {
				result = append(result, item)
				exist[item] = struct{}{}
			}
		}
	}

	return result
}

func getTermPrefix(target *ast.Term) string {
	if target == nil {
		return ""
	}
	switch v := target.Value.(type) {
	case ast.Ref:
		if s, ok := v[len(v)-1].Value.(ast.String); ok {
			return string(s)
		}
		return ""
	default:
		return target.String()
	}
}

func createRuleCompletionItem(rule *ast.Rule) CompletionItem {
	head := rule.Head
	var insertText strings.Builder
	insertText.WriteString(head.Name.String())
	if len(rule.Head.Args) != 0 {
		args := make([]string, len(rule.Head.Args))
		for i, arg := range head.Args {
			args[i] = arg.String()
		}
		insertText.WriteByte('(')
		insertText.WriteString(strings.Join(args, ", "))
		insertText.WriteByte(')')
	} else if head.Key != nil {
		insertText.WriteByte('[')
		insertText.WriteString(head.Key.String())
		insertText.WriteByte(']')
	}

	return CompletionItem{
		Label:      rule.Head.Name.String(),
		Kind:       FunctionItem,
		InsertText: insertText.String(),
	}
}
