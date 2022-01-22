package source_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/regols/langserver/internal/source"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/ast/location"
)

func TestLookupDefinition(t *testing.T) {
	tests := map[string]struct {
		files        map[string]source.File
		location     *location.Location
		expectResult []*ast.Location
		expectErr    error
	}{
		"in file definition": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

violation[msg] {
	m := "hello"
	msg = m
}`,
				},
			},
			location: &location.Location{
				Row: 5,
				Col: 8,
				Offset: len("package main\n\nviolation[msg] {\n	m := \"hello\"\n	msg = m"),
				Text: []byte("m"),
				File: "src.rego",
			},
			expectResult: []*ast.Location{
				{
					Row: 4,
					Col: 2,
					Offset: len("package main\n\nviolation[msg] {\n	m"),
					Text: []byte("m"),
					File: "src.rego",
				},
			},
		},
		"in file definition in args": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

violation[msg] {
	m := "hello"
	msg = m
}`,
				},
			},
			location: &location.Location{
				Row: 5,
				Col: 2,
				Offset: len("package main\n\nviolation[msg] {\n	m := \"hello\"\n	m"),
				Text: []byte("m"),
				File: "src.rego",
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    11,
					Offset: len("package main\n\nviolation[m"),
					Text:   []byte("msg"),
					File:   "src.rego",
				},
			},
		},
		"": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

test(msg) = test {
	msg == "hello"
	test = "hello"
}`,
				},
			},
			location: &location.Location{
				Row: 5,
				Col: 2,
				Offset: len("package main\n\ntest(msg) = test {\n	msg == \"hello\"\n	t"),
				Text: []byte{'t'},
				File: "src.rego",
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    13,
					Offset: len("package main\n\ntest(msg) = t"),
					Text:   []byte("test"),
					File:   "src.rego",
				},
			},
		},
		"same library but other file definition": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

violation[msg] {
	other_method("hello")
	msg := "hello"
}`,
				},
				"src2.rego": {
					RowText: `package main

other_method(msg) {
	msg == "hello"
}`,
				},
			},
			location: &location.Location{
				Row: 4,
				Col: 5,
				Offset: len("package main\n\nviolation[msg] {\n	othe"),
				Text: []byte("e"),
				File: "src.rego",
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					Offset: len("package main\n\no"),
					Text: []byte("other_method(msg) {\n	msg == \"hello\"\n}"),
					File: "src2.rego",
				},
			},
		},
		"in library definition": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

import data.lib

violation[msg] {
	lib.method("hello")
	msg := "hello"
}`,
				},
				"lib.rego": {
					RowText: `package lib

method(msg) {
	msg == "hello"
}`,
				},
			},
			location: &location.Location{
				Row: 6,
				Col: 6,
				Offset: len("package main\n\nimport data.lib\n\nviolation[msg] {\n	lib.m"),
				Text: []byte("m"),
				File: "src.rego",
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					Offset: len("package lib\n\nm"),
					Text: []byte("method(msg) {\n	msg == \"hello\"\n}"),
					File: "lib.rego",
				},
			},
		},
		"jump to import": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

import data.lib

violation[msg] {
	lib.method("hello")
	msg := "hello"
}`,
				},
			},
			location: &location.Location{
				Row: 6,
				Col: 2,
				Offset: len("package main\n\nimport data.lib\n\nviolation[msg] {\n	l"),
				Text: []byte("l"),
				File: "src.rego",
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    13,
					Offset: len("package main\n\nimport data.l"),
					Text:   []byte("lib"),
					File:   "src.rego",
				},
			},
		},
		"no definition because itself is definition": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

violation[msg] {
	m := "hello"
	msg := m
}`,
				},
			},
			location: &location.Location{
				Row: 4,
				Col: 2,
				Offset: len("package main\n\nviolation[msg] {\n	m"),
				Text: []byte("m"),
				File: "src.rego",
			},
			expectResult: []*ast.Location{},
			expectErr:    nil,
		},
		"with not library but has dot": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

violation[msg] {
	containers[container]
	container.name
}

containers[container] {
	container := input.resource.container
}`,
				},
			},
			location: &location.Location{
				Row: 5,
				Col: 12,
				Offset: len("package main\n\nviolation[msg] {\n	containers[container]\n	container.n}"),
				Text: []byte("n"),
				File: "src.rego",
			},
			expectResult: nil,
		},
		"definition has else": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

authorize = "allow" {
	msg := "allow"
	trace(msg)
} else = "deny" {
	msg := "deny"
	trace(msg)
} else = "out" {
	msg := "out"
	trace(msg)
}`,
				},
			},
			location: &ast.Location{
				Row: 5,
				Col: 8,
				Offset: len("package main\n\nauthorize = \"allow\" {\n	msg := \"allow\"\n	trace(m"),
				Text: []byte("m"),
				File: "src.rego",
			},
			expectResult: []*ast.Location{
				{
					Row: 4,
					Col: 2,
					Offset: len("package main\n\nauthorize = \"allow\" {\n	m"),
					Text: []byte("msg"),
					File: "src.rego",
				},
			},
		},
		"else definition": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

authorize = "allow" {
	msg := "allow"
	trace(msg)
} else = "deny" {
	msg := "deny"
	trace(msg)
} else = "out" {
	msg := "out"
	trace(msg)
}`,
				},
			},
			location: &ast.Location{
				Row: 8,
				Col: 8,
				Offset: len("package main\n\nauthorize = \"allow\" {\n	msg := \"allow\"\n	trace(msg)\n} else = \"deny\" {\n	msg := \"deny\"\n	trace(m"),
				Text: []byte("m"),
				File: "src.rego",
			},
			expectResult: []*ast.Location{
				{
					Row: 7,
					Col: 2,
					Offset: len("package main\n\nauthorize = \"allow\" {\n	msg := \"allow\"\n	trace(msg)\n} else = \"deny\" {\n	m"),
					Text: []byte("msg"),
					File: "src.rego",
				},
			},
		},
		"else of else definition": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

authorize = "allow" {
	msg := "allow"
	trace(msg)
} else = "deny" {
	msg := "deny"
	trace(msg)
} else = "out" {
	msg := "out"
	trace(msg)
}`,
				},
			},
			location: &ast.Location{
				Row: 11,
				Col: 8,
				Offset: len("package main\n\nauthorize = \"allow\" {\n	msg := \"allow\"\n	trace(msg)\n} else = \"deny\" {\n	msg := \"deny\"\n	trace(msg)\n} else = \"out\" {\n	msg := \"out\"\n	trace(m"),
				Text: []byte("m"),
				File: "src.rego",
			},
			expectResult: []*ast.Location{
				{
					Row: 10,
					Col: 2,
					Offset: len("package main\n\nauthorize = \"allow\" {\n	msg := \"allow\"\n	trace(msg)\n} else = \"deny\" {\n	msg := \"deny\"\n	trace(msg)\n} else = \"out\" {\n	m"),
					Text: []byte("msg"),
					File: "src.rego",
				},
			},
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			p, err := source.NewProjectWithFiles(tt.files)
			if err != nil {
				t.Fatalf("failed to create project: %v", err)
			}

			got, err := p.LookupDefinition(tt.location)
			if !errors.Is(err, tt.expectErr) {
				t.Fatalf("LookupDefinition should return error expect %v, but got %v", tt.expectErr, err)
			}

			if diff := cmp.Diff(tt.expectResult, got); diff != "" {
				t.Errorf("LookupDefinition result diff (-expect +got):\n%s", diff)
			}
		})
	}
}
