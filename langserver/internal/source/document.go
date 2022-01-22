package source

import (
	"strings"

	"github.com/open-policy-agent/opa/ast"
)

type Document struct {
	Content string
}

func (p *Project) TermDocument(loc *ast.Location) ([]Document, error) {
	term, err := p.SearchTargetTerm(loc)
	if err != nil {
		return nil, err
	}
	if term == nil {
		return nil, nil
	}

	return p.findTermDocument(term), nil
}

func (p *Project) findTermDocument(term *ast.Term) []Document {
	rule := p.findRuleForTerm(term.Loc())
	if rule != nil {
		target := p.findDefinitionInRule(term, rule)
		if target != nil {
			return nil
		}
	}
	return p.findTermDocumentInModule(term)
}

func (p *Project) findTermDocumentInModule(term *ast.Term) []Document {
	searchPackageName := p.findPolicyRef(term)
	searchPolicies := p.cache.FindPolicies(searchPackageName)
	if len(searchPolicies) == 0 {
		return nil
	}

	word := term.String()
	if strings.Contains(word, ".") {
		word = word[strings.Index(word, ".")+1:]
	}

	result := make([]Document, 0)
	for _, mod := range searchPolicies {
		for _, rule := range mod.Rules {
			if rule.Head.Name.String() == word {
				result = append(result, Document{
					Content: createDocForRule(rule),
				})
			}
		}
	}
	return result
}

func createDocForRule(rule *ast.Rule) string {
	detail := string(rule.Loc().Text)
	if detail == "default" {
		return rule.String()
	}
	return detail
}
