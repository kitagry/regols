package source_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/regols/langserver/internal/source"
)

func TestProject_ListCompletionItems(t *testing.T) {
	tests := map[string]struct {
		files          map[string]source.File
		createLocation createLocationFunc
		expectItems    []source.CompletionItem
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
			createLocation: createLocation(3, 1, "src.rego"),
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
			createLocation: createLocation(4, 1, "src.rego"),
			expectItems:    []source.CompletionItem{},
		},
		"completion for else": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package src

authorize = "allow" {
	msg := "allow"
	trace(msg)
} else = "deny" {
	ms := "deny"
	ms
}`,
				},
			},
			createLocation: createLocation(8, 3, "src.rego"),
			expectItems: []source.CompletionItem{
				{
					Label: "ms",
					Kind:  source.VariableItem,
				},
			},
		},
		"completion other args": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package src

func() {
	me
}

mem_multiple("E") = 1000000000000000000000

mem_multiple("P") = 1000000000000000000`,
				},
			},
			createLocation: createLocation(4, 3, "src.rego"),
			expectItems: []source.CompletionItem{
				{
					Label:      "mem_multiple",
					Kind:       source.FunctionItem,
					InsertText: `mem_multiple("E")`,
					Detail: `mem_multiple("E") = 1000000000000000000000

mem_multiple("P") = 1000000000000000000`,
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

			location := tt.createLocation(tt.files)
			got, err := project.ListCompletionItems(location)
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
		files          map[string]source.File
		createLocation createLocationFunc
		expectItems    []source.CompletionItem
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
			createLocation: createLocation(6, 2, "main.rego"),
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
			createLocation: createLocation(6, 2, "main.rego"),
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
			createLocation: createLocation(5, 2, "main.rego"),
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
			createLocation: createLocation(4, 2, "main.rego"),
			expectItems: []source.CompletionItem{
				{
					Label:      "is_hello",
					Kind:       source.FunctionItem,
					InsertText: "is_hello(msg)",
					Detail: `is_hello(msg) {
	msg == "hello"
}`,
				},
			},
		},
		"completion same package but other file": {
			files: map[string]source.File{
				"main.rego": {
					RowText: `package main

violation [msg] {
	he
}`,
				},
				"other.rego": {
					RowText: `package main

hello(msg) {
	msg == "hello"
}`,
				},
			},
			createLocation: createLocation(4, 3, "main.rego"),
			expectItems: []source.CompletionItem{
				{
					Label:      "hello",
					Kind:       source.FunctionItem,
					InsertText: "hello(msg)",
					Detail: `hello(msg) {
	msg == "hello"
}`,
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
			createLocation: createLocation(6, 6, "main.rego"),
			expectItems: []source.CompletionItem{
				{
					Label:      "is_hello",
					Kind:       source.FunctionItem,
					InsertText: "is_hello(msg)",
					Detail: `is_hello(msg) {
	msg == "hello"
}`,
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
			createLocation: createLocation(5, 2, "main.rego"),
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
			createLocation: createLocation(5, 1, "main.rego"),
			expectItems: []source.CompletionItem{
				{
					Label: "msg",
					Kind:  source.VariableItem,
				},
				{
					Label:      "violation",
					Kind:       source.FunctionItem,
					InsertText: "violation[msg]",
					Detail: `violation[msg] {
	msg = "hello"

}`,
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
			createLocation: createLocation(4, 2, "main.rego"),
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
			createLocation: createLocation(4, 7, "main.rego"),
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
			createLocation: createLocation(1, 1, "test/core.rego"),
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
			createLocation: createLocation(1, 1, "test/core.rego"),
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
			createLocation: createLocation(1, 1, "aaa/bbb_test.rego"),
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
			createLocation: createLocation(4, 3, "src.rego"),
			expectItems: []source.CompletionItem{
				{
					Label:      "is_hello",
					Kind:       source.FunctionItem,
					InsertText: "is_hello(msg)",
					Detail: `is_hello(msg) {
	msg == "hello"
}`,
				},
				{
					Label:      "is_test",
					Kind:       source.VariableItem,
					InsertText: "is_test",
					Detail:     "default is_test = true",
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

			location := tt.createLocation(tt.files)
			got, err := project.ListCompletionItems(location)
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
