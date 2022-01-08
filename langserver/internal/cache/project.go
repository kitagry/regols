package cache

import (
	"fmt"
	"strings"

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
	modules, regoErrs, err := loadRegoFiles(rootPath)
	if err != nil {
		return nil, err
	}

	return &Project{
		rootPath:  rootPath,
		modules:   modules,
		openFiles: make(map[string]File),
		errs:      regoErrs,
	}, nil
}

func NewProjectWithFiles(files map[string]File) (*Project, error) {
	modules := make(map[string]*ast.Module)
	errs := make(map[string]ast.Errors)
	for path, file := range files {
		module, err := ast.ParseModule(path, file.RowText)
		if astErr, ok := err.(ast.Errors); ok {
			errs[path] = astErr
		} else if err != nil {
			return nil, err
		}
		modules[path] = module
	}

	if len(modules) == 0 {
		return nil, fmt.Errorf("NewProjectWithFiles should have least one parseable file")
	}
	return &Project{
		openFiles: files,
		modules:   modules,
		errs:      errs,
	}, nil
}

func (p *Project) getModules() (map[string]*ast.Module, error) {
	if p.modules != nil {
		return p.modules, nil
	}
	modules, regoErrs, err := loadRegoFiles(p.rootPath)
	if err != nil {
		return nil, err
	}
	if regoErrs != nil {
		p.errs = regoErrs
		return nil, fmt.Errorf("failed to load rego file: %v", regoErrs)
	}
	p.modules = modules
	return modules, nil
}

func loadRegoFiles(rootPath string) (map[string]*ast.Module, map[string]ast.Errors, error) {
	regoResult, err := loader.AllRegos([]string{rootPath})
	if errs, ok := err.(loader.Errors); ok {
		regoErrs := make(map[string]ast.Errors)
		for _, err := range errs {
			if i := strings.LastIndex(err.Error(), ":"); i != -1 {
				path := err.Error()[:i]
				regoErrs[path] = ast.Errors{
					{
						Message: err.Error(),
						Location: &ast.Location{
							File:   path,
							Row:    1,
							Col:    1,
							Offset: 0,
						},
					},
				}
			}
		}
		return make(map[string]*ast.Module), regoErrs, nil
	} else if err != nil {
		return nil, make(map[string]ast.Errors), fmt.Errorf("failed to load rego files: %w", err)
	}

	modules := regoResult.ParsedModules()
	return modules, make(map[string]ast.Errors), nil
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
	Location *ast.Location
}
