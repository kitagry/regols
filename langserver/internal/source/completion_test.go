package source_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/regols/langserver/internal/source"
	"github.com/open-policy-agent/opa/ast"
)

func TestProject_ListCompletionItems(t *testing.T) {
	tests := map[string]struct {
		files       map[string]source.File
		location    *ast.Location
		expectItems []source.CompletionItem
	}{
		"import completion": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package src

`,
				},
				"lib.rego": {
					RowText: `package lib`,
				},
			},
			location: &ast.Location{
				Row:    3,
				Col:    1,
				Offset: len("pacakge src\n\n"),
				Text:   []byte(""),
				File:   "src.rego",
			},
			expectItems: []source.CompletionItem{
				{
					Label:      "import data.lib",
					Kind:       source.ImportItem,
					InsertText: "import data.lib",
				},
			},
		},
		"ignore already imported": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package src

import data.lib
`,
				},
				"lib.rego": {
					RowText: `package lib`,
				},
			},
			location: &ast.Location{
				Row:    4,
				Col:    1,
				Offset: len("pacakge src\n\nimport data.lib"),
				Text:   []byte(""),
				File:   "src.rego",
			},
			expectItems: []source.CompletionItem{},
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			project, err := source.NewProjectWithFiles(tt.files)
			if err != nil {
				t.Fatal(err)
			}

			got, err := project.ListCompletionItems(tt.location)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tt.expectItems, got); diff != "" {
				t.Errorf("ListCompletionItems result diff (-expect, +got)\n%s", diff)
			}
			for _, e := range tt.expectItems {
				if !in(e, got) {
					t.Errorf("ListCompletionItems should return item %v, got %v", e, got)
				}
			}
		})
	}
}

