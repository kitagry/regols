package cache

import (
	"fmt"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/loader"
)

type Project struct {
	rootPath  string
	openFiles map[string]File
	modules   map[string]*ast.Module
	errs      map[string]ast.Errors
}

type File struct {
	RowText string
	Version int
}

func NewProject(rootPath string) (*Project, error) {
	regoResult, err := loader.AllRegos([]string{rootPath})
	if err != nil {
		return nil, fmt.Errorf("failed to load rego files: %w", err)
	}

	modules := regoResult.ParsedModules()

	return &Project{
		rootPath:  rootPath,
		modules:   modules,
		openFiles: make(map[string]File),
		errs:      make(map[string]ast.Errors),
	}, nil
}

func (p *Project) UpdateFile(path string, text string, version int) error {
	p.openFiles[path] = File{
		RowText: text,
		Version: version,
	}
	module, err := ast.ParseModule(path, text)
	if errs, ok := err.(ast.Errors); ok {
		p.errs[path] = errs
		return nil
	} else if err != nil {
		return err
	}
	p.modules[path] = module
	delete(p.errs, path)
	return nil
}

func (p *Project) GetErrors(path string) ast.Errors {
	if errs, ok := p.errs[path]; ok {
		return errs
	}

	compiler := ast.NewCompiler()
	compiler.Compile(p.modules)
	if !compiler.Failed() {
		return nil
	}

	return compiler.Errors
}

func (p *Project) GetFile(path string) (File, bool) {
	f, ok := p.openFiles[path]
	return f, ok
}

func (p *Project) GetOpenFiles() map[string]File {
	return p.openFiles
}

func (p *Project) DeleteFile(path string) {
	delete(p.openFiles, path)
	delete(p.errs, path)
}

func (p *Project) GetModule(path string) *ast.Module {
	return p.modules[path]
}

type LookUpResult struct {
	Rule *ast.Rule
	Path string
}
