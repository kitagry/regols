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
		expectResult []*ast.Location
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
			expectResult: []*ast.Location{
				{
					Row: 6,
					Col: 2,
					Offset: len("package main\n\nimport data.library\n\nviolation[msg] {\n	m"),
					Text: []byte("m"),
					File: testDataPath + "/src.rego",
				},
			},
		},
		"in file definition in args": {
			path: testDataPath + "/src.rego",
			location: &location.Location{
				Row: 9,
				Col: 2,
				Offset: len("package main\n\nimport data.library\n\nviolation[msg] {\n	m := \"hello\"\n	other_method(m)\n	library.hello(m)\n	m"),
				Text: []byte("m"),
				File: testDataPath + "/src.rego",
			},
			expectResult: []*ast.Location{
				{
					Row:    5,
					Col:    11,
					Offset: len("package main\n\nimport data.library\n\nviolation[m"),
					Text:   []byte("msg"),
					File:   testDataPath + "/src.rego",
				},
			},
		},
		"same library but other definition": {
			path: testDataPath + "/src.rego",
			location: &location.Location{
				Row: 7,
				Col: 5,
				Offset: len("package main\n\nimport data.library\n\nviolation[msg] {\n	m := \"hello\"\n	othe"),
				Text: []byte("e"),
				File: testDataPath + "/src.rego",
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					Offset: len("package main\n\no"),
					Text: []byte("other_method(msg) {\n	msg == \"hello\"\n}"),
					File: testDataPath + "/src2.rego",
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
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					Offset: len("package library\n\nh"),
					Text: []byte("hello(msg) {\n	msg == \"hello\"\n}"),
					File: testDataPath + "/lib/library.rego",
				},
			},
		},
		"no definition because itself is definition": {
			path: testDataPath + "/src.rego",
			location: &location.Location{
				Row: 6,
				Col: 2,
				Offset: len("package main\n\nimport data.library\n\nviolation[msg] {\n	m"),
				Text: []byte("m"),
				File: testDataPath + "/src.rego",
			},
			expectResult: []*ast.Location{},
			expectErr:    nil,
		},
		"with not library but has dot": {
			path: testDataPath + "/src.rego",
			location: &location.Location{
				Row: 14,
				Col: 11,
				Offset: len("package main\n\nimport data.library\n\nviolation[msg] {\n	m := \"hello\"\n	other_method(m)\n	library.hello(m)\n	msg = m\n}\n\nviolation[msg] {\n	library.containers[container]\n	container.n}"),
				Text: []byte("n"),
				File: testDataPath + "/src.rego",
			},
			expectResult: []*ast.Location{
				{
					Row: 13,
					Col: 21,
					Offset: len("package main\n\nimport data.library\n\nviolation[msg] {\n	m := \"hello\"\n	other_method(m)\n	library.hello(m)\n	msg = m\n}\n\nviolation[msg] {\n	library.containers[c"),
					Text: []byte("container"),
					File: testDataPath + "/src.rego",
				},
			},
		},
		"With two library method can jump": {
			path: testDataPath + "/src.rego",
			location: &location.Location{
				Row: 14,
				Col: 11,
				Offset: len("package main\n\nimport data.library\n\nviolation[msg] {\n	m := \"hello\"\n	other_method(m)\n	library.hello(m)\n	msg = m\n}\n\nviolation[msg] {\n	library.containers[container]\n	container.name\n	msg = \"hello\"\n}\n\nviolation[msg] {\n	library.containers[container]\n	library.h"),
				Text: []byte("h"),
				File: testDataPath + "/src.rego",
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					Offset: len("package library\n\nh"),
					Text: []byte("hello(msg) {\n	msg == \"hello\"\n}"),
					File: testDataPath + "/lib/library.rego",
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

			if diff := cmp.Diff(tt.expectResult, got); diff != "" {
				t.Errorf("LookupDefinition result diff (-expect +got):\n%s", diff)
			}
		})
	}
}