func TestProject_ListCompletionItemsExist(t *testing.T) {
	tests := map[string]struct {
		files       map[string]source.File
		location    *ast.Location
		expectItems []source.CompletionItem
	}{
		"list up in rule": {
			files: map[string]source.File{
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
			expectItems: []source.CompletionItem{
				{
					Label: "msg",
					Kind:  source.VariableItem,
				},
				{
					Label: "ms",
					Kind:  source.VariableItem,
				},
				{
					Label: "message",
					Kind:  source.VariableItem,
				},
			},
		},
		"completion package": {
			files: map[string]source.File{
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
			expectItems: []source.CompletionItem{
				{
					Label: "lib",
					Kind:  source.PackageItem,
				},
			},
		},
		"completion ast.Ref body": {
			files: map[string]source.File{
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
			expectItems: []source.CompletionItem{
				{
					Label: "container",
					Kind:  source.VariableItem,
				},
			},
		},
		"completion methods": {
			files: map[string]source.File{
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
			expectItems: []source.CompletionItem{
				{
					Label:      "is_hello",
					Kind:       source.FunctionItem,
					InsertText: "is_hello(msg)",
				},
			},
		},
		"completion library methods": {
			files: map[string]source.File{
				"main.rego": {
					RowText: `package main

import data.lib

violation [msg] {
	lib.i
}`,
				},
				"lib.rego": {
					RowText: `package lib

is_hello(msg) {
	msg == "hello"
}`,
				},
			},
			location: &ast.Location{
				Row: 6,
				Col: 6,
				Offset: len("package main\n\nimport data.lib\n\nviolation [msg] {\n	lib.i"),
				File: "main.rego",
				Text: []byte("i"),
			},
			expectItems: []source.CompletionItem{
				{
					Label:      "is_hello",
					Kind:       source.FunctionItem,
					InsertText: "is_hello(msg)",
				},
			},
		},
		"delete duplicate": {
			files: map[string]source.File{
				"main.rego": {
					RowText: `package main

violation[msg] {
	msg = "hello"
	m
}`,
				},
			},
			location: &ast.Location{
				Row: 5,
				Col: 2,
				Offset: len("package main\n\nviolation[msg] {\n	msg = \"hello\"\n	m"),
				Text: []byte("m"),
				File: "main.rego",
			},
			expectItems: []source.CompletionItem{
				{
					Label: "msg",
					Kind:  source.VariableItem,
				},
			},
		},
		"not prefix term": {
			files: map[string]source.File{
				"main.rego": {
					RowText: `package main

violation[msg] {
	msg = "hello"

}`,
				},
			},
			location: &ast.Location{
				Row: 5,
				Col: 1,
				Offset: len("package main\n\nviolation[msg] {\n	msg = \"hello\"\n	"),
				Text: []byte("	"),
				File: "main.rego",
			},
			expectItems: []source.CompletionItem{
				{
					Label: "msg",
					Kind:  source.VariableItem,
				},
				{
					Label:      "violation",
					Kind:       source.FunctionItem,
					InsertText: "violation[msg]",
				},
			},
		},
		"built-in completion": {
			files: map[string]source.File{
				"main.rego": {
					RowText: `package main

violation[msg] {
	j
}`,
				},
			},
			location: &ast.Location{
				Row: 4,
				Col: 2,
				Offset: len("package main\n\nviolation[msg] {\n	j"),
				Text: []byte("j"),
				File: "main.rego",
			},
			expectItems: []source.CompletionItem{
				{
					Label:      "json.patch",
					Kind:       source.BuiltinFunctionItem,
					Detail:     "json.patch(any, array[object<op: string, path: any>[any: any]])\n\n" + source.BuiltinDetail,
					InsertText: "json.patch(any, array[object<op: string, path: any>[any: any]])",
				},
			},
		},
		"built-in completion with prefix": {
			files: map[string]source.File{
				"main.rego": {
					RowText: `package main

violation[msg] {
	json.p
}`,
				},
			},
			location: &ast.Location{
				Row: 4,
				Col: 7,
				Offset: len("package main\n\nviolation[msg] {\n	json.p"),
				Text: []byte("p"),
				File: "main.rego",
			},
			expectItems: []source.CompletionItem{
				{
					Label:      "patch",
					Kind:       source.BuiltinFunctionItem,
					Detail:     "json.patch(any, array[object<op: string, path: any>[any: any]])\n\n" + source.BuiltinDetail,
					InsertText: "patch(any, array[object<op: string, path: any>[any: any]])",
				},
			},
		},
		"completion empty package": {
			files: map[string]source.File{
				"test/core.rego": {
					RowText: ``,
				},
			},
			location: &ast.Location{
				Row:    1,
				Col:    1,
				Offset: 0,
				Text:   nil,
				File:   "test/core.rego",
			},
			expectItems: []source.CompletionItem{
				{
					Label:      "package core",
					Kind:       source.PackageItem,
					InsertText: "package core",
				},
				{
					Label:      "package test",
					Kind:       source.PackageItem,
					InsertText: "package test",
				},
				{
					Label:      "package test.core",
					Kind:       source.PackageItem,
					InsertText: "package test.core",
				},
			},
		},
		"completion no package": {
			files: map[string]source.File{
				"test/core.rego": {
					RowText: `p`,
				},
			},
			location: &ast.Location{
				Row:    1,
				Col:    1,
				Offset: 1,
				Text:   []byte("p"),
				File:   "test/core.rego",
			},
			expectItems: []source.CompletionItem{
				{
					Label:      "package core",
					Kind:       source.PackageItem,
					InsertText: "package core",
				},
				{
					Label:      "package test",
					Kind:       source.PackageItem,
					InsertText: "package test",
				},
			},
		},
		"completion test package": {
			files: map[string]source.File{
				"aaa/bbb_test.rego": {
					RowText: `p`,
				},
			},
			location: &ast.Location{
				Row:    1,
				Col:    1,
				Offset: 1,
				Text:   []byte("p"),
				File:   "aaa/bbb_test.rego",
			},
			expectItems: []source.CompletionItem{
				{
					Label:      "package aaa",
					Kind:       source.PackageItem,
					InsertText: "package aaa",
				},
				{
					Label:      "package bbb",
					Kind:       source.PackageItem,
					InsertText: "package bbb",
				},
				{
					Label:      "package aaa.bbb",
					Kind:       source.PackageItem,
					InsertText: "package aaa.bbb",
				},
			},
		},
		"some rule is function and others is variable": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package src

violation[msg] {
	is
}

is_hello(msg) {
	msg == "hello"
}

default is_test = true`,
				},
			},
			location: &ast.Location{
				Row: 4,
				Col: 3,
				Offset: len("pacakge src\n\nviolation[msg] {	is"),
				Text: []byte("s"),
				File: "src.rego",
			},
			expectItems: []source.CompletionItem{
				{
					Label:      "is_hello",
					Kind:       source.FunctionItem,
					InsertText: "is_hello(msg)",
				},
				{
					Label:      "is_test",
					Kind:       source.VariableItem,
					InsertText: "is_test",
				},
			},
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			project, err := source.NewProjectWithFiles(tt.files)
			if err != nil {
				t.Fatal(err)
			}

			got, err := project.ListCompletionItems(tt.location)
			if err != nil {
				t.Fatal(err)
			}

			for _, e := range tt.expectItems {
				if !in(e, got) {
					t.Errorf("ListCompletionItems should return item %v, got %v", e, got)
				}
			}
		})
	}
}

func in(item source.CompletionItem, list []source.CompletionItem) bool {
	for _, l := range list {
		if item == l {
			return true
		}
	}
	return false
}
