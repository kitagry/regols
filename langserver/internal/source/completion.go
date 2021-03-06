package source

import (
	"fmt"
	"path"
	"strings"

	"github.com/open-policy-agent/opa/ast"
)

type CompletionItem struct {
	Label               string
	Kind                CompletionKind
	Detail              string
	TextEdit            *TextEdit
	AdditionalTextEdits []TextEdit
}

type TextEdit struct {
	Row  int
	Col  int
	Text string
}

type CompletionKind int

const (
	Unknown CompletionKind = iota
	VariableItem
	PackageItem
	FunctionItem
	BuiltinFunctionItem
	ImportItem
)

func (p *Project) ListCompletionItems(location *ast.Location) ([]CompletionItem, error) {
	term, err := p.SearchTargetTerm(location)
	if err != nil {
		return nil, err
	}

	// update location to use target term location.
	if term != nil && term.Loc() != nil {
		switch t := term.Value.(type) {
		case ast.Ref:
			location = t[len(t)-1].Loc()
		default:
			location = term.Loc()
		}
	}

	// list candidates
	list := p.listCompletionCandidates(location, term)

	// filter items
	list = filterCompletionItems(term, list)

	return list, nil
}

func (p *Project) listCompletionCandidates(location *ast.Location, target *ast.Term) []CompletionItem {
	policy := p.cache.Get(location.File)
	if policy == nil {
		return nil
	}

	if len(policy.Errs) > 0 {
		if policy.Errs[0].Code == ast.ParseErr && (policy.Errs[0].Message == "empty module" || policy.Errs[0].Message == "package expected") {
			// Client returns location which Col is something wrong, because the file cannot be parsed.
			// So, I decide col should be `1`.
			location.Col = 1
			return p.listPackageCompletionItems(location)
		}
	}

	if policy.Module == nil {
		return nil
	}

	for _, r := range policy.Module.Rules {
		if in(location, r.Loc()) {
			return p.listCompletionItemsForTerms(location, target)
		}
	}

	return p.listImportCompletionItems(location)
}

func (p *Project) listPackageCompletionItems(location *ast.Location) []CompletionItem {
	fileNames := make([]string, 0)
	file := path.Base(location.File)
	if ind := strings.LastIndex(file, ".rego"); ind > 0 {
		fileName := file[:ind]

		if strings.HasSuffix(fileName, "_test") {
			packageName := fileName[:len(fileName)-len("_test")]
			fileNames = append(fileNames, packageName)
		} else {
			fileNames = append(fileNames, fileName)
		}
	}

	dirNames := make([]string, 0)
	dir := path.Dir(location.File)
	if dir != "." {
		ind := strings.LastIndex(dir, "/")
		dirNames = append(dirNames, dir[ind+1:])
	}

	for i, fileName := range fileNames {
		fileNames[i] = strings.ReplaceAll(fileName, "-", "_")
	}
	for i, dirName := range dirNames {
		dirNames[i] = strings.ReplaceAll(dirName, "-", "_")
	}

	result := make([]CompletionItem, 0)
	for _, d := range dirNames {
		result = append(result, CompletionItem{
			Label:    fmt.Sprintf("package %s", d),
			Kind:     PackageItem,
			TextEdit: createTextEdit(location, fmt.Sprintf("package %s", d)),
		})
		for _, f := range fileNames {
			result = append(result, CompletionItem{
				Label:    fmt.Sprintf("package %s", f),
				Kind:     PackageItem,
				TextEdit: createTextEdit(location, fmt.Sprintf("package %s", f)),
			}, CompletionItem{
				Label:    fmt.Sprintf("package %s.%s", d, f),
				Kind:     PackageItem,
				TextEdit: createTextEdit(location, fmt.Sprintf("package %s.%s", d, f)),
			})
		}
	}

	return result
}

func (p *Project) listCompletionItemsForTerms(location *ast.Location, target *ast.Term) []CompletionItem {
	result := make([]CompletionItem, 0)

	module := p.GetModule(location.File)
	if module == nil {
		return nil
	}

	if !isLibraryTerm(target) {
		result = append(result, p.listLibraryVariables(location, module)...)

		rule := p.findRuleForTerm(location)
		if rule != nil {
			list := p.listCompletionItemsInRule(location, rule)
			result = append(result, list...)
		}
	}

	result = append(result, p.listRules(location, target)...)
	result = append(result, p.listBuiltinFunctions(location, target)...)

	return result
}

func (p *Project) listLibraryVariables(loc *ast.Location, module *ast.Module) []CompletionItem {
	result := make([]CompletionItem, 0)
	// Already imported libraries
	for _, i := range module.Imports {
		result = append(result, CompletionItem{
			Label: importToLabel(i),
			Kind:  PackageItem,
		})
	}

	// Unimported libraries
	pkgs := p.cache.GetPackages()
	for _, p := range pkgs {
		if !isImported(p, module.Imports) && !p.Equal(module.Package.Path) {
			label := string(p[len(p)-1].Value.(ast.String))

			var importTextEdit TextEdit
			if len(module.Imports) != 0 {
				lastImportedRow := 0
				for _, imp := range module.Imports {
					if lastImportedRow < imp.Location.Row {
						lastImportedRow = imp.Location.Row
					}
					importTextEdit = TextEdit{
						Row:  lastImportedRow + 1,
						Col:  1,
						Text: fmt.Sprintf("import %s\n", p.String()),
					}
				}
			} else {
				importTextEdit = TextEdit{
					Row:  module.Package.Location.Row + 1,
					Col:  1,
					Text: fmt.Sprintf("\nimport %s\n", p.String()),
				}
			}
			result = append(result, CompletionItem{
				Label: label,
				Kind:  PackageItem,
				TextEdit: &TextEdit{
					Row:  loc.Row,
					Col:  loc.Col,
					Text: label,
				},
				AdditionalTextEdits: []TextEdit{
					importTextEdit,
				},
			})
		}
	}
	return result
}

