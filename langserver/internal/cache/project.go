package cache

import (
	"fmt"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/loader"
)

type Project struct {
	rootPath string
	files    map[string]File
	modules  map[string]*ast.Module
	errs     map[string]ast.Errors
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
		rootPath: rootPath,
		modules:  modules,
		files:    make(map[string]File),
		errs:     make(map[string]ast.Errors),
	}, nil
}

func (p *Project) UpdateFile(path string, text string, version int) error {
	p.files[path] = File{
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

func (p *Project) GetFiles() map[string]File {
	return p.files
}

func (p *Project) DeleteFile(path string) {
	delete(p.files, path)
	delete(p.errs, path)
}
