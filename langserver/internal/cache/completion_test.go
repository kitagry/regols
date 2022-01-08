package cache_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/regols/langserver/internal/cache"
	"github.com/open-policy-agent/opa/ast"
)

func TestProject_ListCompletionItems(t *testing.T) {
	tests := map[string]struct {
		files       map[string]cache.File
		location    *ast.Location
		expectItems []cache.CompletionItem
	}{
		"list up in rule": {
			files: map[string]cache.File{
				"main.rego": {
					RowText: `package main

violation[msg] {
	ms := hoge(fuga)
	message := hoge(fuga)
	m
}`,
				},
			},
			location: &ast.Location{
				Row: 6,
				Col: 2,
				Offset: len("package main\n\nviolation[msg] {\n	ms := hoge(fuga)\n	message := hoge(fuga)\nm"),
				File: "main.rego",
				Text: []byte("m"),
			},
			expectItems: []cache.CompletionItem{
				{
					Label: "msg",
					Kind:  cache.VariableItem,
				},
				{
					Label: "ms",
					Kind:  cache.VariableItem,
				},
				{
					Label: "message",
					Kind:  cache.VariableItem,
				},
			},
		},
		"completion package": {
			files: map[string]cache.File{
				"main.rego": {
					RowText: `package main

import data.lib

violation[msg] {
	l
}`,
				},
			},
			location: &ast.Location{
				Row: 6,
				Col: 2,
				Offset: len("package main\n\nimport data.lib\n\nviolation[msg] {\n	l"),
				File: "main.rego",
				Text: []byte("l"),
			},
			expectItems: []cache.CompletionItem{
				{
					Label: "lib",
					Kind:  cache.PackageItem,
				},
			},
		},
		"completion ast.Ref body": {
			files: map[string]cache.File{
				"main.rego": {
					RowText: `package main

violation [msg] {
	containers[container]
	c
}`,
				},
			},
			location: &ast.Location{
				Row: 5,
				Col: 2,
				Offset: len("package main\n\nviolation [msg] {\n	containers[container]\n	c"),
				File: "main.rego",
				Text: []byte("c"),
			},
			expectItems: []cache.CompletionItem{
				{
					Label: "container",
					Kind:  cache.VariableItem,
				},
			},
		},
		"completion methods": {
			files: map[string]cache.File{
				"main.rego": {
					RowText: `package main

violation [msg] {
	i
}

is_hello(msg) {
	msg == "hello"
}`,
				},
			},
			location: &ast.Location{
				Row: 4,
				Col: 2,
				Offset: len("package main\n\nviolation [msg] {\n	i"),
				File: "main.rego",
				Text: []byte("i"),
			},
			expectItems: []cache.CompletionItem{
				{
					Label: "is_hello",
					Kind:  cache.FunctionItem,
				},
			},
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			project, err := cache.NewProjectWithFiles(tt.files)
			if err != nil {
				t.Fatal(err)
			}

			got, err := project.ListCompletionItems(tt.location)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tt.expectItems, got); diff != "" {
				t.Errorf("LookupDefinition result diff (-expect +got):\n%s", diff)
			}
		})
	}
}
