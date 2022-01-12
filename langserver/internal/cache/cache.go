package cache

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/open-policy-agent/opa/ast"
)

type Policy struct {
	RawText string
	Errs    ast.Errors
	Module  *ast.Module
}

type GlobalCache struct {
	mu            sync.RWMutex
	rootPath      string
	pathToPlicies map[string]*Policy
	openFiles     map[string]string
}

func NewGlobalCache(rootPath string) (*GlobalCache, error) {
	g := &GlobalCache{pathToPlicies: make(map[string]*Policy)}

	regoFilePaths, err := loadRegoFiles(rootPath)
	if err != nil {
		return nil, err
	}

	for _, path := range regoFilePaths {
		err = g.putWithPath(path)
		if err != nil {
			return nil, err
		}
	}
	return g, nil
}

func NewGlobalCacheWithFiles(pathToText map[string]string) (*GlobalCache, error) {
	g := &GlobalCache{pathToPlicies: make(map[string]*Policy, len(pathToText))}

	for path, text := range pathToText {
		err := g.Put(path, text)
		if err != nil {
			return nil, err
		}
	}
	return g, nil
}

func loadRegoFiles(rootPath string) ([]string, error) {
	result := make([]string, 0)
	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(d.Name(), ".rego") {
			result = append(result, path)
		}
		return nil
	})
	return result, err
}

func (g *GlobalCache) Get(path string) *Policy {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.pathToPlicies[path]
}

func (g *GlobalCache) putWithPath(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(f)
	if err != nil {
		return err
	}

	return g.Put(path, buf.String())
}

func (g *GlobalCache) Put(path string, rawText string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	policy, ok := g.pathToPlicies[path]
	if !ok {
		policy = &Policy{}
	}
	policy.RawText = rawText
	module, err := ast.ParseModule(path, rawText)
	if errs, ok := err.(ast.Errors); ok {
		policy.Errs = errs
		g.pathToPlicies[path] = policy
		return nil
	} else if errs, ok := err.(*ast.Error); ok {
		policy.Errs = ast.Errors{errs}
		g.pathToPlicies[path] = policy
		return nil
	} else if err != nil {
		return err
	}
	policy.Module = module
	policy.Errs = nil
	g.pathToPlicies[path] = policy
	return nil
}

func (g *GlobalCache) Delete(path string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.pathToPlicies, path)
}

func (g *GlobalCache) FindPolicies(packageName ast.Ref) []*ast.Module {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make([]*ast.Module, 0)
	for _, p := range g.pathToPlicies {
		if p.Module != nil && p.Module.Package.Path.Equal(packageName) {
			result = append(result, p.Module)
		}
	}
	return result
}

func (g *GlobalCache) GetErrors(path string) map[string]ast.Errors {
	// parse error
	if p := g.Get(path); p != nil && len(p.Errs) != 0 {
		return map[string]ast.Errors{path: p.Errs}
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	// compile error
	modules := make(map[string]*ast.Module, len(g.pathToPlicies))
	errs := make(map[string]ast.Errors, len(g.pathToPlicies))
	for path, p := range g.pathToPlicies {
		if p.Module != nil {
			modules[path] = p.Module
		}
		errs[path] = make(ast.Errors, 0)
	}

	compiler := ast.NewCompiler()
	compiler.Compile(modules)
	if !compiler.Failed() {
		return errs
	}

	for _, e := range compiler.Errors {
		errs[e.Location.File] = append(errs[e.Location.File], e)
	}
	return errs
}