func isImported(p ast.Ref, imports []*ast.Import) bool {
	for _, imp := range imports {
		if p.Equal(imp.Path.Value) {
			return true
		}
	}
	return false
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

func (p *Project) listRules(location *ast.Location, term *ast.Term) []CompletionItem {
	searchPackageName := p.findPolicyRef(term)
	if searchPackageName == nil {
		module := p.GetModule(location.File)
		if module == nil {
			return nil
		}
		searchPackageName = module.Package.Path
	}
	searchModules := p.cache.FindPolicies(searchPackageName)
	if len(searchModules) == 0 {
		return nil
	}

	return p.listRulesFromModules(location, searchModules)
}

func (p *Project) listRulesFromModules(location *ast.Location, modules []*ast.Module) []CompletionItem {
	exists := make(map[string]CompletionItem)
	for _, m := range modules {
		for _, r := range m.Rules {
			item := createRuleCompletionItem(location, r)
			alreadyItem, ok := exists[item.Label]
			if !ok {
				exists[item.Label] = item
				continue
			}
			alreadyItem.Detail += "\n\n" + item.Detail
			exists[alreadyItem.Label] = alreadyItem
		}
	}

	result := make([]CompletionItem, 0)
	for _, item := range exists {
		result = append(result, item)
	}
	return result
}

func (p *Project) listBuiltinFunctions(location *ast.Location, term *ast.Term) []CompletionItem {
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
				Label:    b.Name,
				Kind:     BuiltinFunctionItem,
				Detail:   createDocForBuiltinFunction(b),
				TextEdit: createTextEdit(location, fmt.Sprintf("%s%s", b.Name, b.Decl.FuncArgs().String())),
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
				Label:    name,
				Kind:     BuiltinFunctionItem,
				Detail:   createDocForBuiltinFunction(b),
				TextEdit: createTextEdit(location, fmt.Sprintf("%s%s", name, b.Decl.FuncArgs().String())),
			})
		}
	}
	return result
}

func (p *Project) listImportCompletionItems(location *ast.Location) []CompletionItem {
	refs := p.cache.GetPackages()

	alreadyExistPackages := make([]ast.Ref, 0)
	policy := p.cache.Get(location.File)
	if policy.Module != nil {
		alreadyExistPackages = append(alreadyExistPackages, policy.Module.Package.Path)

		for _, imp := range policy.Module.Imports {
			if ref, ok := imp.Path.Value.(ast.Ref); ok {
				alreadyExistPackages = append(alreadyExistPackages, ref)
			}
		}
	}

	locationForImport := &ast.Location{
		Text:   location.Text,
		File:   location.File,
		Row:    location.Row,
		Col:    1,
		Offset: location.Offset,
	}

	result := make([]CompletionItem, 0, len(refs))
	for _, r := range refs {
		if !inRef(r, alreadyExistPackages) {
			result = append(result, CompletionItem{
				Label:    fmt.Sprintf("import %s", r.String()),
				Kind:     ImportItem,
				TextEdit: createTextEdit(locationForImport, fmt.Sprintf("import %s", r.String())),
			})
		}
	}

	return result
}

// When target is "lib." return true, else return false
func isLibraryTerm(target *ast.Term) bool {
	if target == nil {
		return false
	}

	_, ok := target.Value.(ast.Ref)
	return ok
}

func createRuleCompletionItem(location *ast.Location, rule *ast.Rule) CompletionItem {
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

	var itemKind CompletionKind
	if len(rule.Head.Args) != 0 || head.Key != nil {
		itemKind = FunctionItem
	} else {
		itemKind = VariableItem
	}

	return CompletionItem{
		Label:    rule.Head.Name.String(),
		Kind:     itemKind,
		TextEdit: createTextEdit(location, insertText.String()),
		Detail:   createDocForRule(rule),
	}
}

func inRef(target ast.Ref, list []ast.Ref) bool {
	for _, l := range list {
		if l.Equal(target) {
			return true
		}
	}
	return false
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
	exist := make(map[string]struct{})
	for _, item := range list {
		if strings.HasPrefix(item.Label, termPrefix) {
			if _, ok := exist[item.Label]; !ok {
				result = append(result, item)
				exist[item.Label] = struct{}{}
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

func createDocForBuiltinFunction(builtin *ast.Builtin) string {
	return fmt.Sprintf(`%s%s

%s`, builtin.Name, builtin.Decl.FuncArgs().String(), BuiltinDetail)
}

func createTextEdit(location *ast.Location, text string) *TextEdit {
	return &TextEdit{
		Row:  location.Row,
		Col:  location.Col,
		Text: text,
	}
}
