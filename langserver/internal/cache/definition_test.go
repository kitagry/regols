package cache_test

import (
	"errors"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/regols/langserver/internal/cache"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/ast/location"
)

func TestLookupDefinition(t *testing.T) {
	thisPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to find test path: %v", err)
	}
	testDataPath := thisPath + "/testdata"

	p, err := cache.NewProject(testDataPath)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	tests := map[string]struct {
		path         string
		location     *location.Location
		expectResult []cache.LookUpResult
		expectErr    error
	}{
		"in file definition": {
			path: testDataPath + "/src.rego",
			location: &location.Location{
				Row: 9,
				Col: 8,
				Offset: len("package main\n\nimport data.library\n\nviolation[msg] {\n	m := \"hello\"\n	other_method(m)\n	library.hello(m)\n	msg = m"),
				Text: []byte("m"),
				File: testDataPath + "/src.rego",
			},
			expectResult: []cache.LookUpResult{
				{
					Location: &ast.Location{
						Row: 6,
						Col: 2,
						Offset: len("package main\n\nimport data.library\n\nviolation[msg] {\n	m"),
						Text: []byte("m"),
						File: testDataPath + "/src.rego",
					},
					Path: testDataPath + "/src.rego",
				},
			},
		},
		"in library definition": {
			path: testDataPath + "/src.rego",
			location: &location.Location{
				Row: 8,
				Col: 10,
				Offset: len("package main\n\nimport data.library\n\nviolation[msg] {\n	m := \"hello\"\n	other_method(m)\n	library.h"),
				Text: []byte("h"),
				File: testDataPath + "/src.rego",
			},
			expectResult: []cache.LookUpResult{
				{
					Location: &ast.Location{
						Row:    3,
						Col:    1,
						Offset: len("package library\n\nh"),
						Text: []byte("hello(msg) {\n	msg == \"hello\"\n}"),
						File: testDataPath + "/lib/library.rego",
					},
					Path: testDataPath + "/lib/library.rego",
				},
			},
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			got, err := p.LookupDefinition(tt.path, tt.location)
			if !errors.Is(err, tt.expectErr) {
				t.Fatalf("LookupDefinition should return error expect %v, but got %v", tt.expectErr, err)
			}

			if diff := cmp.Diff(got, tt.expectResult); diff != "" {
				t.Errorf("LookupDefinition result diff:\n%s", diff)
			}
		})
	}
}
