package source

import (
	"fmt"
	"os"

	"github.com/kitagry/regols/langserver/internal/cache"
	"github.com/open-policy-agent/opa/ast"
)

type Project struct {
	rootPath string
	cache    *cache.GlobalCache
}

type File struct {
	RowText string
	Version int
}

func NewProject(rootPath string) (*Project, error) {
	cache, err := cache.NewGlobalCache(rootPath)
	if err != nil {
		return nil, err
	}

	return &Project{
		rootPath: rootPath,
		cache:    cache,
	}, nil
}

func NewProjectWithFiles(files map[string]File) (*Project, error) {
	ff := make(map[string]string, len(files))
	for path, file := range files {
		ff[path] = file.RowText
	}

	cache, err := cache.NewGlobalCacheWithFiles(ff)
	if err != nil {
		return nil, err
	}

	return &Project{
		cache: cache,
	}, nil
}

func (p *Project) UpdateFile(path string, text string, version int) error {
	p.cache.Put(path, text)

	return nil
}

func (p *Project) GetErrors(path string) map[string]ast.Errors {
	errs := p.cache.GetErrors(path)
	fmt.Fprintf(os.Stderr, "GetErrors: %+v\n", errs)
	return errs
}

func (p *Project) GetFile(path string) (string, bool) {
	policy := p.cache.Get(path)
	if policy == nil {
		return "", false
	}
	return policy.RawText, true
}

func (p *Project) DeleteFile(path string) {
	p.cache.Delete(path)
}

func (p *Project) GetModule(path string) *ast.Module {
	policy := p.cache.Get(path)
	if policy == nil {
		return nil
	}
	return policy.Module
}

type LookUpResult struct {
	Location *ast.Location
}
